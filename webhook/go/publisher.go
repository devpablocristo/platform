package webhook

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

// Publisher envía HTTP POST a un conjunto estático de URLs agrupadas por tipo
// de evento. Entrega best-effort: loguea errores pero no reintenta.
type Publisher struct {
	client *http.Client
	token  string
	routes map[string][]string
}

// NewPublisher crea un publisher con un token de autenticación y un mapa
// de tipo de evento → URLs destino.
func NewPublisher(token string, routes map[string][]string) *Publisher {
	normalized := make(map[string][]string, len(routes))
	for event, urls := range routes {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		normalized[event] = dedup(urls)
	}
	return &Publisher{
		client: &http.Client{Timeout: 5 * time.Second},
		token:  strings.TrimSpace(token),
		routes: normalized,
	}
}

// Publish serializa payload como JSON y lo envía a todas las URLs registradas
// para el tipo de evento. Retorna el primer error encontrado (si lo hay).
func (p *Publisher) Publish(ctx context.Context, event string, payload any) error {
	targets := p.routes[strings.TrimSpace(event)]
	if len(targets) == 0 {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}
	var firstErr error
	for _, target := range targets {
		if err := p.post(ctx, target, body); err != nil {
			slog.Error("webhook publish failed", "url", target, "event", event, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

func (p *Publisher) post(ctx context.Context, target string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.token != "" {
		req.Header.Set("X-Internal-Service-Token", p.token)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("status %d body %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func dedup(urls []string) []string {
	out := make([]string, 0, len(urls))
	seen := make(map[string]struct{}, len(urls))
	for _, raw := range urls {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}
