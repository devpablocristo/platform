package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/devpablocristo/core/ingestion/go"
)

const defaultPath = "/v1/extract"

// Client llama a un servicio de extracción que implementa el contrato ingestion (JSON).
type Client struct {
	baseURL    string
	httpClient *http.Client
	path       string
}

// Option configura el cliente.
type Option func(*Client)

// WithHTTPClient sustituye el cliente HTTP (timeouts, transport).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithPath cambia la ruta (por defecto /v1/extract).
func WithPath(path string) Option {
	return func(c *Client) {
		if path != "" {
			c.path = path
		}
	}
}

// New crea un cliente. baseURL sin barra final (ej: http://localhost:8091).
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		path: defaultPath,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Extract ejecuta POST JSON y devuelve ExtractResponse o error parseado.
func (c *Client) Extract(ctx context.Context, req ingestion.ExtractRequest) (*ingestion.ExtractResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + c.path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var eb ingestion.ErrorBody
		if json.Unmarshal(respBody, &eb) == nil && eb.Code != "" {
			return nil, ingestion.ExtractError{Code: eb.Code, Message: eb.Message}
		}
		return nil, fmt.Errorf("extract service: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var out ingestion.ExtractResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
