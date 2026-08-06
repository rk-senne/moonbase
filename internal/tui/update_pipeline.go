package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// handlePipelineAdvance handles the "n" key to advance the pipeline to the next phase.
func (a App) handlePipelineAdvance() (tea.Model, tea.Cmd) {
	if a.view == ViewPipeline && a.views.Pipeline.State != nil && !a.views.Pipeline.Running {
		prev := a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current]
		a.views.Pipeline.State.Advance()
		if a.views.Pipeline.State.Current < len(a.views.Pipeline.State.Phases) {
			phase := a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current]
			a.views.Pipeline.Output = append(a.views.Pipeline.Output, "",
				fmt.Sprintf("Phase %d: %s activated...", phase.Number, phase.Operative))
			// Try real execution
			if cmd := a.startNextPhase(); cmd != nil {
				return a, cmd
			}
			// Fallback: manual/simulated advance
			a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current].Status = pipeline.StatusRunning
			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{prev.Operative, fmt.Sprintf("Phase complete. Handing off to %s.", phase.Operative)},
				PipelineMsg{"", "───────────────────────────────────"},
				PipelineMsg{phase.Operative, fmt.Sprintf("Received handoff from %s. Starting %s phase.", prev.Operative, phase.Name)},
			)
		}
	}
	return a, nil
}

// handlePipelineRetry handles the "r" key to retry the current pipeline phase.
func (a App) handlePipelineRetry() (tea.Model, tea.Cmd) {
	if a.view == ViewPipeline && a.views.Pipeline.State != nil && !a.views.Pipeline.Running {
		a.views.Pipeline.State.Retry()
		phase := a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current]
		a.views.Pipeline.Output = append(a.views.Pipeline.Output,
			fmt.Sprintf("⚠️ RETRYING Phase %d: %s...", phase.Number, phase.Operative))
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{phase.Operative, "⚠️ Retrying... Let me take another look at this."},
		)
		// Try real execution on retry
		if cmd := a.startNextPhase(); cmd != nil {
			return a, cmd
		}
	}
	return a, nil
}

// handlePipelineSkip handles the "s" key to skip the current pipeline phase.
// If a phase is actively running, it interrupts the backend first (so the agent
// actually stops), then advances and continues the mission.
func (a App) handlePipelineSkip() (tea.Model, tea.Cmd) {
	if a.view != ViewPipeline || a.views.Pipeline.State == nil {
		return a, nil
	}

	if a.views.Pipeline.Running {
		// Interrupt the running phase: stop its backend stream and discard its
		// late messages via a generation bump.
		a.flushLiveBuf()
		if a.views.Pipeline.PhaseStreamCancel != nil {
			a.views.Pipeline.PhaseStreamCancel()
			a.views.Pipeline.PhaseStreamCancel = nil
		}
		a.views.Pipeline.PhaseStreamCh = nil
		a.views.Pipeline.Running = false
		a.views.Pipeline.Gen++
	}

	phase := a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current]
	a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
		PipelineMsg{"", fmt.Sprintf("⊘ Phase %d (%s) skipped by operator.", phase.Number, phase.Operative)},
	)
	a.views.Pipeline.State.Skip()

	// Continue with the next phase if the mission isn't complete.
	if a.views.Pipeline.State.Active {
		if cmd := a.startNextPhase(); cmd != nil {
			return a, cmd
		}
		next := a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current]
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", fmt.Sprintf("─── %s stepping in ───", next.Operative)},
		)
	}
	return a, nil
}

// handlePipelineScroll handles conversation scrollback in the pipeline view.
// ChatScroll counts lines up from the bottom (0 = follow the latest output).
// It returns handled=true only for scroll keys while on the pipeline view, so
// all other keys fall through to the normal handler.
func (a App) handlePipelineScroll(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	if a.view != ViewPipeline || a.views.Pipeline.State == nil {
		return a, nil, false
	}
	maxScroll := len(a.pipelineChatLines(a.pipelineMainWidth())) - a.pipelineChatHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	page := a.pipelineChatHeight() - 1
	if page < 1 {
		page = 1
	}
	cur := a.views.Pipeline.ChatScroll
	switch {
	case key.Matches(msg, a.keys.Up):
		cur++
	case key.Matches(msg, a.keys.Down):
		cur--
	case key.Matches(msg, a.keys.DocsPageUp):
		cur += page
	case key.Matches(msg, a.keys.DocsPageDown):
		cur -= page
	case msg.String() == "home":
		cur = maxScroll
	case msg.String() == "end":
		cur = 0
	default:
		return a, nil, false
	}
	if cur < 0 {
		cur = 0
	}
	if cur > maxScroll {
		cur = maxScroll
	}
	a.views.Pipeline.ChatScroll = cur
	return a, nil, true
}
