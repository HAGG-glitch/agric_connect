package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agriconnect-ai/internal/ai"
)

func TestAIClient_ChatRespectsRequestModel(t *testing.T) {
	var lastRequest struct {
		Model string `json:"model"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&lastRequest); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	client := ai.NewClient("test-key", srv.URL, "llama-3.1-8b-instant", 10)

	t.Run("uses request model when set", func(t *testing.T) {
		req := ai.ChatRequest{
			Model:    "meta-llama/llama-4-scout-17b-16e-instruct",
			Messages: []ai.Message{{Role: "user", Content: "hello"}},
		}

		_, err := client.Chat(context.Background(), req)
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}

		if lastRequest.Model != "meta-llama/llama-4-scout-17b-16e-instruct" {
			t.Errorf("expected vision model, got %q", lastRequest.Model)
		}
	})

	t.Run("falls back to client default when request model is empty", func(t *testing.T) {
		req := ai.ChatRequest{
			Model:    "",
			Messages: []ai.Message{{Role: "user", Content: "hello"}},
		}

		_, err := client.Chat(context.Background(), req)
		if err != nil {
			t.Fatalf("Chat failed: %v", err)
		}

		if lastRequest.Model != "llama-3.1-8b-instant" {
			t.Errorf("expected chat model default, got %q", lastRequest.Model)
		}
	})
}

func TestAIClient_ModelNotOverridden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	client := ai.NewClient("test-key", srv.URL, "llama-3.1-8b-instant", 10)

	req := ai.ChatRequest{
		Model:    "meta-llama/llama-4-scout-17b-16e-instruct",
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	}

	_, err := client.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
}
