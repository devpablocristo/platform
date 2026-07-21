package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DeliveryStore is the dispatcher-facing persistence contract.
type DeliveryStore interface {
	Lease(context.Context, LeaseRequest) ([]LeasedMessage, error)
	MarkPublished(context.Context, LeasedMessage) error
	MarkFailed(context.Context, LeasedMessage, error, time.Duration) (FailureDisposition, error)
}

// Backoff calculates retry delay from a one-based attempt number.
type Backoff interface {
	Delay(attempt int) time.Duration
}

// DispatcherConfig controls one dispatch batch.
type DispatcherConfig struct {
	BatchSize     int
	LeaseDuration time.Duration
	Backoff       Backoff
}

// Dispatcher leases, publishes, and transitions durable messages.
type Dispatcher struct {
	store     DeliveryStore
	publisher Publisher
	config    DispatcherConfig
}

// DispatchResult summarizes one batch. Retried and Failed refer to publisher
// failures that were persisted successfully.
type DispatchResult struct {
	Leased    int
	Published int
	Retried   int
	Failed    int
}

// NewDispatcher validates and creates a dispatcher.
func NewDispatcher(store DeliveryStore, publisher Publisher, config DispatcherConfig) (*Dispatcher, error) {
	if store == nil {
		return nil, ErrNilDeliveryStore
	}
	if publisher == nil {
		return nil, ErrNilPublisher
	}
	if config.BatchSize < 1 || config.LeaseDuration <= 0 || config.Backoff == nil {
		return nil, ErrInvalidLease
	}
	return &Dispatcher{store: store, publisher: publisher, config: config}, nil
}

// Dispatch processes one leased batch. Publisher errors are persisted, then
// joined into the returned error so callers can observe degraded delivery.
func (dispatcher *Dispatcher) Dispatch(ctx context.Context) (DispatchResult, error) {
	if dispatcher == nil {
		return DispatchResult{}, ErrNilDeliveryStore
	}
	messages, err := dispatcher.store.Lease(ctx, LeaseRequest{
		Limit:    dispatcher.config.BatchSize,
		Duration: dispatcher.config.LeaseDuration,
	})
	if err != nil {
		return DispatchResult{}, err
	}

	result := DispatchResult{Leased: len(messages)}
	var dispatchErr error
	for _, message := range messages {
		publication := Publication{
			MessageID:      message.ID,
			IdempotencyKey: message.IdempotencyKey,
			Topic:          message.Topic,
			Payload:        copyBytes(message.Payload),
			Headers:        copyHeaders(message.Headers),
			Attempt:        message.Attempts,
			CreatedAt:      message.CreatedAt,
		}
		if publishErr := dispatcher.publisher.Publish(ctx, publication); publishErr != nil {
			disposition, markErr := dispatcher.store.MarkFailed(
				ctx,
				message,
				publishErr,
				dispatcher.config.Backoff.Delay(message.Attempts),
			)
			if markErr != nil {
				dispatchErr = errors.Join(dispatchErr, fmt.Errorf("mark message %s failed: %w", message.ID, markErr))
			} else if disposition == FailureAttemptsExhausted {
				result.Failed++
			} else {
				result.Retried++
			}
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("publish message %s: %w", message.ID, publishErr))
			continue
		}
		if err := dispatcher.store.MarkPublished(ctx, message); err != nil {
			dispatchErr = errors.Join(dispatchErr, fmt.Errorf("mark message %s published: %w", message.ID, err))
			continue
		}
		result.Published++
	}
	return result, dispatchErr
}
