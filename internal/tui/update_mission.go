package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// handleMissionKeys handles key messages when the view is ViewMission.
func (a App) handleMissionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
		a.mission.Input.Reset()
		a.mission.Input.Blur()
	case key.Matches(msg, a.keys.Enter):
		task := a.mission.Input.Value()
		if task != "" {
			a.addIntel("Mission briefed: %s", task)
			a.pipeline.State = pipeline.New(task)
			a.pipeline.Output = []string{
				fmt.Sprintf("━━━ MISSION: %s ━━━", task),
			}
			a.pipeline.Chat = []PipelineMsg{
				{"", fmt.Sprintf("━━━ MISSION: %s ━━━", task)},
			}
			a.mission.Input.Reset()
			a.mission.Input.Blur()
			a.view = ViewPipeline
			a.pipeline.MissionStart = time.Now()

			// Create pipeline context for graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			a.pipeline.Ctx = ctx
			a.pipeline.Cancel = cancel

			// Start real pipeline execution if backend available
			if cmd := a.startNextPhase(); cmd != nil {
				a.addIntel("Pipeline executing via %s...", a.backend.Active.Name())
				return a, cmd
			}
			// No backend — show simulated mode
			a.addIntel("No AI backend — simulated pipeline mode")
			a.pipeline.Chat = append(a.pipeline.Chat,
				PipelineMsg{"", "⚠️ No AI backend available. Showing pipeline simulation."},
				PipelineMsg{"", "Install kiro-cli for real execution, or use [n] to advance manually."},
				PipelineMsg{"", ""},
				PipelineMsg{"Numbuh 1", "Receiving mission brief... Analyzing requirements."},
			)
			a.pipeline.State.Phases[0].Status = pipeline.StatusRunning
		}
	default:
		var cmd tea.Cmd
		a.mission.Input, cmd = a.mission.Input.Update(msg)
		return a, cmd
	}
	return a, nil
}
