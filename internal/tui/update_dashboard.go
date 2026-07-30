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
		if a.views.Pipeline.Cancel != nil {
			a.views.Pipeline.Cancel()
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
			if a.views.Pipeline.Running {
				if a.views.Pipeline.AbortPending && time.Since(a.views.Pipeline.AbortAt) < 3*time.Second {
					// Second esc within 3s — actually abort
					if a.views.Pipeline.Cancel != nil {
						a.views.Pipeline.Cancel()
					}
					a.views.Pipeline.State.Stop("Aborted by human")
					a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
						PipelineMsg{"", "🛑 Mission aborted by human."},
					)
					a.addIntel("Mission aborted: %s", a.views.Pipeline.State.Task)
					a.views.Pipeline.Running = false
					a.views.Pipeline.AbortPending = false
				} else {
					// First esc — show warning
					a.views.Pipeline.AbortPending = true
					a.views.Pipeline.AbortAt = time.Now()
				}
			} else {
				a.view = ViewDashboard
				a.views.Pipeline.AbortPending = false
			}
		} else {
			a.view = ViewDashboard
			a.views.Pipeline.AbortPending = false
		}
	case key.Matches(msg, a.keys.NextPhase):
		return a.handlePipelineAdvance()
	case key.Matches(msg, a.keys.RetryPhase):
		return a.handlePipelineRetry()
	case key.Matches(msg, a.keys.SkipPhase):
		return a.handlePipelineSkip()
	case key.Matches(msg, a.keys.Up):
		if a.view == ViewDashboard || a.view == ViewDossier {
			if a.views.Dashboard.Cursor > 0 {
				a.views.Dashboard.Cursor--
			}
			a.views.Dashboard.Selected = a.views.Dashboard.Cursor
			a.chrome.Anim.TriggerSelectPulse()
		}
	case key.Matches(msg, a.keys.Down):
		if a.view == ViewDashboard || a.view == ViewDossier {
			if a.views.Dashboard.Cursor < a.registry.Count()-1 {
				a.views.Dashboard.Cursor++
			}
			a.views.Dashboard.Selected = a.views.Dashboard.Cursor
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
		a.views.Mission.Input.Focus()
		return a, textinput.Blink
	case key.Matches(msg, a.keys.CycleTheme):
		a.cycleTheme()
		a.addIntel("Theme: %s", a.theme.Name)
	case key.Matches(msg, a.keys.Search):
		a.views.Search.Active = true
		a.views.Search.Input.Focus()
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
		a.views.Docs = newDocsState(a.width, a.height)
		a.view = ViewDocs
	case key.Matches(msg, a.keys.Projects):
		if a.view == ViewDashboard {
			a.views.ProjectNav = newProjectsState()
			a.view = ViewProjects
		}
	case key.Matches(msg, a.keys.SpawnHook):
		if a.view == ViewDossier {
			return a, a.runSpawnHook()
		}
	case key.Matches(msg, a.keys.JumpToAgent):
		idx := int(msg.String()[0] - '0')
		a.views.Dashboard.Cursor = idx
		a.views.Dashboard.Selected = a.views.Dashboard.Cursor
		a.view = ViewDossier
	}

	return a, nil
}
