package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const DefaultMetricsPath = "/metrics"

var defaultDurationBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// MetricsConfig define la configuración base de RED metrics HTTP.
type MetricsConfig struct {
	Namespace                string
	DurationBuckets          []float64
	DisableRuntimeCollectors bool
}

// DefaultMetricsConfig devuelve una configuración razonable para servicios HTTP.
func DefaultMetricsConfig(namespace string) MetricsConfig {
	return MetricsConfig{
		Namespace:       normalizeMetricLabel(namespace),
		DurationBuckets: append([]float64(nil), defaultDurationBuckets...),
	}
}

// Metrics expone colectores Prometheus para RED metrics HTTP.
type Metrics struct {
	registry *prometheus.Registry

	httpRequests *prometheus.CounterVec
	httpErrors   *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
}

// NewMetrics construye una registry local para métricas RED HTTP.
func NewMetrics(config MetricsConfig) *Metrics {
	config = normalizeMetricsConfig(config)

	registry := prometheus.NewRegistry()
	httpRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: config.Namespace,
		Name:      "http_requests_total",
		Help:      "Total HTTP requests handled by route and status code.",
	}, []string{"method", "route", "status_code"})
	httpErrors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: config.Namespace,
		Name:      "http_request_errors_total",
		Help:      "Total HTTP requests that completed with an error status.",
	}, []string{"method", "route", "status_code"})
	httpDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: config.Namespace,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency by route.",
		Buckets:   config.DurationBuckets,
	}, []string{"method", "route"})

	if !config.DisableRuntimeCollectors {
		registry.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)
	}
	registry.MustRegister(httpRequests, httpErrors, httpDuration)

	return &Metrics{
		registry:     registry,
		httpRequests: httpRequests,
		httpErrors:   httpErrors,
		httpDuration: httpDuration,
	}
}

// Registry devuelve la registry local para integraciones avanzadas.
func (m *Metrics) Registry() *prometheus.Registry {
	if m == nil {
		return nil
	}
	return m.registry
}

// Handler devuelve el handler Prometheus para scrapes.
func (m *Metrics) Handler() http.Handler {
	if m == nil || m.registry == nil {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// WithMetricsEndpoint enruta el path de métricas y delega el resto.
func WithMetricsEndpoint(next http.Handler, metricsHandler http.Handler) http.Handler {
	return WithMetricsEndpointPath(DefaultMetricsPath, next, metricsHandler)
}

// WithMetricsEndpointPath enruta un path de métricas configurable y delega el resto.
func WithMetricsEndpointPath(metricsPath string, next http.Handler, metricsHandler http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	if metricsHandler == nil {
		return next
	}
	metricsPath = normalizeMetricsPath(metricsPath)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestPath(r) == metricsPath {
			r.Pattern = metricsPath
			metricsHandler.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ObserveHTTPRequest registra métricas RED de un request completado.
func (m *Metrics) ObserveHTTPRequest(r *http.Request, status int, duration time.Duration) {
	if m == nil || r == nil {
		return
	}
	route := routeLabel(r)
	statusCode := strconv.Itoa(status)
	m.httpRequests.WithLabelValues(r.Method, route, statusCode).Inc()
	if status >= http.StatusBadRequest {
		m.httpErrors.WithLabelValues(r.Method, route, statusCode).Inc()
	}
	m.httpDuration.WithLabelValues(r.Method, route).Observe(duration.Seconds())
}

func normalizeMetricsConfig(config MetricsConfig) MetricsConfig {
	defaults := DefaultMetricsConfig(config.Namespace)
	if config.Namespace == "" {
		config.Namespace = defaults.Namespace
	}
	if len(config.DurationBuckets) == 0 {
		config.DurationBuckets = defaults.DurationBuckets
	}
	return config
}

func normalizeMetricsPath(metricsPath string) string {
	metricsPath = strings.TrimSpace(metricsPath)
	if metricsPath == "" {
		return DefaultMetricsPath
	}
	if !strings.HasPrefix(metricsPath, "/") {
		metricsPath = "/" + metricsPath
	}
	return metricsPath
}

func routeLabel(r *http.Request) string {
	if r == nil {
		return "unmatched"
	}
	pattern := strings.TrimSpace(r.Pattern)
	if pattern == "" {
		if requestPath(r) == DefaultMetricsPath {
			return DefaultMetricsPath
		}
		return "unmatched"
	}
	if _, route, ok := strings.Cut(pattern, " "); ok {
		pattern = route
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "unmatched"
	}
	return pattern
}

func normalizeMetricLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "app"
	}
	return value
}

func requestPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Path)
}
