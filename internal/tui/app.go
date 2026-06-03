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
	"github.com/f5508037/moonbase/internal/pipeline"
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
	view           View
	registry       *agents.Registry
	backends       []backend.Backend
	selected       int
	cursor         int
	width          int
	height         int
	ready          bool
	bootStep       int
	spinner        spinner.Model
	intel          []IntelEntry
	gitBranch      string
	gitClean       bool
	dockerCount    int
	missionInput   textinput.Model
	searchInput    textinput.Model
	searching      bool
	filtered       []int
	theme          string
	pipelineState  *pipeline.Pipeline
	pipelineOutput []string
}

func NewApp() App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorActive)

	ti := textinput.New()
	ti.Placeholder = "Describe the mission objective..."
	ti.CharLimit = 500
	ti.Width = 60

	si := textinput.New()
	si.Placeholder = "Search operatives..."
	si.CharLimit = 40
	si.Width = 30

	reg := agents.NewRegistry("./agents")
	return App{
		view:         ViewBoot,
		registry:     reg,
		spinner:      s,
		intel:        []IntelEntry{},
		missionInput: ti,
		searchInput:  si,
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
		if a.searching {
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
					a.selected = a.cursorToAgent()
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
					a.pipelineState = pipeline.New(task)
					a.pipelineState.Phases[0].Status = 1 // Running
					a.pipelineOutput = []string{
						fmt.Sprintf("━━━ MISSION: %s ━━━", task),
						"",
						"Phase 1: Numbuh 1 (Analyst) activated...",
						"Generating requirements and acceptance criteria...",
					}
					a.missionInput.Reset()
					a.missionInput.Blur()
					a.view = ViewPipeline
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
			if a.view == ViewPipeline {
				a.view = ViewDashboard
			} else {
				a.view = ViewDashboard
			}
		case "n":
			if a.view == ViewPipeline && a.pipelineState != nil {
				cur := a.pipelineState.Current
				a.pipelineState.Advance()
				if a.pipelineState.Current < len(a.pipelineState.Phases) {
					phase := a.pipelineState.Phases[a.pipelineState.Current]
					a.pipelineOutput = append(a.pipelineOutput, "",
						fmt.Sprintf("Phase %d: %s activated...", phase.Number, phase.Operative))
					a.pipelineState.Phases[a.pipelineState.Current].Status = 1
				}
				_ = cur
			}
		case "r":
			if a.view == ViewPipeline && a.pipelineState != nil {
				a.pipelineState.Retry()
				phase := a.pipelineState.Phases[a.pipelineState.Current]
				a.pipelineOutput = append(a.pipelineOutput,
					fmt.Sprintf("⚠️ RETRYING Phase %d: %s...", phase.Number, phase.Operative))
			}
		case "s":
			if a.view == ViewPipeline && a.pipelineState != nil {
				phase := a.pipelineState.Phases[a.pipelineState.Current]
				a.pipelineOutput = append(a.pipelineOutput,
					fmt.Sprintf("⊘ SKIPPED Phase %d: %s", phase.Number, phase.Operative))
				a.pipelineState.Skip()
				if a.pipelineState.Current < len(a.pipelineState.Phases) {
					next := a.pipelineState.Phases[a.pipelineState.Current]
					a.pipelineOutput = append(a.pipelineOutput,
						fmt.Sprintf("Phase %d: %s activated...", next.Number, next.Operative))
				}
			}
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
		case "/":
			a.searching = true
			a.searchInput.Focus()
			return a, textinput.Blink
		case "d":
			return a, a.runGitCmd("git diff --stat")
		case "g":
			return a, a.runGitCmd("git status --short")
		case "t":
			if a.view == ViewDossier {
				return a, a.runSpawnHook()
			}
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

	case gitOutputMsg:
		lines := strings.Split(msg.output, "\n")
		for _, line := range lines {
			if line != "" {
				a.addIntel("  %s", line)
			}
		}

	case spawnHookMsg:
		a.addIntel("Spawn hook [%s]:", msg.agent)
		lines := strings.Split(msg.output, "\n")
		for _, line := range lines {
			if line != "" {
				a.addIntel("  %s", line)
			}
		}
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

func (a *App) filterAgents() {
	query := strings.ToLower(a.searchInput.Value())
	if query == "" {
		a.filtered = nil
		return
	}
	a.filtered = nil
	for i, agent := range a.registry.All() {
		name := strings.ToLower(agent.Name)
		desc := strings.ToLower(agent.Description)
		if strings.Contains(name, query) || strings.Contains(desc, query) {
			a.filtered = append(a.filtered, i)
		}
	}
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
type gitOutputMsg struct{ output string }
type spawnHookMsg struct {
	agent  string
	output string
}

func (a App) runGitCmd(command string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(command)
		out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
		if err != nil {
			return gitOutputMsg{output: fmt.Sprintf("(%s failed: %v)", command, err)}
		}
		result := strings.TrimSpace(string(out))
		if result == "" {
			result = "(clean — no output)"
		}
		return gitOutputMsg{output: result}
	}
}

func (a App) runSpawnHook() tea.Cmd {
	agent := a.registry.Get(a.selected)
	hooks, ok := agent.Hooks["agentSpawn"]
	if !ok || len(hooks) == 0 {
		return func() tea.Msg {
			return spawnHookMsg{agent: agent.Name, output: "(no spawn hook configured)"}
		}
	}
	cmd := hooks[0].Command
	return func() tea.Msg {
		out, err := exec.Command("bash", "-c", cmd).CombinedOutput()
		if err != nil {
			return spawnHookMsg{agent: agent.Name, output: fmt.Sprintf("error: %v", err)}
		}
		return spawnHookMsg{agent: agent.Name, output: strings.TrimSpace(string(out))}
	}
}

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
