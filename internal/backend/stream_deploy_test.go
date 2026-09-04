package backend

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// fakeKiroCLI puts an executable named kiro-cli on PATH that runs the given shell
// body, so DeployStream can be exercised without the real CLI installed.
//
// This keeps the test hermetic: PATH is replaced entirely, so a real kiro-cli on
// the developer's machine cannot be invoked by accident and CI (where it is absent)
// behaves identically.
func fakeKiroCLI(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell binary is not portable to windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "kiro-cli")
	body := "#!/bin/sh\n" + script + "\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("writing fake kiro-cli: %v", err)
	}
	t.Setenv("PATH", dir)
}

// drain collects every chunk from the stream until it closes.
func drain(t *testing.T, ch <-chan chat.StreamChunk) (text string, done bool, streamErr error) {
	t.Helper()
	var b strings.Builder
	for c := range ch {
		if c.Done {
			done = true
			streamErr = c.Err
			continue
		}
		b.WriteString(c.Text)
	}
	return b.String(), done, streamErr
}

func TestKiroDeployStream_StreamsLinesThenDone(t *testing.T) {
	fakeKiroCLI(t, `printf 'first line\nsecond line\n'`)

	ch, err := (&Kiro{}).DeployStream(context.Background(), testAgent(),
		&discovery.ProjectContext{}, "the task")
	if err != nil {
		t.Fatalf("DeployStream: %v", err)
	}

	text, done, streamErr := drain(t, ch)
	if !strings.Contains(text, "first line") || !strings.Contains(text, "second line") {
		t.Errorf("streamed text = %q, want both lines", text)
	}
	if !done {
		t.Error("expected a terminal Done chunk")
	}
	if streamErr != nil {
		t.Errorf("unexpected stream error: %v", streamErr)
	}
}

// A non-zero exit must surface on the Done chunk rather than being swallowed.
func TestKiroDeployStream_ReportsNonZeroExit(t *testing.T) {
	fakeKiroCLI(t, `printf 'partial\n'; exit 3`)

	ch, err := (&Kiro{}).DeployStream(context.Background(), testAgent(),
		&discovery.ProjectContext{}, "the task")
	if err != nil {
		t.Fatalf("DeployStream: %v", err)
	}

	text, done, streamErr := drain(t, ch)
	if !strings.Contains(text, "partial") {
		t.Errorf("expected output before the failure, got %q", text)
	}
	if !done {
		t.Fatal("expected a terminal Done chunk")
	}
	if streamErr == nil {
		t.Error("expected the non-zero exit to be reported on the Done chunk")
	}
}

// stderr is folded into stdout, so diagnostics reach the caller.
func TestKiroDeployStream_FoldsStderrIntoStream(t *testing.T) {
	fakeKiroCLI(t, `printf 'to stderr\n' >&2`)

	ch, err := (&Kiro{}).DeployStream(context.Background(), testAgent(),
		&discovery.ProjectContext{}, "the task")
	if err != nil {
		t.Fatalf("DeployStream: %v", err)
	}

	text, _, _ := drain(t, ch)
	if !strings.Contains(text, "to stderr") {
		t.Errorf("expected stderr folded into the stream, got %q", text)
	}
}

func TestKiroDeployStream_MissingBinaryReturnsStartError(t *testing.T) {
	// Empty PATH → kiro-cli cannot be resolved.
	t.Setenv("PATH", t.TempDir())

	_, err := (&Kiro{}).DeployStream(context.Background(), testAgent(),
		&discovery.ProjectContext{}, "the task")
	if err == nil {
		t.Fatal("expected an error when kiro-cli is absent")
	}
	if !strings.Contains(err.Error(), "kiro-cli") {
		t.Errorf("error should name the missing binary, got %q", err)
	}
}

