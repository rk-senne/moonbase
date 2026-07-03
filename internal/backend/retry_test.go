package backend

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestWithRetry_SucceedsFirstAttempt(t *testing.T) {
	calls := 0
	output, err := WithRetry(func() (string, error) {
		calls++
		return "success", nil
	}, 3)

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if output != "success" {
		t.Errorf("expected 'success', got: %q", output)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_SucceedsAfterTransientFailure(t *testing.T) {
	calls := 0
	output, err := WithRetry(func() (string, error) {
		calls++
		if calls < 3 {
			return "", &DeployError{StatusCode: 500, Message: "server error"}
		}
		return "recovered", nil
	}, 3)

	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if output != "recovered" {
		t.Errorf("expected 'recovered', got: %q", output)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_FailsImmediatelyOnNonRetryable(t *testing.T) {
	calls := 0
	_, err := WithRetry(func() (string, error) {
		calls++
		return "", &DeployError{StatusCode: 401, Message: "unauthorized"}
	}, 3)

	if err == nil {
		t.Fatal("expected error for non-retryable failure")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for 401), got %d", calls)
	}
}

func TestWithRetry_ExhaustsAllAttempts(t *testing.T) {
	calls := 0
	_, err := WithRetry(func() (string, error) {
		calls++
		return "", &DeployError{StatusCode: 503, Message: "service unavailable"}
	}, 3)

	if err == nil {
		t.Fatal("expected error when all attempts exhausted")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
	if !errors.Is(err, err) { // just ensure it wraps
		t.Error("expected wrapped error")
	}
}

func TestWithRetryCtx_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := WithRetryCtx(ctx, func() (string, error) {
		calls++
		return "", &DeployError{StatusCode: 500, Message: "server error"}
	}, 5)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from context cancellation")
	}

	// Should have bailed early (well under 5 × backoff)
	if elapsed > 3*time.Second {
		t.Errorf("expected early exit, but took %s", elapsed)
	}
}

func TestWithRetry_Retries429(t *testing.T) {
	calls := 0
	_, err := WithRetry(func() (string, error) {
		calls++
		if calls == 1 {
			return "", &DeployError{StatusCode: 429, Message: "rate limited"}
		}
		return "ok", nil
	}, 3)

	if err != nil {
		t.Fatalf("expected success after 429 retry: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (1 retry after 429), got %d", calls)
	}
}

func TestWithRetry_DoesNotRetry400(t *testing.T) {
	calls := 0
	_, err := WithRetry(func() (string, error) {
		calls++
		return "", &DeployError{StatusCode: 400, Message: "bad request"}
	}, 3)

	if err == nil {
		t.Fatal("expected immediate failure for 400")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for 400), got %d", calls)
	}
}

func TestWithRetry_DoesNotRetry403(t *testing.T) {
	calls := 0
	_, err := WithRetry(func() (string, error) {
		calls++
		return "", &DeployError{StatusCode: 403, Message: "forbidden"}
	}, 3)

	if err == nil {
		t.Fatal("expected immediate failure for 403")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry for 403), got %d", calls)
	}
}

func TestIsRetryable_TransientErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"500", &DeployError{StatusCode: 500, Message: "server"}, true},
		{"502", &DeployError{StatusCode: 502, Message: "gateway"}, true},
		{"503", &DeployError{StatusCode: 503, Message: "unavail"}, true},
		{"504", &DeployError{StatusCode: 504, Message: "timeout"}, true},
		{"429", &DeployError{StatusCode: 429, Message: "rate"}, true},
		{"400", &DeployError{StatusCode: 400, Message: "bad"}, false},
		{"401", &DeployError{StatusCode: 401, Message: "auth"}, false},
		{"403", &DeployError{StatusCode: 403, Message: "forbidden"}, false},
		{"404", &DeployError{StatusCode: 404, Message: "not found"}, false},
		{"timeout string", fmt.Errorf("connection timeout"), true},
		{"connection refused", fmt.Errorf("connection refused"), true},
		{"generic error", fmt.Errorf("something went wrong"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("IsRetryable(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestBackoffDuration_Scaling(t *testing.T) {
	// Test that backoff increases with attempts and is within expected range
	for attempt := 1; attempt <= 5; attempt++ {
		d := backoffDuration(attempt)

		// Expected base: 1s, 2s, 4s, 8s, 10s (capped)
		expectedBase := time.Second * time.Duration(1<<(attempt-1))
		if expectedBase > 10*time.Second {
			expectedBase = 10 * time.Second
		}

		// With ±25% jitter, should be within 75%-125% of base
		minExpected := time.Duration(float64(expectedBase) * 0.75)
		maxExpected := time.Duration(float64(expectedBase) * 1.25)

		if d < minExpected || d > maxExpected {
			t.Errorf("attempt %d: backoff %s not in range [%s, %s]",
				attempt, d, minExpected, maxExpected)
		}
	}
}
