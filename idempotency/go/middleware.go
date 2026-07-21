package idempotency

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultHeaderName       = "Idempotency-Key"
	DefaultWaitTimeout      = 2 * time.Second
	DefaultInitialBackoff   = 10 * time.Millisecond
	DefaultMaxBackoff       = 100 * time.Millisecond
	DefaultStoreTimeout     = 5 * time.Second
	DefaultMaxRequestBytes  = int64(1 << 20)
	DefaultMaxResponseBytes = int64(1 << 20)
)

// ScopeResolver returns a trusted, business-agnostic namespace for a request.
type ScopeResolver func(*http.Request) (string, error)

// Fingerprinter returns a stable fingerprint while preserving request.Body.
type Fingerprinter func(*http.Request, int64) (string, error)

// ErrorWriter maps middleware errors to the consumer's HTTP error contract.
type ErrorWriter func(http.ResponseWriter, *http.Request, error)

// SleepFunc waits for a retry or returns when the context is canceled.
type SleepFunc func(context.Context, time.Duration) error

// MiddlewareConfig configures key extraction, polling, limits, and error output.
type MiddlewareConfig struct {
	HeaderName       string
	Scope            ScopeResolver
	Fingerprint      Fingerprinter
	WriteError       ErrorWriter
	WaitTimeout      time.Duration
	InitialBackoff   time.Duration
	MaxBackoff       time.Duration
	StoreTimeout     time.Duration
	MaxRequestBytes  int64
	MaxResponseBytes int64
	Now              func() time.Time
	Sleep            SleepFunc
}

// DefaultMiddlewareConfig returns bounded defaults for buffered HTTP commands.
func DefaultMiddlewareConfig() MiddlewareConfig {
	return MiddlewareConfig{
		HeaderName:       DefaultHeaderName,
		Fingerprint:      FingerprintRequest,
		WriteError:       writeDefaultError,
		WaitTimeout:      DefaultWaitTimeout,
		InitialBackoff:   DefaultInitialBackoff,
		MaxBackoff:       DefaultMaxBackoff,
		StoreTimeout:     DefaultStoreTimeout,
		MaxRequestBytes:  DefaultMaxRequestBytes,
		MaxResponseBytes: DefaultMaxResponseBytes,
		Now:              time.Now,
		Sleep:            sleepContext,
	}
}

// Middleware makes net/http handlers idempotent with a durable Store.
type Middleware struct {
	store  Store
	config MiddlewareConfig
}

// NewMiddleware validates config and creates an HTTP idempotency middleware.
func NewMiddleware(store Store, config MiddlewareConfig) (*Middleware, error) {
	if store == nil {
		return nil, ErrNilStore
	}
	if config.Scope == nil {
		return nil, fmt.Errorf("%w: scope resolver is nil", ErrInvalidConfig)
	}
	defaults := DefaultMiddlewareConfig()
	if strings.TrimSpace(config.HeaderName) == "" {
		config.HeaderName = defaults.HeaderName
	}
	config.HeaderName = http.CanonicalHeaderKey(strings.TrimSpace(config.HeaderName))
	if config.Fingerprint == nil {
		config.Fingerprint = defaults.Fingerprint
	}
	if config.WriteError == nil {
		config.WriteError = defaults.WriteError
	}
	if config.WaitTimeout == 0 {
		config.WaitTimeout = defaults.WaitTimeout
	}
	if config.InitialBackoff == 0 {
		config.InitialBackoff = defaults.InitialBackoff
	}
	if config.MaxBackoff == 0 {
		config.MaxBackoff = defaults.MaxBackoff
	}
	if config.StoreTimeout == 0 {
		config.StoreTimeout = defaults.StoreTimeout
	}
	if config.MaxRequestBytes == 0 {
		config.MaxRequestBytes = defaults.MaxRequestBytes
	}
	if config.MaxResponseBytes == 0 {
		config.MaxResponseBytes = defaults.MaxResponseBytes
	}
	if config.Now == nil {
		config.Now = defaults.Now
	}
	if config.Sleep == nil {
		config.Sleep = defaults.Sleep
	}
	if config.WaitTimeout < 0 || config.InitialBackoff < 0 || config.MaxBackoff < config.InitialBackoff {
		return nil, fmt.Errorf("%w: invalid wait or backoff duration", ErrInvalidConfig)
	}
	if config.StoreTimeout < 0 {
		return nil, fmt.Errorf("%w: store timeout must be positive", ErrInvalidConfig)
	}
	if config.MaxRequestBytes < 1 || config.MaxResponseBytes < 1 {
		return nil, fmt.Errorf("%w: body limits must be positive", ErrInvalidConfig)
	}

	return &Middleware{store: store, config: config}, nil
}

// Wrap applies idempotency to next. Responses are buffered until Complete succeeds.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	if next == nil {
		panic("idempotency: next handler is nil")
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		m.serveHTTP(writer, request, next)
	})
}

