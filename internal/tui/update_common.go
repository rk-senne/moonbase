package tui

import (
	"fmt"
	"image/color"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/rk-senne/moonbase/internal/backend"
)

// handleWindowSize processes tea.WindowSizeMsg.
func (a App) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	a.width = msg.Width
	a.height = msg.Height
	a.boot.Ready = true
	return a, nil
}

// handleBootTick processes boot animation steps.
func (a App) handleBootTick() (tea.Model, tea.Cmd) {
	if a.view == ViewBoot {
		a.boot.Step++
		if a.boot.Step >= len(bootMessages) {
			return a, bootDone()
		}
		if a.boot.Step == len(bootMessages)-1 {
			a.chrome.Anim.TriggerTypewriter()
		}
		return a, bootTick()
	}
	return a, nil
}

// handleBootDone transitions from boot to dashboard.
func (a App) handleBootDone() (tea.Model, tea.Cmd) {
	a.view = ViewDashboard
	a.addIntel("Moonbase online. %d operatives loaded.", a.registry.Count())
	a.addIntel("AI backends detected: %s", a.detectedBackends())
	a.addIntel("Git: %s %s", a.env.System.Branch, a.gitStatus())
	return a, nil
}

// handleAgentsLoaded processes the registry load result.
func (a App) handleAgentsLoaded(msg AgentsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.env.Backend.Available = backend.DetectAll()
	}
	return a, nil
}

// handleSpinnerTick updates the spinner animation.
func (a App) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.spinner, cmd = a.spinner.Update(msg)
	return a, cmd
}

// handleSearchKeys handles key messages when in search mode.
func (a App) handleSearchKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.SearchCancel):
		a.views.Search.Active = false
		a.views.Search.Input.Reset()
		a.views.Search.Input.Blur()
		a.views.Search.Filtered = nil
	case key.Matches(msg, a.keys.SearchConfirm):
		a.views.Search.Active = false
		a.views.Search.Input.Blur()
		if len(a.views.Search.Filtered) > 0 {
			a.views.Dashboard.Cursor = a.views.Search.Filtered[0]
			a.views.Dashboard.Selected = a.views.Dashboard.Cursor
			a.view = ViewDossier
		}
		a.views.Search.Input.Reset()
		a.views.Search.Filtered = nil
	default:
		var cmd tea.Cmd
		a.views.Search.Input, cmd = a.views.Search.Input.Update(msg)
		a.filterAgents()
		return a, cmd
	}
	return a, nil
}

// handleTerminalKeys handles key messages when the embedded terminal is active.
func (a App) handleTerminalKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	wasBrowserSwitch := key.Matches(msg, a.keys.TerminalToBrowser)
	var cmd tea.Cmd
	a.views.Terminal, cmd = a.views.Terminal.Update(msg, a.appContext())
	// If user pressed the browser-switch key, activate file browser mode
	if wasBrowserSwitch {
		a.views.Browser.Active = true
	}
	return a, cmd
}

// handleFileBrowserKeys handles key messages when the file browser is active.
func (a App) handleFileBrowserKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.BrowserToTerminal):
		a.views.Browser.Active = false
		a.views.Terminal.Active = true
		a.views.Terminal.Input.Focus()
		return a, textinput.Blink
	case key.Matches(msg, a.keys.BrowserUp):
		a.views.Browser.FileBrowser.Up()
	case key.Matches(msg, a.keys.BrowserDown):
		a.views.Browser.FileBrowser.Down()
	case key.Matches(msg, a.keys.BrowserEnter):
		a.views.Browser.FileBrowser.Enter()
		a.views.Terminal.Cwd = a.views.Browser.FileBrowser.dir
	case key.Matches(msg, a.keys.BrowserBack):
		a.views.Browser.FileBrowser.Back()
		a.views.Terminal.Cwd = a.views.Browser.FileBrowser.dir
	case key.Matches(msg, a.keys.BrowserEdit):
		if a.views.Browser.FileBrowser.SelectedIsFile() {
			return a, a.editFile(a.views.Browser.FileBrowser.SelectedPath())
		}
	case key.Matches(msg, a.keys.BrowserRefresh):
		a.views.Browser.FileBrowser.refresh()
	case key.Matches(msg, a.keys.BrowserEsc):
		a.views.Browser.Active = false
	}
	return a, nil
}

