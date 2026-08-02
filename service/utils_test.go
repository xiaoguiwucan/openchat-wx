package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/openai/openai-go/v3"
)

func TestStreamChatCompletionFallsBackToNonStreaming(t *testing.T) {
	t.Setenv("OPENCHAT_DOCKER", "0")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if stream, _ := payload["stream"].(bool); stream {
			http.Error(w, `{"error":{"message":"streaming unsupported"}}`, http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"test","object":"chat.completion","created":1,"model":"test-model","choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"OK"}}]}`))
	}))
	defer server.Close()

	client := newOpenAIClient("test-key", server.URL+"/v1")
	req := openai.ChatCompletionNewParams{
		Model:    "test-model",
		Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage("ping")},
	}
	message, err := streamChatCompletionMessage(context.Background(), &client, req)
	if err != nil {
		t.Fatalf("stream fallback failed: %v", err)
	}
	if message.Content != "OK" {
		t.Fatalf("content = %q, want OK", message.Content)
	}
	if requests.Load() != 2 {
		t.Fatalf("request count = %d, want 2", requests.Load())
	}
}
