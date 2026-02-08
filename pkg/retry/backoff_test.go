package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prxgr4mmer/price-snapshot-service/pkg/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_Success(t *testing.T) {
	callCount := 0
	err := retry.Do(context.Background(), retry.DefaultConfig(), func(ctx context.Context) error {
		callCount++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestDo_RetryOnRetryableError(t *testing.T) {
	callCount := 0
	cfg := retry.Config{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         0,
	}

	err := retry.Do(context.Background(), cfg, func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return retry.NewRetryableError(errors.New("temporary error"))
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestDo_NoRetryOnNonRetryableError(t *testing.T) {
	callCount := 0
	permanentErr := errors.New("permanent error")

	err := retry.Do(context.Background(), retry.DefaultConfig(), func(ctx context.Context) error {
		callCount++
		return permanentErr
	})

	assert.ErrorIs(t, err, permanentErr)
	assert.Equal(t, 1, callCount)
}

func TestDo_ExhaustsRetries(t *testing.T) {
	callCount := 0
	cfg := retry.Config{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         0,
	}

	retryableErr := errors.New("always fails")

	err := retry.Do(context.Background(), cfg, func(ctx context.Context) error {
		callCount++
		return retry.NewRetryableError(retryableErr)
	})

	assert.Error(t, err)
	assert.Equal(t, 3, callCount) // Initial + 2 retries
}

func TestDo_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	cfg := retry.Config{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		Multiplier:     2.0,
		Jitter:         0,
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retry.Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		return retry.NewRetryableError(errors.New("temporary"))
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.LessOrEqual(t, callCount, 2)
}

func TestDoWithResult_Success(t *testing.T) {
	result, err := retry.DoWithResult(context.Background(), retry.DefaultConfig(), func(ctx context.Context) (int, error) {
		return 42, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestDoWithResult_RetryAndSucceed(t *testing.T) {
	callCount := 0
	cfg := retry.Config{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         0,
	}

	result, err := retry.DoWithResult(context.Background(), cfg, func(ctx context.Context) (string, error) {
		callCount++
		if callCount < 2 {
			return "", retry.NewRetryableError(errors.New("temporary"))
		}
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 2, callCount)
}

func TestIsRetryable(t *testing.T) {
	t.Run("retryable error", func(t *testing.T) {
		err := retry.NewRetryableError(errors.New("temporary"))
		assert.True(t, retry.IsRetryable(err))
	})

	t.Run("non-retryable error", func(t *testing.T) {
		err := errors.New("permanent")
		assert.False(t, retry.IsRetryable(err))
	})

	t.Run("wrapped retryable error", func(t *testing.T) {
		inner := retry.NewRetryableError(errors.New("temporary"))
		wrapped := errors.New("wrapper: " + inner.Error())
		// Note: this won't be detected as retryable because we're not using errors.Join
		assert.False(t, retry.IsRetryable(wrapped))
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := retry.DefaultConfig()
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialBackoff)
	assert.Equal(t, 10*time.Second, cfg.MaxBackoff)
	assert.Equal(t, 2.0, cfg.Multiplier)
	assert.Equal(t, 0.1, cfg.Jitter)
}
