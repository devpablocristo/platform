package outbox

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestNewExponentialBackoffRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	valid := ExponentialBackoffConfig{
		Initial:    time.Second,
		Maximum:    time.Minute,
		Multiplier: 2,
	}
	tests := []struct {
		name   string
		mutate func(*ExponentialBackoffConfig)
	}{
		{name: "initial", mutate: func(config *ExponentialBackoffConfig) { config.Initial = 0 }},
		{name: "maximum", mutate: func(config *ExponentialBackoffConfig) { config.Maximum = time.Millisecond }},
		{name: "multiplier", mutate: func(config *ExponentialBackoffConfig) { config.Multiplier = 0.5 }},
		{name: "multiplier NaN", mutate: func(config *ExponentialBackoffConfig) { config.Multiplier = math.NaN() }},
		{name: "multiplier infinity", mutate: func(config *ExponentialBackoffConfig) { config.Multiplier = math.Inf(1) }},
		{name: "negative jitter", mutate: func(config *ExponentialBackoffConfig) { config.Jitter = -0.1 }},
		{name: "large jitter", mutate: func(config *ExponentialBackoffConfig) { config.Jitter = 1.1 }},
		{name: "jitter NaN", mutate: func(config *ExponentialBackoffConfig) { config.Jitter = math.NaN() }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := valid
			test.mutate(&config)
			_, err := NewExponentialBackoff(config)
			if !errors.Is(err, ErrInvalidBackoff) {
				t.Fatalf("error = %v, want ErrInvalidBackoff", err)
			}
		})
	}
}

func TestExponentialBackoffDelayAndCap(t *testing.T) {
	t.Parallel()

	backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial:    100 * time.Millisecond,
		Maximum:    800 * time.Millisecond,
		Multiplier: 2,
	})
	if err != nil {
		t.Fatalf("NewExponentialBackoff: %v", err)
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: -1, want: 100 * time.Millisecond},
		{attempt: 1, want: 100 * time.Millisecond},
		{attempt: 2, want: 200 * time.Millisecond},
		{attempt: 3, want: 400 * time.Millisecond},
		{attempt: 4, want: 800 * time.Millisecond},
		{attempt: 100, want: 800 * time.Millisecond},
	}
	for _, test := range tests {
		if got := backoff.Delay(test.attempt); got != test.want {
			t.Errorf("Delay(%d) = %s, want %s", test.attempt, got, test.want)
		}
	}
}

func TestExponentialBackoffUsesInjectableJitter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		random float64
		want   time.Duration
	}{
		{name: "lower bound", random: 0, want: 500 * time.Millisecond},
		{name: "midpoint", random: 0.5, want: time.Second},
		{name: "upper bound", random: 1, want: 1500 * time.Millisecond},
		{name: "NaN falls back to midpoint", random: math.NaN(), want: time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backoff, err := NewExponentialBackoff(ExponentialBackoffConfig{
				Initial:    time.Second,
				Maximum:    2 * time.Second,
				Multiplier: 2,
				Jitter:     0.5,
				Random:     RandomFunc(func() float64 { return test.random }),
			})
			if err != nil {
				t.Fatalf("NewExponentialBackoff: %v", err)
			}
			if got := backoff.Delay(1); got != test.want {
				t.Fatalf("Delay(1) = %s, want %s", got, test.want)
			}
		})
	}
}
