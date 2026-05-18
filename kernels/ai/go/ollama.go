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

// --- Ollama Provider (local LLM via /api/chat) ---

// Ollama llama al API de Ollama (OpenAI-compatible) para modelos locales.
type Ollama struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// OllamaOption configura el provider de Ollama.
type OllamaOption func(*Ollama)

// WithOllamaModel setea el modelo (default: qwen2.5:3b).
func WithOllamaModel(model string) OllamaOption {
	return func(o *Ollama) {
		if model != "" {
			o.model = model
		}
	}
}

// WithOllamaTimeout setea el timeout HTTP (default: 120s, modelos locales son lentos).
func WithOllamaTimeout(d time.Duration) OllamaOption {
	return func(o *Ollama) { o.httpClient.Timeout = d }
}

// NewOllama crea un provider para Ollama local.
func NewOllama(baseURL string, opts ...OllamaOption) *Ollama {
	o := &Ollama{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		model:      "qwen2.5:3b",
		httpClient: &http.Client{Timeout: 120 * time.Second},
	}
	if o.baseURL == "" {
		o.baseURL = "http://localhost:11434"
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Chat implementa Provider usando /api/chat de Ollama.
func (o *Ollama) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	messages := buildOllamaMessages(req.SystemPrompt, req.Messages)

	body := map[string]any{
		"model":    o.model,
		"messages": messages,
		"stream":   false,
	}

	options := map[string]any{}
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	}
	if len(options) > 0 {
		body["options"] = options
	}

	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  t.Parameters,
				},
			})
		}
		body["tools"] = tools
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("ollama http: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("ollama_api_error", "status", resp.StatusCode, "body", truncate(string(data), 200))
		return ChatResponse{}, fmt.Errorf("ollama api: status %d", resp.StatusCode)
	}

	return parseOllamaResponse(data)
}

func buildOllamaMessages(systemPrompt string, msgs []Message) []map[string]any {
	out := make([]map[string]any, 0, len(msgs)+1)

	if systemPrompt != "" {
		out = append(out, map[string]any{
			"role":    "system",
			"content": systemPrompt,
		})
	}

	for _, m := range msgs {
		msg := map[string]any{
			"role":    m.Role,
			"content": m.Content,
		}
		out = append(out, msg)
	}

	return out
}

func parseOllamaResponse(data []byte) (ChatResponse, error) {
	var resp struct {
		Message struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Function struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ChatResponse{}, fmt.Errorf("decode ollama response: %w", err)
	}

	var result ChatResponse
	result.Text = resp.Message.Content

	for i, tc := range resp.Message.ToolCalls {
		args := tc.Function.Arguments
		if len(args) == 0 {
			args = json.RawMessage(`{}`)
		}
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:   fmt.Sprintf("ollama_tc_%d", i),
			Name: tc.Function.Name,
			Args: args,
		})
	}

	return result, nil
}
