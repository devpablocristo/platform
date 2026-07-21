package idempotency

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type manualClock struct {
	mu  sync.Mutex
	now time.Time
}

func (clock *manualClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *manualClock) Add(duration time.Duration) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(duration)
}

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("PLATFORM_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("PLATFORM_POSTGRES_TEST_DSN is not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	migration, err := Migrations.ReadFile(InitialMigration)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply idempotency migration: %v", err)
	}
	return pool
}

func integrationScope(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	token, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	scope := "test-" + token
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM platform_idempotency_records WHERE scope = $1", scope)
	})
	return scope
}

func TestPostgresIntegrationLeaseReplayMismatchAndExpiry(t *testing.T) {
	pool := integrationPool(t)
	scope := integrationScope(t, pool)
	clock := &manualClock{now: time.Date(2026, 7, 21, 15, 0, 0, 0, time.UTC)}
	store, err := NewPostgresStore(pool, StoreConfig{
		LeaseDuration: time.Minute,
		RecordTTL:     10 * time.Minute,
		Now:           clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	claim := ClaimRequest{Scope: scope, Key: "lease-replay", Fingerprint: "fingerprint-a"}

	first, err := store.Claim(context.Background(), claim)
	if err != nil || first.Outcome != OutcomeAcquired {
		t.Fatalf("first claim: result=%#v err=%v", first, err)
	}
	if _, err := store.Claim(context.Background(), claim); !errors.Is(err, ErrInProgress) {
		t.Fatalf("live lease should be in progress, got %v", err)
	}

	clock.Add(time.Minute + time.Nanosecond)
	reclaimed, err := store.Claim(context.Background(), claim)
	if err != nil || reclaimed.Outcome != OutcomeAcquired {
		t.Fatalf("expired lease was not reclaimed: result=%#v err=%v", reclaimed, err)
	}
	if reclaimed.Lease.Token == first.Lease.Token {
		t.Fatal("reclaimed lease reused the owner token")
	}
	if err := store.Complete(context.Background(), first.Lease, Response{StatusCode: http.StatusAccepted}); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("stale owner completed reclaimed lease: %v", err)
	}

	response := Response{
		StatusCode: http.StatusCreated,
		Header:     http.Header{"Content-Type": {"application/json"}, "X-Result": {"stable"}},
		Body:       []byte(`{"id":"one"}`),
	}
	if err := store.Complete(context.Background(), reclaimed.Lease, response); err != nil {
		t.Fatal(err)
	}
	replay, err := store.Claim(context.Background(), claim)
	if err != nil || replay.Outcome != OutcomeReplay {
		t.Fatalf("completed claim did not replay: result=%#v err=%v", replay, err)
	}
	if replay.Response.StatusCode != response.StatusCode || replay.Response.Header.Get("X-Result") != "stable" || string(replay.Response.Body) != string(response.Body) {
		t.Fatalf("replay changed response: %#v", replay.Response)
	}

	mismatch := claim
	mismatch.Fingerprint = "fingerprint-b"
	if _, err := store.Claim(context.Background(), mismatch); !errors.Is(err, ErrFingerprintMismatch) {
		t.Fatalf("mismatched fingerprint returned %v", err)
	}

	clock.Add(10*time.Minute + time.Nanosecond)
	afterExpiry, err := store.Claim(context.Background(), mismatch)
	if err != nil || afterExpiry.Outcome != OutcomeAcquired {
		t.Fatalf("expired record was not reusable: result=%#v err=%v", afterExpiry, err)
	}
	if err := store.Abandon(context.Background(), afterExpiry.Lease); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresIntegrationConcurrentHTTPRequestsExecuteOnce(t *testing.T) {
	pool := integrationPool(t)
	scope := integrationScope(t, pool)
	store, err := NewPostgresStore(pool, StoreConfig{
		LeaseDuration: 5 * time.Second,
		RecordTTL:     time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultMiddlewareConfig()
	config.Scope = func(*http.Request) (string, error) { return scope, nil }
	config.WaitTimeout = 3 * time.Second
	config.InitialBackoff = 5 * time.Millisecond
	config.MaxBackoff = 25 * time.Millisecond
	middleware, err := NewMiddleware(store, config)
	if err != nil {
		t.Fatal(err)
	}

	var executions atomic.Int32
	handler := middleware.Wrap(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		executions.Add(1)
		time.Sleep(100 * time.Millisecond)
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Result", "postgres")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"id":"one"}`))
	}))

	const requestCount = 16
	start := make(chan struct{})
	responses := make(chan *httptest.ResponseRecorder, requestCount)
	var waitGroup sync.WaitGroup
	for range requestCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			request := httptest.NewRequest(http.MethodPost, "/commands", strings.NewReader(`{"amount":"10.00"}`))
			request.Header.Set(DefaultHeaderName, "postgres-concurrent-key")
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
		if recorder.Code != http.StatusCreated || recorder.Header().Get("X-Result") != "postgres" || recorder.Body.String() != `{"id":"one"}` {
			t.Fatalf("response was not replayed exactly: code=%d header=%q body=%q", recorder.Code, recorder.Header().Get("X-Result"), recorder.Body.String())
		}
	}
}
