package tui

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/agents"
)

// newTestRegistry creates a registry pre-loaded with agents from the real agents directory.
func newTestRegistry() *agents.Registry {
	// When running tests, CWD is the package dir. Walk up to project root.
	dir, err := agents.FindAgentsDir("")
	if err != nil {
		// Fallback: try the relative path from package to project root
		dir = "../../agents"
	}
	reg := agents.NewRegistry(dir)
	reg.Reload()
	return reg
}

func TestNewApp_Initializes(t *testing.T) {
	app := NewApp()

	if app.view != ViewBoot {
		t.Errorf("expected initial view=ViewBoot (%d), got %d", ViewBoot, app.view)
	}
	if app.ready {
		t.Error("expected ready=false on new app")
	}
	if app.bootStep != 0 {
		t.Errorf("expected bootStep=0, got %d", app.bootStep)
	}
	if app.dashboard.Cursor != 0 {
		t.Errorf("expected cursor=0, got %d", app.dashboard.Cursor)
	}
	if app.theme != "moonbase" {
		t.Errorf("expected theme=moonbase, got %s", app.theme)
	}
	if app.focus != FocusSidebar {
		t.Errorf("expected focus=FocusSidebar, got %d", app.focus)
	}
}

func TestApp_BootSequence(t *testing.T) {
	app := NewApp()
	app.ready = true

	if app.view != ViewBoot {
		t.Fatalf("expected ViewBoot, got %d", app.view)
	}

	model, _ := app.Update(bootTickMsg{})
	result := model.(App)
	if result.bootStep != 1 {
		t.Errorf("expected bootStep=1 after first tick, got %d", result.bootStep)
	}
	if result.view != ViewBoot {
		t.Errorf("expected still in ViewBoot during sequence, got %d", result.view)
	}

	for i := 0; i < 3; i++ {
		model, _ = result.Update(bootTickMsg{})
		result = model.(App)
	}
	if result.bootStep != 4 {
		t.Errorf("expected bootStep=4 after 4 ticks total, got %d", result.bootStep)
	}
}

func TestApp_BootSkipOnKeyPress(t *testing.T) {
	app := NewApp()
	app.ready = true

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after boot skip, got %d", result.view)
	}
}

func TestApp_BootDoneMsg(t *testing.T) {
	app := NewApp()
	app.ready = true
	app.view = ViewBoot

	model, _ := app.Update(bootDoneMsg{})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after bootDone, got %d", result.view)
	}
}

func TestApp_KeyNavigation_MissionView(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	result := model.(App)
	if result.view != ViewMission {
		t.Errorf("expected ViewMission after 'm', got %d", result.view)
	}
}

func TestApp_KeyNavigation_HelpToggle(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	result := model.(App)
	if result.view != ViewHelp {
		t.Errorf("expected ViewHelp after '?', got %d", result.view)
	}

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	result = model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after second '?', got %d", result.view)
	}
}

func TestApp_KeyNavigation_CursorUpDown(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false
	app.registry = newTestRegistry()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := model.(App)
	if result.dashboard.Cursor != 1 {
		t.Errorf("expected cursor=1 after 'j', got %d", result.dashboard.Cursor)
	}
	if result.dashboard.Selected != 1 {
		t.Errorf("expected selected=1 after 'j', got %d", result.dashboard.Selected)
	}

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result = model.(App)
	if result.dashboard.Cursor != 0 {
		t.Errorf("expected cursor=0 after 'k', got %d", result.dashboard.Cursor)
	}
}

func TestApp_KeyNavigation_DossierView(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after enter, got %d", result.view)
	}
}

func TestApp_KeyNavigation_HistoryView(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	result := model.(App)
	if result.view != ViewHistory {
		t.Errorf("expected ViewHistory after 'H', got %d", result.view)
	}
}

func TestApp_MissionInput(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.ready = true
	app.missionInput.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	result := model.(App)
	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	result = model.(App)

	if result.missionInput.Value() != "hi" {
		t.Errorf("expected mission input value='hi', got '%s'", result.missionInput.Value())
	}
}

func TestApp_MissionInput_EscReturns(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.ready = true
	app.missionInput.Focus()

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in mission, got %d", result.view)
	}
}

func TestApp_MissionInput_Submit(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.ready = true
	app.missionInput.Focus()
	app.missionInput.SetValue("add pagination")

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewPipeline {
		t.Errorf("expected ViewPipeline after mission submit, got %d", result.view)
	}
	if result.pipeline.State == nil {
		t.Fatal("expected pipelineState to be set after mission submit")
	}
	if result.pipeline.State.Task != "add pagination" {
		t.Errorf("expected pipeline task='add pagination', got '%s'", result.pipeline.State.Task)
	}
}

