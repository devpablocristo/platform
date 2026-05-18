package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// PromptOption configura un prompt one-shot.
type PromptOption func(*promptConfig)

type promptConfig struct {
	maxTokens        int
	temperature      float64
	jsonMode         bool
	frequencyPenalty float64
	maxRetries       int
	retryDelay       time.Duration
}

// WithMaxTokens setea los tokens máximos de output.
func WithMaxTokens(n int) PromptOption {
	return func(c *promptConfig) { c.maxTokens = n }
}

// WithTemperature setea la temperatura.
func WithTemperature(t float64) PromptOption {
	return func(c *promptConfig) { c.temperature = t }
}

// WithJSONMode fuerza que el LLM devuelva JSON válido.
func WithJSONMode() PromptOption {
	return func(c *promptConfig) { c.jsonMode = true }
}

// WithFrequencyPenalty setea la penalización de frecuencia.
func WithFrequencyPenalty(p float64) PromptOption {
	return func(c *promptConfig) { c.frequencyPenalty = p }
}

// WithRetries setea el número de reintentos con backoff.
func WithRetries(n int, delay time.Duration) PromptOption {
	return func(c *promptConfig) {
		c.maxRetries = n
		c.retryDelay = delay
	}
}

// RawPrompt envía un prompt y espera JSON válido. Reintenta si falla.
// Usa VertexAI o cualquier Provider — stripea code fences y valida JSON.
func RawPrompt(ctx context.Context, provider Provider, prompt string, opts ...PromptOption) ([]byte, error) {
	cfg := &promptConfig{
		maxTokens:   65536,
		temperature: 0.2,
		maxRetries:  3,
		retryDelay:  5 * time.Second,
		jsonMode:    true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error
	for attempt := 0; attempt <= cfg.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(cfg.retryDelay * time.Duration(attempt))
		}

		resp, err := provider.Chat(ctx, ChatRequest{
			Messages:  []Message{{Role: "user", Content: prompt}},
			MaxTokens: cfg.maxTokens,
		})
		if err != nil {
			lastErr = err
			slog.Debug("raw_prompt_retry", "attempt", attempt+1, "error", err)
			continue
		}

		text := stripCodeFences(strings.TrimSpace(resp.Text))

		if cfg.jsonMode && !json.Valid([]byte(text)) {
			lastErr = fmt.Errorf("LLM returned invalid JSON (len=%d)", len(text))
			slog.Debug("raw_prompt_invalid_json", "attempt", attempt+1, "len", len(text))
			continue
		}

		return []byte(text), nil
	}

	return nil, fmt.Errorf("after %d attempts: %w", cfg.maxRetries+1, lastErr)
}

// TextPrompt envía un prompt y espera texto libre. Sanitiza markdown.
func TextPrompt(ctx context.Context, provider Provider, prompt string, opts ...PromptOption) (string, error) {
	cfg := &promptConfig{
		maxTokens:   4096,
		temperature: 0.7,
		maxRetries:  3,
		retryDelay:  5 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	var lastErr error
	for attempt := 0; attempt <= cfg.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(cfg.retryDelay * time.Duration(attempt))
		}

		resp, err := provider.Chat(ctx, ChatRequest{
			Messages:  []Message{{Role: "user", Content: prompt}},
			MaxTokens: cfg.maxTokens,
		})
		if err != nil {
			lastErr = err
			continue
		}

		return SanitizeMarkdown(resp.Text), nil
	}

	return "", fmt.Errorf("after %d attempts: %w", cfg.maxRetries+1, lastErr)
}

// SanitizeMarkdown limita líneas largas y trunca output excesivo.
func SanitizeMarkdown(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		if len(line) > 1000 {
			line = line[:1000] + " ..."
		}
		out = append(out, line)
	}
	result := strings.Join(out, "\n")
	const maxSize = 25000
	if len(result) > maxSize {
		result = result[:maxSize] + "\n\n*(truncated — output exceeded size limit)*"
	}
	return result
}

// stripCodeFences remueve ```json ... ``` que los LLMs a veces agregan.
func stripCodeFences(text string) string {
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if idx := strings.LastIndex(text, "```"); idx >= 0 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}
	return text
}
