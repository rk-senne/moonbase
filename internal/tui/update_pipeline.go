package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/pipeline"
)

// handlePipelineAdvance handles the "n" key to advance the pipeline to the next phase.
func (a App) handlePipelineAdvance() (tea.Model, tea.Cmd) {
	if a.view == ViewPipeline && a.pipeline.State != nil && !a.pipeline.Running {
		prev := a.pipeline.State.Phases[a.pipeline.State.Current]
		a.pipeline.State.Advance()
		if a.pipeline.State.Current < len(a.pipeline.State.Phases) {
			phase := a.pipeline.State.Phases[a.pipeline.State.Current]
			a.pipeline.Output = append(a.pipeline.Output, "",
				fmt.Sprintf("Phase %d: %s activated...", phase.Number, phase.Operative))
			// Try real execution
			if cmd := a.startNextPhase(); cmd != nil {
				return a, cmd
			}
			// Fallback: manual/simulated advance
			a.pipeline.State.Phases[a.pipeline.State.Current].Status = pipeline.StatusRunning
			a.pipeline.Chat = append(a.pipeline.Chat,
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
	if a.view == ViewPipeline && a.pipeline.State != nil && !a.pipeline.Running {
		a.pipeline.State.Retry()
		phase := a.pipeline.State.Phases[a.pipeline.State.Current]
		a.pipeline.Output = append(a.pipeline.Output,
			fmt.Sprintf("⚠️ RETRYING Phase %d: %s...", phase.Number, phase.Operative))
		a.pipeline.Chat = append(a.pipeline.Chat,
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
	if a.view == ViewPipeline && a.pipeline.State != nil {
		phase := a.pipeline.State.Phases[a.pipeline.State.Current]
		a.pipeline.Output = append(a.pipeline.Output,
			fmt.Sprintf("⊘ SKIPPED Phase %d: %s", phase.Number, phase.Operative))
		a.pipeline.Chat = append(a.pipeline.Chat,
			PipelineMsg{phase.Operative, "⊘ Phase skipped by operator."},
		)
		a.pipeline.State.Skip()
		if a.pipeline.State.Current < len(a.pipeline.State.Phases) {
			next := a.pipeline.State.Phases[a.pipeline.State.Current]
			a.pipeline.Output = append(a.pipeline.Output,
				fmt.Sprintf("Phase %d: %s activated...", next.Number, next.Operative))
			a.pipeline.Chat = append(a.pipeline.Chat,
				PipelineMsg{next.Operative, fmt.Sprintf("Stepping in. Previous phase was skipped. Starting %s.", next.Name)},
			)
		}
	}
	return a, nil
}