func TestApp_QuitKey(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestApp_CtrlC_Quit(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command from ctrl+c, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg from ctrl+c, got %T", msg)
	}
}

func TestApp_WindowSizeMsg(t *testing.T) {
	app := NewApp()
	app.ready = false

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := model.(App)
	if !result.ready {
		t.Error("expected ready=true after WindowSizeMsg")
	}
	if result.width != 120 {
		t.Errorf("expected width=120, got %d", result.width)
	}
	if result.height != 40 {
		t.Errorf("expected height=40, got %d", result.height)
	}
}

func TestApp_TabCyclesFocus(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyTab})
	result := model.(App)
	if result.focus != FocusMain {
		t.Errorf("expected focus=FocusMain after first tab, got %d", result.focus)
	}

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyTab})
	result = model.(App)
	if result.focus != FocusRight {
		t.Errorf("expected focus=FocusRight after second tab, got %d", result.focus)
	}

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyTab})
	result = model.(App)
	if result.focus != FocusSidebar {
		t.Errorf("expected focus=FocusSidebar after third tab (wrap), got %d", result.focus)
	}
}

func TestApp_ThemeCycle(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	result := model.(App)
	if result.theme != "treehouse" {
		t.Errorf("expected theme=treehouse after first 'T', got %s", result.theme)
	}

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	result = model.(App)
	if result.theme != "classified" {
		t.Errorf("expected theme=classified after second 'T', got %s", result.theme)
	}

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	result = model.(App)
	if result.theme != "nerv" {
		t.Errorf("expected theme=nerv after third 'T', got %s", result.theme)
	}

	model, _ = result.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	result = model.(App)
	if result.theme != "moonbase" {
		t.Errorf("expected theme=moonbase after fourth 'T' (wrap), got %s", result.theme)
	}
}

func TestApp_PipelineEscAbort(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.ready = true
	app.browsing = false
	app.terminal.Active = false
	app.pipeline.Running = true

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := model.(App)
	if !result.pipeline.AbortPending {
		t.Error("expected abortPending=true after first esc during running pipeline")
	}

	// When pipeline is NOT running, esc returns to dashboard
	app2 := NewApp()
	app2.view = ViewPipeline
	app2.ready = true
	app2.browsing = false
	app2.terminal.Active = false
	app2.pipeline.Running = false

	model, _ = app2.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result = model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc on idle pipeline, got %d", result.view)
	}
}

func TestApp_View_ReturnsInitializing(t *testing.T) {
	app := NewApp()
	app.ready = false

	output := app.View()
	if output != "  Initializing..." {
		t.Errorf("expected 'Initializing...' when not ready, got %q", output)
	}
}

func TestApp_CursorBoundsCheck(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.ready = true
	app.browsing = false
	app.terminal.Active = false
	app.dashboard.Cursor = 0

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result := model.(App)
	if result.dashboard.Cursor != 0 {
		t.Errorf("expected cursor stays at 0, got %d", result.dashboard.Cursor)
	}
}

// TestApp_FieldCountBounded asserts the top-level App struct field budget.
// The design target is ≤15 fields (orchestration/shared only). The current count
// is higher because several field groups have not yet been extracted into sub-models:
//
// Remaining groups that could be extracted in future phases:
//   - Search state (searchInput, searching, filtered) → SearchModel
//   - Git state (gitBranch, gitClean, gitDiffLines) → GitModel or into DashboardModel
//   - Comms-related (comms, commsInput, snippetPicker, snippetList, snippetCursor, contextFile, contextInput) → into CommsState
//   - Mission input (missionInput, missions) → MissionModel
//   - Boot state (bootStep, ready) → BootModel
//   - Visual state (clock, startTime, focus, blink, anim) → part of rendering context
//   - Infra (fileWatcher, toolCache, toolCacheTime, ctx) → infra/platform grouping
//
// Each extraction is its own task; this test documents the current count to catch
// unintentional field additions.
func TestApp_FieldCountBounded(t *testing.T) {
	count := reflect.TypeOf(App{}).NumField()
	// Current field count after Phase 5 extractions (TerminalModel, DashboardModel, PipelineModel).
	// Target was ≤15, actual is 48 because only 3 of ~8 potential sub-models have been
	// extracted. The remaining fields are documented above for future extraction work.
	const maxFields = 48
	if count > maxFields {
		t.Errorf("App has %d fields, expected ≤ %d — did you add fields without extracting? See comment above.", count, maxFields)
	}
}
