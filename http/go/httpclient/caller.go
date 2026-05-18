// Package httpclient proporciona un cliente HTTP reutilizable para comunicación service-to-service.
// Soporta headers estáticos y dinámicos, retry con backoff, body size limit, y timeout per-request.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
)

// Caller ejecuta peticiones HTTP JSON contra un baseURL fijo.
// Headers estáticos se configuran en Header; headers dinámicos se pasan por request via RequestOption.
type Caller struct {
	HTTP    *http.Client
	BaseURL string
	Header  http.Header

	// MaxRetries número de reintentos en error 5xx o de red (default 0 = sin retry).
	MaxRetries int
	// RetryBaseDelay delay inicial entre reintentos (default 200ms).
	RetryBaseDelay time.Duration
	// MaxBodySize límite de lectura del body de respuesta (default 0 = sin límite).
	MaxBodySize int64
}

// RequestOption modifica una petición individual sin cambiar el Caller.
type RequestOption func(*http.Request)

// WithHeader agrega un header a esta petición.
func WithHeader(key, value string) RequestOption {
	return func(r *http.Request) {
		r.Header.Set(key, value)
	}
}

// WithHeaders agrega múltiples headers a esta petición.
func WithHeaders(headers http.Header) RequestOption {
	return func(r *http.Request) {
		for k, vv := range headers {
			for _, v := range vv {
				r.Header.Add(k, v)
			}
		}
	}
}

// WithIdempotencyKey agrega un Idempotency-Key header.
func WithIdempotencyKey(key string) RequestOption {
	return WithHeader("Idempotency-Key", key)
}

func (c *Caller) join(pathOrURL string) string {
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		return pathOrURL
	}
	b := strings.TrimSuffix(strings.TrimSpace(c.BaseURL), "/")
	if !strings.HasPrefix(pathOrURL, "/") {
		pathOrURL = "/" + pathOrURL
	}
	return b + pathOrURL
}

// DoJSON ejecuta method en path. body nil para GET/DELETE sin cuerpo.
// Devuelve status HTTP, cuerpo crudo y error de red/construcción.
// Reintenta en errores 5xx y de red si MaxRetries > 0.
func (c *Caller) DoJSON(ctx context.Context, method, path string, body any, opts ...RequestOption) (int, []byte, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal body: %w", err)
		}
	}
	return c.doWithRetry(ctx, method, path, bodyBytes, opts)
}

// DoRaw ejecuta una petición con body pre-serializado y content-type custom.
func (c *Caller) DoRaw(ctx context.Context, method, path string, body []byte, contentType string, opts ...RequestOption) (int, []byte, error) {
	allOpts := append([]RequestOption{WithHeader("Content-Type", contentType)}, opts...)
	return c.doWithRetry(ctx, method, path, body, allOpts)
}

// DoForm ejecuta una petición POST con body application/x-www-form-urlencoded.
func (c *Caller) DoForm(ctx context.Context, path string, formData string, opts ...RequestOption) (int, []byte, error) {
	return c.DoRaw(ctx, http.MethodPost, path, []byte(formData), "application/x-www-form-urlencoded", opts...)
}

// doWithRetry centraliza el loop de retry para DoJSON y DoRaw.
func (c *Caller) doWithRetry(ctx context.Context, method, path string, bodyBytes []byte, opts []RequestOption) (int, []byte, error) {
	maxAttempts := 1 + c.MaxRetries
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	var status int
	var raw []byte

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay(attempt)
			select {
			case <-ctx.Done():
				return 0, nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		status, raw, lastErr = c.doOnce(ctx, method, path, bodyBytes, opts)
		if lastErr != nil {
			slog.Debug("httpclient_retry", "attempt", attempt+1, "method", method, "path", path, "error", lastErr)
			continue
		}
		if status >= 500 && attempt < maxAttempts-1 {
			slog.Debug("httpclient_retry_5xx", "attempt", attempt+1, "method", method, "path", path, "status", status)
			continue
		}
		return status, raw, nil
	}

	if lastErr != nil {
		return 0, nil, lastErr
	}
	return status, raw, nil
}

func (c *Caller) doOnce(ctx context.Context, method, path string, bodyBytes []byte, opts []RequestOption) (int, []byte, error) {
	var rdr io.Reader
	if bodyBytes != nil {
		rdr = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.join(path), rdr)
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}

	for k, vv := range c.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	for _, opt := range opts {
		opt(req)
	}

	if bodyBytes != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}

	resp, err := hc.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if c.MaxBodySize > 0 {
		reader = io.LimitReader(resp.Body, c.MaxBodySize)
	}

	raw, err := io.ReadAll(reader)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read body: %w", err)
	}

	return resp.StatusCode, raw, nil
}

func (c *Caller) retryDelay(attempt int) time.Duration {
	base := c.RetryBaseDelay
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	delay := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	if delay > 10*time.Second {
		delay = 10 * time.Second
	}
	return delay
}
