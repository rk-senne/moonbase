package backend

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"
)

// DefaultMaxAttempts is the default number of attempts (initial + retries).
const DefaultMaxAttempts = 3

// WithRetry wraps a function with retry logic using exponential backoff.
// It only retries on transient errors (5xx, timeout, connection refused).
// Non-retryable errors (4xx, validation) fail immediately.
//
// Backoff schedule: 1s, 2s, 4s (base × 2^attempt) with ±25% jitter.
func WithRetry(fn func() (string, error), maxAttempts int) (string, error) {
	return WithRetryCtx(context.Background(), fn, maxAttempts)
}

// WithRetryCtx wraps a function with retry logic, respecting context cancellation.
// When the context is cancelled, the retry loop terminates immediately.
func WithRetryCtx(ctx context.Context, fn func() (string, error), maxAttempts int) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = DefaultMaxAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return "", fmt.Errorf("retry cancelled: %w", ctx.Err())
		}

		output, err := fn()
		if err == nil {
			return output, nil
		}

		lastErr = err

		// Don't retry non-transient errors
		if !IsRetryable(err) {
			return "", err
		}

		// Don't sleep after the last attempt
		if attempt == maxAttempts {
			break
		}

		// Exponential backoff: 1s, 2s, 4s ... capped at 10s
		backoff := backoffDuration(attempt)

		slog.Warn("deploy attempt failed, retrying",
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"backoff", backoff.String(),
			"error", err.Error(),
		)

		// Wait with context awareness
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", fmt.Errorf("retry cancelled during backoff: %w", ctx.Err())
		case <-timer.C:
		}
	}

	return "", fmt.Errorf("all %d attempts failed, last error: %w", maxAttempts, lastErr)
}

// backoffDuration calculates the sleep duration for a given attempt (1-indexed).
// Base = 1s, multiplier = 2x, cap = 10s, jitter = ±25%.
func backoffDuration(attempt int) time.Duration {
	// Base: 1s × 2^(attempt-1) → 1s, 2s, 4s, 8s, ...
	base := time.Second * time.Duration(1<<(attempt-1))

	// Cap at 10s
	const maxBackoff = 10 * time.Second
	if base > maxBackoff {
		base = maxBackoff
	}

	// Apply ±25% jitter
	jitter := float64(base) * 0.25 * (2*rand.Float64() - 1) // range: -25% to +25%
	return base + time.Duration(jitter)
}
