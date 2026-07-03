package tui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/f5508037/moonbase/internal/watcher"
)

// handleDashboardKeys handles key messages when the current view is the
// general dashboard/dossier default case (after all specific views have
// been checked). This covers ViewDashboard, ViewDossier, ViewPipeline,
// ViewHelp, ViewHistory, and other non-specific key handling.
func (a App) handleDashboardKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if a.cancelPipeline != nil {
			a.cancelPipeline()
		}
		return a, tea.Quit
	case "?":
		if a.view == ViewHelp {
			a.view = ViewDashboard
		} else {
			a.view = ViewHelp
		}
	case "F1":
		a.view = ViewProtocol
	case "esc":
		if a.view == ViewPipeline {
			if a.pipelineRunning {
				if a.abortPending && time.Since(a.abortPendingAt) < 3*time.Second {
					// Second esc within 3s — actually abort
					if a.cancelPipeline != nil {
						a.cancelPipeline()
					}
					a.pipelineState.Stop("Aborted by human")
					a.pipelineChat = append(a.pipelineChat,
						PipelineMsg{"", "🛑 Mission aborted by human."},
					)
					a.addIntel("Mission aborted: %s", a.pipelineState.Task)
					a.pipelineRunning = false
					a.abortPending = false
				} else {
					// First esc — show warning
					a.abortPending = true
					a.abortPendingAt = time.Now()
				}
			} else {
				a.view = ViewDashboard
				a.abortPending = false
			}
		} else {
			a.view = ViewDashboard
			a.abortPending = false
		}
	case "n":
		return a.handlePipelineAdvance()
	case "r":
		return a.handlePipelineRetry()
	case "s":
		return a.handlePipelineSkip()
	case "up", "k":
		if a.view == ViewDashboard || a.view == ViewDossier {
			if a.cursor > 0 {
				a.cursor--
			}
			a.selected = a.cursor
			a.anim.TriggerSelectPulse()
		}
	case "down", "j":
		if a.view == ViewDashboard || a.view == ViewDossier {
			if a.cursor < a.registry.Count()-1 {
				a.cursor++
			}
			a.selected = a.cursor
			a.anim.TriggerSelectPulse()
		}
	case "enter":
		if a.view == ViewDashboard {
			a.view = ViewDossier
			a.anim.TriggerReveal()
		} else if a.view == ViewDossier {
			return a, a.deployAgent()
		}
	case "c":
		if a.view == ViewDossier {
			return a, a.copyPrompt()
		}
	case "C":
		if a.view == ViewDossier || a.view == ViewDashboard {
			a.openComms()
			return a, textinput.Blink
		}
	case "m":
		a.view = ViewMission
		a.missionInput.Focus()
		return a, textinput.Blink
	case "T":
		a.cycleTheme()
		a.addIntel("Theme: %s", a.theme)
	case "/":
		a.searching = true
		a.searchInput.Focus()
		return a, textinput.Blink
	case "d":
		return a, a.runGitCmd("git diff --stat")
	case "g":
		return a, a.runGitCmd("git status --short")
	case "tab":
		a.focus = (a.focus + 1) % 3
	case "L":
		return a, a.launchTool("lazygit")
	case "B":
		return a, a.launchTool("btop")
	case "V":
		return a, a.launchNvim()
	case "M":
		return a, a.launchCmux()
	case "F":
		return a, a.launchTool("fish")
	case "w":
		if a.fileWatcher != nil {
			if a.fileWatcher.Running() {
				a.fileWatcher.Stop()
				a.addIntel("File watcher stopped.")
			} else {
				cwd, _ := os.Getwd()
				fw, _ := watcher.New()
				if fw != nil {
					fw.Start(cwd)
					a.fileWatcher = fw
					a.addIntel("File watcher started: %s", cwd)
				}
			}
		}
	case "P":
		if a.ctx.IsPersonal() {
			return a, a.createPR()
		}
		a.addIntel("PR: not available in this context.")
	case "H":
		a.view = ViewHistory
	case "W":
		a.docs = NewDocsState(a.width, a.height)
		a.view = ViewDocs
	case "p":
		if a.view == ViewDashboard {
			a.projectNav = NewProjectsState()
			a.view = ViewProjects
		}
	case "t":
		if a.view == ViewDossier {
			return a, a.runSpawnHook()
		}
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(msg.String()[0] - '0')
		a.cursor = idx
		a.selected = a.cursor
		a.view = ViewDossier
	}

	return a, nil
}
