package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/exabits/gpu-cli/internal/types"
	"github.com/spf13/viper"
)

const (
	defaultBaseURL  = "https://gpu-api.exabits.ai"
	defaultBasePath = "/api/v1"

	maxRetries  = 3
	retryDelay  = 500 * time.Millisecond
)

// authMode controls which authentication header scheme is used.
type authMode int

const (
	// authJWT sends both Authorization: Bearer <access_token> and refresh-token: <refresh_token>.
	// Tokens expire: access_token 30 min, refresh_token 2 hours.
	authJWT authMode = iota

	// authAPIToken sends only Authorization: Bearer <api_token>.
	// API Tokens do not expire and do not require a refresh cycle.
	authAPIToken
)

// Client is the authenticated HTTP client for the Exabits API.
type Client struct {
	baseURL      string // e.g. https://gpu-api.exabits.ai/api/v1
	mode         authMode
	accessToken  string // JWT mode: short-lived access token
	refreshToken string // JWT mode: longer-lived refresh token
	apiToken     string // API Token mode: non-expiring token
	httpClient   *http.Client
}

// NewClient creates a Client from Viper configuration.
//
// Auth precedence (first match wins):
//  1. api_token            → API Token mode (single header, no expiry)
//  2. access_token +
//     refresh_token        → JWT mode (two headers, 30 min / 2 h expiry)
//
// Optional config keys:
//   - api_url: overrides the base URL (default: https://gpu-api.exabits.ai)
func NewClient() (*Client, error) {
	baseURL := viper.GetString("api_url")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	fullBase := baseURL + defaultBasePath

	// Prefer API Token when present.
	if tok := viper.GetString("api_token"); tok != "" {
		return &Client{
			baseURL:    fullBase,
			mode:       authAPIToken,
			apiToken:   tok,
			httpClient: &http.Client{Timeout: 30 * time.Second},
		}, nil
	}

	// Fall back to JWT (access_token + refresh_token).
	accessToken := viper.GetString("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf(
			"no credentials found — set api_token, or both access_token and refresh_token " +
				"in ~/.exabits/config.yaml (or via EXABITS_ env vars). " +
				"Run 'exabits auth login' to obtain tokens.",
		)
	}
	refreshToken := viper.GetString("refresh_token")
	if refreshToken == "" {
		return nil, fmt.Errorf(
			"refresh_token is not set — run 'exabits auth login' or add it to ~/.exabits/config.yaml",
		)
	}

	return &Client{
		baseURL:      fullBase,
		mode:         authJWT,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// isRetryable reports whether err is a transient network error that is safe
// to retry. It matches EOF (server closed the connection before responding —
// common on the first request after a TLS handshake) and any *net.OpError
// (connection reset, broken pipe, etc.). HTTP-level errors (4xx / 5xx) are
// intentionally excluded because they are deterministic.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr *net.OpError
	return errors.As(err, &netErr)
}

