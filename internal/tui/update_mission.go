package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/pipeline"
)

// handleMissionKeys handles key messages when the view is ViewMission.
func (a App) handleMissionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.view = ViewDashboard
		a.missionInput.Reset()
		a.missionInput.Blur()
	case "enter":
		task := a.missionInput.Value()
		if task != "" {
			a.addIntel("Mission briefed: %s", task)
			a.pipelineState = pipeline.New(task)
			a.pipelineOutput = []string{
				fmt.Sprintf("━━━ MISSION: %s ━━━", task),
			}
			a.pipelineChat = []PipelineMsg{
				{"", fmt.Sprintf("━━━ MISSION: %s ━━━", task)},
			}
			a.missionInput.Reset()
			a.missionInput.Blur()
			a.view = ViewPipeline
			a.missionStart = time.Now()

			// Create pipeline context for graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			a.pipelineCtx = ctx
			a.cancelPipeline = cancel

			// Start real pipeline execution if backend available
			if cmd := a.startNextPhase(); cmd != nil {
				a.addIntel("Pipeline executing via %s...", a.activeBackend.Name())
				return a, cmd
			}
			// No backend — show simulated mode
			a.addIntel("No AI backend — simulated pipeline mode")
			a.pipelineChat = append(a.pipelineChat,
				PipelineMsg{"", "⚠️ No AI backend available. Showing pipeline simulation."},
				PipelineMsg{"", "Install kiro-cli for real execution, or use [n] to advance manually."},
				PipelineMsg{"", ""},
				PipelineMsg{"Numbuh 1", "Receiving mission brief... Analyzing requirements."},
			)
			a.pipelineState.Phases[0].Status = pipeline.StatusRunning
		}
	default:
		var cmd tea.Cmd
		a.missionInput, cmd = a.missionInput.Update(msg)
		return a, cmd
	}
	return a, nil
}
