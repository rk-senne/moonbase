package tui

// Automatic phase retry with exponential backoff.
//
// A failed phase is retried up to MaxRetries times before the pipeline stops and
// asks the human. Retry lives here rather than in pipeline_exec.go because it
// changes for a different reason: backoff policy and retry accounting, not phase
// execution.

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// PhaseRetryMsg is sent after a backoff delay to trigger an automatic retry
// of a failed phase. The pipeline retries up to MaxRetries times with
// exponential backoff before stopping and asking the user.
type PhaseRetryMsg struct {
	Phase   int
	Attempt int // 1-indexed: which retry attempt this is
	Gen     int // owning mission generation
}

// scheduleRetry schedules a delayed re-run of a failed phase and returns the
// command that will deliver the PhaseRetryMsg.
//
// It returns nil when no retry should happen — either there is no pipeline state
// or the retry budget for this phase is spent — which is the caller's signal to
// fall through to asking the human.
func (a *App) scheduleRetry(msg PhaseResultMsg) tea.Cmd {
	state := a.views.Pipeline.State
	if state != nil {
		attempt := state.Retries[msg.Phase] + 1
		if attempt <= state.MaxRetries {
			// Auto-retry: compute backoff and schedule a delayed retry.
			cfg := config.Load()
			baseMs := cfg.RetryBackoffBase
			if baseMs <= 0 {
				baseMs = 1000
			}
			backoffMs := baseMs * (1 << (attempt - 1)) // exponential: 1s, 2s, 4s...
			backoff := time.Duration(backoffMs) * time.Millisecond

			state.Retries[msg.Phase] = attempt

			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{"", fmt.Sprintf("⚠️ Phase %d failed: %v", msg.Phase, msg.Err)},
				PipelineMsg{"", fmt.Sprintf("🔄 Auto-retrying (%d/%d) in %s...", attempt, state.MaxRetries, backoff.Round(time.Millisecond))},
			)

			// Log the retry to flywheel. CurrentPhase and Context are read
			// defensively: this runs on a failure path, where a panic would
			// replace a recoverable retry with a crashed TUI.
			agentName := ""
			if cp := state.CurrentPhase(); cp != nil {
				agentName = cp.AgentName
			}
			reworkCount := 0
			if state.Context != nil {
				reworkCount = state.Context.ReworkCount
			}

			flywheel := pipeline.NewFlywheelLog()
			flywheel.Append(pipeline.FlywheelEntry{
				Timestamp:   time.Now().UTC(),
				TraceID:     state.TraceID,
				Phase:       msg.Phase,
				Agent:       agentName,
				Task:        state.Task,
				Outcome:     "retried",
				DurationMs:  msg.Elapsed.Milliseconds(),
				OutputSize:  0,
				ReworkCount: reworkCount,
			})

			gen := a.views.Pipeline.Gen
			return tea.Tick(backoff, func(time.Time) tea.Msg {
				return PhaseRetryMsg{Phase: msg.Phase, Attempt: attempt, Gen: gen}
			})
		}
	}

	return nil
}

// handlePhaseRetryMsg is triggered after a backoff delay. It re-executes the
// failed phase as an automatic retry. If the mission was superseded in the
// meantime (gen mismatch), the retry is silently discarded.
func (a *App) handlePhaseRetryMsg(msg PhaseRetryMsg) (tea.Model, tea.Cmd) {
	if msg.Gen != a.views.Pipeline.Gen {
		return a, nil
	}
	if a.views.Pipeline.State == nil || !a.views.Pipeline.State.Active {
		return a, nil
	}

	phase := a.views.Pipeline.State.CurrentPhase()
	if phase == nil || phase.Number != msg.Phase {
		return a, nil
	}

	a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
		PipelineMsg{phase.Operative, fmt.Sprintf("🔄 Retry %d — taking another run at this...", msg.Attempt)},
	)

	// Reset phase state and re-execute.
	phase.StartPhase()
	a.views.Pipeline.Running = true

	cmd := executePhase(
		a.views.Pipeline.Ctx,
		*phase,
		a.registry,
		a.env.Backend.Active,
		a.projectCtx,
		a.views.Pipeline.State.Context,
		a.views.Pipeline.State.PhaseTimeout,
		a.views.Pipeline.Gen,
	)
	return a, cmd
}
