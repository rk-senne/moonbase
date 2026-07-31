package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/logging"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// PhaseTimeout is kept as a package-level default for backward compatibility in tests.
// The actual timeout used during execution comes from the pipeline's configured PhaseTimeout.
var PhaseTimeout = 120 * time.Second

// PhaseResultMsg is sent when a pipeline phase completes (or fails).
type PhaseResultMsg struct {
	Phase   int
	Output  string
	Err     error
	Elapsed time.Duration
}

// FanOutCompleteMsg carries all specialist results as a single batch from the
// fan-out goroutine back to the Elm update loop. This replaces N individual
// PhaseResultMsg events with a single atomic update for deterministic merging.
type FanOutCompleteMsg struct {
	Results []pipeline.FanOutResult
}

// PipelineAbortedMsg is sent when the user aborts the pipeline.
type PipelineAbortedMsg struct{}

// pipelineRunning is tracked on the App struct to prevent double-dispatch.

// executePhase returns a tea.Cmd that runs an agent via the backend.
// It's non-blocking: runs in a goroutine, returns result as PhaseResultMsg.
func executePhase(
	ctx context.Context,
	phase pipeline.Phase,
	reg *agents.Registry,
	be backend.Backend,
	projectCtx *discovery.ProjectContext,
	pipelineCtx *pipeline.PipelineContext,
	phaseTimeout time.Duration,
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
		timeoutCtx, cancel := context.WithTimeout(ctx, phaseTimeout)
		defer cancel()

		// Run backend deployment in a channel so we can respect timeout
		type result struct {
			output string
			err    error
		}
		ch := make(chan result, 1)
		go func() {
			// Wrap with retry for transient failures (5xx, timeout, connection refused).
			// Clipboard backend is never retried (local operation).
			if be.Name() == "clipboard" {
				output, err := be.Deploy(*agent, projectCtx, composed)
				ch <- result{output, err}
			} else {
				output, err := backend.WithRetryCtx(timeoutCtx, func() (string, error) {
					return be.Deploy(*agent, projectCtx, composed)
				}, backend.DefaultMaxAttempts)
				ch <- result{output, err}
			}
		}()

		select {
		case <-timeoutCtx.Done():
			if ctx.Err() != nil {
				return PhaseResultMsg{
					Phase:   phase.Number,
					Err:     fmt.Errorf("phase %d cancelled", phase.Number),
					Elapsed: time.Since(start),
				}
			}
			return PhaseResultMsg{
				Phase:   phase.Number,
				Err:     fmt.Errorf("phase %d timed out after %s", phase.Number, phaseTimeout),
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
	if a.views.Pipeline.State == nil || !a.views.Pipeline.State.Active {
		a.views.Pipeline.Running = false
		return nil
	}

	// Check if we have a backend that can actually execute
	if a.env.Backend.Active == nil || a.env.Backend.Active.Name() == "clipboard" {
		// No real backend — stay in simulated mode
		a.views.Pipeline.Running = false
		return nil
	}

	// Ensure pipeline context exists for graceful shutdown
	if a.views.Pipeline.Ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		a.views.Pipeline.Ctx = ctx
		a.views.Pipeline.Cancel = cancel
	}

	phase := a.views.Pipeline.State.CurrentPhase()
	if phase == nil {
		a.views.Pipeline.Running = false
		return nil
	}

	// Check conditional phases
	if phase.Conditional {
		trigger := a.views.Pipeline.State.ShouldInvokeConditional(phase)
		if !trigger.Invoke {
			// Skip this phase
			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{"", fmt.Sprintf("⏭️ %s — skipped (%s)", phase.Name, trigger.Reason)},
			)
			a.views.Pipeline.State.Skip()
			return a.startNextPhase()
		}
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", fmt.Sprintf("⚡ %s — triggered (%s)", phase.Name, trigger.Reason)},
		)
	}

	// Start the phase (records start time for elapsed tracking)
	phase.StartPhase()
	a.views.Pipeline.Running = true

	if logging.Logger != nil {
		logging.Logger.Info("pipeline phase starting",
			"phase", phase.Number,
			"name", phase.Name,
			"operative", phase.Operative,
		)
	}

	a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
		PipelineMsg{"", "───────────────────────────────────"},
		PipelineMsg{phase.Operative, fmt.Sprintf("Phase %d: %s starting...", phase.Number, phase.Name)},
	)

	return executePhase(
		a.views.Pipeline.Ctx,
		*phase,
		a.registry,
		a.env.Backend.Active,
		a.projectCtx,
		a.views.Pipeline.State.Context,
		a.views.Pipeline.State.PhaseTimeout,
	)
}

