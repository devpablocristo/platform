package observability

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMiddlewareWithMetricsRecordsREDMetrics(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics(DefaultMetricsConfig("svc_test"))
	handler := MiddlewareWithMetrics(NewJSONLoggerWriter("svc-test", nil), metrics, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Pattern, "GET /v1/resources/{id}"; got != want {
			t.Fatalf("unexpected pattern in handler: got=%q want=%q", got, want)
		}
		w.WriteHeader(http.StatusCreated)
	}))

	mux := http.NewServeMux()
	mux.Handle("GET /v1/resources/{id}", handler)

	req := httptest.NewRequest(http.MethodGet, "/v1/resources/123", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if got := testutil.ToFloat64(metrics.httpRequests.WithLabelValues(http.MethodGet, "/v1/resources/{id}", "201")); got != 1 {
		t.Fatalf("unexpected request counter: %v", got)
	}
	if got := testutil.ToFloat64(metrics.httpErrors.WithLabelValues(http.MethodGet, "/v1/resources/{id}", "201")); got != 0 {
		t.Fatalf("unexpected error counter: %v", got)
	}
	metricsRec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, DefaultMetricsPath, nil))
	body, err := io.ReadAll(metricsRec.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	if !strings.Contains(string(body), `svc_test_http_request_duration_seconds_bucket{method="GET",route="/v1/resources/{id}"`) {
		t.Fatalf("expected duration metric in scrape output, got=%q", string(body))
	}
}

func TestMiddlewareWithMetricsTracksErrorsAndMetricsEndpoint(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics(DefaultMetricsConfig("svc_test"))
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	wrapped := WithMetricsEndpoint(base, metrics.Handler())
	handler := MiddlewareWithMetrics(NewJSONLoggerWriter("svc-test", nil), metrics, wrapped)

	req := httptest.NewRequest(http.MethodGet, DefaultMetricsPath, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected /metrics status: %d", rec.Code)
	}
	if got := testutil.ToFloat64(metrics.httpRequests.WithLabelValues(http.MethodGet, DefaultMetricsPath, "200")); got != 1 {
		t.Fatalf("unexpected /metrics request counter: %v", got)
	}

	errReq := httptest.NewRequest(http.MethodPost, "/missing", nil)
	errRec := httptest.NewRecorder()
	handler.ServeHTTP(errRec, errReq)

	if got := testutil.ToFloat64(metrics.httpErrors.WithLabelValues(http.MethodPost, "unmatched", "401")); got != 1 {
		t.Fatalf("unexpected error counter: %v", got)
	}
}

func TestWithMetricsEndpointPathUsesConfiguredPath(t *testing.T) {
	t.Parallel()

	metrics := NewMetrics(DefaultMetricsConfig("svc_test"))
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := WithMetricsEndpointPath("internal/metrics", base, metrics.Handler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/metrics", nil)
	MiddlewareWithMetrics(NewJSONLoggerWriter("svc-test", nil), metrics, handler).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected configured metrics status: %d", rec.Code)
	}
	if got := testutil.ToFloat64(metrics.httpRequests.WithLabelValues(http.MethodGet, "/internal/metrics", "200")); got != 1 {
		t.Fatalf("unexpected configured metrics counter: %v", got)
	}
}
