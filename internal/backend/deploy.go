package backend

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	clip "github.com/rk-senne/moonbase/internal/clipboard"
)

// DeployComposed deploys a pre-composed prompt via the preferred backend with retry.
// Uses WithRetryCtx for exponential backoff with jitter.
//
// If the backend implements RawUsageReporter, it sends the pre-composed prompt
// directly and captures token usage. Falls back to RawDeployer (no usage),
// UsageReporter, and finally Backend.Deploy (no usage).
//
// The timeout parameter bounds the total retry budget: the overall context
// deadline is set to timeout, and per-attempt timeouts are derived as
// timeout / DefaultMaxAttempts (capped at a sensible minimum of 30s).
//
// The parent ctx is respected for cancellation; if it is cancelled (e.g. SIGINT),
// the derived context will be cancelled immediately, aborting any in-flight call.
//
// Falls back to clipboard + stdin if no AI backend is available.
// SECURITY: All subprocess execution uses SafeEnv() via the backend implementations.
func DeployComposed(ctx context.Context, composed, task string, timeout time.Duration) (string, *UsageInfo, error) {
	be := Preferred()

	if be.Name() != "clipboard" {
		// Derive per-attempt timeout from the phase timeout so the total retry
		// budget never exceeds the phase timeout.
		perAttempt := timeout / time.Duration(DefaultMaxAttempts)
		if perAttempt < 30*time.Second {
			perAttempt = 30 * time.Second
		}

		deployCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		var capturedUsage *UsageInfo

		output, err := WithRetryCtx(deployCtx, func() (string, error) {
			attemptCtx, attemptCancel := context.WithTimeout(deployCtx, perAttempt)
			defer attemptCancel()

			var result string
			var deployErr error

			// Prefer RawUsageReporter > RawContextDeployer > RawDeployer > UsageReporter > Backend.Deploy.
			//
			// RawContextDeployer sits above RawDeployer because it is the only
			// non-streaming path that can honour attemptCtx: without a context the
			// backend's subprocess cannot be killed, so the deadline below can only
			// be observed after the child exits on its own.
			if raw, ok := be.(RawUsageReporter); ok {
				var usage *UsageInfo
				result, usage, deployErr = raw.DeployRawWithUsage(composed, task)
				if deployErr == nil {
					capturedUsage = usage
				}
			} else if raw, ok := be.(RawContextDeployer); ok {
				result, deployErr = raw.DeployRawCtx(attemptCtx, composed, task)
				// usage stays nil
			} else if raw, ok := be.(RawDeployer); ok {
				result, deployErr = raw.DeployRaw(composed, task)
				// usage stays nil
			} else if ur, ok := be.(UsageReporter); ok {
				var usage *UsageInfo
				result, usage, deployErr = ur.DeployWithUsage(agents.Agent{}, nil, composed)
				if deployErr == nil {
					capturedUsage = usage
				}
			} else {
				// Fallback: pass composed prompt as task with a zero-value agent.
				// This causes a double-composition, but it's the best we can do
				// for backends that don't implement RawDeployer.
				result, deployErr = be.Deploy(agents.Agent{}, nil, composed)
			}

			if deployErr != nil {
				if attemptCtx.Err() == context.DeadlineExceeded {
					return "", fmt.Errorf("backend timed out after %s: %w", perAttempt, deployErr)
				}
				return "", deployErr
			}

			if attemptCtx.Err() != nil {
				return "", fmt.Errorf("deploy cancelled: %w", attemptCtx.Err())
			}
			return result, nil
		}, DefaultMaxAttempts)

		if err != nil {
			return "", nil, fmt.Errorf("%s failed after %d attempts: %w", be.Name(), DefaultMaxAttempts, err)
		}
		return output, capturedUsage, nil
	}

	// Clipboard backend — fall back to clipboard/stdin
	output, err := clipboardFallback(composed, task)
	return output, nil, err
}

// clipboardFallback handles the case where no AI backend (kiro-cli) is available.
// It copies the composed prompt to the clipboard and reads the response from stdin.
// Returns the user-provided response or an error if no fallback mechanism is available.
func clipboardFallback(composed, task string) (string, error) {
	if err := clip.Copy(composed); err == nil {
		fmt.Printf("\n   📋 Prompt copied to clipboard (%d chars).\n", len(composed))
		fmt.Printf("   Paste into your AI tool. When done, paste the response below.\n")
		fmt.Printf("   (Type END on a line by itself to finish, or press Ctrl+C to abort)\n\n")

		// Read multi-line input until "END" with size limit
		var lines []string
		totalSize := 0
		const maxInputSize = 1 << 20 // 1MB
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "END" {
				break
			}
			totalSize += len(line) + 1
			if totalSize > maxInputSize {
				fmt.Fprintf(os.Stderr, "   ⚠️  Input truncated at 1MB\n")
				break
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n"), nil
	}

	return "", fmt.Errorf("no backend available — install kiro-cli or ensure clipboard is accessible")
}
