package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
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
func (a App) handlePipelineSkip() (tea.Model, tea.Cmd) {
	if a.view == ViewPipeline && a.views.Pipeline.State != nil {
		phase := a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current]
		a.views.Pipeline.Output = append(a.views.Pipeline.Output,
			fmt.Sprintf("⊘ SKIPPED Phase %d: %s", phase.Number, phase.Operative))
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{phase.Operative, "⊘ Phase skipped by operator."},
		)
		a.views.Pipeline.State.Skip()
		if a.views.Pipeline.State.Current < len(a.views.Pipeline.State.Phases) {
			next := a.views.Pipeline.State.Phases[a.views.Pipeline.State.Current]
			a.views.Pipeline.Output = append(a.views.Pipeline.Output,
				fmt.Sprintf("Phase %d: %s activated...", next.Number, next.Operative))
			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{next.Operative, fmt.Sprintf("Stepping in. Previous phase was skipped. Starting %s.", next.Name)},
			)
		}
	}
	return a, nil
}
