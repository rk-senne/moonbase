package backend

import (
	"context"
	"strings"
	"testing"
	"time"
)

// A phase deadline must be able to interrupt a running backend. Before
// DeployRawCtx existed, Kiro shelled out with exec.Command, so cancellation could
// not kill the child: the call blocked until kiro-cli exited by itself and the
// expired context only surfaced afterwards. A 5-minute phase timeout was observed
// taking 11 minutes, and with auto-retry a single stuck phase could hold the
// pipeline for most of an hour.

func TestKiroDeployRawCtx_CancellationInterrupts(t *testing.T) {
	// A fake kiro-cli that would otherwise run far longer than the deadline.
	fakeKiroCLI(t, `/bin/sleep 60`)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := (&Kiro{}).DeployRawCtx(ctx, "composed prompt", "task")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error when the context expires")
	}
	// The whole point: we return promptly rather than waiting out the child.
	if elapsed > 10*time.Second {
		t.Errorf("took %s — cancellation did not interrupt the subprocess", elapsed)
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("error should identify cancellation, got %q", err)
	}
}

func TestKiroDeployRawCtx_AlreadyCancelledContext(t *testing.T) {
	fakeKiroCLI(t, `/bin/sleep 60`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before the call

	start := time.Now()
	if _, err := (&Kiro{}).DeployRawCtx(ctx, "composed", "task"); err == nil {
		t.Fatal("expected an error for an already-cancelled context")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("took %s — should fail immediately", elapsed)
	}
}

func TestKiroDeployRawCtx_SucceedsWithinDeadline(t *testing.T) {
	fakeKiroCLI(t, `printf 'agent output\n'`)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := (&Kiro{}).DeployRawCtx(ctx, "composed", "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "agent output") {
		t.Errorf("output = %q", out)
	}
}

// DeployRaw must keep working for callers that have no context.
func TestKiroDeployRaw_StillWorks(t *testing.T) {
	fakeKiroCLI(t, `printf 'legacy path\n'`)

	out, err := (&Kiro{}).DeployRaw("composed", "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "legacy path") {
		t.Errorf("output = %q", out)
	}
}

// DeployComposed must select the cancellable path, or the fix is inert.
func TestKiro_IsPreferredAsContextDeployer(t *testing.T) {
	var be Backend = &Kiro{}

	if _, ok := be.(RawUsageReporter); ok {
		t.Fatal("Kiro implements RawUsageReporter, which DeployComposed prefers " +
			"over RawContextDeployer — the cancellable path would be skipped")
	}
	if _, ok := be.(RawContextDeployer); !ok {
		t.Fatal("Kiro must implement RawContextDeployer so phase timeouts are enforceable")
	}
}
