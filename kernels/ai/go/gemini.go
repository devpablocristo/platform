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

// --- Google AI Studio (Gemini con API key) ---

// Gemini llama al API de Google AI Studio (generativelanguage.googleapis.com).
// Para Vertex AI (Google Cloud con ADC/bearer), usar VertexAI.
type Gemini struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// GeminiOption configura el provider de Gemini.
type GeminiOption func(*Gemini)

// WithGeminiModel setea el modelo (default: gemini-2.0-flash).
func WithGeminiModel(model string) GeminiOption {
	return func(g *Gemini) {
		if model != "" {
			g.model = model
		}
	}
}

// WithGeminiTimeout setea el timeout HTTP.
func WithGeminiTimeout(d time.Duration) GeminiOption {
	return func(g *Gemini) { g.httpClient.Timeout = d }
}

// NewGemini crea un provider para Google AI Studio.
func NewGemini(apiKey string, opts ...GeminiOption) *Gemini {
	g := &Gemini{
		apiKey:     apiKey,
		model:      "gemini-2.0-flash",
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Chat implementa Provider.
func (g *Gemini) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	contents := buildGeminiContents(req.SystemPrompt, req.Messages)

	body := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": maxTokensOrDefault(req.MaxTokens, 1024),
			"temperature":     0.7,
		},
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		g.model, g.apiKey,
	)

	respBody, err := doPostRaw(ctx, g.httpClient, url, body, nil)
	if err != nil {
		return ChatResponse{}, err
	}

	return parseGeminiResponse(respBody)
}

// SimpleChat envía un mensaje simple.
func (g *Gemini) SimpleChat(ctx context.Context, systemPrompt, userMessage string, maxTokens int) (string, error) {
	resp, err := g.Chat(ctx, ChatRequest{
		SystemPrompt: systemPrompt,
		Messages:     []Message{{Role: "user", Content: userMessage}},
		MaxTokens:    maxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// --- Vertex AI (Google Cloud con bearer token) ---

// VertexAI llama a Vertex AI (aiplatform.googleapis.com) con bearer token.
// El token se obtiene externamente (ADC, metadata server, etc.) y se inyecta via TokenSource.
type VertexAI struct {
	projectID   string
	region      string
	model       string
	tokenSource func(ctx context.Context) (string, error)
	httpClient  *http.Client
}

// VertexAIOption configura el provider de Vertex AI.
type VertexAIOption func(*VertexAI)

// WithVertexModel setea el modelo.
func WithVertexModel(model string) VertexAIOption {
	return func(v *VertexAI) {
		if model != "" {
			v.model = model
		}
	}
}

// WithVertexTimeout setea el timeout HTTP.
func WithVertexTimeout(d time.Duration) VertexAIOption {
	return func(v *VertexAI) { v.httpClient.Timeout = d }
}

// NewVertexAI crea un provider para Vertex AI.
// tokenSource es una función que devuelve un bearer token válido.
func NewVertexAI(projectID, region string, tokenSource func(ctx context.Context) (string, error), opts ...VertexAIOption) *VertexAI {
	v := &VertexAI{
		projectID:   projectID,
		region:      region,
		model:       "gemini-2.5-flash",
		tokenSource: tokenSource,
		httpClient:  &http.Client{Timeout: 5 * time.Minute},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Chat implementa Provider.
func (v *VertexAI) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	token, err := v.tokenSource(ctx)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("vertex token: %w", err)
	}

	contents := buildGeminiContents(req.SystemPrompt, req.Messages)

	body := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"maxOutputTokens": maxTokensOrDefault(req.MaxTokens, 1024),
			"temperature":     0.2,
		},
	}

	url := fmt.Sprintf(
		"https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		v.region, v.projectID, v.region, v.model,
	)

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)

	respBody, err := doPostRaw(ctx, v.httpClient, url, body, headers)
	if err != nil {
		return ChatResponse{}, err
	}

	return parseGeminiResponse(respBody)
}

// SimpleChat envía un mensaje simple.
func (v *VertexAI) SimpleChat(ctx context.Context, systemPrompt, userMessage string, maxTokens int) (string, error) {
	resp, err := v.Chat(ctx, ChatRequest{
		SystemPrompt: systemPrompt,
		Messages:     []Message{{Role: "user", Content: userMessage}},
		MaxTokens:    maxTokens,
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// --- Shared Gemini/Vertex helpers ---

func buildGeminiContents(systemPrompt string, msgs []Message) []map[string]any {
	var contents []map[string]any

	// System prompt como primer mensaje de usuario (Gemini no tiene system role nativo)
	if systemPrompt != "" {
		contents = append(contents, map[string]any{
			"role":  "user",
			"parts": []map[string]any{{"text": "[System] " + systemPrompt}},
		})
		contents = append(contents, map[string]any{
			"role":  "model",
			"parts": []map[string]any{{"text": "Entendido."}},
		})
	}

	for _, m := range msgs {
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]any{{"text": m.Content}},
		})
	}

	return contents
}

func parseGeminiResponse(data []byte) (ChatResponse, error) {
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return ChatResponse{}, fmt.Errorf("decode gemini response: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return ChatResponse{}, fmt.Errorf("empty gemini response")
	}

	text := strings.TrimSpace(resp.Candidates[0].Content.Parts[0].Text)
	return ChatResponse{Text: text}, nil
}

func doPostRaw(ctx context.Context, client *http.Client, url string, body any, extraHeaders http.Header) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, vv := range extraHeaders {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Error("llm_api_error", "status", resp.StatusCode, "body", truncate(string(data), 200))
		return nil, fmt.Errorf("llm api: status %d", resp.StatusCode)
	}

	return data, nil
}