// handlePhaseResult processes a completed phase and advances the pipeline.
func (a *App) handlePhaseResult(msg PhaseResultMsg) tea.Cmd {
	a.views.Pipeline.Running = false

	if msg.Err != nil {
		// Phase failed
		if logging.Logger != nil {
			logging.Logger.Error("pipeline phase failed",
				"phase", msg.Phase,
				"error", msg.Err.Error(),
				"elapsed", msg.Elapsed.String(),
			)
		}
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", fmt.Sprintf("❌ Phase %d failed: %v", msg.Phase, msg.Err)},
			PipelineMsg{"", "Press [r] to retry or [s] to skip."},
		)
		if a.views.Pipeline.State != nil && a.views.Pipeline.State.CurrentPhase() != nil {
			a.views.Pipeline.State.CurrentPhase().Status = pipeline.StatusFailed
		}
		return nil
	}

	// Phase succeeded — record output and mark completion time
	if logging.Logger != nil {
		logging.Logger.Info("pipeline phase complete",
			"phase", msg.Phase,
			"elapsed", msg.Elapsed.String(),
		)
	}
	// Mark phase as complete with timing
	if a.views.Pipeline.State != nil && a.views.Pipeline.State.CurrentPhase() != nil {
		a.views.Pipeline.State.CurrentPhase().CompletePhase()
	}
	a.views.Pipeline.State.Context.RecordPhase(msg.Phase, msg.Output)

	// Send cmux notification on phase completion
	if backend.CmuxAvailable() {
		phase := a.views.Pipeline.State.CurrentPhase()
		if phase != nil {
			backend.CmuxNotify(
				fmt.Sprintf("Phase %d Complete", msg.Phase),
				fmt.Sprintf("%s — %s", phase.Name, phase.Operative),
			)
		}
	}

	// Show summary in chat (truncated)
	summary := strings.TrimSpace(msg.Output)
	if len(summary) > maxSummaryChars {
		summary = summary[:maxSummaryChars] + "..."
	}
	elapsed := msg.Elapsed.Round(time.Millisecond)
	a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
		PipelineMsg{a.views.Pipeline.State.CurrentPhase().Operative, summary},
		PipelineMsg{"", fmt.Sprintf("✅ Phase %d complete (%s)", msg.Phase, elapsed)},
	)

	// Apply risk gate if this was QA (phase 4)
	if msg.Phase == 4 {
		routing, err := a.views.Pipeline.State.ApplyRiskGate(msg.Output)
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", fmt.Sprintf("🎯 Risk Gate: %s — %s", routing.Level, routing.Action)},
		)

		if routing.Level == pipeline.RiskCritical {
			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{"", "🛑 CRITICAL — Pipeline stopped. Human intervention required."},
			)
			// Notify via cmux on critical risk
			if backend.CmuxAvailable() {
				backend.CmuxNotify("🛑 CRITICAL Risk", "Pipeline stopped. Human intervention required.")
			}
			return nil
		}
		if err != nil {
			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{"", fmt.Sprintf("🛑 %v", err)},
			)
			return nil
		}
		if routing.Level != pipeline.RiskLow {
			// Rework — pipeline was already rerouted by ApplyRiskGate
			return a.startNextPhase()
		}

		// RiskLow — dispatch parallel specialists if enabled.
		if a.views.Pipeline.State.ParallelSpecialists {
			return a.startFanOut()
		}
	}

	// Advance to next phase
	a.views.Pipeline.State.Advance()

	// Check if pipeline is complete
	if !a.views.Pipeline.State.Active {
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", ""},
			PipelineMsg{"", "━━━ MISSION COMPLETE ━━━"},
		)
		a.addIntel("Mission complete: %s", a.views.Pipeline.State.Task)
		// Notify via cmux on mission completion
		if backend.CmuxAvailable() {
			backend.CmuxNotify("━━━ MISSION COMPLETE ━━━", a.views.Pipeline.State.Task)
		}
		return nil
	}

	// Start next phase
	return a.startNextPhase()
}

// startFanOut dispatches parallel specialist execution as a tea.Cmd.
// It identifies triggered specialists from the pipeline's conditional phases,
// runs them concurrently via RunSpecialists, and returns a FanOutCompleteMsg.
func (a *App) startFanOut() tea.Cmd {
	return func() tea.Msg {
		state := a.views.Pipeline.State
		if state == nil {
			return FanOutCompleteMsg{Results: nil}
		}

		// Identify conditional phases that are pending (eligible for fan-out).
		var candidates []pipeline.Phase
		for _, p := range state.Phases {
			if p.Conditional && p.Status == pipeline.StatusPending {
				candidates = append(candidates, p)
			}
		}

		// Filter to only triggered specialists.
		triggered := pipeline.TriggeredSpecialists(candidates, state.Context)

		if len(triggered) == 0 {
			return FanOutCompleteMsg{Results: nil}
		}

		cfg := pipeline.FanOutConfig{
			MaxConcurrency: state.MaxSpecialistConcurrency,
			PhaseTimeout:   state.PhaseTimeout,
			TraceID:        state.TraceID,
		}

		// Build the execute function wrapping the backend call.
		execute := func(ctx context.Context, phase pipeline.Phase) (string, error) {
			agent := a.registry.GetByName(phase.AgentName)
			if agent == nil {
				return "", fmt.Errorf("agent %s not found", phase.AgentName)
			}
			phaseInput := state.Context.ForPhase(phase.Number)
			composed := discovery.ComposePrompt(agent.Prompt, a.projectCtx, phaseInput)

			output, _, err := backend.DeployComposed(ctx, composed, phaseInput, state.PhaseTimeout)
			return output, err
		}

		results := pipeline.RunSpecialists(a.views.Pipeline.Ctx, triggered, execute, cfg)
		return FanOutCompleteMsg{Results: results}
	}
}

