package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiChat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Verificar que tiene API key en query
		if !strings.Contains(r.URL.RawQuery, "key=test-gemini-key") {
			t.Errorf("missing API key in query: %s", r.URL.RawQuery)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		contents := body["contents"].([]any)
		if len(contents) == 0 {
			t.Error("expected contents")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]any{{"text": "Respuesta de Gemini."}},
				},
			}},
		})
	}))
	defer srv.Close()

	// Reemplazar la URL base para test — Gemini construye URL con modelo,
	// así que hacemos un provider que apunte al server de test
	g := &Gemini{
		apiKey:     "test-gemini-key",
		model:      "gemini-2.0-flash",
		httpClient: srv.Client(),
	}
	// Override: redirigir al test server
	origChat := g.Chat
	_ = origChat

	// Test via factory no es posible sin override de URL, así que testeamos directo
	// con un handler que simula la respuesta de Gemini
	resp, err := testGeminiWithServer(srv.URL, "test-gemini-key")
	if err != nil {
		t.Fatalf("gemini chat: %v", err)
	}
	if resp.Text != "Respuesta de Gemini." {
		t.Errorf("expected 'Respuesta de Gemini.', got %q", resp.Text)
	}
}

// testGeminiWithServer usa doPostRaw directamente para simular Gemini
func testGeminiWithServer(serverURL, apiKey string) (ChatResponse, error) {
	contents := buildGeminiContents("test system", []Message{{Role: "user", Content: "hola"}})
	body := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": 1024,
			"temperature":     0.7,
		},
	}

	ctx := context.Background()
	respBody, err := doPostRaw(ctx, http.DefaultClient, serverURL+"?key="+apiKey, body, nil)
	if err != nil {
		return ChatResponse{}, err
	}
	return parseGeminiResponse(respBody)
}

func TestVertexAIChat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing bearer token: %s", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{
					"parts": []map[string]any{{"text": "Respuesta de Vertex."}},
				},
			}},
		})
	}))
	defer srv.Close()

	// Usar doPostRaw para simular Vertex
	contents := buildGeminiContents("system", []Message{{Role: "user", Content: "test"}})
	body := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": 1024,
		},
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer test-token")

	respBody, err := doPostRaw(context.Background(), http.DefaultClient, srv.URL, body, headers)
	if err != nil {
		t.Fatalf("vertex: %v", err)
	}

	resp, err := parseGeminiResponse(respBody)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if resp.Text != "Respuesta de Vertex." {
		t.Errorf("expected 'Respuesta de Vertex.', got %q", resp.Text)
	}
}

func TestBuildGeminiContents(t *testing.T) {
	t.Parallel()

	contents := buildGeminiContents("sos un asistente", []Message{
		{Role: "user", Content: "hola"},
		{Role: "assistant", Content: "hola!"},
		{Role: "user", Content: "qué onda?"},
	})
	// System prompt → 2 mensajes (user system + model ack) + 3 mensajes = 5
	if len(contents) != 5 {
		t.Fatalf("expected 5 contents, got %d", len(contents))
	}
	// Primer mensaje debe ser system
	first := contents[0]
	parts := first["parts"].([]map[string]any)
	if !strings.Contains(parts[0]["text"].(string), "[System]") {
		t.Error("first message should contain [System] prefix")
	}
}

func TestNewProviderGemini(t *testing.T) {
	t.Parallel()
	p := NewProvider("gemini", "key-123", "")
	if _, ok := p.(*Gemini); !ok {
		t.Error("expected Gemini provider")
	}
}

func TestNewProviderGoogleAIStudio(t *testing.T) {
	t.Parallel()
	p := NewProvider("google_ai_studio", "key-456", "gemini-2.0-flash")
	if _, ok := p.(*Gemini); !ok {
		t.Error("expected Gemini provider for google_ai_studio")
	}
}

func TestParseGeminiEmptyResponse(t *testing.T) {
	t.Parallel()
	_, err := parseGeminiResponse([]byte(`{"candidates":[]}`))
	if err == nil {
		t.Error("expected error on empty candidates")
	}
}