// handlePhaseResultUpdate processes a pipeline phase result message.
func (a App) handlePhaseResultUpdate(msg PhaseResultMsg) (tea.Model, tea.Cmd) {
	if cmd := a.handlePhaseResult(msg); cmd != nil {
		return a, cmd
	}
	return a, nil
}

// handlePipelineAborted processes a pipeline abort message.
func (a App) handlePipelineAborted() (tea.Model, tea.Cmd) {
	if a.views.Pipeline.Cancel != nil {
		a.views.Pipeline.Cancel()
	}
	if a.views.Pipeline.State != nil {
		a.views.Pipeline.State.Stop("Aborted by human")
		a.views.Pipeline.Chat = append(a.views.Pipeline.Chat,
			PipelineMsg{"", "🛑 Mission aborted by human."},
		)
		a.addIntel("Mission aborted: %s", a.views.Pipeline.State.Task)
	}
	a.views.Pipeline.Running = false
	return a, nil
}

// handleSystemInfo processes system detection results.
func (a App) handleSystemInfo(msg systemInfoMsg) (tea.Model, tea.Cmd) {
	a.env.System = SystemModel{
		Branch:         msg.branch,
		Clean:          msg.clean,
		Docker:         msg.dockerCount,
		ChangedLines:   msg.diffLines,
		FilesChanged:   msg.filesChanged,
		UntrackedFiles: msg.untrackedFiles,
		SensitiveHits:  msg.sensitiveHits,
		NoRepo:         msg.noRepo,
	}
	return a, nil
}

// handleFileChange processes file watcher events.
func (a App) handleFileChange(msg fileChangeMsg) (tea.Model, tea.Cmd) {
	a.addIntel("📁 %s modified", msg.path)
	return a, a.pollWatcher()
}

// handleTermOutput processes embedded terminal command output.
func (a App) handleTermOutput(msg termOutputMsg) (tea.Model, tea.Cmd) {
	a.views.Terminal = a.views.Terminal.HandleOutput(msg, a.theme.Data)
	return a, nil
}

// ringBell sends terminal bell and triggers animation.
func (a App) ringBell() {
	fmt.Print("\a")
	a.chrome.Anim.TriggerIntelFlash()
}

// abortPendingTimedOut checks if the abort pending is within the 3s window.
func (a App) abortPendingTimedOut() bool {
	return time.Since(a.views.Pipeline.AbortAt) < 3*time.Second
}

// --- Init helper commands and message types ---

type fileChangeMsg struct{ path string }
type agentReloadMsg struct{}

func (a App) pollWatcher() tea.Cmd {
	if a.env.Infra.Watcher == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case ev, ok := <-a.env.Infra.Watcher.Events:
			if !ok {
				// Watcher closed, stop polling
				return nil
			}
			return fileChangeMsg{path: ev.Path}
		}
	}
}

func (a App) pollAgentDir() tea.Cmd {
	return tea.Every(2*time.Second, func(t time.Time) tea.Msg {
		return agentReloadMsg{}
	})
}

type clockTickMsg time.Time
type blinkTickMsg time.Time

func clockTick() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return clockTickMsg(t)
	})
}

func blinkTick() tea.Cmd {
	return tea.Every(800*time.Millisecond, func(t time.Time) tea.Msg {
		return blinkTickMsg(t)
	})
}

type systemInfoMsg struct {
	branch         string
	clean          bool
	dockerCount    int
	diffLines      int
	filesChanged   int
	untrackedFiles int
	sensitiveHits  int
	noRepo         bool
}

