// Kiro backend: deploys agents via the kiro-cli command-line tool.
//
// Split from backends.go because it changes for its own reasons — kiro-cli's flag
// surface, streaming behaviour, and cancellation semantics — independently of the
// other backend integrations.
package backend

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/chat"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// cancelGrace bounds how long a cancelled backend command may keep the caller
// waiting on its output pipes. Descendants of the child process can hold those
// pipes open after the child is killed, so without this the caller blocks until
// they exit on their own — which is how a 5-minute phase timeout was measured
// taking 11 minutes.
const cancelGrace = 5 * time.Second

// Kiro deploys agents via kiro-cli
type Kiro struct {
	// TrustTools enables --trust-all-tools and --no-interactive for headless execution.
	TrustTools bool
}

func (k *Kiro) Name() string    { return "kiro-cli" }
func (k *Kiro) Available() bool { _, err := exec.LookPath("kiro-cli"); return err == nil }

func (k *Kiro) Deploy(agent agents.Agent, context *discovery.ProjectContext, task string) (string, error) {
	composed := discovery.ComposePrompt(agent.Prompt, context, task)
	return k.DeployRaw(composed, task)
}

// DeployRaw sends a pre-composed prompt to kiro-cli without additional prompt composition.
// Use this when the caller has already built the full prompt (e.g., mission pipeline
// which injects per-phase context like file contents and git diffs).
//
// Prefer DeployRawCtx where a deadline matters: this variant cannot be cancelled,
// so a hung kiro-cli blocks the caller until the child exits on its own.
func (k *Kiro) DeployRaw(composed string, task string) (string, error) {
	return k.DeployRawCtx(context.Background(), composed, task)
}

// DeployRawCtx is DeployRaw with cancellation. Implements RawContextDeployer.
//
// exec.CommandContext is what makes the caller's deadline effective: on
// cancellation it kills the child process. DeployRaw previously used exec.Command,
// so a phase timeout could not interrupt kiro-cli and only surfaced after the
// subprocess finished by itself.
func (k *Kiro) DeployRawCtx(ctx context.Context, composed string, task string) (string, error) {
	// The installed kiro-cli accepts the prompt as a positional [INPUT] argument.
	// (Older builds used --system-prompt/--message, which no longer exist and cause
	// "unexpected argument" errors.) `composed` already includes the task via
	// ComposePrompt, so it is passed directly. `task` is retained for signature
	// stability across backends.
	_ = task

	args := []string{"chat"}
	// Enhancement 1: headless execution flags for pipeline/pipe mode
	if k.TrustTools {
		args = append(args, "--trust-all-tools", "--no-interactive")
	}
	// "--" ends flag parsing so a prompt beginning with "---" (ComposePrompt emits
	// "--- PROJECT RULES ---" first) is not misread as a CLI flag.
	args = append(args, "--", composed)
	cmd := exec.CommandContext(ctx, "kiro-cli", args...)
	// SECURITY: SafeEnv prevents leaking user's full environment to child process.
	cmd.Env = SafeEnv()
	// Killing the direct child is not sufficient. kiro-cli spawns descendants
	// (kiro-cli-chat, bun) that inherit stdout, and CombinedOutput blocks on those
	// pipes even after kiro-cli itself is killed — cancellation appeared to do
	// nothing because the read never returned. WaitDelay bounds that wait so the
	// caller regains control shortly after the deadline fires.
	cmd.WaitDelay = cancelGrace

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Surface cancellation distinctly: "signal: killed" alone is misleading when
		// the real cause was the phase deadline expiring.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return string(output), fmt.Errorf("kiro-cli cancelled: %w", ctxErr)
		}
		return string(output), fmt.Errorf("kiro-cli execution: %w\noutput: %s", err, string(output))
	}

	return string(output), nil
}

// DeployNative deploys an agent via kiro-cli's native agent JSON support.
// Uses `kiro-cli chat --agent <name>` which loads the compiled JSON, resolves
// file:// prompt references, and applies Kiro's permission/hook/MCP engine.
//
// SECURITY: When deploying natively, moonbase delegates safety to Kiro's engine.
// SafeEnv is NOT applied — Kiro manages env isolation for the agent session.
// This is intentional: the agent JSON already declares permissions via toolsSettings.
func (k *Kiro) DeployNative(agentName string) ([]string, error) {
	if agentName == "" {
		return nil, fmt.Errorf("agent name is empty")
	}

	args := []string{"chat", "--agent", agentName}
	if k.TrustTools {
		args = append(args, "--trust-all-tools", "--no-interactive")
	}

	return append([]string{"kiro-cli"}, args...), nil
}

// DeployStream implements StreamingBackend for Kiro. It launches kiro-cli with
// the same arguments as DeployRaw, but reads stdout incrementally via StdoutPipe
// + bufio.Scanner, emitting one StreamChunk per line. On process exit, a terminal
// Done chunk (carrying any non-zero-exit error) is emitted. The channel is closed
// after the Done chunk.
//
// Timeout/cancel: exec.CommandContext kills the process on ctx cancellation.
// The goroutine also checks ctx.Done() on each send to avoid blocking on a full channel.
func (k *Kiro) DeployStream(ctx context.Context, agent agents.Agent,
	pc *discovery.ProjectContext, task string) (<-chan chat.StreamChunk, error) {

	// task is the pre-composed prompt (same as DeployRaw receives via composed)
	args := []string{"chat"}
	if k.TrustTools {
		args = append(args, "--trust-all-tools", "--no-interactive")
	}
	args = append(args, "--", task)

	cmd := exec.CommandContext(ctx, "kiro-cli", args...)
	cmd.Env = SafeEnv()
	// See cancelGrace: descendants can hold the pipe open past the child's death.
	cmd.WaitDelay = cancelGrace

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("kiro-cli stdout pipe: %w", err)
	}
	// Fold stderr into stdout for CombinedOutput parity
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("kiro-cli start: %w", err)
	}

	// Unblock the scanner on cancellation. The loop below sits in sc.Scan(),
	// waiting on a pipe that kiro-cli's descendants keep open after kiro-cli is
	// killed — so ctx.Done() alone never gets a chance to be observed and the
	// stream appeared to hang. Closing the pipe makes Scan return, after which the
	// normal ctx.Done() handling runs. WaitDelay cannot cover this case because
	// nothing has called Wait yet.
	watchdogDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = stdout.Close()
		case <-watchdogDone:
		}
	}()

	ch := make(chan chat.StreamChunk)
	go func() {
		defer close(ch)
		defer close(watchdogDone)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 64*1024), 1024*1024) // 1 MB max line

		for sc.Scan() {
			line := sc.Text() + "\n"
			select {
			case ch <- chat.StreamChunk{Text: line}:
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				return
			}
		}

		werr := cmd.Wait()
		select {
		case ch <- chat.StreamChunk{Done: true, Err: werr}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

// Compile-time assertion: Kiro implements StreamingBackend.
var _ StreamingBackend = (*Kiro)(nil)

// Kiro must stay cancellable: without this, DeployComposed silently falls back to
// the un-cancellable RawDeployer path and phase timeouts stop working.
var _ RawContextDeployer = (*Kiro)(nil)
