package idempotency

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryRecord struct {
	fingerprint string
	token       string
	processing  bool
	response    Response
}

type memoryStore struct {
	mu      sync.Mutex
	records map[string]memoryRecord
	next    uint64
}

func newMemoryStore() *memoryStore {
	return &memoryStore{records: make(map[string]memoryRecord)}
}

func (store *memoryStore) Claim(_ context.Context, claim ClaimRequest) (ClaimResult, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	mapKey := claim.Scope + "\x00" + claim.Key
	record, ok := store.records[mapKey]
	if !ok {
		store.next++
		token := time.Unix(0, int64(store.next)).UTC().Format(time.RFC3339Nano)
		store.records[mapKey] = memoryRecord{fingerprint: claim.Fingerprint, token: token, processing: true}
		return ClaimResult{
			Outcome: OutcomeAcquired,
			Lease:   Lease{Scope: claim.Scope, Key: claim.Key, Fingerprint: claim.Fingerprint, Token: token},
		}, nil
	}
	if record.fingerprint != claim.Fingerprint {
		return ClaimResult{}, ErrFingerprintMismatch
	}
	if record.processing {
		return ClaimResult{}, ErrInProgress
	}
	return ClaimResult{Outcome: OutcomeReplay, Response: cloneResponse(record.response)}, nil
}

func (store *memoryStore) Complete(_ context.Context, lease Lease, response Response) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	mapKey := lease.Scope + "\x00" + lease.Key
	record, ok := store.records[mapKey]
	if !ok || !record.processing || record.token != lease.Token || record.fingerprint != lease.Fingerprint {
		return ErrLeaseLost
	}
	record.processing = false
	record.response = cloneResponse(response)
	store.records[mapKey] = record
	return nil
}

func (store *memoryStore) Abandon(_ context.Context, lease Lease) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	mapKey := lease.Scope + "\x00" + lease.Key
	record, ok := store.records[mapKey]
	if !ok || !record.processing || record.token != lease.Token {
		return ErrLeaseLost
	}
	delete(store.records, mapKey)
	return nil
}

func cloneResponse(response Response) Response {
	return Response{
		StatusCode: response.StatusCode,
		Header:     cloneHeader(response.Header),
		Body:       append([]byte(nil), response.Body...),
	}
}

func testMiddleware(t *testing.T, store Store) *Middleware {
	t.Helper()
	config := DefaultMiddlewareConfig()
	config.Scope = func(*http.Request) (string, error) { return "trusted-scope", nil }
	config.WaitTimeout = time.Second
	config.InitialBackoff = time.Millisecond
	config.MaxBackoff = 5 * time.Millisecond
	middleware, err := NewMiddleware(store, config)
	if err != nil {
		t.Fatal(err)
	}
	return middleware
}

func TestMiddlewareConcurrentSameKeyExecutesOnceAndReplays(t *testing.T) {
	store := newMemoryStore()
	middleware := testMiddleware(t, store)
	var executions atomic.Int32
	handler := middleware.Wrap(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		executions.Add(1)
		time.Sleep(40 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Result", "stable")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"id":"one"}`))
	}))

	const requestCount = 12
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, requestCount)
	var waitGroup sync.WaitGroup
	for range requestCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			request := httptest.NewRequest(http.MethodPost, "/commands", strings.NewReader(`{"amount":"10.00"}`))
			request.Header.Set(DefaultHeaderName, "same-key")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			responses <- recorder
		}()
	}
	close(start)
	waitGroup.Wait()
	close(responses)

	if got := executions.Load(); got != 1 {
		t.Fatalf("handler executed %d times", got)
	}
	for recorder := range responses {
		if recorder.Code != http.StatusCreated {
			t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("X-Result"); got != "stable" {
			t.Fatalf("unexpected header %q", got)
		}
		if got := recorder.Body.String(); got != `{"id":"one"}` {
			t.Fatalf("unexpected body %q", got)
		}
	}
}

func TestMiddlewareRejectsMissingKeyAndFingerprintMismatch(t *testing.T) {
	store := newMemoryStore()
	middleware := testMiddleware(t, store)
	var executions atomic.Int32
	handler := middleware.Wrap(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		executions.Add(1)
		writer.WriteHeader(http.StatusNoContent)
	}))

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/commands", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing key returned %d", missing.Code)
	}

	firstRequest := httptest.NewRequest(http.MethodPost, "/commands", strings.NewReader("first"))
	firstRequest.Header.Set(DefaultHeaderName, "reused-key")
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, firstRequest)
	if first.Code != http.StatusNoContent {
		t.Fatalf("first request returned %d", first.Code)
	}

	secondRequest := httptest.NewRequest(http.MethodPost, "/commands", strings.NewReader("different"))
	secondRequest.Header.Set(DefaultHeaderName, "reused-key")
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, secondRequest)
	if second.Code != http.StatusConflict {
		t.Fatalf("mismatched request returned %d", second.Code)
	}
	if got := executions.Load(); got != 1 {
		t.Fatalf("mismatched request executed handler; count=%d", got)
	}
}

