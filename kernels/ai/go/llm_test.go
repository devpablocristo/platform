package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicChat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/messages" {
			t.Errorf("expected /v1/messages, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("missing api key")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("missing anthropic-version")
		}

		// Verificar que el body tiene system prompt y messages
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["system"] != "test system" {
			t.Errorf("expected system=test system, got %v", body["system"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Hola, soy Nexus."},
			},
		})
	}))
	defer srv.Close()

	provider := NewAnthropic("test-key", WithAnthropicBaseURL(srv.URL))
	resp, err := provider.Chat(context.Background(), ChatRequest{
		SystemPrompt: "test system",
		Messages:     []Message{{Role: "user", Content: "Hola"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Text != "Hola, soy Nexus." {
		t.Errorf("expected 'Hola, soy Nexus.', got %q", resp.Text)
	}
}

func TestAnthropicToolCalls(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		// Verificar que tools fueron enviados
		tools, ok := body["tools"].([]any)
		if !ok || len(tools) == 0 {
			t.Error("expected tools in request")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Voy a verificar."},
				{
					"type":  "tool_use",
					"id":    "tc_123",
					"name":  "check_approvals",
					"input": map[string]any{},
				},
			},
		})
	}))
	defer srv.Close()

	provider := NewAnthropic("test-key", WithAnthropicBaseURL(srv.URL))
	resp, err := provider.Chat(context.Background(), ChatRequest{
		SystemPrompt: "test",
		Messages:     []Message{{Role: "user", Content: "Qué hay pendiente?"}},
		Tools: []Tool{{
			Name:        "check_approvals",
			Description: "Lista aprobaciones pendientes",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Text != "Voy a verificar." {
		t.Errorf("expected text, got %q", resp.Text)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "check_approvals" {
		t.Errorf("expected check_approvals, got %s", resp.ToolCalls[0].Name)
	}
}

func TestAnthropicSimpleChat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Resumen: acción de bajo riesgo."},
			},
		})
	}))
	defer srv.Close()

	provider := NewAnthropic("test-key", WithAnthropicBaseURL(srv.URL))
	text, err := provider.SimpleChat(context.Background(), "resumir", "acción X", 300)
	if err != nil {
		t.Fatalf("simple chat: %v", err)
	}
	if text != "Resumen: acción de bajo riesgo." {
		t.Errorf("unexpected: %q", text)
	}
}

func TestAnthropicAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	provider := NewAnthropic("test-key", WithAnthropicBaseURL(srv.URL))
	_, err := provider.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if err == nil {
		t.Fatal("expected error on 429")
	}
}

func TestEchoProvider(t *testing.T) {
	t.Parallel()

	echo := NewEcho()
	resp, err := echo.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hola"}},
	})
	if err != nil {
		t.Fatalf("echo: %v", err)
	}
	if resp.Text == "" {
		t.Error("expected non-empty reply")
	}
	if len(resp.ToolCalls) != 0 {
		t.Error("echo should not return tool calls")
	}
}

func TestNewProviderAnthropic(t *testing.T) {
	t.Parallel()
	p := NewProvider("anthropic", "key-123", "")
	if _, ok := p.(*Anthropic); !ok {
		t.Error("expected Anthropic provider")
	}
}

func TestNewProviderNoKey(t *testing.T) {
	t.Parallel()
	p := NewProvider("anthropic", "", "")
	if _, ok := p.(*Echo); !ok {
		t.Error("expected Echo fallback when no API key")
	}
}

func TestNewProviderUnknown(t *testing.T) {
	t.Parallel()
	p := NewProvider("unknown", "", "")
	if _, ok := p.(*Echo); !ok {
		t.Error("expected Echo for unknown provider")
	}
}

func TestBuildAnthropicMessagesToolResult(t *testing.T) {
	t.Parallel()

	msgs := buildAnthropicMessages([]Message{
		{Role: "user", Content: "hola"},
		{Role: "assistant", Content: "checking", ToolCalls: []ToolCall{{ID: "tc1", Name: "check", Args: json.RawMessage(`{}`)}}},
		{Role: "tool", Content: `{"result":"ok"}`, ToolCallID: "tc1"},
	})
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
	// Tool result should have role=user with tool_result content
	if msgs[2]["role"] != "user" {
		t.Errorf("tool result role should be user, got %v", msgs[2]["role"])
	}
}
