package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaChat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["model"] != "qwen2.5:3b" {
			t.Errorf("unexpected model: %v", body["model"])
		}
		if body["stream"] != false {
			t.Error("expected stream=false")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "Hola, soy Ollama local.",
			},
		})
	}))
	defer srv.Close()

	provider := NewOllama(srv.URL)

	resp, err := provider.Chat(context.Background(), ChatRequest{
		SystemPrompt: "Sos un asistente.",
		Messages:     []Message{{Role: "user", Content: "Hola"}},
		MaxTokens:    100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "Hola, soy Ollama local." {
		t.Fatalf("unexpected text: %q", resp.Text)
	}
}

func TestOllamaChatWithToolCalls(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		// Verificar que los tools se envían
		tools, ok := body["tools"].([]any)
		if !ok || len(tools) != 1 {
			t.Errorf("expected 1 tool, got %v", body["tools"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]any{
				"role":    "assistant",
				"content": "",
				"tool_calls": []map[string]any{
					{
						"function": map[string]any{
							"name":      "get_overview",
							"arguments": map[string]any{"scope": "all"},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	provider := NewOllama(srv.URL)

	resp, err := provider.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Dame un resumen"}},
		Tools: []Tool{{
			Name:        "get_overview",
			Description: "Obtiene resumen",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_overview" {
		t.Fatalf("unexpected tool name: %s", resp.ToolCalls[0].Name)
	}
}

func TestOllamaFactoryCreation(t *testing.T) {
	t.Parallel()

	// Con baseURL vacío, usa default
	p := NewProvider("ollama", "", "qwen2.5:3b")
	if _, ok := p.(*Ollama); !ok {
		t.Fatal("expected Ollama provider")
	}

	// Con baseURL explícito
	p = NewProvider("ollama", "http://custom:11434", "llama3.2:1b")
	ollama, ok := p.(*Ollama)
	if !ok {
		t.Fatal("expected Ollama provider")
	}
	if ollama.baseURL != "http://custom:11434" {
		t.Fatalf("unexpected baseURL: %s", ollama.baseURL)
	}
	if ollama.model != "llama3.2:1b" {
		t.Fatalf("unexpected model: %s", ollama.model)
	}
}