// handleFanOutComplete processes the batch result from parallel specialist execution.
// It merges results into the pipeline context, appends per-specialist chat messages,
// marks specialist phases appropriately, and advances to Phase 5 (Review).
func (a *App) handleFanOutComplete(msg FanOutCompleteMsg) tea.Cmd {
	a.views.Pipeline.Running = false

	if a.views.Pipeline.State == nil {
		return nil
	}

	state := a.views.Pipeline.State

	if len(msg.Results) == 0 {
		// No specialists triggered — skip to Phase 5 directly.
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", "  ⏭️ No specialists triggered — advancing to Review."},
		)
	} else {
		// Merge results into pipeline context (deterministic, phase-order).
		state.Context.MergeSpecialistResults(msg.Results)

		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", fmt.Sprintf("⚡ Fan-out complete: %d specialist(s)", len(msg.Results))},
		)

		// Mark phases and append per-specialist chat.
		for _, r := range msg.Results {
			// Find the phase in the pipeline and update its status.
			for i := range state.Phases {
				if state.Phases[i].Number == r.Phase {
					if r.Err != nil {
						state.Phases[i].Status = pipeline.StatusFailed
						state.Phases[i].Summary = r.Err.Error()
					} else {
						state.Phases[i].CompletePhase()
					}
					break
				}
			}

			// Chat message per specialist.
			if r.Err != nil {
				a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
					PipelineMsg{"", fmt.Sprintf("  ❌ %s (Phase %d) — failed: %v", r.Agent, r.Phase, r.Err)},
				)
			} else {
				elapsed := r.Duration.Round(time.Millisecond)
				a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
					PipelineMsg{"", fmt.Sprintf("  ✅ %s (Phase %d) — %s", r.Agent, r.Phase, elapsed)},
				)
			}
		}
	}

	// Skip untriggered conditional phases (mark as skipped).
	for i := range state.Phases {
		if state.Phases[i].Conditional && state.Phases[i].Status == pipeline.StatusPending {
			state.Phases[i].Status = pipeline.StatusSkipped
		}
	}

	// Advance to Phase 5 (Review).
	// Find the Review phase index and set current to it.
	for i := range state.Phases {
		if state.Phases[i].Number == 5 && state.Phases[i].Status == pipeline.StatusPending {
			state.Current = i
			state.Phases[i].Status = pipeline.StatusRunning
			a.views.Pipeline.Running = true
			return a.startNextPhaseFrom(i)
		}
	}

	// If Review is already complete or not found, mark pipeline done.
	state.Active = false
	a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
		PipelineMsg{"", ""},
		PipelineMsg{"", "━━━ MISSION COMPLETE ━━━"},
	)
	return nil
}

// startNextPhaseFrom starts a specific phase by index (used after fan-out to jump to Review).
func (a *App) startNextPhaseFrom(idx int) tea.Cmd {
	if a.views.Pipeline.State == nil || !a.views.Pipeline.State.Active {
		a.views.Pipeline.Running = false
		return nil
	}

	if a.env.Backend.Active == nil || a.env.Backend.Active.Name() == "clipboard" {
		a.views.Pipeline.Running = false
		return nil
	}

	if a.views.Pipeline.Ctx == nil {
		ctx, cancel := context.WithCancel(context.Background())
		a.views.Pipeline.Ctx = ctx
		a.views.Pipeline.Cancel = cancel
	}

	phase := &a.views.Pipeline.State.Phases[idx]
	phase.StartPhase()
	a.views.Pipeline.Running = true

	if logging.Logger != nil {
		logging.Logger.Info("pipeline phase starting (post fan-out)",
			"phase", phase.Number,
			"name", phase.Name,
			"operative", phase.Operative,
		)
	}

	a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
		PipelineMsg{"", "───────────────────────────────────"},
		PipelineMsg{phase.Operative, fmt.Sprintf("Phase %d: %s starting...", phase.Number, phase.Name)},
	)

	return executePhase(
		a.views.Pipeline.Ctx,
		*phase,
		a.registry,
		a.env.Backend.Active,
		a.projectCtx,
		a.views.Pipeline.State.Context,
		a.views.Pipeline.State.PhaseTimeout,
	)
}
