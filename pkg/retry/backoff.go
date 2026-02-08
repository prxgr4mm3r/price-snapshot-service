package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Config defines retry configuration
type Config struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
	Jitter         float64 // Random jitter factor (0-1)
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Multiplier:     2.0,
		Jitter:         0.1,
	}
}

// RetryableError wraps an error that should be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error) *RetryableError {
	return &RetryableError{Err: err}
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	var retryable *RetryableError
	return errors.As(err, &retryable)
}

// Do executes a function with retry logic
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := calculateBackoff(cfg, attempt)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Only retry if error is retryable
		if !IsRetryable(err) {
			return err
		}
	}

	return lastErr
}

// DoWithResult executes a function with retry logic and returns a result
func DoWithResult[T any](ctx context.Context, cfg Config, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := calculateBackoff(cfg, attempt)

			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(backoff):
			}
		}

		var err error
		result, err = fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Only retry if error is retryable
		if !IsRetryable(err) {
			return result, err
		}
	}

	return result, lastErr
}

func calculateBackoff(cfg Config, attempt int) time.Duration {
	backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.Multiplier, float64(attempt-1))

	// Apply max backoff
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	// Apply jitter (random value between -jitter% and +jitter%)
	if cfg.Jitter > 0 {
		jitter := backoff * cfg.Jitter * (rand.Float64()*2 - 1)
		backoff += jitter
	}

	// Ensure backoff is not negative
	if backoff < 0 {
		backoff = float64(cfg.InitialBackoff)
	}

	return time.Duration(backoff)
}
