package outbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mutableClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (clock *mutableClock) Now() time.Time {
	clock.mu.RLock()
	defer clock.mu.RUnlock()
	return clock.now
}

func (clock *mutableClock) Add(duration time.Duration) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.now = clock.now.Add(duration)
}

type sequenceTokenGenerator struct {
	prefix string
	next   atomic.Int64
}

func (generator *sequenceTokenGenerator) NewToken() (string, error) {
	return fmt.Sprintf("%s-%d", generator.prefix, generator.next.Add(1)), nil
}

type integrationHarness struct {
	pool  *pgxpool.Pool
	store *Store
	clock *mutableClock
	table pgx.Identifier
}

func newIntegrationHarness(t *testing.T) *integrationHarness {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("PLATFORM_POSTGRES_TEST_DSN"))
	if dsn == "" {
		if strings.EqualFold(os.Getenv("CI"), "true") {
			t.Fatal("PLATFORM_POSTGRES_TEST_DSN is required in CI")
		}
		t.Skip("PLATFORM_POSTGRES_TEST_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	schemaName := fmt.Sprintf("outbox_%d", time.Now().UnixNano())
	schemaSQL := pgx.Identifier{schemaName}.Sanitize()
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+schemaSQL); err != nil {
		pool.Close()
		t.Fatalf("create schema: %v", err)
	}
	table := pgx.Identifier{schemaName, "messages"}
	ddl, err := SchemaSQL(table)
	if err != nil {
		pool.Close()
		t.Fatalf("render schema: %v", err)
	}
	if _, err := pool.Exec(ctx, ddl); err != nil {
		pool.Close()
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := pool.Exec(ctx, ddl); err != nil {
		pool.Close()
		t.Fatalf("reapply idempotent schema: %v", err)
	}

	clock := &mutableClock{now: time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)}
	store, err := NewStore(pool, StoreConfig{
		Table:               table,
		DefaultMaxAttempts:  3,
		Clock:               clock,
		MessageIDGenerator:  &sequenceTokenGenerator{prefix: "message"},
		LeaseTokenGenerator: &sequenceTokenGenerator{prefix: "lease"},
	})
	if err != nil {
		pool.Close()
		t.Fatalf("NewStore: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+schemaSQL+" CASCADE")
		pool.Close()
	})
	return &integrationHarness{pool: pool, store: store, clock: clock, table: table}
}

func (harness *integrationHarness) appendCommitted(t *testing.T, input MessageInput) Message {
	t.Helper()
	tx, err := harness.pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("begin append: %v", err)
	}
	defer tx.Rollback(t.Context()) //nolint:errcheck
	message, err := harness.store.Append(t.Context(), tx, input)
	if err != nil {
		t.Fatalf("append message: %v", err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatalf("commit append: %v", err)
	}
	return message
}

func TestAppendRollbackIsNeverLeasedOrPublished(t *testing.T) {
	harness := newIntegrationHarness(t)

	tx, err := harness.pool.Begin(t.Context())
	if err != nil {
		t.Fatalf("begin append: %v", err)
	}
	if _, err := harness.store.Append(t.Context(), tx, MessageInput{
		Topic: "entity.created", Payload: []byte(`{"id":"1"}`),
	}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := tx.Rollback(t.Context()); err != nil {
		t.Fatalf("rollback append: %v", err)
	}

	leased, err := harness.store.Lease(t.Context(), LeaseRequest{Limit: 10, Duration: time.Minute})
	if err != nil {
		t.Fatalf("Lease: %v", err)
	}
	if len(leased) != 0 {
		t.Fatalf("leased rolled-back messages: %d", len(leased))
	}

	var publishCalls atomic.Int64
	backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: time.Second, Maximum: time.Minute, Multiplier: 2,
	})
	if err != nil {
		t.Fatalf("NewExponentialBackoff: %v", err)
	}
	dispatcher, err := NewDispatcher(
		harness.store,
		PublisherFunc(func(context.Context, Publication) error {
			publishCalls.Add(1)
			return nil
		}),
		DispatcherConfig{BatchSize: 10, LeaseDuration: time.Minute, Backoff: backoff},
	)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	result, err := dispatcher.Dispatch(t.Context())
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if result.Leased != 0 || publishCalls.Load() != 0 {
		t.Fatalf("result = %+v, publish calls = %d", result, publishCalls.Load())
	}
}

func TestConcurrentWorkersLeaseEachMessageOnce(t *testing.T) {
	harness := newIntegrationHarness(t)
	const (
		messageCount = 24
		workerCount  = 8
	)
	for index := 0; index < messageCount; index++ {
		harness.appendCommitted(t, MessageInput{
			Topic: "batch.item", Payload: []byte(fmt.Sprintf(`{"index":%d}`, index)),
		})
	}

	start := make(chan struct{})
	results := make(chan []LeasedMessage, workerCount)
	errorsChannel := make(chan error, workerCount)
	var workers sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			messages, err := harness.store.Lease(t.Context(), LeaseRequest{
				Limit: messageCount / workerCount, Duration: time.Minute,
			})
			if err != nil {
				errorsChannel <- err
				return
			}
			results <- messages
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	close(errorsChannel)
	for err := range errorsChannel {
		t.Fatalf("concurrent Lease: %v", err)
	}

	counts := make(map[string]int, messageCount)
	for messages := range results {
		for _, message := range messages {
			counts[message.ID]++
		}
	}
	if len(counts) != messageCount {
		t.Fatalf("unique leased messages = %d, want %d", len(counts), messageCount)
	}
	for id, count := range counts {
		if count != 1 {
			t.Errorf("message %s leased %d times", id, count)
		}
	}
}

