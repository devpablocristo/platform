package outbox

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeDeliveryStore struct {
	mu                 sync.Mutex
	messages           []LeasedMessage
	leaseRequest       LeaseRequest
	published          []LeasedMessage
	failed             []LeasedMessage
	failureDelay       time.Duration
	failureDisposition FailureDisposition
	leaseErr           error
	markPublishedErr   error
	markFailedErr      error
}

func (store *fakeDeliveryStore) Lease(_ context.Context, request LeaseRequest) ([]LeasedMessage, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.leaseRequest = request
	return append([]LeasedMessage(nil), store.messages...), store.leaseErr
}

func (store *fakeDeliveryStore) MarkPublished(_ context.Context, message LeasedMessage) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.published = append(store.published, message)
	return store.markPublishedErr
}

func (store *fakeDeliveryStore) MarkFailed(_ context.Context, message LeasedMessage, _ error, delay time.Duration) (FailureDisposition, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.failed = append(store.failed, message)
	store.failureDelay = delay
	return store.failureDisposition, store.markFailedErr
}

func TestNewDispatcherValidation(t *testing.T) {
	t.Parallel()

	store := &fakeDeliveryStore{}
	publisher := PublisherFunc(func(context.Context, Publication) error { return nil })
	backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: time.Second, Maximum: time.Minute, Multiplier: 2,
	})
	if err != nil {
		t.Fatalf("NewExponentialBackoff: %v", err)
	}
	valid := DispatcherConfig{BatchSize: 10, LeaseDuration: time.Minute, Backoff: backoff}
	if _, err := NewDispatcher(nil, publisher, valid); !errors.Is(err, ErrNilDeliveryStore) {
		t.Fatalf("nil store error = %v, want ErrNilDeliveryStore", err)
	}
	if _, err := NewDispatcher(store, nil, valid); !errors.Is(err, ErrNilPublisher) {
		t.Fatalf("nil publisher error = %v, want ErrNilPublisher", err)
	}
	valid.BatchSize = 0
	if _, err := NewDispatcher(store, publisher, valid); !errors.Is(err, ErrInvalidLease) {
		t.Fatalf("invalid config error = %v, want ErrInvalidLease", err)
	}
}

func TestDispatcherPassesStableMetadataAndSchedulesRetry(t *testing.T) {
	t.Parallel()

	message := LeasedMessage{
		Message: Message{
			ID:             "message-1",
			IdempotencyKey: "command-9",
			Topic:          "entity.created",
			Payload:        []byte(`{"id":"9"}`),
			Headers:        map[string]string{"trace": "abc"},
			Attempts:       2,
			MaxAttempts:    5,
			CreatedAt:      time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC),
		},
		LeaseToken:     "lease-1",
		LeaseExpiresAt: time.Date(2026, 7, 21, 12, 1, 0, 0, time.UTC),
	}
	store := &fakeDeliveryStore{
		messages:           []LeasedMessage{message},
		failureDisposition: FailureRetryScheduled,
	}
	var got Publication
	publishFailure := errors.New("broker unavailable")
	publisher := PublisherFunc(func(_ context.Context, publication Publication) error {
		got = publication
		publication.Headers["trace"] = "mutated"
		publication.Payload[0] = 'x'
		return publishFailure
	})
	backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: time.Second, Maximum: time.Minute, Multiplier: 2,
	})
	if err != nil {
		t.Fatalf("NewExponentialBackoff: %v", err)
	}
	dispatcher, err := NewDispatcher(store, publisher, DispatcherConfig{
		BatchSize: 8, LeaseDuration: 30 * time.Second, Backoff: backoff,
	})
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}

	result, err := dispatcher.Dispatch(t.Context())
	if !errors.Is(err, publishFailure) {
		t.Fatalf("Dispatch error = %v, want publisher failure", err)
	}
	if result != (DispatchResult{Leased: 1, Retried: 1}) {
		t.Fatalf("Dispatch result = %+v", result)
	}
	if got.MessageID != message.ID || got.IdempotencyKey != message.IdempotencyKey || got.Attempt != 2 {
		t.Fatalf("publication metadata = %+v", got)
	}
	if store.failureDelay != 2*time.Second {
		t.Fatalf("retry delay = %s, want 2s", store.failureDelay)
	}
	if message.Headers["trace"] != "abc" || string(message.Payload) != `{"id":"9"}` {
		t.Fatal("publisher mutated leased message")
	}
}

func TestDispatcherMarksSuccessfulPublication(t *testing.T) {
	t.Parallel()

	message := LeasedMessage{
		Message:        Message{ID: "message-1", Attempts: 1, MaxAttempts: 2},
		LeaseToken:     "lease-1",
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	store := &fakeDeliveryStore{messages: []LeasedMessage{message}}
	backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: time.Second, Maximum: time.Minute, Multiplier: 2,
	})
	if err != nil {
		t.Fatalf("NewExponentialBackoff: %v", err)
	}
	dispatcher, err := NewDispatcher(
		store,
		PublisherFunc(func(context.Context, Publication) error { return nil }),
		DispatcherConfig{BatchSize: 1, LeaseDuration: time.Minute, Backoff: backoff},
	)
	if err != nil {
		t.Fatalf("NewDispatcher: %v", err)
	}
	result, err := dispatcher.Dispatch(t.Context())
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if result != (DispatchResult{Leased: 1, Published: 1}) || len(store.published) != 1 {
		t.Fatalf("result = %+v, published = %d", result, len(store.published))
	}
}
