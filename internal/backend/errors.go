package backend

import (
	"errors"
	"net"
	"os"
	"strings"
)

// DeployError represents an HTTP error from a backend API call.
// It embeds the status code so retry logic can distinguish transient from permanent failures.
type DeployError struct {
	StatusCode int
	Message    string
}

func (e *DeployError) Error() string {
	return e.Message
}

// IsRetryable determines whether an error is transient and worth retrying.
// Retryable: 429 (rate limit), 5xx (server errors), timeouts, connection refused.
// Non-retryable: 4xx (except 429), validation errors, auth failures.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for DeployError with HTTP status code
	var deployErr *DeployError
	if errors.As(err, &deployErr) {
		switch {
		case deployErr.StatusCode == 429:
			return true // rate limit — retry
		case deployErr.StatusCode >= 500:
			return true // server error — retry
		default:
			return false // 4xx (auth, bad request) — don't retry
		}
	}

	// Check for network-level transient errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	// Check for connection refused
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check for OS-level connection errors (ECONNREFUSED, ECONNRESET)
	var sysErr *os.SyscallError
	if errors.As(err, &sysErr) {
		return true
	}

	// Check error message for common transient patterns
	msg := err.Error()
	transientPatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"no such host", // DNS resolution failure — may be transient
	}
	for _, pattern := range transientPatterns {
		if strings.Contains(strings.ToLower(msg), pattern) {
			return true
		}
	}

	return false
}
