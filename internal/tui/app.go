package tui

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
	"github.com/f5508037/moonbase/internal/chat"
	"github.com/f5508037/moonbase/internal/discovery"
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
	projectCtx     *discovery.ProjectContext
	activeBackend  backend.Backend
	abortPending   bool
	abortPendingAt time.Time
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
	fileBrowser     *FileBrowser
	browsing        bool // true = file browser mode, false = terminal mode
	pipelineRunning bool // prevents double-dispatch of pipeline phases
	streamCh        <-chan chat.StreamChunk // active stream channel for continued polling
	pipelineCtx     context.Context    // context for active pipeline execution
	cancelPipeline  context.CancelFunc // cancels the active pipeline context
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
	projectCtx, _ := discovery.Discover(cwd)
	
	// Select preferred backend
	activeBackend := backend.Preferred()

	return App{
		view:          ViewBoot,
		registry:      reg,
		spinner:       s,
		intel:         []IntelEntry{},
		projectCtx:    projectCtx,
		activeBackend: activeBackend,
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

// Update dispatches messages to the appropriate handler based on type and view.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return a.handleWindowSize(msg)

	case bootTickMsg:
		return a.handleBootTick()

	case bootDoneMsg:
		return a.handleBootDone()

	case agents.AgentsLoadedMsg:
		return a.handleAgentsLoaded(msg)

	case PhaseResultMsg:
		return a.handlePhaseResultUpdate(msg)

	case PipelineAbortedMsg:
		return a.handlePipelineAborted()

	case systemInfoMsg:
		return a.handleSystemInfo(msg)

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

	case tea.KeyMsg:
		// Boot view: any key skips
		if a.view == ViewBoot {
			a.view = ViewDashboard
			a.addIntel("Boot skipped by operative.")
			return a, nil
		}
		// Search mode
		if a.searching {
			return a.handleSearchKeys(msg)
		}
		// Embedded terminal
		if a.termActive && a.view == ViewDashboard {
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
