// Package outbox provides a PostgreSQL transactional outbox with leased,
// retryable delivery.
package outbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

var (
	ErrNilPool             = errors.New("outbox: PostgreSQL pool is nil")
	ErrNilTransaction      = errors.New("outbox: transaction is nil")
	ErrNilPublisher        = errors.New("outbox: publisher is nil")
	ErrNilDeliveryStore    = errors.New("outbox: delivery store is nil")
	ErrInvalidTable        = errors.New("outbox: invalid table identifier")
	ErrInvalidMessage      = errors.New("outbox: invalid message")
	ErrInvalidLease        = errors.New("outbox: invalid lease")
	ErrLeaseLost           = errors.New("outbox: lease lost")
	ErrMessageNotFound     = errors.New("outbox: message not found")
	ErrNilFailure          = errors.New("outbox: failure is nil")
	ErrClockReturnedZero   = errors.New("outbox: clock returned zero time")
	ErrTokenGeneratorEmpty = errors.New("outbox: token generator returned an empty token")
)

// Clock supplies operation timestamps. Implementations must be safe for
// concurrent use.
type Clock interface {
	Now() time.Time
}

// ClockFunc adapts a function to Clock.
type ClockFunc func() time.Time

// Now implements Clock.
func (f ClockFunc) Now() time.Time { return f() }

// TokenGenerator creates opaque message IDs or lease tokens. Implementations
// must be safe for concurrent use.
type TokenGenerator interface {
	NewToken() (string, error)
}

// TokenGeneratorFunc adapts a function to TokenGenerator.
type TokenGeneratorFunc func() (string, error)

// NewToken implements TokenGenerator.
func (f TokenGeneratorFunc) NewToken() (string, error) { return f() }

// MessageInput is appended inside a caller-owned transaction. ID and
// IdempotencyKey are optional; generated values remain stable across retries.
type MessageInput struct {
	ID             string
	IdempotencyKey string
	Topic          string
	Payload        []byte
	Headers        map[string]string
	AvailableAt    time.Time
	MaxAttempts    int
}

// Message is the durable outbox representation.
type Message struct {
	ID             string
	IdempotencyKey string
	Topic          string
	Payload        []byte
	Headers        map[string]string
	AvailableAt    time.Time
	Attempts       int
	MaxAttempts    int
	LastAttemptAt  *time.Time
	LastError      string
	LastErrorAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	PublishedAt    *time.Time
	FailedAt       *time.Time
}

// LeasedMessage is a message owned by one dispatcher until LeaseExpiresAt.
type LeasedMessage struct {
	Message
	LeaseToken     string
	LeasedAt       time.Time
	LeaseExpiresAt time.Time
}

// LeaseRequest controls one atomic leasing operation.
type LeaseRequest struct {
	Limit    int
	Duration time.Duration
}

// Publication is the immutable delivery metadata passed to Publisher.
// MessageID and IdempotencyKey are stable across every retry.
type Publication struct {
	MessageID      string
	IdempotencyKey string
	Topic          string
	Payload        []byte
	Headers        map[string]string
	Attempt        int
	CreatedAt      time.Time
}

// Publisher delivers one leased message. Downstream adapters should forward
// IdempotencyKey to transports that support deduplication.
type Publisher interface {
	Publish(context.Context, Publication) error
}

// PublisherFunc adapts a function to Publisher.
type PublisherFunc func(context.Context, Publication) error

// Publish implements Publisher.
func (f PublisherFunc) Publish(ctx context.Context, message Publication) error {
	return f(ctx, message)
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

type cryptoTokenGenerator struct{}

func (cryptoTokenGenerator) NewToken() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

func generatedToken(generator TokenGenerator) (string, error) {
	token, err := generator.NewToken()
	if err != nil {
		return "", err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrTokenGeneratorEmpty
	}
	return token, nil
}

func copyBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}

func copyHeaders(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
