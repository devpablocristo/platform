// Package ai proporciona primitivas agnósticas para interactuar con LLMs desde Go.
// Soporta Anthropic Messages API (con tool_use), Google Gemini, y un provider echo para dev.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// --- Interfaces ---

// Provider abstracción de un modelo de lenguaje.
type Provider interface {
	// Chat envía mensajes al LLM y devuelve respuesta (texto + tool calls opcionales).
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// ChatRequest petición al LLM.
type ChatRequest struct {
	SystemPrompt string    `json:"system"`
	Messages     []Message `json:"messages"`
	Tools        []Tool    `json:"tools,omitempty"`
	MaxTokens    int       `json:"max_tokens,omitempty"`
}

// Message mensaje en el hilo del LLM.
type Message struct {
	Role       string     `json:"role"` // user, assistant, tool
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall invocación de tool por el LLM.
type ToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"arguments"`
}

// Tool declaración de un tool para el LLM.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"input_schema"`
}

// ChatResponse respuesta del LLM.
type ChatResponse struct {
	Text      string     `json:"text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// --- Anthropic Provider ---

// Anthropic llama al API de Anthropic Messages (v1/messages) con soporte de tool_use.
type Anthropic struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// AnthropicOption configura el provider de Anthropic.
type AnthropicOption func(*Anthropic)

// WithAnthropicModel setea el modelo (default: claude-sonnet-4-20250514).
func WithAnthropicModel(model string) AnthropicOption {
	return func(a *Anthropic) {
		if model != "" {
			a.model = model
		}
	}
}

// WithAnthropicTimeout setea el timeout HTTP.
func WithAnthropicTimeout(d time.Duration) AnthropicOption {
	return func(a *Anthropic) {
		a.httpClient.Timeout = d
	}
}

// WithAnthropicBaseURL setea la base URL (default: https://api.anthropic.com).
func WithAnthropicBaseURL(url string) AnthropicOption {
	return func(a *Anthropic) {
		if url != "" {
			a.baseURL = strings.TrimRight(url, "/")
		}
	}
}

// NewAnthropic crea un provider para Anthropic Messages API.
func NewAnthropic(apiKey string, opts ...AnthropicOption) *Anthropic {
	a := &Anthropic{
		apiKey:     apiKey,
		model:      "claude-sonnet-4-20250514",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.anthropic.com",
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Chat implementa Provider.
func (a *Anthropic) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	messages := buildAnthropicMessages(req.Messages)

	body := map[string]any{
		"model":      a.model,
		"max_tokens": maxTokensOrDefault(req.MaxTokens, 1024),
		"system":     req.SystemPrompt,
		"messages":   messages,
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"name":         t.Name,
				"description":  t.Description,
				"input_schema": t.Parameters,
			})
		}
		body["tools"] = tools
	}

	respBody, err := a.doPost(ctx, "/v1/messages", body)
	if err != nil {
		return ChatResponse{}, err
	}

	return parseAnthropicResponse(respBody)
}

// SimpleChat envía un mensaje simple sin tools. Convenience para summarize y otros one-shot.
func (a *Anthropic) SimpleChat(ctx context.Context, systemPrompt, userMessage string, maxTokens int) (string, error) {
	resp, err := a.Chat(ctx, ChatRequest{
		SystemPrompt: systemPrompt,
		Messages:     []Message{{Role: "user", Content: userMessage}},
		MaxTokens:    maxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

func (a *Anthropic) doPost(ctx context.Context, path string, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic http: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("anthropic_api_error", "status", resp.StatusCode, "body", truncate(string(data), 200))
		return nil, fmt.Errorf("anthropic api: status %d", resp.StatusCode)
	}

	return data, nil
}

func buildAnthropicMessages(msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		msg := map[string]any{"role": m.Role}

		if len(m.ToolCalls) > 0 {
			// Asistente con tool_use
			content := make([]map[string]any, 0)
			if m.Content != "" {
				content = append(content, map[string]any{"type": "text", "text": m.Content})
			}
			for _, tc := range m.ToolCalls {
				var args any
				if err := json.Unmarshal(tc.Args, &args); err != nil {
					args = map[string]any{}
				}
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": args,
				})
			}
			msg["content"] = content
		} else if m.ToolCallID != "" {
			// Resultado de tool
			msg["role"] = "user"
			msg["content"] = []map[string]any{{
				"type":        "tool_result",
				"tool_use_id": m.ToolCallID,
				"content":     m.Content,
			}}
		} else {
			msg["content"] = m.Content
		}

		out = append(out, msg)
	}
	return out
}

func parseAnthropicResponse(data []byte) (ChatResponse, error) {
	var resp struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}

	var result ChatResponse
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			result.Text += block.Text
		case "tool_use":
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: block.Input,
			})
		}
	}
	return result, nil
}

// --- Echo Provider (dev/test/fallback) ---

// Echo devuelve respuestas deterministas sin LLM.
type Echo struct{}

// NewEcho crea un provider echo para desarrollo y fallback.
func NewEcho() *Echo { return &Echo{} }

// Chat implementa Provider con respuestas deterministas.
func (e *Echo) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	var lastUser string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" && req.Messages[i].Content != "" {
			lastUser = req.Messages[i].Content
			break
		}
	}
	return ChatResponse{Text: echoReply(lastUser)}, nil
}

// SimpleChat implementa la interfaz simple.
func (e *Echo) SimpleChat(_ context.Context, _, userMessage string, _ int) (string, error) {
	return echoReply(userMessage), nil
}

func echoReply(input string) string {
	lower := strings.ToLower(input)
	switch {
	case strings.Contains(lower, "resumen") || strings.Contains(lower, "summary") || strings.Contains(lower, "summar"):
		return "Resumen no disponible en modo echo."
	default:
		return "Recibido (modo echo). LLM no configurado."
	}
}

// --- Factory ---

// NewProvider crea el provider adecuado según configuración.
// Providers: "anthropic"/"claude", "gemini"/"google_ai_studio", "ollama", "echo" (default).
// Para Vertex AI usar NewVertexAI directamente (requiere tokenSource, no API key).
// Para Ollama, apiKey se usa como baseURL (default: http://localhost:11434).
func NewProvider(provider, apiKey, model string) Provider {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "anthropic", "claude":
		if apiKey == "" {
			slog.Warn("ai_provider_no_api_key", "provider", provider)
			return NewEcho()
		}
		opts := []AnthropicOption{}
		if model != "" {
			opts = append(opts, WithAnthropicModel(model))
		}
		return NewAnthropic(apiKey, opts...)
	case "gemini", "google_ai_studio", "google":
		if apiKey == "" {
			slog.Warn("ai_provider_no_api_key", "provider", provider)
			return NewEcho()
		}
		opts := []GeminiOption{}
		if model != "" {
			opts = append(opts, WithGeminiModel(model))
		}
		return NewGemini(apiKey, opts...)
	case "ollama":
		baseURL := strings.TrimSpace(apiKey) // apiKey se reutiliza como baseURL para Ollama
		opts := []OllamaOption{}
		if model != "" {
			opts = append(opts, WithOllamaModel(model))
		}
		return NewOllama(baseURL, opts...)
	default:
		return NewEcho()
	}
}

// --- Helpers ---

func maxTokensOrDefault(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