func TestMiddlewareRequiresTrustedNonEmptyScope(t *testing.T) {
	store := newMemoryStore()
	_, err := NewMiddleware(store, DefaultMiddlewareConfig())
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("missing scope resolver returned %v", err)
	}

	config := DefaultMiddlewareConfig()
	config.Scope = func(*http.Request) (string, error) { return "  ", nil }
	middleware, err := NewMiddleware(store, config)
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("empty scope reached handler")
	}))
	request := httptest.NewRequest(http.MethodPost, "/commands", nil)
	request.Header.Set(DefaultHeaderName, "scopeless-key")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("empty scope returned %d", recorder.Code)
	}
}

func TestMiddlewareDoesNotPersistHeadersMutatedAfterWriteHeader(t *testing.T) {
	store := newMemoryStore()
	middleware := testMiddleware(t, store)
	handler := middleware.Wrap(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Committed", "yes")
		writer.WriteHeader(http.StatusCreated)
		writer.Header().Set("X-Late", "no")
		_, _ = writer.Write([]byte("created"))
	}))

	for range 2 {
		request := httptest.NewRequest(http.MethodPost, "/commands", nil)
		request.Header.Set(DefaultHeaderName, "header-key")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if got := recorder.Header().Get("X-Committed"); got != "yes" {
			t.Fatalf("committed header = %q", got)
		}
		if got := recorder.Header().Get("X-Late"); got != "" {
			t.Fatalf("late header leaked into response: %q", got)
		}
	}
}

func TestMiddlewareEnforcesBodyLimitsAndAbandonsOversizedResponse(t *testing.T) {
	store := newMemoryStore()
	config := DefaultMiddlewareConfig()
	config.Scope = func(*http.Request) (string, error) { return "trusted-scope", nil }
	config.MaxRequestBytes = 4
	config.MaxResponseBytes = 4
	middleware, err := NewMiddleware(store, config)
	if err != nil {
		t.Fatal(err)
	}

	requestTooLarge := middleware.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("oversized request reached handler")
	}))
	request := httptest.NewRequest(http.MethodPost, "/commands", strings.NewReader("12345"))
	request.Header.Set(DefaultHeaderName, "large-request")
	recorder := httptest.NewRecorder()
	requestTooLarge.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized request returned %d", recorder.Code)
	}

	var executions atomic.Int32
	responseTooLarge := middleware.Wrap(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if executions.Add(1) == 1 {
			_, _ = writer.Write([]byte("12345"))
			return
		}
		_, _ = writer.Write([]byte("okay"))
	}))
	for attempt, want := range []int{http.StatusInternalServerError, http.StatusOK} {
		request := httptest.NewRequest(http.MethodPost, "/commands", nil)
		request.Header.Set(DefaultHeaderName, "large-response")
		recorder := httptest.NewRecorder()
		responseTooLarge.ServeHTTP(recorder, request)
		if recorder.Code != want {
			t.Fatalf("attempt %d returned %d, want %d", attempt+1, recorder.Code, want)
		}
	}
}

func TestMiddlewareAbandonsOnPanic(t *testing.T) {
	store := newMemoryStore()
	middleware := testMiddleware(t, store)
	handler := middleware.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	request := httptest.NewRequest(http.MethodPost, "/commands", nil)
	request.Header.Set(DefaultHeaderName, "panic-key")

	func() {
		defer func() {
			if recovered := recover(); recovered == nil {
				t.Fatal("expected panic")
			}
		}()
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}()

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.records) != 0 {
		t.Fatalf("panic left processing records: %#v", store.records)
	}
}

func TestMiddlewareWaitIsBounded(t *testing.T) {
	store := &alwaysInProgressStore{}
	config := DefaultMiddlewareConfig()
	config.Scope = func(*http.Request) (string, error) { return "trusted-scope", nil }
	config.WaitTimeout = time.Nanosecond
	config.InitialBackoff = time.Nanosecond
	config.MaxBackoff = time.Nanosecond
	middleware, err := NewMiddleware(store, config)
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware.Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("in-progress request reached handler")
	}))
	request := httptest.NewRequest(http.MethodPost, "/commands", nil)
	request.Header.Set(DefaultHeaderName, "busy-key")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("in-progress request returned %d", recorder.Code)
	}
	if store.claims.Load() > 2 {
		t.Fatalf("bounded wait claimed %d times", store.claims.Load())
	}
}

type alwaysInProgressStore struct {
	claims atomic.Int32
}

func (store *alwaysInProgressStore) Claim(context.Context, ClaimRequest) (ClaimResult, error) {
	store.claims.Add(1)
	return ClaimResult{}, ErrInProgress
}

func (*alwaysInProgressStore) Complete(context.Context, Lease, Response) error {
	return errors.New("unexpected complete")
}

func (*alwaysInProgressStore) Abandon(context.Context, Lease) error {
	return errors.New("unexpected abandon")
}
