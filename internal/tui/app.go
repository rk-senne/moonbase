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
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
)

type View int

const (
	ViewBoot View = iota
	ViewDashboard
	ViewDossier
	ViewPipeline
	ViewHelp
	ViewMission
)

type IntelEntry struct {
	Time    string
	Message string
}

type App struct {
	view        View
	registry    *agents.Registry
	backends    []backend.Backend
	selected    int
	cursor      int
	width       int
	height      int
	ready       bool
	bootStep    int
	spinner     spinner.Model
	intel       []IntelEntry
	gitBranch   string
	gitClean    bool
	dockerCount int
	missionInput textinput.Model
	theme       string
}

func NewApp() App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorActive)

	ti := textinput.New()
	ti.Placeholder = "Describe the mission objective..."
	ti.CharLimit = 500
	ti.Width = 60

	reg := agents.NewRegistry("./agents")
	return App{
		view:         ViewBoot,
		registry:     reg,
		spinner:      s,
		intel:        []IntelEntry{},
		missionInput: ti,
		theme:        "moonbase",
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		bootTick(),
		a.registry.Load(),
		detectSystem(),
	)
}

type systemInfoMsg struct {
	branch      string
	clean       bool
	dockerCount int
}

func detectSystem() tea.Cmd {
	return func() tea.Msg {
		branch := "unknown"
		clean := false
		dockerCount := 0

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

		return systemInfoMsg{branch: branch, clean: clean, dockerCount: dockerCount}
	}
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.view == ViewBoot {
			a.view = ViewDashboard
			a.addIntel("Boot skipped by operative.")
			return a, nil
		}
		if a.view == ViewMission {
			switch msg.String() {
			case "esc":
				a.view = ViewDashboard
				a.missionInput.Reset()
				a.missionInput.Blur()
			case "enter":
				task := a.missionInput.Value()
				if task != "" {
					a.addIntel("Mission briefed: %s", task)
					a.addIntel("Deploying KND Council pipeline...")
					a.missionInput.Reset()
					a.missionInput.Blur()
					a.view = ViewDashboard
					return a, a.deployMission(task)
				}
			default:
				var cmd tea.Cmd
				a.missionInput, cmd = a.missionInput.Update(msg)
				return a, cmd
			}
			return a, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "?":
			if a.view == ViewHelp {
				a.view = ViewDashboard
			} else {
				a.view = ViewHelp
			}
		case "esc":
			a.view = ViewDashboard
		case "up", "k":
			if a.view == ViewDashboard || a.view == ViewDossier {
				if a.cursor > 0 {
					a.cursor--
				}
				a.selected = a.cursorToAgent()
			}
		case "down", "j":
			if a.view == ViewDashboard || a.view == ViewDossier {
				if a.cursor < a.registry.Count()-1 {
					a.cursor++
				}
				a.selected = a.cursorToAgent()
			}
		case "enter":
			if a.view == ViewDashboard {
				a.view = ViewDossier
			} else if a.view == ViewDossier {
				return a, a.deployAgent()
			}
		case "c":
			if a.view == ViewDossier {
				return a, a.copyPrompt()
			}
		case "m":
			a.view = ViewMission
			a.missionInput.Focus()
			return a, textinput.Blink
		case "T":
			a.cycleTheme()
			a.addIntel("Theme: %s", a.theme)
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			idx := int(msg.String()[0] - '0')
			a.cursor = idx
			a.selected = a.cursorToAgent()
			a.view = ViewDossier
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

	case bootTickMsg:
		if a.view == ViewBoot {
			a.bootStep++
			if a.bootStep >= len(bootMessages) {
				return a, bootDone()
			}
			return a, bootTick()
		}

	case bootDoneMsg:
		a.view = ViewDashboard
		a.addIntel("Moonbase online. %d operatives loaded.", a.registry.Count())
		a.addIntel("AI backends detected: %s", a.detectedBackends())
		a.addIntel("Git: %s %s", a.gitBranch, a.gitStatus())

	case agents.AgentsLoadedMsg:
		if msg.Err == nil {
			a.backends = backend.DetectAll()
		}

	case systemInfoMsg:
		a.gitBranch = msg.branch
		a.gitClean = msg.clean
		a.dockerCount = msg.dockerCount

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case copyDoneMsg:
		a.addIntel("Prompt copied to clipboard: %s", msg.agent)
		a.view = ViewDashboard

	case deployDoneMsg:
		a.addIntel("Deployed operative: %s", msg.agent)
	}

	return a, nil
}

