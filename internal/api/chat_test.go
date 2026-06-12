package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/viper"
)

func newChatTestClient(t *testing.T, srvURL string) *Client {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("api_url", srvURL)
	viper.Set("api_token", "test-token")

	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func TestChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/chat/completions" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}

		var req types.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Stream {
			t.Errorf("stream = true, want false")
		}
		if req.Model != "MiniMaxAI/MiniMax-M2.7" || len(req.Messages) != 2 {
			t.Errorf("request = %+v", req)
		}
		if req.Temperature == nil || *req.Temperature != 0.2 {
			t.Errorf("temperature = %v, want 0.2", req.Temperature)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "1fddec966dacea7a7b9ec380684e201f",
			"object": "chat.completion",
			"created": 1780986485,
			"model": "minimax/minimax-m2.7",
			"choices": [{
				"index": 0,
				"finish_reason": "stop",
				"message": {
					"role": "assistant",
					"content": "Hello! How can I help you today?",
					"name": "MiniMax AI",
					"reasoning_content": "The user greeted me."
				}
			}],
			"usage": {
				"prompt_tokens": 48,
				"completion_tokens": 39,
				"total_tokens": 87,
				"completion_tokens_details": {"reasoning_tokens": 30}
			}
		}`))
	}))
	defer srv.Close()

	client := newChatTestClient(t, srv.URL)

	temp := 0.2
	resp, err := client.ChatCompletion(types.ChatCompletionRequest{
		Model: "MiniMaxAI/MiniMax-M2.7",
		Messages: []types.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "hello"},
		},
		Temperature: &temp,
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "Hello! How can I help you today?" {
		t.Fatalf("choices = %+v", resp.Choices)
	}
	if resp.Choices[0].FinishReason != "stop" || resp.Usage == nil || resp.Usage.TotalTokens != 87 {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Usage.CompletionTokensDetails == nil || resp.Usage.CompletionTokensDetails.ReasoningTokens != 30 {
		t.Fatalf("usage details = %+v", resp.Usage.CompletionTokensDetails)
	}
}

func TestChatCompletionErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "model not found"}}`))
	}))
	defer srv.Close()

	client := newChatTestClient(t, srv.URL)

	_, err := client.ChatCompletion(types.ChatCompletionRequest{
		Model:    "nope",
		Messages: []types.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err == nil || !strings.Contains(err.Error(), "model not found") {
		t.Fatalf("err = %v, want OpenAI-style error message surfaced", err)
	}
}

func TestChatCompletionStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !req.Stream {
			t.Errorf("stream = false, want true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"id":"abc","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"thinking"},"finish_reason":null}]}` + "\n\n" +
				`data: {"id":"abc","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n" +
				`data: {"id":"abc","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":"stop"}]}` + "\n\n" +
				`data: {"id":"abc","object":"chat.completion.chunk","choices":[],"usage":{"prompt_tokens":48,"completion_tokens":37,"total_tokens":85}}` + "\n\n" +
				"data: [DONE]\n",
		))
	}))
	defer srv.Close()

	client := newChatTestClient(t, srv.URL)

	var content strings.Builder
	var usage *types.ChatUsage
	chunks := 0

	err := client.ChatCompletionStream(types.ChatCompletionRequest{
		Model:    "MiniMaxAI/MiniMax-M2.7",
		Messages: []types.ChatMessage{{Role: "user", Content: "hello"}},
	}, func(chunk types.ChatCompletionChunk) error {
		chunks++
		for _, c := range chunk.Choices {
			content.WriteString(c.Delta.Content)
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	if chunks != 4 {
		t.Fatalf("chunks = %d, want 4 (DONE must not be delivered)", chunks)
	}
	if content.String() != "Hello there" {
		t.Fatalf("content = %q, want %q", content.String(), "Hello there")
	}
	if usage == nil || usage.TotalTokens != 85 {
		t.Fatalf("usage = %+v", usage)
	}
}
