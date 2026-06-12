package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/exabits-xyz/gpu-cli/internal/types"
)

// chatTimeout bounds non-streaming chat completions, which can take far
// longer than the regular 30 s management-API timeout.
const chatTimeout = 5 * time.Minute

// setAuthHeaders injects the same auth header scheme used by Client.do.
func (c *Client) setAuthHeaders(req *http.Request) {
	switch c.mode {
	case authAPIToken:
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	case authJWT:
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("refresh-token", c.refreshToken)
	}
}

// chatErrorFromBody extracts a useful message from a chat endpoint error.
// The chat route may return either the Exabits envelope ({"message": ...})
// or an OpenAI-style error object ({"error": {"message": ...}}).
func chatErrorFromBody(statusCode int, body []byte) error {
	var envelope struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil {
		switch {
		case envelope.Error.Message != "":
			return fmt.Errorf("API error %d: %s", statusCode, envelope.Error.Message)
		case envelope.Message != "":
			return fmt.Errorf("API error %d: %s", statusCode, envelope.Message)
		}
	}
	return fmt.Errorf("API error %d: %s", statusCode, string(body))
}

// ChatCompletion performs a non-streaming POST /chat/completions request.
// The response is decoded directly — this endpoint does not use the
// {status, message, data} envelope.
func (c *Client) ChatCompletion(req types.ChatCompletionRequest) (*types.ChatCompletionResponse, error) {
	req.Stream = false

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode chat request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build chat request: %w", err)
	}
	c.setAuthHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: chatTimeout}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read chat response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, chatErrorFromBody(resp.StatusCode, respBody)
	}

	var completion types.ChatCompletionResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return nil, fmt.Errorf("failed to decode chat response: %w", err)
	}
	return &completion, nil
}

// ChatCompletionStream performs a streaming POST /chat/completions request
// and invokes onChunk for every SSE "data:" event until "[DONE]" or EOF.
// No overall timeout is applied — generation length is model-dependent.
func (c *Client) ChatCompletionStream(req types.ChatCompletionRequest, onChunk func(types.ChatCompletionChunk) error) error {
	req.Stream = true

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode chat request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to build chat request: %w", err)
	}
	c.setAuthHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	httpClient := &http.Client{} // no Timeout: streams run until generation completes
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("chat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return chatErrorFromBody(resp.StatusCode, respBody)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Allow large SSE events (long content deltas, reasoning blocks).
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}

		var chunk types.ChatCompletionChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("failed to decode stream chunk: %w", err)
		}
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read failed: %w", err)
	}
	return nil
}
