package vision

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// Default retry backoff parameters. Exposed as a configuration type so callers
// can override any of them; the zero value of RetryConfig falls back to these.
const (
	defaultRetryAttempts   = 3
	defaultRetryInitial    = 500 * time.Millisecond
	defaultRetryMaxBackoff = 30 * time.Second
	defaultRetryMultiplier = 2.0

	// jitterFloor + jitterRange*rand gives a multiplier in [0.75, 1.25): ±25%.
	jitterFloor = 0.75
	jitterRange = 0.5
)

// RetryConfig controls automatic retry of transient (retryable) failures.
// The zero value is usable and falls back to [DefaultRetryConfig].
type RetryConfig struct {
	// MaxAttempts is the total number of attempts including the first call.
	// Zero defaults to 3.
	MaxAttempts int

	// InitialBackoff is the delay before the first retry. Zero defaults to 500ms.
	InitialBackoff time.Duration

	// MaxBackoff caps the delay between retries. Zero defaults to 30s.
	MaxBackoff time.Duration

	// Multiplier grows the backoff between successive retries. Zero defaults to 2.0.
	Multiplier float64

	// Jitter adds up to ±25% randomness to each delay to avoid thundering herds.
	// Enabled by default (a false RetryConfig still gets defaults via the methods
	// below only for the numeric fields; set Jitter explicitly to disable).
	Jitter bool
}

// DefaultRetryConfig returns a sensible policy: up to 3 attempts, starting at
// 500ms, doubling up to a 30s cap, with jitter.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:    defaultRetryAttempts,
		InitialBackoff: defaultRetryInitial,
		MaxBackoff:     defaultRetryMaxBackoff,
		Multiplier:     defaultRetryMultiplier,
		Jitter:         true,
	}
}

func (c RetryConfig) attempts() int {
	if c.MaxAttempts <= 0 {
		return defaultRetryAttempts
	}

	return c.MaxAttempts
}

func (c RetryConfig) initial() time.Duration {
	if c.InitialBackoff <= 0 {
		return defaultRetryInitial
	}

	return c.InitialBackoff
}

func (c RetryConfig) cap() time.Duration {
	if c.MaxBackoff <= 0 {
		return defaultRetryMaxBackoff
	}

	return c.MaxBackoff
}

func (c RetryConfig) multiplier() float64 {
	if c.Multiplier <= 0 {
		return defaultRetryMultiplier
	}

	return c.Multiplier
}

// delayFor returns the backoff before the retry that follows the given
// zero-based attempt index, applying exponential growth, the cap, and jitter.
func (c RetryConfig) delayFor(attempt int) time.Duration {
	grown := float64(c.initial()) * math.Pow(c.multiplier(), float64(attempt))
	if grown > float64(c.cap()) {
		grown = float64(c.cap())
	}

	if c.Jitter {
		grown *= jitterFloor + jitterRange*rand.Float64() //nolint:gosec // G404: jitter intentionally uses math/rand, not crypto-sensitive
	}

	return time.Duration(grown)
}

// WithRetry runs fn, retrying on transient ([IsRetryable]) errors using
// exponential backoff configured by cfg. Non-retryable errors and successful
// results return immediately. The context is honored between attempts.
//
// It is generic over the result type so it can wrap any analysis method:
//
//	result, err := vision.WithRetry(ctx, vision.DefaultRetryConfig(),
//	    func(ctx context.Context) (*vision.AnalyzeResult, error) {
//	        return agent.Analyze(ctx, "review this", img)
//	    })
func WithRetry[T any](
	ctx context.Context,
	cfg RetryConfig,
	fn func(context.Context) (*T, error),
) (*T, error) {
	var lastErr error

	for attempt := range cfg.attempts() {
		if err := ctx.Err(); err != nil {
			return nil, err //nolint:wrapcheck // context sentinel errors are idiomatic to return raw
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		if !IsRetryable(err) {
			return nil, err
		}

		if attempt == cfg.attempts()-1 {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err() //nolint:wrapcheck // context sentinel errors are idiomatic to return raw
		case <-time.After(cfg.delayFor(attempt)):
		}
	}

	return nil, lastErr
}