func (m *Middleware) serveHTTP(writer http.ResponseWriter, request *http.Request, next http.Handler) {
	key := strings.TrimSpace(request.Header.Get(m.config.HeaderName))
	if key == "" {
		m.config.WriteError(writer, request, ErrKeyRequired)
		return
	}
	if utf8.RuneCountInString(key) > maxKeyRunes {
		m.config.WriteError(writer, request, fmt.Errorf("%w: key exceeds %d characters", ErrInvalidKey, maxKeyRunes))
		return
	}

	scope, err := m.config.Scope(request)
	if err != nil {
		m.config.WriteError(writer, request, fmt.Errorf("resolve idempotency scope: %w", err))
		return
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		m.config.WriteError(writer, request, ErrScopeRequired)
		return
	}

	fingerprint, err := m.config.Fingerprint(request, m.config.MaxRequestBytes)
	if err != nil {
		m.config.WriteError(writer, request, err)
		return
	}
	claim := ClaimRequest{Scope: scope, Key: key, Fingerprint: fingerprint}
	result, err := m.waitForClaim(request.Context(), claim)
	if err != nil {
		m.config.WriteError(writer, request, err)
		return
	}
	if result.Outcome == OutcomeReplay {
		writeResponse(writer, result.Response)
		return
	}
	if result.Outcome != OutcomeAcquired {
		m.config.WriteError(writer, request, errors.New("idempotency: store returned an unknown outcome"))
		return
	}

	capture := newResponseCapture(m.config.MaxResponseBytes)
	defer func() {
		if recovered := recover(); recovered != nil {
			m.abandon(request.Context(), result.Lease)
			panic(recovered)
		}
	}()

	next.ServeHTTP(capture, request)
	if capture.err != nil {
		m.abandon(request.Context(), result.Lease)
		m.config.WriteError(writer, request, capture.err)
		return
	}

	response := capture.response()
	storeCtx, cancel := context.WithTimeout(context.WithoutCancel(request.Context()), m.config.StoreTimeout)
	err = m.store.Complete(storeCtx, result.Lease, response)
	cancel()
	if err != nil {
		m.config.WriteError(writer, request, fmt.Errorf("persist idempotent response: %w", err))
		return
	}
	writeResponse(writer, response)
}

func (m *Middleware) waitForClaim(ctx context.Context, claim ClaimRequest) (ClaimResult, error) {
	deadline := m.config.Now().Add(m.config.WaitTimeout)
	backoff := m.config.InitialBackoff

	for {
		result, err := m.store.Claim(ctx, claim)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrInProgress) {
			return ClaimResult{}, err
		}

		remaining := deadline.Sub(m.config.Now())
		if remaining <= 0 {
			return ClaimResult{}, ErrInProgress
		}
		wait := min(backoff, remaining)
		if err := m.config.Sleep(ctx, wait); err != nil {
			return ClaimResult{}, err
		}
		if backoff < m.config.MaxBackoff {
			if backoff > m.config.MaxBackoff/2 {
				backoff = m.config.MaxBackoff
			} else {
				backoff *= 2
			}
		}
	}
}

func (m *Middleware) abandon(requestCtx context.Context, lease Lease) {
	storeCtx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), m.config.StoreTimeout)
	defer cancel()
	_ = m.store.Abandon(storeCtx, lease)
}

type responseCapture struct {
	header    http.Header
	status    int
	committed http.Header
	body      bytes.Buffer
	maxBytes  int64
	err       error
	wroteHead bool
}

func newResponseCapture(maxBytes int64) *responseCapture {
	return &responseCapture{header: make(http.Header), maxBytes: maxBytes}
}

func (capture *responseCapture) Header() http.Header {
	return capture.header
}

func (capture *responseCapture) WriteHeader(status int) {
	if capture.wroteHead {
		return
	}
	capture.status = status
	capture.committed = replayableHeader(capture.header)
	capture.wroteHead = true
}

func (capture *responseCapture) Write(data []byte) (int, error) {
	if !capture.wroteHead {
		capture.WriteHeader(http.StatusOK)
	}
	if capture.err != nil {
		return 0, capture.err
	}
	if int64(capture.body.Len()+len(data)) > capture.maxBytes {
		capture.err = ErrResponseTooLarge
		return 0, capture.err
	}
	return capture.body.Write(data)
}

func (capture *responseCapture) response() Response {
	status := capture.status
	if status == 0 {
		status = http.StatusOK
	}
	header := capture.committed
	if header == nil {
		header = replayableHeader(capture.header)
	}
	return Response{
		StatusCode: status,
		Header:     header,
		Body:       append([]byte(nil), capture.body.Bytes()...),
	}
}

func replayableHeader(header http.Header) http.Header {
	result := cloneHeader(header)
	for _, value := range result.Values("Connection") {
		for _, name := range strings.Split(value, ",") {
			result.Del(strings.TrimSpace(name))
		}
	}
	for _, name := range []string{
		"Connection",
		"Content-Length",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		result.Del(name)
	}
	return result
}

func writeResponse(writer http.ResponseWriter, response Response) {
	for name, values := range response.Header {
		for _, value := range values {
			writer.Header().Add(name, value)
		}
	}
	writer.WriteHeader(response.StatusCode)
	_, _ = writer.Write(response.Body)
}

func writeDefaultError(writer http.ResponseWriter, _ *http.Request, err error) {
	status := http.StatusServiceUnavailable
	switch {
	case errors.Is(err, ErrKeyRequired), errors.Is(err, ErrInvalidKey), errors.Is(err, ErrScopeRequired), errors.Is(err, ErrFingerprintRequired):
		status = http.StatusBadRequest
	case errors.Is(err, ErrFingerprintMismatch):
		status = http.StatusConflict
	case errors.Is(err, ErrInProgress):
		status = http.StatusConflict
		writer.Header().Set("Retry-After", strconv.Itoa(max(1, int(DefaultMaxBackoff/time.Second))))
	case errors.Is(err, ErrRequestTooLarge):
		status = http.StatusRequestEntityTooLarge
	case errors.Is(err, ErrResponseTooLarge):
		status = http.StatusInternalServerError
	}
	http.Error(writer, http.StatusText(status), status)
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