func TestDispatcherRetriesThenExhaustsAttemptsWithStableMetadata(t *testing.T) {
	harness := newIntegrationHarness(t)
	appended := harness.appendCommitted(t, MessageInput{
		IdempotencyKey: "command-42",
		Topic:          "entity.created",
		Payload:        []byte(`{"id":"42"}`),
		Headers:        map[string]string{"trace-id": "trace-7"},
		MaxAttempts:    2,
	})

	var mu sync.Mutex
	publications := make([]Publication, 0, 2)
	publishFailure := errors.New("transport unavailable")
	publisher := PublisherFunc(func(_ context.Context, publication Publication) error {
		mu.Lock()
		defer mu.Unlock()
		publications = append(publications, publication)
		return publishFailure
	})
	backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: 5 * time.Second, Maximum: 20 * time.Second, Multiplier: 2,
	})
	if err != nil {
		t.Fatalf("NewExponentialBackoff: %v", err)
	}
	dispatcher, err := NewDispatcher(harness.store, publisher, DispatcherConfig{
		BatchSize: 1, LeaseDuration: time.Minute, Backoff: backoff,
	})
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	first, err := dispatcher.Dispatch(t.Context())
	if !errors.Is(err, publishFailure) || first != (DispatchResult{Leased: 1, Retried: 1}) {
		t.Fatalf("first dispatch = %+v, error = %v", first, err)
	}
	tooSoon, err := dispatcher.Dispatch(t.Context())
	if err != nil || tooSoon.Leased != 0 {
		t.Fatalf("early dispatch = %+v, error = %v", tooSoon, err)
	}
	harness.clock.Add(5 * time.Second)
	second, err := dispatcher.Dispatch(t.Context())
	if !errors.Is(err, publishFailure) || second != (DispatchResult{Leased: 1, Failed: 1}) {
		t.Fatalf("second dispatch = %+v, error = %v", second, err)
	}
	harness.clock.Add(time.Hour)
	afterExhaustion, err := dispatcher.Dispatch(t.Context())
	if err != nil || afterExhaustion.Leased != 0 {
		t.Fatalf("post-exhaustion dispatch = %+v, error = %v", afterExhaustion, err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(publications) != 2 {
		t.Fatalf("publications = %d, want 2", len(publications))
	}
	sort.Slice(publications, func(i, j int) bool { return publications[i].Attempt < publications[j].Attempt })
	for _, publication := range publications {
		if publication.MessageID != appended.ID || publication.IdempotencyKey != "command-42" {
			t.Errorf("unstable publication metadata: %+v", publication)
		}
	}
	if publications[0].Attempt != 1 || publications[1].Attempt != 2 {
		t.Fatalf("attempts = %d, %d", publications[0].Attempt, publications[1].Attempt)
	}

	var attempts int
	var lastError string
	var lastErrorAt, failedAt *time.Time
	if err := harness.pool.QueryRow(t.Context(), `SELECT attempts, last_error, last_error_at, failed_at
        FROM `+harness.store.tableSQL+` WHERE id = $1`, appended.ID).Scan(
		&attempts, &lastError, &lastErrorAt, &failedAt,
	); err != nil {
		t.Fatalf("inspect failed message: %v", err)
	}
	if attempts != 2 || lastError != publishFailure.Error() || lastErrorAt == nil || failedAt == nil {
		t.Fatalf("failed row: attempts=%d error=%q last_error_at=%v failed_at=%v", attempts, lastError, lastErrorAt, failedAt)
	}
}

func TestPublishedStateIsIdempotentAndNeverLeasedAgain(t *testing.T) {
	harness := newIntegrationHarness(t)
	appended := harness.appendCommitted(t, MessageInput{Topic: "entity.created"})
	leased, err := harness.store.Lease(t.Context(), LeaseRequest{Limit: 1, Duration: time.Minute})
	if err != nil {
		t.Fatalf("Lease: %v", err)
	}
	if len(leased) != 1 || leased[0].ID != appended.ID {
		t.Fatalf("leased = %+v", leased)
	}
	if err := harness.store.MarkPublished(t.Context(), leased[0]); err != nil {
		t.Fatalf("first MarkPublished: %v", err)
	}
	if err := harness.store.MarkPublished(t.Context(), leased[0]); err != nil {
		t.Fatalf("idempotent MarkPublished: %v", err)
	}

	harness.clock.Add(time.Hour)
	again, err := harness.store.Lease(t.Context(), LeaseRequest{Limit: 1, Duration: time.Minute})
	if err != nil {
		t.Fatalf("Lease after publish: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("published message leased again: %+v", again)
	}
}

func TestPublishedAndFailedTransitionsRequireCurrentLeaseToken(t *testing.T) {
	harness := newIntegrationHarness(t)
	harness.appendCommitted(t, MessageInput{Topic: "entity.created", MaxAttempts: 2})
	leased, err := harness.store.Lease(t.Context(), LeaseRequest{Limit: 1, Duration: time.Minute})
	if err != nil {
		t.Fatalf("Lease: %v", err)
	}
	if len(leased) != 1 {
		t.Fatalf("leased = %d, want 1", len(leased))
	}

	stale := leased[0]
	stale.LeaseToken = "not-the-current-token"
	if err := harness.store.MarkPublished(t.Context(), stale); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("MarkPublished with stale token error = %v, want ErrLeaseLost", err)
	}
	if _, err := harness.store.MarkFailed(t.Context(), stale, errors.New("failure"), time.Second); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("MarkFailed with stale token error = %v, want ErrLeaseLost", err)
	}
	if err := harness.store.MarkPublished(t.Context(), leased[0]); err != nil {
		t.Fatalf("MarkPublished with current token: %v", err)
	}
}
