package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/exabits-xyz/gpu-cli/internal/securestore"
	"github.com/spf13/viper"
)

func TestNewClientUsesEncryptedAPIToken(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)

	encrypted, err := securestore.EncryptToken("secret-token")
	if err != nil {
		t.Fatalf("EncryptToken: %v", err)
	}
	viper.Set("api_url", "https://example.test")
	viper.Set("api_token_encrypted", encrypted)

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.mode != authAPIToken {
		t.Fatalf("mode = %v, want authAPIToken", client.mode)
	}
	if client.apiToken != "secret-token" {
		t.Fatalf("apiToken = %q, want %q", client.apiToken, "secret-token")
	}
}

func TestResolveBaseURLTrimsTrailingSlash(t *testing.T) {
	got := ResolveBaseURL("https://gpu-api.exascalelabs.ai/")
	if got != DefaultBaseURL() {
		t.Fatalf("ResolveBaseURL = %q, want %q", got, DefaultBaseURL())
	}
}

func TestDeviceAuthFlowResponses(t *testing.T) {
	var validateCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != deviceAuthClientID || password != deviceAuthClientSecret {
			t.Fatalf("missing device auth Basic credentials")
		}
		switch r.URL.Path {
		case "/api/v1/authenticate/auth-access-code":
			writeNestedEnvelope(t, w, true, map[string]any{
				"state":      "state-123",
				"expires_in": 900,
			})
		case "/api/v1/authenticate/auth-access-code/state-123/validate":
			validateCalls++
			if validateCalls == 1 {
				writeNestedEnvelope(t, w, true, map[string]any{})
				return
			}
			writeNestedEnvelope(t, w, true, map[string]any{"token": "device-token"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	start, err := RequestDeviceAuth(srv.URL)
	if err != nil {
		t.Fatalf("RequestDeviceAuth: %v", err)
	}
	if start.State != "state-123" || start.ExpiresIn != 900 {
		t.Fatalf("start = %+v", start)
	}

	if token, ok, err := ValidateDeviceAuth(srv.URL, "state-123"); err != nil || ok || token != nil {
		t.Fatalf("first validate = token:%v ok:%v err:%v, want pending", token, ok, err)
	}

	token, ok, err := ValidateDeviceAuth(srv.URL, "state-123")
	if err != nil {
		t.Fatalf("second ValidateDeviceAuth: %v", err)
	}
	if !ok || token == nil || token.Token != "device-token" {
		t.Fatalf("second validate = token:%+v ok:%v, want device-token", token, ok)
	}
}

func writeNestedEnvelope(t *testing.T, w http.ResponseWriter, status bool, data any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{
			"status": status,
			"data":   data,
		},
	}); err != nil {
		t.Fatalf("write envelope: %v", err)
	}
}