// do is the single request path.
//
// It:
//   - marshals body (if non-nil) to JSON once, then builds a fresh
//     http.Request on each attempt so the body reader is always at position 0
//   - retries up to maxRetries times on transient network errors (EOF, etc.)
//   - injects the appropriate auth header(s) based on authMode
//   - on HTTP 4xx/5xx: returns an error using the API's "message" field
//   - on HTTP 2xx: unwraps the Exabits envelope {"status":bool,"message":string,"total":int,"data":...}
//     and reports status:false as an error, then JSON-decodes "data" into dst
//
// total is optional: when non-nil it is set to the envelope's "total" field,
// which list endpoints use to indicate the server-side record count before
// limit/offset are applied.
func (c *Client) do(method, path string, body any, dst any, total *int) error {
	// Marshal the body once; re-use the bytes across retry attempts.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode request body: %w", err)
		}
	}

	var (
		resp    *http.Response
		lastErr error
	)

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		// Build a fresh request each attempt so the body reader starts at 0.
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to build request: %w", err)
		}

		switch c.mode {
		case authAPIToken:
			req.Header.Set("Authorization", "Bearer "+c.apiToken)
		case authJWT:
			req.Header.Set("Authorization", "Bearer "+c.accessToken)
			req.Header.Set("refresh-token", c.refreshToken)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, lastErr = c.httpClient.Do(req)
		if lastErr == nil || !isRetryable(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return fmt.Errorf("request failed: %w", lastErr)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle HTTP-level errors.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &envelope) == nil && envelope.Message != "" {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, envelope.Message)
		}
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if len(respBody) == 0 {
		return nil
	}

	// Unwrap the Exabits API envelope:
	// {"status": bool, "message": string, "total": int, "data": ...}
	var envelope struct {
		Status  bool            `json:"status"`
		Message string          `json:"message"`
		Total   int             `json:"total"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("failed to decode response envelope: %w", err)
	}

	// The API signals application-level errors via status:false even on HTTP 200.
	if !envelope.Status {
		msg := envelope.Message
		if msg == "" {
			msg = "unknown API error (status: false)"
		}
		return fmt.Errorf("API error: %s", msg)
	}

	// Expose the total record count to the caller when requested.
	if total != nil {
		*total = envelope.Total
	}

	// Decode the inner data payload into the caller's destination.
	if dst != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, dst); err != nil {
			return fmt.Errorf("failed to decode response data: %w", err)
		}
	}

	return nil
}

// Get performs a GET request and decodes the response data into dst.
func (c *Client) Get(path string, dst any) error {
	return c.do(http.MethodGet, path, nil, dst, nil)
}

// GetPaged performs a GET request, decodes the response data into dst,
// and sets *total to the envelope's "total" field (server-side record count
// before limit/offset). Use this for list endpoints that support pagination.
func (c *Client) GetPaged(path string, dst any, total *int) error {
	return c.do(http.MethodGet, path, nil, dst, total)
}

// Post performs a POST request with body and decodes the response data into dst.
func (c *Client) Post(path string, body any, dst any) error {
	return c.do(http.MethodPost, path, body, dst, nil)
}

// Put performs a PUT request with body and decodes the response data into dst.
func (c *Client) Put(path string, body any, dst any) error {
	return c.do(http.MethodPut, path, body, dst, nil)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) error {
	return c.do(http.MethodDelete, path, nil, nil, nil)
}

// Login performs the unauthenticated POST /authenticate/login request.
// It is a package-level function (not a Client method) because no auth
// headers are needed — this is the call that obtains the tokens.
//
// baseURL should be the raw host (e.g. "https://gpu-api.exabits.ai");
// the base path and endpoint are appended internally.
func Login(baseURL, username, md5Password string) (*types.LoginData, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	url := baseURL + defaultBasePath + "/authenticate/login"

	reqBytes, err := json.Marshal(types.LoginRequest{
		Username: username,
		Password: md5Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode login request: %w", err)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	var (
		httpResp *http.Response
		lastErr  error
	)

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(retryDelay)
		}

		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBytes))
		if err != nil {
			return nil, fmt.Errorf("failed to build login request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		httpResp, lastErr = httpClient.Do(req)
		if lastErr == nil || !isRetryable(lastErr) {
			break
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("login request failed: %w", lastErr)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read login response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		var envelope struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &envelope) == nil && envelope.Message != "" {
			return nil, fmt.Errorf("login failed (%d): %s", httpResp.StatusCode, envelope.Message)
		}
		return nil, fmt.Errorf("login failed (%d): %s", httpResp.StatusCode, string(body))
	}

	var envelope struct {
		Status  bool            `json:"status"`
		Message string          `json:"message"`
		Data    types.LoginData `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}
	if !envelope.Status {
		msg := envelope.Message
		if msg == "" {
			msg = "login failed (status: false)"
		}
		return nil, fmt.Errorf("login failed: %s", msg)
	}

	return &envelope.Data, nil
}
