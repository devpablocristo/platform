package resilience

import (
	"context"
	"errors"
	"time"
)

const (
	defaultAttempts   = 3
	defaultDelay      = 200 * time.Millisecond
	defaultMaxDelay   = 2 * time.Second
	defaultMultiplier = 2.0
)

// Config define retries con backoff exponencial.
type Config struct {
	Attempts     int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	ShouldRetry  func(error) bool
	Sleep        func(context.Context, time.Duration) error
}

// DefaultConfig devuelve defaults razonables para I/O.
func DefaultConfig() Config {
	return Config{
		Attempts:     defaultAttempts,
		InitialDelay: defaultDelay,
		MaxDelay:     defaultMaxDelay,
		Multiplier:   defaultMultiplier,
	}
}

// Retry ejecuta `fn` hasta éxito, agotamiento o cancelación.
func Retry(ctx context.Context, config Config, fn func(context.Context) error) error {
	_, err := RetryValue(ctx, config, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}

// RetryValue ejecuta `fn` y retorna el valor exitoso.
func RetryValue[T any](ctx context.Context, config Config, fn func(context.Context) (T, error)) (T, error) {
	config = normalizeConfig(config)

	var zero T
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.Attempts; attempt++ {
		value, err := fn(ctx)
		if err == nil {
			return value, nil
		}

		lastErr = err
		if !config.ShouldRetry(err) || attempt == config.Attempts {
			return zero, lastErr
		}

		if err := config.Sleep(ctx, delay); err != nil {
			return zero, errors.Join(err, lastErr)
		}
		delay = nextDelay(delay, config)
	}

	return zero, lastErr
}

func normalizeConfig(config Config) Config {
	defaults := DefaultConfig()
	if config.Attempts <= 0 {
		config.Attempts = defaults.Attempts
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = defaults.InitialDelay
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = defaults.MaxDelay
	}
	if config.Multiplier <= 1 {
		config.Multiplier = defaults.Multiplier
	}
	if config.ShouldRetry == nil {
		config.ShouldRetry = func(error) bool { return true }
	}
	if config.Sleep == nil {
		config.Sleep = sleep
	}
	return config
}

func nextDelay(current time.Duration, config Config) time.Duration {
	next := time.Duration(float64(current) * config.Multiplier)
	if next > config.MaxDelay {
		return config.MaxDelay
	}
	if next <= 0 {
		return config.MaxDelay
	}
	return next
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
