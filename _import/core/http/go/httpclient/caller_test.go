package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestDoJSON_Get(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ping" {
			t.Fatalf("path %s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "k" {
			t.Fatalf("missing api key")
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "1"})
	}))
	t.Cleanup(srv.Close)

	h := make(http.Header)
	h.Set("X-API-Key", "k")
	c := Caller{HTTP: srv.Client(), BaseURL: srv.URL, Header: h}
	st, raw, err := c.DoJSON(context.Background(), http.MethodGet, "/v1/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusOK {
		t.Fatalf("status %d", st)
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil || m["ok"] != "1" {
		t.Fatalf("body %s err %v", raw, err)
	}
}

func TestDoJSON_Post(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]int
		json.NewDecoder(r.Body).Decode(&in)
		if in["n"] != 42 {
			t.Fatalf("body %+v", in)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"x"}`))
	}))
	t.Cleanup(srv.Close)

	c := Caller{HTTP: srv.Client(), BaseURL: srv.URL}
	st, raw, err := c.DoJSON(context.Background(), http.MethodPost, "/v1/x", map[string]int{"n": 42})
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusCreated {
		t.Fatalf("status %d", st)
	}
	if string(raw) != `{"id":"x"}` {
		t.Fatalf("raw %s", raw)
	}
}

func TestDoJSON_DynamicHeaders(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "static" {
			t.Errorf("expected static header, got %s", r.Header.Get("X-API-Key"))
		}
		if r.Header.Get("X-User-ID") != "user-123" {
			t.Errorf("expected dynamic header X-User-ID=user-123, got %s", r.Header.Get("X-User-ID"))
		}
		if r.Header.Get("X-Project-ID") != "proj-456" {
			t.Errorf("expected dynamic header X-Project-ID=proj-456, got %s", r.Header.Get("X-Project-ID"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	h := make(http.Header)
	h.Set("X-API-Key", "static")
	c := Caller{HTTP: srv.Client(), BaseURL: srv.URL, Header: h}

	st, _, err := c.DoJSON(context.Background(), http.MethodGet, "/v1/test", nil,
		WithHeader("X-User-ID", "user-123"),
		WithHeader("X-Project-ID", "proj-456"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusOK {
		t.Fatalf("status %d", st)
	}
}

func TestDoJSON_IdempotencyKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Idempotency-Key") != "idem-abc" {
			t.Errorf("expected Idempotency-Key=idem-abc, got %s", r.Header.Get("Idempotency-Key"))
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	c := Caller{HTTP: srv.Client(), BaseURL: srv.URL}
	st, _, err := c.DoJSON(context.Background(), http.MethodPost, "/v1/x", map[string]string{"a": "b"},
		WithIdempotencyKey("idem-abc"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusCreated {
		t.Fatalf("status %d", st)
	}
}

func TestDoJSON_Retry5xx(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"temporary"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	c := Caller{HTTP: srv.Client(), BaseURL: srv.URL, MaxRetries: 3, RetryBaseDelay: 1}
	st, _, err := c.DoJSON(context.Background(), http.MethodGet, "/v1/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusOK {
		t.Fatalf("expected 200 after retry, got %d", st)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

func TestDoJSON_NoRetryOn4xx(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"bad"}`))
	}))
	t.Cleanup(srv.Close)

	c := Caller{HTTP: srv.Client(), BaseURL: srv.URL, MaxRetries: 3}
	st, _, err := c.DoJSON(context.Background(), http.MethodGet, "/v1/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", st)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 call (no retry on 4xx), got %d", calls.Load())
	}
}

func TestDoJSON_MaxBodySize(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Devolver un body grande
		for range 1000 {
			w.Write([]byte("x"))
		}
	}))
	t.Cleanup(srv.Close)

	c := Caller{HTTP: srv.Client(), BaseURL: srv.URL, MaxBodySize: 100}
	st, raw, err := c.DoJSON(context.Background(), http.MethodGet, "/v1/big", nil)
	if err != nil {
		t.Fatal(err)
	}
	if st != http.StatusOK {
		t.Fatalf("status %d", st)
	}
	if len(raw) > 100 {
		t.Fatalf("expected max 100 bytes, got %d", len(raw))
	}
}
