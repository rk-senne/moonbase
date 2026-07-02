package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
	"github.com/f5508037/moonbase/internal/discovery"
	"github.com/f5508037/moonbase/internal/pipeline"
)

// PhaseTimeout is the maximum duration a single pipeline phase can run before being cancelled.
const PhaseTimeout = 120 * time.Second

// PhaseResultMsg is sent when a pipeline phase completes (or fails).
type PhaseResultMsg struct {
	Phase   int
	Output  string
	Err     error
	Elapsed time.Duration
}

// PipelineAbortedMsg is sent when the user aborts the pipeline.
type PipelineAbortedMsg struct{}

// pipelineRunning tracks whether a phase is currently executing.
// Prevents double-dispatch.
var pipelineRunning bool

// executePhase returns a tea.Cmd that runs an agent via the backend.
// It's non-blocking: runs in a goroutine, returns result as PhaseResultMsg.
func executePhase(
	phase pipeline.Phase,
	reg *agents.Registry,
	be backend.Backend,
	projectCtx *discovery.ProjectContext,
	pipelineCtx *pipeline.PipelineContext,
) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()

		agent := reg.GetByName(phase.AgentName)
		if agent == nil {
			return PhaseResultMsg{
				Phase:   phase.Number,
				Err:     fmt.Errorf("agent %s not found", phase.AgentName),
				Elapsed: time.Since(start),
			}
		}

		// Compose the input for this phase
		phaseInput := pipelineCtx.ForPhase(phase.Number)

		// Compose full prompt: project context + agent prompt + phase input
		composed := discovery.ComposePrompt(agent.Prompt, projectCtx, phaseInput)

		// Execute with timeout
		ctx, cancel := context.WithTimeout(context.Background(), PhaseTimeout)
		defer cancel()

		// Run backend deployment in a channel so we can respect timeout
		type result struct {
			output string
			err    error
		}
		ch := make(chan result, 1)
		go func() {
			output, err := be.Deploy(*agent, projectCtx, composed)
			ch <- result{output, err}
		}()

		select {
		case <-ctx.Done():
			return PhaseResultMsg{
				Phase:   phase.Number,
				Err:     fmt.Errorf("phase %d timed out after 120s", phase.Number),
				Elapsed: time.Since(start),
			}
		case r := <-ch:
			return PhaseResultMsg{
				Phase:   phase.Number,
				Output:  r.output,
				Err:     r.err,
				Elapsed: time.Since(start),
			}
		}
	}
}

// startNextPhase determines and starts the next phase in the pipeline.
// Returns the tea.Cmd to execute, or nil if pipeline is complete.
func (a *App) startNextPhase() tea.Cmd {
	if a.pipelineState == nil || !a.pipelineState.Active {
		pipelineRunning = false
		return nil
	}

	// Check if we have a backend that can actually execute
	if a.activeBackend == nil || a.activeBackend.Name() == "clipboard" {
		// No real backend — stay in simulated mode
		pipelineRunning = false
		return nil
	}

	phase := a.pipelineState.CurrentPhase()
	if phase == nil {
		pipelineRunning = false
		return nil
	}

	// Check conditional phases
	if phase.Conditional {
		trigger := a.pipelineState.ShouldInvokeConditional(phase)
		if !trigger.Invoke {
			// Skip this phase
			a.pipelineChat = append(a.pipelineChat,
				PipelineMsg{"", fmt.Sprintf("⏭️ %s — skipped (%s)", phase.Name, trigger.Reason)},
			)
			a.pipelineState.Skip()
			return a.startNextPhase()
		}
		a.pipelineChat = append(a.pipelineChat,
			PipelineMsg{"", fmt.Sprintf("⚡ %s — triggered (%s)", phase.Name, trigger.Reason)},
		)
	}

	// Start the phase
	phase.Status = pipeline.StatusRunning
	pipelineRunning = true

	a.pipelineChat = append(a.pipelineChat,
		PipelineMsg{"", "───────────────────────────────────"},
		PipelineMsg{phase.Operative, fmt.Sprintf("Phase %d: %s starting...", phase.Number, phase.Name)},
	)

	return executePhase(
		*phase,
		a.registry,
		a.activeBackend,
		a.projectCtx,
		a.pipelineState.Context,
	)
}

// handlePhaseResult processes a completed phase and advances the pipeline.
func (a *App) handlePhaseResult(msg PhaseResultMsg) tea.Cmd {
	pipelineRunning = false

	if msg.Err != nil {
		// Phase failed
		a.pipelineChat = append(a.pipelineChat,
			PipelineMsg{"", fmt.Sprintf("❌ Phase %d failed: %v", msg.Phase, msg.Err)},
			PipelineMsg{"", "Press [r] to retry or [s] to skip."},
		)
		if a.pipelineState != nil && a.pipelineState.CurrentPhase() != nil {
			a.pipelineState.CurrentPhase().Status = pipeline.StatusFailed
		}
		return nil
	}

	// Phase succeeded — record output
	a.pipelineState.Context.RecordPhase(msg.Phase, msg.Output)

	// Show summary in chat (truncated)
	summary := strings.TrimSpace(msg.Output)
	if len(summary) > 300 {
		summary = summary[:300] + "..."
	}
	elapsed := msg.Elapsed.Round(time.Millisecond)
	a.pipelineChat = append(a.pipelineChat,
		PipelineMsg{a.pipelineState.CurrentPhase().Operative, summary},
		PipelineMsg{"", fmt.Sprintf("✅ Phase %d complete (%s)", msg.Phase, elapsed)},
	)

	// Apply risk gate if this was QA (phase 4)
	if msg.Phase == 4 {
		routing, err := a.pipelineState.ApplyRiskGate(msg.Output)
		a.pipelineChat = append(a.pipelineChat,
			PipelineMsg{"", fmt.Sprintf("🎯 Risk Gate: %s — %s", routing.Level, routing.Action)},
		)

		if routing.Level == pipeline.RiskCritical {
			a.pipelineChat = append(a.pipelineChat,
				PipelineMsg{"", "🛑 CRITICAL — Pipeline stopped. Human intervention required."},
			)
			return nil
		}
		if err != nil {
			a.pipelineChat = append(a.pipelineChat,
				PipelineMsg{"", fmt.Sprintf("🛑 %v", err)},
			)
			return nil
		}
		if routing.Level != pipeline.RiskLow {
			// Rework — pipeline was already rerouted by ApplyRiskGate
			return a.startNextPhase()
		}
	}

	// Advance to next phase
	a.pipelineState.Advance()

	// Check if pipeline is complete
	if !a.pipelineState.Active {
		a.pipelineChat = append(a.pipelineChat,
			PipelineMsg{"", ""},
			PipelineMsg{"", "━━━ MISSION COMPLETE ━━━"},
		)
		a.addIntel("Mission complete: %s", a.pipelineState.Task)
		return nil
	}

	// Start next phase
	return a.startNextPhase()
}
