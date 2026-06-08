package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
	"github.com/f5508037/moonbase/internal/chat"
	"github.com/f5508037/moonbase/internal/history"
	"github.com/f5508037/moonbase/internal/pipeline"
	"github.com/f5508037/moonbase/internal/platform"
	"github.com/f5508037/moonbase/internal/snippets"
	"github.com/f5508037/moonbase/internal/watcher"
)

type View int

const (
	ViewBoot View = iota
	ViewDashboard
	ViewDossier
	ViewPipeline
	ViewHelp
	ViewMission
	ViewComms
	ViewHistory
	ViewDocs
	ViewProjects
	ViewProtocol
)

type IntelEntry struct {
	Time    string
	Message string
}

type FocusPanel int

const (
	FocusSidebar FocusPanel = iota
	FocusMain
	FocusRight
)

// PipelineMsg represents an agent chat message in the pipeline view
type PipelineMsg struct {
	Agent   string // agent name (empty = system message)
	Content string
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
	gitDiffLines   int
	dockerCount    int
	missionInput   textinput.Model
	searchInput    textinput.Model
	searching      bool
	filtered       []int
	theme          string
	pipelineState  *pipeline.Pipeline
	pipelineOutput []string
	pipelineChat   []PipelineMsg
	clock          string
	startTime      time.Time
	focus          FocusPanel
	blink          bool
	missions       []MissionEntry
	comms          *CommsState
	commsInput     textinput.Model
	anim           AnimState
	fileWatcher    *watcher.Watcher
	ctx            platform.Context
	snippetPicker  bool
	snippetList    []snippets.Snippet
	snippetCursor  int
	contextFile    bool // ctrl+f mode: typing file path
	contextInput   textinput.Model
	missionStart   time.Time
	docs           *DocsState
	projectNav     *ProjectsState
	termInput      textinput.Model
	termOutput     []string
	termActive     bool
	cwd            string
	fileBrowser    *FileBrowser
	browsing       bool // true = file browser mode, false = terminal mode
}

type MissionEntry struct {
	Name   string
	Status string
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

	ci := textinput.New()
	ci.Placeholder = "Type message..."
	ci.CharLimit = 2000
	ci.Width = 80

	fi := textinput.New()
	fi.Placeholder = "File path to attach..."
	fi.CharLimit = 256
	fi.Width = 60

	ti2 := textinput.New()
	ti2.Placeholder = "$ "
	ti2.CharLimit = 500
	ti2.Width = 80

	cwd, _ := os.Getwd()

	// Start file watcher
	fw, _ := watcher.New()
	if fw != nil {
		cwd, _ := os.Getwd()
		fw.Start(cwd)
	}

	// Load mission history for sidebar display
	hist := history.Load()
	var missionEntries []MissionEntry
	for i := len(hist) - 1; i >= 0 && len(missionEntries) < 5; i-- {
		status := "✅"
		if hist[i].Outcome == "aborted" {
			status = "❌"
		} else if hist[i].Outcome == "in-progress" {
			status = "🔄"
		}
		missionEntries = append(missionEntries, MissionEntry{Name: hist[i].Task, Status: status})
	}
	if len(missionEntries) == 0 {
		missionEntries = []MissionEntry{
			{Name: "init scaffold", Status: "✅"},
			{Name: "tui views", Status: "✅"},
			{Name: "pipeline+deploy", Status: "✅"},
		}
	}

	reg := agents.NewRegistry("./agents")
	return App{
		view:         ViewBoot,
		registry:     reg,
		spinner:      s,
		intel:        []IntelEntry{},
		missionInput: ti,
		searchInput:  si,
		commsInput:   ci,
		contextInput: fi,
		theme:        "moonbase",
		clock:        time.Now().Format("15:04:05"),
		startTime:    time.Now(),
		focus:        FocusSidebar,
		missions:     missionEntries,
		fileWatcher:  fw,
		ctx:          platform.Detect(),
		termInput:    ti2,
		cwd:          cwd,
		fileBrowser:  NewFileBrowser(),
		browsing:     true, // start in file browser mode
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		bootTick(),
		a.registry.Load(),
		detectSystem(),
		clockTick(),
		blinkTick(),
		animTick(),
		a.pollWatcher(),
		a.pollAgentDir(),
	)
}

