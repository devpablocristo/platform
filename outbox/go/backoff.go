package outbox

import (
	"errors"
	"math"
	randv2 "math/rand/v2"
	"time"
)

var ErrInvalidBackoff = errors.New("outbox: invalid exponential backoff")

// Random supplies values in [0, 1) for backoff jitter. Implementations must be
// safe for concurrent use.
type Random interface {
	Float64() float64
}

// RandomFunc adapts a function to Random.
type RandomFunc func() float64

// Float64 implements Random.
func (f RandomFunc) Float64() float64 { return f() }

// ExponentialBackoffConfig defines explicit retry timing.
type ExponentialBackoffConfig struct {
	Initial    time.Duration
	Maximum    time.Duration
	Multiplier float64
	Jitter     float64
	Random     Random
}

// ExponentialBackoff computes a capped exponential delay. Attempt numbering
// starts at one.
type ExponentialBackoff struct {
	initial    time.Duration
	maximum    time.Duration
	multiplier float64
	jitter     float64
	random     Random
}

// NewExponentialBackoff validates and builds a retry policy.
func NewExponentialBackoff(config ExponentialBackoffConfig) (*ExponentialBackoff, error) {
	if config.Initial <= 0 || config.Maximum < config.Initial || config.Multiplier < 1 ||
		math.IsNaN(config.Multiplier) || math.IsInf(config.Multiplier, 0) ||
		config.Jitter < 0 || config.Jitter > 1 || math.IsNaN(config.Jitter) {
		return nil, ErrInvalidBackoff
	}
	if config.Random == nil {
		config.Random = RandomFunc(randv2.Float64)
	}
	return &ExponentialBackoff{
		initial:    config.Initial,
		maximum:    config.Maximum,
		multiplier: config.Multiplier,
		jitter:     config.Jitter,
		random:     config.Random,
	}, nil
}

// Delay returns the delay for a one-based attempt number.
func (backoff *ExponentialBackoff) Delay(attempt int) time.Duration {
	if backoff == nil {
		return 0
	}
	if attempt < 1 {
		attempt = 1
	}

	delay := float64(backoff.initial)
	maximum := float64(backoff.maximum)
	for current := 1; current < attempt && delay < maximum; current++ {
		if delay >= maximum/backoff.multiplier {
			delay = maximum
			break
		}
		delay *= backoff.multiplier
	}
	if delay > maximum {
		delay = maximum
	}

	if backoff.jitter > 0 {
		random := backoff.random.Float64()
		if math.IsNaN(random) || math.IsInf(random, 0) {
			random = 0.5
		}
		if random < 0 {
			random = 0
		}
		if random > 1 {
			random = 1
		}
		delay *= 1 + backoff.jitter*(2*random-1)
	}
	if delay < 0 {
		return 0
	}
	if delay > maximum {
		delay = maximum
	}
	return time.Duration(delay)
}
