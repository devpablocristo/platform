package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPublisher_Publish_sendsToAllURLs(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var received []map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		mu.Lock()
		received = append(received, body)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pub := NewPublisher("test-token", map[string][]string{
		"order.created": {srv.URL + "/a", srv.URL + "/b"},
	})

	err := pub.Publish(context.Background(), "order.created", map[string]string{"id": "123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("expected 2 deliveries, got %d", len(received))
	}
}

func TestPublisher_Publish_setsTokenHeader(t *testing.T) {
	t.Parallel()

	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Internal-Service-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pub := NewPublisher("secret-123", map[string][]string{
		"ping": {srv.URL},
	})

	if err := pub.Publish(context.Background(), "ping", "ok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotToken != "secret-123" {
		t.Fatalf("expected token %q, got %q", "secret-123", gotToken)
	}
}

func TestPublisher_Publish_noTokenHeaderWhenEmpty(t *testing.T) {
	t.Parallel()

	var hasHeader bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasHeader = r.Header["X-Internal-Service-Token"]
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pub := NewPublisher("", map[string][]string{
		"ping": {srv.URL},
	})

	if err := pub.Publish(context.Background(), "ping", "ok"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasHeader {
		t.Fatal("expected no token header when token is empty")
	}
}

func TestPublisher_Publish_unknownEventIsNoop(t *testing.T) {
	t.Parallel()

	pub := NewPublisher("tok", map[string][]string{
		"known": {"http://localhost:9999"},
	})

	err := pub.Publish(context.Background(), "unknown", "data")
	if err != nil {
		t.Fatalf("unexpected error for unknown event: %v", err)
	}
}

func TestPublisher_Publish_returnsErrorOnFailedTarget(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	pub := NewPublisher("", map[string][]string{
		"fail": {srv.URL},
	})

	err := pub.Publish(context.Background(), "fail", "data")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestPublisher_Publish_dedupURLs(t *testing.T) {
	t.Parallel()

	var count int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pub := NewPublisher("", map[string][]string{
		"dup": {srv.URL, srv.URL, " " + srv.URL + " "},
	})

	if err := pub.Publish(context.Background(), "dup", "x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 delivery after dedup, got %d", count)
	}
}
