package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryEventuallySucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := Retry(context.Background(), Config{
		Attempts:     3,
		InitialDelay: 1,
		MaxDelay:     1,
		Sleep:        func(context.Context, time.Duration) error { return nil },
	}, func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Retry returned error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("unexpected attempts: %d", attempts)
	}
}

func TestRetryValueStopsOnNonRetryableError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("fatal")
	_, err := RetryValue(context.Background(), Config{
		Attempts:     5,
		InitialDelay: 1,
		MaxDelay:     1,
		ShouldRetry:  func(err error) bool { return !errors.Is(err, sentinel) },
		Sleep:        func(context.Context, time.Duration) error { return nil },
	}, func(context.Context) (string, error) {
		return "", sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}
