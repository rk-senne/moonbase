package tui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// handleMissionKeys handles key messages when the view is ViewMission.
func (a App) handleMissionKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
		a.views.Mission.Input.Reset()
		a.views.Mission.Input.Blur()
	case key.Matches(msg, a.keys.Enter):
		task := a.views.Mission.Input.Value()
		if task != "" {
			a.addIntel("Mission briefed: %s", task)
			a.views.Pipeline.State = pipeline.New(task)
			a.views.Pipeline.Output = []string{
				fmt.Sprintf("━━━ MISSION: %s ━━━", task),
			}
			a.views.Pipeline.Chat = []PipelineMsg{
				{"", fmt.Sprintf("━━━ MISSION: %s ━━━", task)},
			}
			a.views.Mission.Input.Reset()
			a.views.Mission.Input.Blur()
			a.view = ViewPipeline
			a.views.Pipeline.MissionStart = time.Now()

			// Create pipeline context for graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			a.views.Pipeline.Ctx = ctx
			a.views.Pipeline.Cancel = cancel

			// Start real pipeline execution if backend available
			if cmd := a.startNextPhase(); cmd != nil {
				a.addIntel("Pipeline executing via %s...", a.env.Backend.Active.Name())
				return a, cmd
			}
			// No backend — show simulated mode
			a.addIntel("No AI backend — simulated pipeline mode")
			a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
				PipelineMsg{"", "⚠️ No AI backend available. Showing pipeline simulation."},
				PipelineMsg{"", "Install kiro-cli for real execution, or use [n] to advance manually."},
				PipelineMsg{"", ""},
				PipelineMsg{"Numbuh 1", "Receiving mission brief... Analyzing requirements."},
			)
			a.views.Pipeline.State.Phases[0].Status = pipeline.StatusRunning
		}
	default:
		var cmd tea.Cmd
		a.views.Mission.Input, cmd = a.views.Mission.Input.Update(msg)
		return a, cmd
	}
	return a, nil
}