func (a App) View() string {
	if !a.ready {
		return "  Initializing..."
	}
	switch a.view {
	case ViewBoot:
		return a.renderBoot()
	case ViewHelp:
		return a.renderHelp()
	case ViewDossier:
		return a.renderDossier()
	case ViewPipeline:
		return a.renderPipeline()
	case ViewMission:
		return a.renderMission()
	default:
		return a.renderDashboard()
	}
}

// --- Helpers ---

func (a *App) addIntel(format string, args ...any) {
	entry := IntelEntry{
		Time:    time.Now().Format("15:04"),
		Message: fmt.Sprintf(format, args...),
	}
	a.intel = append(a.intel, entry)
	if len(a.intel) > 50 {
		a.intel = a.intel[len(a.intel)-50:]
	}
}

func (a App) cursorToAgent() int {
	return a.cursor
}

func (a App) gitStatus() string {
	if a.gitClean {
		return "✓ clean"
	}
	return "● dirty"
}

func (a App) detectedBackends() string {
	var names []string
	for _, b := range a.backends {
		if b.Available() {
			names = append(names, b.Name())
		}
	}
	if len(names) == 0 {
		return "clipboard only"
	}
	return strings.Join(names, ", ")
}

type copyDoneMsg struct{ agent string }
type deployDoneMsg struct{ agent string }

func (a *App) cycleTheme() {
	switch a.theme {
	case "moonbase":
		a.theme = "treehouse"
		ColorActive = lipgloss.Color("#33CC33")
		ColorInfo = lipgloss.Color("#8B4513")
		ColorBrand = lipgloss.Color("#228B22")
		ColorHeader = lipgloss.Color("#006400")
	case "treehouse":
		a.theme = "classified"
		ColorActive = lipgloss.Color("#FF0000")
		ColorInfo = lipgloss.Color("#CC0000")
		ColorBrand = lipgloss.Color("#FF3333")
		ColorHeader = lipgloss.Color("#990000")
	default:
		a.theme = "moonbase"
		ColorActive = lipgloss.Color("#00FF88")
		ColorInfo = lipgloss.Color("#00BBFF")
		ColorBrand = lipgloss.Color("#FFD700")
		ColorHeader = lipgloss.Color("#FF6600")
	}
}

func (a App) deployMission(task string) tea.Cmd {
	return func() tea.Msg {
		// Deploy council with the task via kiro-cli or copy to clipboard
		prompt := fmt.Sprintf("MISSION: %s\n\nExecute the full KND Council pipeline.", task)
		if _, err := exec.LookPath("kiro-cli"); err == nil {
			cmd := exec.Command("kiro-cli", "chat", "--agent", "knd-council")
			cmd.Stdin = strings.NewReader(prompt)
			cmd.Start()
		} else {
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(prompt)
			cmd.Run()
		}
		return deployDoneMsg{agent: "knd-council"}
	}
}

func (a App) copyPrompt() tea.Cmd {
	agent := a.registry.Get(a.selected)
	return func() tea.Msg {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(agent.Prompt)
		cmd.Run()
		return copyDoneMsg{agent: agent.Name}
	}
}

func (a App) deployAgent() tea.Cmd {
	agent := a.registry.Get(a.selected)
	return func() tea.Msg {
		// Try kiro-cli first
		if _, err := exec.LookPath("kiro-cli"); err == nil {
			exec.Command("kiro-cli", "chat", "--agent", agent.Name).Start()
		} else {
			// Fallback: copy to clipboard
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(agent.Prompt)
			cmd.Run()
		}
		return deployDoneMsg{agent: agent.Name}
	}
}