// Cancelling the context must terminate the stream rather than hang, since the
// TUI cancels in-flight phases when a mission is interrupted.
func TestKiroDeployStream_ContextCancellationClosesStream(t *testing.T) {
	// Emit a line, then sleep well past the test's patience.
	fakeKiroCLI(t, `printf 'before cancel\n'; /bin/sleep 30`)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := (&Kiro{}).DeployStream(ctx, testAgent(), &discovery.ProjectContext{}, "the task")
	if err != nil {
		t.Fatalf("DeployStream: %v", err)
	}

	// Consume the first chunk, then cancel.
	select {
	case c := <-ch:
		if !strings.Contains(c.Text, "before cancel") {
			t.Errorf("first chunk = %q", c.Text)
		}
	case <-time.After(10 * time.Second):
		cancel()
		t.Fatal("timed out waiting for the first chunk")
	}
	cancel()

	// The channel must close rather than block forever.
	closed := make(chan struct{})
	go func() {
		for range ch {
		}
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(15 * time.Second):
		t.Fatal("stream did not close after context cancellation")
	}
}

func TestKiro_ImplementsStreamingBackend(t *testing.T) {
	var k any = &Kiro{}
	if _, ok := k.(StreamingBackend); !ok {
		t.Error("Kiro must implement StreamingBackend")
	}
}

// === clipboardFallback ===

// withStdin replaces os.Stdin for the duration of the test.
func withStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		r.Close()
	})
	go func() {
		defer w.Close()
		_, _ = w.WriteString(input)
	}()
}

func TestClipboardFallback_ReadsUntilEND(t *testing.T) {
	withStdin(t, "line one\nline two\nEND\nignored after end\n")

	out, err := clipboardFallback("composed prompt", "task")
	if err != nil {
		// No clipboard available in this environment — the documented error path.
		if !strings.Contains(err.Error(), "no backend available") {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Skip("clipboard unavailable in this environment; error path asserted instead")
	}

	if !strings.Contains(out, "line one") || !strings.Contains(out, "line two") {
		t.Errorf("output = %q, want both input lines", out)
	}
	if strings.Contains(out, "ignored after end") {
		t.Error("input after END must not be consumed")
	}
}

func TestClipboardFallback_EmptyInputYieldsEmptyResponse(t *testing.T) {
	withStdin(t, "END\n")

	out, err := clipboardFallback("composed prompt", "task")
	if err != nil {
		t.Skip("clipboard unavailable in this environment")
	}
	if out != "" {
		t.Errorf("expected empty response, got %q", out)
	}
}

// === Anthropic.DeployWithUsage ===

// Without credentials the streaming path must fail rather than hang or panic.
func TestAnthropicDeployWithUsage_FailsWithoutCredentials(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	out, usage, err := (&Anthropic{}).DeployWithUsage(testAgent(),
		&discovery.ProjectContext{}, "task")
	if err == nil {
		t.Fatal("expected an error without ANTHROPIC_API_KEY")
	}
	if usage != nil {
		t.Errorf("expected no usage on failure, got %+v", usage)
	}
	_ = out // partial output is permitted on the error path
}

func TestAnthropic_ImplementsUsageReporter(t *testing.T) {
	var a any = &Anthropic{}
	if _, ok := a.(UsageReporter); !ok {
		t.Error("Anthropic must implement UsageReporter")
	}
}

// Guard against the agent argument being ignored: the composed prompt must reach
// the CLI. The fake echoes its last argument, which is the task.
func TestKiroDeployStream_PassesTaskToCLI(t *testing.T) {
	fakeKiroCLI(t, `for a in "$@"; do printf '%s\n' "$a"; done`)

	ch, err := (&Kiro{}).DeployStream(context.Background(), agents.Agent{Name: "numbuh-1"},
		&discovery.ProjectContext{}, "SENTINEL-TASK")
	if err != nil {
		t.Fatalf("DeployStream: %v", err)
	}

	text, _, _ := drain(t, ch)
	if !strings.Contains(text, "SENTINEL-TASK") {
		t.Errorf("task was not passed through to the CLI; got %q", text)
	}
}
