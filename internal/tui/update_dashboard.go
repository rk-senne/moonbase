package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/watcher"
)

// handleDashboardKeys handles key messages when the current view is the
// general dashboard/dossier default case (after all specific views have
// been checked). This covers ViewDashboard, ViewDossier, ViewPipeline,
// ViewHelp, ViewHistory, and other non-specific key handling.
func (a App) handleDashboardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit):
		if a.pipeline.Cancel != nil {
			a.pipeline.Cancel()
		}
		return a, tea.Quit
	case key.Matches(msg, a.keys.Help):
		if a.view == ViewHelp {
			a.view = ViewDashboard
		} else {
			a.view = ViewHelp
		}
	case key.Matches(msg, a.keys.Protocol):
		a.view = ViewProtocol
	case key.Matches(msg, a.keys.Back):
		if a.view == ViewPipeline {
			if a.pipeline.Running {
				if a.pipeline.AbortPending && time.Since(a.pipeline.AbortAt) < 3*time.Second {
					// Second esc within 3s — actually abort
					if a.pipeline.Cancel != nil {
						a.pipeline.Cancel()
					}
					a.pipeline.State.Stop("Aborted by human")
					a.pipeline.Chat = append(a.pipeline.Chat,
						PipelineMsg{"", "🛑 Mission aborted by human."},
					)
					a.addIntel("Mission aborted: %s", a.pipeline.State.Task)
					a.pipeline.Running = false
					a.pipeline.AbortPending = false
				} else {
					// First esc — show warning
					a.pipeline.AbortPending = true
					a.pipeline.AbortAt = time.Now()
				}
			} else {
				a.view = ViewDashboard
				a.pipeline.AbortPending = false
			}
		} else {
			a.view = ViewDashboard
			a.pipeline.AbortPending = false
		}
	case key.Matches(msg, a.keys.NextPhase):
		return a.handlePipelineAdvance()
	case key.Matches(msg, a.keys.RetryPhase):
		return a.handlePipelineRetry()
	case key.Matches(msg, a.keys.SkipPhase):
		return a.handlePipelineSkip()
	case key.Matches(msg, a.keys.Up):
		if a.view == ViewDashboard || a.view == ViewDossier {
			if a.dashboard.Cursor > 0 {
				a.dashboard.Cursor--
			}
			a.dashboard.Selected = a.dashboard.Cursor
			a.chrome.Anim.TriggerSelectPulse()
		}
	case key.Matches(msg, a.keys.Down):
		if a.view == ViewDashboard || a.view == ViewDossier {
			if a.dashboard.Cursor < a.registry.Count()-1 {
				a.dashboard.Cursor++
			}
			a.dashboard.Selected = a.dashboard.Cursor
			a.chrome.Anim.TriggerSelectPulse()
		}
	case key.Matches(msg, a.keys.Enter):
		if a.view == ViewDashboard {
			a.view = ViewDossier
			a.chrome.Anim.TriggerReveal()
		} else if a.view == ViewDossier {
			return a, a.deployAgent()
		}
	case key.Matches(msg, a.keys.CopyPrompt):
		if a.view == ViewDossier {
			return a, a.copyPrompt()
		}
	case key.Matches(msg, a.keys.OpenComms):
		if a.view == ViewDossier || a.view == ViewDashboard {
			a.openComms()
			return a, textinput.Blink
		}
	case key.Matches(msg, a.keys.NewMission):
		a.view = ViewMission
		a.mission.Input.Focus()
		return a, textinput.Blink
	case key.Matches(msg, a.keys.CycleTheme):
		a.cycleTheme()
		a.addIntel("Theme: %s", a.theme.Name)
	case key.Matches(msg, a.keys.Search):
		a.search.Active = true
		a.search.Input.Focus()
		return a, textinput.Blink
	case key.Matches(msg, a.keys.GitDiff):
		return a, a.runGitCmd("git diff --stat")
	case key.Matches(msg, a.keys.GitStatus):
		return a, a.runGitCmd("git status --short")
	case key.Matches(msg, a.keys.Tab):
		a.chrome.Focus = (a.chrome.Focus + 1) % 3
	case key.Matches(msg, a.keys.LaunchLazygit):
		return a, a.launchTool("lazygit")
	case key.Matches(msg, a.keys.LaunchBtop):
		return a, a.launchTool("btop")
	case key.Matches(msg, a.keys.LaunchNvim):
		return a, a.launchNvim()
	case key.Matches(msg, a.keys.LaunchCmux):
		return a, a.launchCmux()
	case key.Matches(msg, a.keys.LaunchFish):
		return a, a.launchTool("fish")
	case key.Matches(msg, a.keys.ToggleWatcher):
		if a.env.Infra.Watcher != nil {
			if a.env.Infra.Watcher.Running() {
				a.env.Infra.Watcher.Stop()
				a.addIntel("File watcher stopped.")
			} else {
				cwd, _ := os.Getwd()
				fw, _ := watcher.New()
				if fw != nil {
					fw.Start(cwd)
					a.env.Infra.Watcher = fw
					a.addIntel("File watcher started: %s", cwd)
				}
			}
		}
	case key.Matches(msg, a.keys.CreatePR):
		if a.env.Infra.Ctx.IsPersonal() {
			return a, a.createPR()
		}
		a.addIntel("PR: not available in this context.")
	case key.Matches(msg, a.keys.History):
		a.view = ViewHistory
	case key.Matches(msg, a.keys.Docs):
		a.docs = newDocsState(a.width, a.height)
		a.view = ViewDocs
	case key.Matches(msg, a.keys.Projects):
		if a.view == ViewDashboard {
			a.projectNav = newProjectsState()
			a.view = ViewProjects
		}
	case key.Matches(msg, a.keys.SpawnHook):
		if a.view == ViewDossier {
			return a, a.runSpawnHook()
		}
	case key.Matches(msg, a.keys.JumpToAgent):
		idx := int(msg.String()[0] - '0')
		a.dashboard.Cursor = idx
		a.dashboard.Selected = a.dashboard.Cursor
		a.view = ViewDossier
	}

	return a, nil
}
