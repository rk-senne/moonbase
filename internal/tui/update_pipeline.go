package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/pipeline"
)

// handlePipelineAdvance handles the "n" key to advance the pipeline to the next phase.
func (a App) handlePipelineAdvance() (tea.Model, tea.Cmd) {
	if a.view == ViewPipeline && a.pipelineState != nil && !a.pipelineRunning {
		prev := a.pipelineState.Phases[a.pipelineState.Current]
		a.pipelineState.Advance()
		if a.pipelineState.Current < len(a.pipelineState.Phases) {
			phase := a.pipelineState.Phases[a.pipelineState.Current]
			a.pipelineOutput = append(a.pipelineOutput, "",
				fmt.Sprintf("Phase %d: %s activated...", phase.Number, phase.Operative))
			// Try real execution
			if cmd := a.startNextPhase(); cmd != nil {
				return a, cmd
			}
			// Fallback: manual/simulated advance
			a.pipelineState.Phases[a.pipelineState.Current].Status = pipeline.StatusRunning
			a.pipelineChat = append(a.pipelineChat,
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
	if a.view == ViewPipeline && a.pipelineState != nil && !a.pipelineRunning {
		a.pipelineState.Retry()
		phase := a.pipelineState.Phases[a.pipelineState.Current]
		a.pipelineOutput = append(a.pipelineOutput,
			fmt.Sprintf("⚠️ RETRYING Phase %d: %s...", phase.Number, phase.Operative))
		a.pipelineChat = append(a.pipelineChat,
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
	if a.view == ViewPipeline && a.pipelineState != nil {
		phase := a.pipelineState.Phases[a.pipelineState.Current]
		a.pipelineOutput = append(a.pipelineOutput,
			fmt.Sprintf("⊘ SKIPPED Phase %d: %s", phase.Number, phase.Operative))
		a.pipelineChat = append(a.pipelineChat,
			PipelineMsg{phase.Operative, "⊘ Phase skipped by operator."},
		)
		a.pipelineState.Skip()
		if a.pipelineState.Current < len(a.pipelineState.Phases) {
			next := a.pipelineState.Phases[a.pipelineState.Current]
			a.pipelineOutput = append(a.pipelineOutput,
				fmt.Sprintf("Phase %d: %s activated...", next.Number, next.Operative))
			a.pipelineChat = append(a.pipelineChat,
				PipelineMsg{next.Operative, fmt.Sprintf("Stepping in. Previous phase was skipped. Starting %s.", next.Name)},
			)
		}
	}
	return a, nil
}
