package outbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type validationTx struct{ pgx.Tx }

func TestNewStoreValidation(t *testing.T) {
	t.Parallel()

	valid := StoreConfig{DefaultMaxAttempts: 3}
	if _, err := NewStore(nil, valid); !errors.Is(err, ErrNilPool) {
		t.Fatalf("nil pool error = %v, want ErrNilPool", err)
	}
	if _, err := NewStore(new(pgxpool.Pool), StoreConfig{}); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("zero max attempts error = %v, want ErrInvalidMessage", err)
	}
	invalidTable := valid
	invalidTable.Table = pgx.Identifier{""}
	if _, err := NewStore(new(pgxpool.Pool), invalidTable); !errors.Is(err, ErrInvalidTable) {
		t.Fatalf("invalid table error = %v, want ErrInvalidTable", err)
	}
}

func TestAppendRequiresCallerTransaction(t *testing.T) {
	t.Parallel()

	store, err := NewStore(new(pgxpool.Pool), StoreConfig{DefaultMaxAttempts: 3})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if _, err := store.Append(t.Context(), nil, MessageInput{Topic: "created"}); !errors.Is(err, ErrNilTransaction) {
		t.Fatalf("Append error = %v, want ErrNilTransaction", err)
	}
}

func TestAppendValidatesMessageBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	store, err := NewStore(new(pgxpool.Pool), StoreConfig{
		DefaultMaxAttempts: 3,
		Clock: ClockFunc(func() time.Time {
			return time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
		}),
	})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	for _, input := range []MessageInput{
		{Topic: "   "},
		{Topic: "entity.created", MaxAttempts: -1},
	} {
		if _, err := store.Append(context.Background(), &validationTx{}, input); !errors.Is(err, ErrInvalidMessage) {
			t.Errorf("Append(%+v) error = %v, want ErrInvalidMessage", input, err)
		}
	}
}

func TestAppendUsesInjectedClockAndTokenGenerator(t *testing.T) {
	t.Parallel()

	store, err := NewStore(new(pgxpool.Pool), StoreConfig{
		DefaultMaxAttempts: 3,
		Clock: ClockFunc(func() time.Time {
			return time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
		}),
		MessageIDGenerator: TokenGeneratorFunc(func() (string, error) { return "", nil }),
	})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if _, err := store.Append(context.Background(), &validationTx{}, MessageInput{Topic: "entity.created"}); !errors.Is(err, ErrTokenGeneratorEmpty) {
		t.Fatalf("Append error = %v, want ErrTokenGeneratorEmpty", err)
	}

	zeroClockStore, err := NewStore(new(pgxpool.Pool), StoreConfig{
		DefaultMaxAttempts: 3,
		Clock:              ClockFunc(func() time.Time { return time.Time{} }),
	})
	if err != nil {
		t.Fatalf("NewStore with zero clock: %v", err)
	}
	if _, err := zeroClockStore.Append(context.Background(), &validationTx{}, MessageInput{
		ID: "message-1", Topic: "entity.created",
	}); !errors.Is(err, ErrClockReturnedZero) {
		t.Fatalf("Append error = %v, want ErrClockReturnedZero", err)
	}
}

func TestGeneratedTokenRejectsEmptyValue(t *testing.T) {
	t.Parallel()

	_, err := generatedToken(TokenGeneratorFunc(func() (string, error) { return "   ", nil }))
	if !errors.Is(err, ErrTokenGeneratorEmpty) {
		t.Fatalf("generatedToken error = %v, want ErrTokenGeneratorEmpty", err)
	}
}

func TestValidateLeasedMessage(t *testing.T) {
	t.Parallel()

	valid := LeasedMessage{
		Message:        Message{ID: "message", Attempts: 1, MaxAttempts: 3},
		LeaseToken:     "lease",
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}
	if err := validateLeasedMessage(valid); err != nil {
		t.Fatalf("valid lease: %v", err)
	}
	valid.LeaseToken = ""
	if err := validateLeasedMessage(valid); !errors.Is(err, ErrInvalidLease) {
		t.Fatalf("invalid lease error = %v, want ErrInvalidLease", err)
	}
}