type fileChangeMsg struct{ path string }
type agentReloadMsg struct{}

func (a App) pollWatcher() tea.Cmd {
	if a.fileWatcher == nil {
		return nil
	}
	return func() tea.Msg {
		ev := <-a.fileWatcher.Events
		return fileChangeMsg{path: ev.Path}
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
					// Parse summary line like " 3 files changed, 40 insertions(+), 10 deletions(-)"
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
		if a.termActive && a.view == ViewDashboard {
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
		if a.browsing && a.view == ViewDashboard && a.fileBrowser != nil {
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
				// Toggle hidden files — refresh
				a.fileBrowser.refresh()
			case "esc":
				a.browsing = false
			}
			return a, nil
		}
		if a.view == ViewComms {
			// Context file input mode
			if a.contextFile {
				switch msg.String() {
				case "esc":
					a.contextFile = false
					a.contextInput.Blur()
				case "enter":
					path := a.contextInput.Value()
					if data, err := os.ReadFile(path); err == nil {
						inject := fmt.Sprintf("[attached: %s]\n```\n%s\n```", path, string(data))
						a.commsInput.SetValue(a.commsInput.Value() + inject)
					} else {
						a.addIntel("File not found: %s", path)
					}
					a.contextFile = false
					a.contextInput.Reset()
					a.contextInput.Blur()
				default:
					var cmd tea.Cmd
					a.contextInput, cmd = a.contextInput.Update(msg)
					return a, cmd
				}
				return a, nil
			}
			// Snippet picker mode
			if a.snippetPicker {
				switch msg.String() {
				case "esc":
					a.snippetPicker = false
				case "up", "k":
					if a.snippetCursor > 0 {
						a.snippetCursor--
					}
				case "down", "j":
					if a.snippetCursor < len(a.snippetList)-1 {
						a.snippetCursor++
					}
				case "enter":
					if len(a.snippetList) > 0 {
						a.commsInput.SetValue(a.snippetList[a.snippetCursor].Content)
						a.snippetPicker = false
					}
				}
				return a, nil
			}
			switch msg.String() {
			case "esc":
				a.view = ViewDossier
				a.commsInput.Blur()
			case "enter":
				if !a.comms.streaming {
					val := a.commsInput.Value()
					// @agent — switch active agent
					if strings.HasPrefix(val, "@") {
						agentName := strings.TrimPrefix(val, "@")
						a.switchCommsAgent(agentName)
						a.commsInput.Reset()
						return a, nil
					}
					// >>agent message — relay custom message to agent
					if strings.HasPrefix(val, ">>") {
						parts := strings.SplitN(strings.TrimPrefix(val, ">>"), " ", 2)
						if len(parts) >= 2 {
							a.commsInput.Reset()
							cmd := a.relayToAgent(parts[0], parts[1])
							if cmd != nil {
								return a, cmd
							}
						}
						return a, nil
					}
					// >agent — relay last assistant response to agent
					if strings.HasPrefix(val, ">") {
						target := strings.TrimPrefix(val, ">")
						a.commsInput.Reset()
						cmd := a.relayToAgent(target, "")
						if cmd != nil {
							return a, cmd
						}
						return a, nil
					}
					cmd := a.sendCommsMessage()
					if cmd != nil {
						return a, cmd
					}
				}
			case "ctrl+f":
				a.contextFile = true
				a.contextInput.Focus()
				return a, textinput.Blink
			case "ctrl+s":
				a.snippetList = snippets.ForAgent(a.comms.agent)
				a.snippetCursor = 0
				a.snippetPicker = true
			case "ctrl+c":
				return a, tea.Quit
			default:
				if !a.comms.streaming {
					var cmd tea.Cmd
					a.commsInput, cmd = a.commsInput.Update(msg)
					return a, cmd
				}
			}
			return a, nil
		}
		if a.view == ViewProjects && a.projectNav != nil {
			switch msg.String() {
			case "esc":
				a.view = ViewDashboard
			case "up", "k":
				if a.projectNav.cursor > 0 {
					a.projectNav.cursor--
				}
			case "down", "j":
				if a.projectNav.cursor < len(a.projectNav.list)-1 {
					a.projectNav.cursor++
				}
			case "enter":
				a.selectProject()
			case "M":
				return a, a.launchCmux()
			case "F":
				return a, a.launchTool("fish")
			}
			return a, nil
		}
		if a.view == ViewDocs && a.docs != nil {
			switch msg.String() {
			case "esc":
				a.view = ViewDashboard
			case "up", "k":
				if a.docs.cursor > 0 {
					a.docs.cursor--
				}
			case "down", "j":
				if a.docs.cursor < len(a.docs.files)-1 {
					a.docs.cursor++
				}
			case "enter":
				a.docs.loadDoc(a.docs.cursor, a.width-30)
			case "pgdown", " ":
				a.docs.viewport.HalfViewDown()
			case "pgup":
				a.docs.viewport.HalfViewUp()
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
					}
					a.pipelineChat = []PipelineMsg{
						{"", fmt.Sprintf("━━━ MISSION: %s ━━━", task)},
						{"Numbuh 1", "Receiving mission brief... Analyzing requirements."},
						{"Numbuh 1", "Breaking down objectives into acceptance criteria."},
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
		case "F1":
			a.view = ViewProtocol
		case "esc":
			if a.view == ViewPipeline {
				a.view = ViewDashboard
			} else {
				a.view = ViewDashboard
			}
		case "n":
			if a.view == ViewPipeline && a.pipelineState != nil {
				prev := a.pipelineState.Phases[a.pipelineState.Current]
				a.pipelineState.Advance()
				if a.pipelineState.Current < len(a.pipelineState.Phases) {
					phase := a.pipelineState.Phases[a.pipelineState.Current]
					a.pipelineOutput = append(a.pipelineOutput, "",
						fmt.Sprintf("Phase %d: %s activated...", phase.Number, phase.Operative))
					a.pipelineState.Phases[a.pipelineState.Current].Status = 1
					// Inter-agent handoff chat
					a.pipelineChat = append(a.pipelineChat,
						PipelineMsg{prev.Operative, fmt.Sprintf("Phase complete. Handing off to %s.", phase.Operative)},
						PipelineMsg{"", "───────────────────────────────────"},
						PipelineMsg{phase.Operative, fmt.Sprintf("Received handoff from %s. Starting %s phase.", prev.Operative, phase.Name)},
					)
				}
			}
		case "r":
			if a.view == ViewPipeline && a.pipelineState != nil {
				a.pipelineState.Retry()
				phase := a.pipelineState.Phases[a.pipelineState.Current]
				a.pipelineOutput = append(a.pipelineOutput,
					fmt.Sprintf("⚠️ RETRYING Phase %d: %s...", phase.Number, phase.Operative))
				a.pipelineChat = append(a.pipelineChat,
					PipelineMsg{phase.Operative, "⚠️ Retrying... Let me take another look at this."},
				)
			}
		case "s":
			if a.view == ViewPipeline && a.pipelineState != nil {
				phase := a.pipelineState.Phases[a.pipelineState.Current]
				a.pipelineOutput = append(a.pipelineOutput,
					fmt.Sprintf("⊘ SKIPPED Phase %d: %s", phase.Number, phase.Operative))
				a.pipelineChat = append(a.pipelineChat,
					PipelineMsg{phase.Operative, "⊘ Phase skipped by operator."},
				)
				a.pipelineState.Skip()
				if a.pipelineState.Current < len(a.pipelineState.Phases) {
					next := a.pipelineState.Phases[a.pipelineState.Current]
					a.pipelineOutput = append(a.pipelineOutput,
						fmt.Sprintf("Phase %d: %s activated...", next.Number, next.Operative))
					a.pipelineChat = append(a.pipelineChat,
						PipelineMsg{next.Operative, fmt.Sprintf("Stepping in. Previous phase was skipped. Starting %s.", next.Name)},
					)
				}
			}
		case "up", "k":
			if a.view == ViewDashboard || a.view == ViewDossier {
				if a.cursor > 0 {
					a.cursor--
				}
				a.selected = a.cursorToAgent()
				a.anim.TriggerSelectPulse()
			}
		case "down", "j":
			if a.view == ViewDashboard || a.view == ViewDossier {
				if a.cursor < a.registry.Count()-1 {
					a.cursor++
				}
				a.selected = a.cursorToAgent()
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
			// Trigger typewriter for the final message
			if a.bootStep == len(bootMessages)-1 {
				a.anim.TriggerTypewriter()
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
		a.gitDiffLines = msg.diffLines

	case clockTickMsg:
		a.clock = time.Time(msg).Format("15:04:05")
		return a, nil

	case blinkTickMsg:
		a.blink = !a.blink
		return a, nil

	case animTickMsg:
		a.anim.Advance()
		return a, nil

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

	case toolExitMsg:
		a.addIntel("Tool exited: %s", msg.tool)

	case streamChunkMsg:
		if a.comms == nil {
			return a, nil
		}
		if msg.err != nil {
			a.comms.buffer += fmt.Sprintf("\n[ERROR: %s]", msg.err)
			a.comms.FinishStream()
			return a, nil
		}
		if msg.done {
			a.comms.FinishStream()
			a.ringBell()
			return a, nil
		}
		a.comms.AppendStreamToken(msg.text)
		// Continue polling the stream
		return a, pollStream(streamCh)

	case fileChangeMsg:
		a.addIntel("📁 %s modified", msg.path)
		return a, a.pollWatcher()

	case agentReloadMsg:
		// Hot-reload: re-read agents dir
		a.registry.Reload()
		return a, nil

	case prCreatedMsg:
		a.addIntel("PR: %s", msg.output)

	case termOutputMsg:
		// Add command prompt line
		a.termOutput = append(a.termOutput, lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+msg.cmd))
		// Add output lines
		for _, line := range strings.Split(msg.output, "\n") {
			a.termOutput = append(a.termOutput, line)
		}
		// Keep last 100 lines
		if len(a.termOutput) > 100 {
			a.termOutput = a.termOutput[len(a.termOutput)-100:]
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
	case ViewComms:
		return a.renderComms()
	case ViewHistory:
		return a.renderHistory()
	case ViewDocs:
		return a.renderDocs()
	case ViewProjects:
		return a.renderProjects()
	case ViewProtocol:
		return a.renderProtocol()
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
	a.anim.TriggerIntelFlash()
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

func (a App) uptime() string {
	d := time.Since(a.startTime)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
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
type streamChunkMsg struct {
	text string
	done bool
	err  error
}

type termOutputMsg struct {
	cmd    string
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
	case "classified":
		a.theme = "nerv"
		ColorActive = lipgloss.Color("#FF6600")
		ColorInfo = lipgloss.Color("#FF3399")
		ColorBrand = lipgloss.Color("#9900CC")
		ColorHeader = lipgloss.Color("#FF6600")
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

type toolExitMsg struct{ tool string }

func (a App) launchTool(name string) tea.Cmd {
	bin, err := exec.LookPath(name)
	if err != nil {
		return func() tea.Msg {
			return toolExitMsg{tool: name + " (not found)"}
		}
	}
	c := exec.Command(bin)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return toolExitMsg{tool: name}
	})
}

func (a App) launchNvim() tea.Cmd {
	bin, err := exec.LookPath("nvim")
	if err != nil {
		return func() tea.Msg { return toolExitMsg{tool: "nvim (not found)"} }
	}
	var args []string
	if a.view == ViewProjects && a.projectNav != nil && len(a.projectNav.list) > 0 {
		args = append(args, a.projectNav.list[a.projectNav.cursor].Path)
	} else if a.browsing && a.fileBrowser != nil && len(a.fileBrowser.entries) > 0 {
		entry := a.fileBrowser.entries[a.fileBrowser.cursor]
		if !entry.IsDir {
			args = append(args, filepath.Join(a.fileBrowser.dir, entry.Name))
		}
	}
	c := exec.Command(bin, args...)
	return tea.ExecProcess(c, func(err error) tea.Msg { return toolExitMsg{tool: "nvim"} })
}

func (a App) launchCmux() tea.Cmd {
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return func() tea.Msg { return toolExitMsg{tool: "tmux (not found)"} }
	}
	// Check if session "moonbase" exists
	check := exec.Command(bin, "has-session", "-t", "moonbase")
	if check.Run() == nil {
		c := exec.Command(bin, "attach-session", "-t", "moonbase")
		return tea.ExecProcess(c, func(err error) tea.Msg { return toolExitMsg{tool: "tmux"} })
	}
	c := exec.Command(bin, "new-session", "-s", "moonbase")
	return tea.ExecProcess(c, func(err error) tea.Msg { return toolExitMsg{tool: "tmux"} })
}

func (a App) editFile(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	bin, err := exec.LookPath(editor)
	if err != nil {
		return func() tea.Msg {
			return toolExitMsg{tool: editor + " (not found)"}
		}
	}
	c := exec.Command(bin, path)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return toolExitMsg{tool: editor}
	})
}

func (a *App) openComms() {
	agent := a.registry.Get(a.selected)
	// Load existing chat history or start fresh
	conv := chat.Load(agent.Name, agent.Prompt)
	if conv != nil {
		a.comms = &CommsState{
			conv:     conv,
			viewport: viewport.New(a.width-6, a.height-6),
			agent:    agent.Name,
		}
		a.comms.rebuildContent()
	} else {
		a.comms = NewCommsState(agent.Name, agent.Prompt, a.width, a.height)
	}
	a.view = ViewComms
	a.commsInput.Focus()
	a.commsInput.Width = a.width - 8
}

func (a *App) sendCommsMessage() tea.Cmd {
	msg := a.commsInput.Value()
	if msg == "" {
		return nil
	}
	a.commsInput.Reset()
	a.comms.AddUserMessage(msg)
	a.comms.streaming = true

	// Start streaming and store channel for continued polling
	streamCh = chat.Stream(a.comms.conv)
	return pollStream(streamCh)
}

func pollStream(ch <-chan chat.StreamChunk) tea.Cmd {
	return func() tea.Msg {
		chunk, ok := <-ch
		if !ok {
			return streamChunkMsg{done: true}
		}
		return streamChunkMsg{text: chunk.Text, done: chunk.Done, err: chunk.Err}
	}
}

// streamCh holds the active stream channel for continued polling
var streamCh <-chan chat.StreamChunk

// --- GitHub PR ---

type prCreatedMsg struct{ output string }

func (a App) createPR() tea.Cmd {
	return func() tea.Msg {
		// Check if gh is available
		if _, err := exec.LookPath("gh"); err != nil {
			return prCreatedMsg{output: "gh CLI not found. Install: https://cli.github.com"}
		}
		// Get current branch
		branch, _ := exec.Command("git", "branch", "--show-current").Output()
		branchName := strings.TrimSpace(string(branch))
		if branchName == "main" || branchName == "master" {
			return prCreatedMsg{output: "Cannot create PR from main/master. Switch to a feature branch."}
		}
		// Create PR
		out, err := exec.Command("gh", "pr", "create", "--fill").CombinedOutput()
		if err != nil {
			return prCreatedMsg{output: fmt.Sprintf("PR failed: %s", string(out))}
		}
		return prCreatedMsg{output: strings.TrimSpace(string(out))}
	}
}

// --- Multi-Agent COMMS ---

func (a *App) switchCommsAgent(name string) {
	// Find agent by partial name match
	for _, agent := range a.registry.All() {
		if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(name)) {
			a.comms.agent = agent.Name
			a.comms.conv.System = agent.Prompt
			a.addIntel("COMMS switched to: %s", agent.Name)
			return
		}
	}
	a.addIntel("Agent not found: %s", name)
}

// relayToAgent sends a message to another agent's conversation and streams the response.
// If msg is empty, relays the last assistant message from the current conversation.
func (a *App) relayToAgent(targetName, msg string) tea.Cmd {
	// Find target agent
	var target *agents.Agent
	for _, agent := range a.registry.All() {
		if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(targetName)) {
			ag := agent
			target = &ag
			break
		}
	}
	if target == nil {
		a.addIntel("Relay failed — agent not found: %s", targetName)
		return nil
	}

	// Build relay content
	fromAgent := a.comms.agent
	if msg == "" {
		// Get last assistant message from current conversation
		for i := len(a.comms.conv.Messages) - 1; i >= 0; i-- {
			if a.comms.conv.Messages[i].Role == chat.RoleAssistant {
				msg = a.comms.conv.Messages[i].Content
				break
			}
		}
		if msg == "" {
			a.addIntel("Relay failed — no response to relay")
			return nil
		}
	}

	relayMsg := fmt.Sprintf("[Relayed from %s]\n%s", fromAgent, msg)

	// Switch to target agent's conversation
	conv := chat.Load(target.Name, target.Prompt)
	if conv == nil {
		conv = chat.NewConversation(target.Name, target.Prompt)
	}
	a.comms = &CommsState{
		conv:     conv,
		viewport: viewport.New(a.width-6, a.height-6),
		agent:    target.Name,
	}
	a.comms.rebuildContent()

	// Send the relay message and stream response
	a.comms.AddUserMessage(relayMsg)
	a.comms.streaming = true
	a.addIntel("Relayed to %s from %s", target.Name, fromAgent)

	streamCh = chat.Stream(a.comms.conv)
	return pollStream(streamCh)
}

// --- Mission History View ---

func (a App) renderHistory() string {
	header := a.renderHeader("MISSION HISTORY")

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)
	labelStyle := lipgloss.NewStyle().Foreground(ColorInfo)

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ◆ MISSION LOG") + "\n")
	b.WriteString(labelStyle.Render("  ─────────────────────────────────────────────") + "\n\n")

	missions := history.Load()
	if len(missions) == 0 {
		b.WriteString(dimStyle.Render("  No missions logged yet.") + "\n")
		b.WriteString(dimStyle.Render("  Run a pipeline to start recording history.") + "\n")
	} else {
		b.WriteString(fmt.Sprintf("  %-4s %-30s %-10s %s\n", "ID", "TASK", "OUTCOME", "DURATION"))
		b.WriteString(labelStyle.Render("  ─────────────────────────────────────────────") + "\n")
		// Show most recent first
		for i := len(missions) - 1; i >= 0 && i >= len(missions)-20; i-- {
			m := missions[i]
			status := "✅"
			if m.Outcome == "aborted" {
				status = "❌"
			}
			task := m.Task
			if len(task) > 28 {
				task = task[:28] + ".."
			}
			b.WriteString(fmt.Sprintf("  %-4d %-30s %s %-10s %s\n",
				m.ID, task, status, m.Outcome, m.Duration))
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  [esc] Back") + "\n")

	body := StylePanel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderStatusBar("[esc] BACK TO DASHBOARD")
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
}

// --- Notification Bell ---

func (a *App) ringBell() {
	fmt.Print("\a") // terminal bell
	a.anim.TriggerIntelFlash()
}

// --- Embedded Terminal ---

func (a *App) execTermCmd(input string) tea.Cmd {
	// Handle built-in cd
	if strings.HasPrefix(input, "cd ") {
		dir := strings.TrimPrefix(input, "cd ")
		dir = strings.TrimSpace(dir)
		if dir == "~" {
			dir, _ = os.UserHomeDir()
		} else if strings.HasPrefix(dir, "~/") {
			home, _ := os.UserHomeDir()
			dir = home + dir[1:]
		}
		if err := os.Chdir(dir); err != nil {
			a.termOutput = append(a.termOutput,
				lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+input),
				lipgloss.NewStyle().Foreground(ColorError).Render(err.Error()))
		} else {
			a.cwd, _ = os.Getwd()
			a.termOutput = append(a.termOutput,
				lipgloss.NewStyle().Foreground(ColorActive).Render("$ "+input))
			a.addIntel("cd → %s", a.cwd)
		}
		return nil
	}
	// Handle clear
	if input == "clear" {
		a.termOutput = nil
		return nil
	}

	return func() tea.Msg {
		out, err := exec.Command("bash", "-c", input).CombinedOutput()
		result := strings.TrimRight(string(out), "\n")
		if err != nil && result == "" {
			result = err.Error()
		}
		return termOutputMsg{cmd: input, output: result}
	}
}
