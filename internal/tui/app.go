package tui

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/history"
	"github.com/rk-senne/moonbase/internal/platform"
	"github.com/rk-senne/moonbase/internal/watcher"
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

// Buffer size limits for TUI state
const (
	maxIntelEntries   = 50
	maxTerminalLines  = 100
	maxSummaryChars   = 300
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

// testMode disables resource-heavy operations (file watcher) during testing.
// Overridden to true by TestMain in test files.
var testMode = false

type App struct {
	keys           KeyMap
	view           View
	registry       *agents.Registry
	backends       []backend.Backend
	dashboard      DashboardModel
	pipeline       PipelineModel
	width          int
	height         int
	ready          bool
	bootStep       int
	spinner        spinner.Model
	intel          []IntelEntry
	system         SystemModel
	mission        MissionModel
	search         SearchModel
	theme          string
	themeData      Theme
	styles         Styles
	clock          string
	startTime      time.Time
	focus          FocusPanel
	blink          bool
	comms          *CommsState
	commsInput     textinput.Model
	anim           AnimState
	fileWatcher    *watcher.Watcher
	ctx            platform.Context
	projectCtx     *discovery.ProjectContext
	activeBackend  backend.Backend
	snippetPick    SnippetPickerModel
	ctxFile        ContextFileModel
	docs           *DocsState
	projectNav     *ProjectsState
	terminal       TerminalModel
	fileBrowser     *FileBrowser
	browsing        bool // true = file browser mode, false = terminal mode
	toolCache       map[string]bool // cached exec.LookPath results (refreshed on timer)
	toolCacheTime   time.Time       // last time toolCache was refreshed
}

type MissionEntry struct {
	Name   string
	Status string
}


func NewApp() App {
	initialTheme := moonbaseTheme
	initialStyles := NewStyles(initialTheme)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(initialTheme.Active)

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

	cwd, _ := os.Getwd()

	// Start file watcher (skip in test mode to avoid FD exhaustion)
	var fw *watcher.Watcher
	if !testMode {
		fw, _ = watcher.New()
		if fw != nil {
			cwd, _ := os.Getwd()
			fw.Start(cwd)
		}
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

	dir, _ := agents.FindAgentsDir("")
	reg := agents.NewRegistry(dir)
	
	// Discover project context for pipeline execution
	projectCtx := discovery.Discover(cwd)
	
	// Select preferred backend
	activeBackend := backend.Preferred()

	term := NewTerminalModel()
	term.Cwd = cwd

	return App{
		keys:          DefaultKeyMap(),
		view:          ViewBoot,
		registry:      reg,
		spinner:       s,
		intel:         []IntelEntry{},
		projectCtx:    projectCtx,
		activeBackend: activeBackend,
		mission:      MissionModel{Input: ti, History: missionEntries},
		search:       SearchModel{Input: si},
		commsInput:   ci,
		ctxFile:      ContextFileModel{Input: fi},
		theme:        "moonbase",
		themeData:    initialTheme,
		styles:       initialStyles,
		clock:        time.Now().Format("15:04:05"),
		startTime:    time.Now(),
		focus:        FocusSidebar,
		fileWatcher:  fw,
		ctx:          platform.Detect(),
		terminal:     term,
		fileBrowser:  newFileBrowser(),
		browsing:     true, // start in file browser mode
		toolCache:    refreshToolCache(),
		toolCacheTime: time.Now(),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		bootTick(),
		LoadAgentsCmd(a.registry),
		detectSystem(),
		clockTick(),
		blinkTick(),
		animTick(),
		a.pollWatcher(),
		a.pollAgentDir(),
	)
}

// Update dispatches messages to the appropriate handler based on type and view.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleWindowSize(msg)

	case bootTickMsg:
		return a.handleBootTick()

	case bootDoneMsg:
		return a.handleBootDone()

	case AgentsLoadedMsg:
		return a.handleAgentsLoaded(msg)

	case PhaseResultMsg:
		return a.handlePhaseResultUpdate(msg)

	case PipelineAbortedMsg:
		return a.handlePipelineAborted()

	case systemInfoMsg:
		return a.handleSystemInfo(msg)

	case clockTickMsg:
		a.clock = time.Time(msg).Format("15:04:05")
		// Refresh tool cache every 30 seconds
		if time.Since(a.toolCacheTime) > 30*time.Second {
			a.toolCache = refreshToolCache()
			a.toolCacheTime = time.Now()
		}
		return a, nil

	case blinkTickMsg:
		a.blink = !a.blink
		return a, nil

	case animTickMsg:
		a.anim.Advance()
		return a, nil

	case spinner.TickMsg:
		return a.handleSpinnerTick(msg)

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
		return a.handleStreamChunk(msg)

	case fileChangeMsg:
		return a.handleFileChange(msg)

	case agentReloadMsg:
		a.registry.Reload()
		return a, nil

	case prCreatedMsg:
		a.addIntel("PR: %s", msg.output)

	case termOutputMsg:
		return a.handleTermOutput(msg)

	case termCdMsg:
		a.terminal = a.terminal.HandleCd(msg, a.themeData)
		if msg.err == nil && msg.newCwd != "" {
			a.addIntel("cd → %s", msg.newCwd)
		}
		return a, nil

	case termClearMsg:
		a.terminal = a.terminal.HandleClear()
		return a, nil

	case tea.KeyMsg:
		// Boot view: any key skips
		if a.view == ViewBoot {
			a.view = ViewDashboard
			a.addIntel("Boot skipped by operative.")
			return a, nil
		}
		// Search mode
		if a.search.Active {
			return a.handleSearchKeys(msg)
		}
		// Embedded terminal
		if a.terminal.Active && a.view == ViewDashboard {
			return a.handleTerminalKeys(msg)
		}
		// File browser
		if a.browsing && a.view == ViewDashboard && a.fileBrowser != nil {
			return a.handleFileBrowserKeys(msg)
		}
		// View-specific key handlers
		switch a.view {
		case ViewComms:
			return a.handleCommsKeys(msg)
		case ViewProjects:
			return a.handleProjectsKeys(msg)
		case ViewDocs:
			return a.handleDocsKeys(msg)
		case ViewMission:
			return a.handleMissionKeys(msg)
		default:
			return a.handleDashboardKeys(msg)
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
