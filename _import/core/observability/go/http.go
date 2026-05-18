package observability

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

const RequestIDHeader = "X-Request-Id"

type contextKey string

const (
	requestIDContextKey contextKey = "core.backend.observability.request_id"
	loggerContextKey    contextKey = "core.backend.observability.logger"
)

// NewJSONLogger construye un logger JSON sobre stdout.
func NewJSONLogger(service string) *slog.Logger {
	return NewJSONLoggerWriter(service, os.Stdout)
}

// NewJSONLoggerWriter construye un logger JSON sobre el writer indicado.
func NewJSONLoggerWriter(service string, w io.Writer) *slog.Logger {
	if w == nil {
		w = io.Discard
	}
	service = strings.TrimSpace(service)
	if service == "" {
		service = "unknown"
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{})).With("service", service)
}

// Middleware agrega request IDs, logger por request y access logs JSON.
func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	return MiddlewareWithMetrics(logger, nil, next)
}

// MiddlewareWithMetrics extiende el middleware con RED metrics HTTP.
func MiddlewareWithMetrics(logger *slog.Logger, metrics *Metrics, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if logger == nil {
		logger = NewJSONLogger("unknown")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(RequestIDHeader))
		if requestID == "" {
			requestID = newRequestID()
		}

		start := time.Now()
		w.Header().Set(RequestIDHeader, requestID)

		ctx := ContextWithRequestID(r.Context(), requestID)
		requestLogger := logger.With("request_id", requestID)
		ctx = ContextWithLogger(ctx, requestLogger)
		r = r.WithContext(ctx)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		duration := time.Since(start)
		route := routeLabel(r)
		if metrics != nil {
			metrics.ObserveHTTPRequest(r, rec.status, duration)
		}

		requestLogger.Info("http request completed",
			"event", "http_request_completed",
			"method", r.Method,
			"path", requestPath(r),
			"route", route,
			"status", rec.status,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

// ContextWithRequestID guarda el request ID en el contexto.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// RequestIDFromContext devuelve el request ID si existe.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	if !ok || requestID == "" {
		return "", false
	}
	return requestID, true
}

// ContextWithLogger guarda el logger request-scoped en el contexto.
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerContextKey, logger)
}

// LoggerFromContext devuelve el logger request-scoped cuando existe.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerContextKey).(*slog.Logger)
	if !ok || logger == nil {
		return slog.Default()
	}
	return logger
}

// ApplyRequestID propaga el request ID del contexto a un request saliente.
func ApplyRequestID(r *http.Request, ctx context.Context) {
	if r == nil {
		return
	}
	if requestID, ok := RequestIDFromContext(ctx); ok {
		r.Header.Set(RequestIDHeader, requestID)
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.wroteHeader {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func (w *statusRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		if !w.wroteHeader {
			w.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}

func (w *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (w *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func newRequestID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buf[:])
}