func detectSystem() tea.Cmd {
	return func() tea.Msg {
		msg := systemInfoMsg{branch: "unknown", clean: true}

		// Not a git repo: report docker only and mark noRepo so the threat
		// gauge reads LOW/"no git repo" instead of a misleading empty state.
		if err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Run(); err != nil {
			msg.noRepo = true
			msg.dockerCount = dockerContainerCount()
			return msg
		}

		if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
			if b := strings.TrimSpace(string(out)); b != "" {
				msg.branch = b
			}
		}

		// A single porcelain scan yields dirtiness, files changed, untracked
		// count, and security-sensitivity — for staged, unstaged, and new files.
		if out, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
			for _, ln := range strings.Split(string(out), "\n") {
				if strings.TrimSpace(ln) == "" {
					continue
				}
				msg.clean = false
				status, path := parsePorcelainLine(ln)
				if status == "??" {
					msg.untrackedFiles++
				} else {
					msg.filesChanged++
				}
				if isSensitivePath(path) {
					msg.sensitiveHits++
				}
			}
		}

		// Changed lines vs HEAD captures staged + unstaged volume. Fall back to
		// the working-tree diff when there is no HEAD yet (fresh repo).
		if out, err := exec.Command("git", "diff", "HEAD", "--shortstat").Output(); err == nil {
			msg.diffLines = parseShortstat(string(out))
		} else if out, err := exec.Command("git", "diff", "--shortstat").Output(); err == nil {
			msg.diffLines = parseShortstat(string(out))
		}

		msg.dockerCount = dockerContainerCount()
		return msg
	}
}

// dockerContainerCount returns the number of running docker containers, or 0
// when docker is unavailable.
func dockerContainerCount() int {
	if out, err := exec.Command("docker", "ps", "-q").Output(); err == nil {
		if lines := strings.TrimSpace(string(out)); lines != "" {
			return len(strings.Split(lines, "\n"))
		}
	}
	return 0
}

// parsePorcelainLine splits a `git status --porcelain` line into its two-char
// status code and path, resolving rename entries ("old -> new") to the new path.
func parsePorcelainLine(ln string) (status, path string) {
	if len(ln) < 3 {
		return strings.TrimSpace(ln), ""
	}
	status = strings.TrimSpace(ln[:2])
	if strings.HasPrefix(ln, "??") {
		status = "??"
	}
	path = strings.TrimSpace(ln[3:])
	if idx := strings.Index(path, " -> "); idx >= 0 {
		path = path[idx+4:]
	}
	return status, path
}

// parseShortstat extracts total insertions+deletions from `git diff --shortstat`
// output, e.g. " 3 files changed, 12 insertions(+), 4 deletions(-)".
func parseShortstat(out string) int {
	total := 0
	for _, field := range strings.Split(out, ",") {
		field = strings.TrimSpace(field)
		if strings.Contains(field, "insertion") || strings.Contains(field, "deletion") {
			n := 0
			fmt.Sscanf(field, "%d", &n)
			total += n
		}
	}
	return total
}

// agentColor returns a unique color for each pipeline agent
func agentColor(name string) color.Color {
	switch {
	case strings.Contains(name, "1"):
		return lipgloss.Color("#FF6B6B") // red
	case strings.Contains(name, "2"):
		return lipgloss.Color("#4ECDC4") // teal
	case strings.Contains(name, "3"):
		return lipgloss.Color("#A8E6CF") // mint
	case strings.Contains(name, "4"):
		return lipgloss.Color("#FFE66D") // yellow
	case strings.Contains(name, "5"):
		return lipgloss.Color("#C4B5FD") // purple
	case strings.Contains(name, "0"):
		return lipgloss.Color("#F97316") // orange
	case strings.Contains(name, "274"):
		return lipgloss.Color("#EF4444") // crimson
	case strings.Contains(name, "362"):
		return lipgloss.Color("#06B6D4") // cyan
	default:
		return lipgloss.Color("#00FF88")
	}
}
