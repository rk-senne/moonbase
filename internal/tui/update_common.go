package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/backend"
)

// handleWindowSize processes tea.WindowSizeMsg.
func (a App) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	a.width = msg.Width
	a.height = msg.Height
	a.ready = true
	return a, nil
}

// handleBootTick processes boot animation steps.
func (a App) handleBootTick() (tea.Model, tea.Cmd) {
	if a.view == ViewBoot {
		a.bootStep++
		if a.bootStep >= len(bootMessages) {
			return a, bootDone()
		}
		if a.bootStep == len(bootMessages)-1 {
			a.anim.TriggerTypewriter()
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
	a.addIntel("Git: %s %s", a.gitBranch, a.gitStatus())
	return a, nil
}

// handleAgentsLoaded processes the registry load result.
func (a App) handleAgentsLoaded(msg AgentsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.backends = backend.DetectAll()
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
func (a App) handleSearchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.searching = false
		a.searchInput.Reset()
		a.searchInput.Blur()
		a.filtered = nil
	case "enter":
		a.searching = false
		a.searchInput.Blur()
		if len(a.filtered) > 0 {
			a.cursor = a.filtered[0]
			a.selected = a.cursor
			a.view = ViewDossier
		}
		a.searchInput.Reset()
		a.filtered = nil
	default:
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		a.filterAgents()
		return a, cmd
	}
	return a, nil
}

// handleTerminalKeys handles key messages when the embedded terminal is active.
func (a App) handleTerminalKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.termActive = false
		a.termInput.Blur()
	case "`":
		a.termActive = false
		a.termInput.Blur()
		a.browsing = true
	case "enter":
		cmd := a.termInput.Value()
		a.termInput.Reset()
		if cmd != "" {
			return a, a.execTermCmd(cmd)
		}
	default:
		var c tea.Cmd
		a.termInput, c = a.termInput.Update(msg)
		return a, c
	}
	return a, nil
}

// handleFileBrowserKeys handles key messages when the file browser is active.
func (a App) handleFileBrowserKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "`":
		a.browsing = false
		a.termActive = true
		a.termInput.Focus()
		return a, textinput.Blink
	case "up", "k":
		a.fileBrowser.Up()
	case "down", "j":
		a.fileBrowser.Down()
	case "enter", "l":
		a.fileBrowser.Enter()
		a.cwd = a.fileBrowser.dir
	case "backspace", "h":
		a.fileBrowser.Back()
		a.cwd = a.fileBrowser.dir
	case "e":
		if a.fileBrowser.SelectedIsFile() {
			return a, a.editFile(a.fileBrowser.SelectedPath())
		}
	case ".":
		a.fileBrowser.refresh()
	case "esc":
		a.browsing = false
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
	if a.cancelPipeline != nil {
		a.cancelPipeline()
	}
	if a.pipelineState != nil {
		a.pipelineState.Stop("Aborted by human")
		a.pipelineChat = append(a.pipelineChat,
			PipelineMsg{"", "🛑 Mission aborted by human."},
		)
		a.addIntel("Mission aborted: %s", a.pipelineState.Task)
	}
	a.pipelineRunning = false
	return a, nil
}

// handleSystemInfo processes system detection results.
func (a App) handleSystemInfo(msg systemInfoMsg) (tea.Model, tea.Cmd) {
	a.gitBranch = msg.branch
	a.gitClean = msg.clean
	a.dockerCount = msg.dockerCount
	a.gitDiffLines = msg.diffLines
	return a, nil
}

// handleFileChange processes file watcher events.
func (a App) handleFileChange(msg fileChangeMsg) (tea.Model, tea.Cmd) {
	a.addIntel("📁 %s modified", msg.path)
	return a, a.pollWatcher()
}

// handleTermOutput processes embedded terminal command output.
func (a App) handleTermOutput(msg termOutputMsg) (tea.Model, tea.Cmd) {
	a.termOutput = append(a.termOutput, lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+msg.cmd))
	for _, line := range strings.Split(msg.output, "\n") {
		a.termOutput = append(a.termOutput, line)
	}
	if len(a.termOutput) > maxTerminalLines {
		a.termOutput = a.termOutput[len(a.termOutput)-maxTerminalLines:]
	}
	return a, nil
}

// ringBell sends terminal bell and triggers animation.
func (a App) ringBell() {
	fmt.Print("\a")
	a.anim.TriggerIntelFlash()
}

// abortPendingTimedOut checks if the abort pending is within the 3s window.
func (a App) abortPendingTimedOut() bool {
	return time.Since(a.abortPendingAt) < 3*time.Second
}

// --- Init helper commands and message types ---

type fileChangeMsg struct{ path string }
type agentReloadMsg struct{}

func (a App) pollWatcher() tea.Cmd {
	if a.fileWatcher == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case ev, ok := <-a.fileWatcher.Events:
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
	branch      string
	clean       bool
	dockerCount int
	diffLines   int
}

func detectSystem() tea.Cmd {
	return func() tea.Msg {
		branch := "unknown"
		clean := false
		dockerCount := 0
		diffLines := 0

		if out, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
			branch = strings.TrimSpace(string(out))
		}
		if out, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
			clean = len(strings.TrimSpace(string(out))) == 0
		}
		if out, err := exec.Command("docker", "ps", "-q").Output(); err == nil {
			lines := strings.TrimSpace(string(out))
			if lines != "" {
				dockerCount = len(strings.Split(lines, "\n"))
			}
		}
		if out, err := exec.Command("git", "diff", "--stat").Output(); err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "insertion") || strings.Contains(line, "deletion") {
					parts := strings.Fields(line)
					for i, p := range parts {
						if strings.HasPrefix(p, "insertion") || strings.HasPrefix(p, "deletion") {
							if i > 0 {
								n := 0
								fmt.Sscanf(parts[i-1], "%d", &n)
								diffLines += n
							}
						}
					}
				}
			}
		}

		return systemInfoMsg{branch: branch, clean: clean, dockerCount: dockerCount, diffLines: diffLines}
	}
}

// agentColor returns a unique color for each pipeline agent
func agentColor(name string) lipgloss.Color {
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
