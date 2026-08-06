package tui

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/agents"
)

// newTestRegistry creates a registry pre-loaded with agents from the real agents
// directory. The path is resolved from THIS source file's location (not the
// process CWD) so the registry is immune to os.Chdir side effects from other
// tests (e.g. the file browser's Enter/Back), keeping agent count deterministic.
func newTestRegistry() *agents.Registry {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(filename), "..", "..", "agents")
	reg := agents.NewRegistry(dir)
	reg.Reload()
	return reg
}

func TestNewApp_Initializes(t *testing.T) {
	app := NewApp()

	if app.view != ViewBoot {
		t.Errorf("expected initial view=ViewBoot (%d), got %d", ViewBoot, app.view)
	}
	if app.boot.Ready {
		t.Error("expected ready=false on new app")
	}
	if app.boot.Step != 0 {
		t.Errorf("expected bootStep=0, got %d", app.boot.Step)
	}
	if app.views.Dashboard.Cursor != 0 {
		t.Errorf("expected cursor=0, got %d", app.views.Dashboard.Cursor)
	}
	if app.theme.Name != "moonbase" {
		t.Errorf("expected theme=moonbase, got %s", app.theme.Name)
	}
	if app.chrome.Focus != FocusSidebar {
		t.Errorf("expected focus=FocusSidebar, got %d", app.chrome.Focus)
	}
}

func TestApp_BootSequence(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true

	if app.view != ViewBoot {
		t.Fatalf("expected ViewBoot, got %d", app.view)
	}

	model, _ := app.Update(bootTickMsg{})
	result := model.(App)
	if result.boot.Step != 1 {
		t.Errorf("expected bootStep=1 after first tick, got %d", result.boot.Step)
	}
	if result.view != ViewBoot {
		t.Errorf("expected still in ViewBoot during sequence, got %d", result.view)
	}

	for i := 0; i < 3; i++ {
		model, _ = result.Update(bootTickMsg{})
		result = model.(App)
	}
	if result.boot.Step != 4 {
		t.Errorf("expected bootStep=4 after 4 ticks total, got %d", result.boot.Step)
	}
}

func TestApp_BootSkipOnKeyPress(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true

	model, _ := app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after boot skip, got %d", result.view)
	}
}

func TestApp_BootDoneMsg(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
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
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	result := model.(App)
	if result.view != ViewMission {
		t.Errorf("expected ViewMission after 'm', got %d", result.view)
	}
}

func TestApp_KeyNavigation_HelpToggle(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	result := model.(App)
	if result.view != ViewHelp {
		t.Errorf("expected ViewHelp after '?', got %d", result.view)
	}

	model, _ = result.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	result = model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after second '?', got %d", result.view)
	}
}

func TestApp_KeyNavigation_CursorUpDown(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.registry = newTestRegistry()

	// Navigation follows the sidebar's VISUAL display order, not raw registry
	// index order (which would appear to skip grouped agents).
	order := sidebarDisplayOrder(app.registry)
	if len(order) < 2 {
		t.Skip("need at least 2 displayed agents")
	}
	app.views.Dashboard.Cursor = order[0]
	app.views.Dashboard.Selected = order[0]

	model, _ := app.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	result := model.(App)
	if result.views.Dashboard.Cursor != order[1] {
		t.Errorf("expected cursor=%d (2nd displayed) after 'j', got %d", order[1], result.views.Dashboard.Cursor)
	}
	if result.views.Dashboard.Selected != order[1] {
		t.Errorf("expected selected=%d after 'j', got %d", order[1], result.views.Dashboard.Selected)
	}

	model, _ = result.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	result = model.(App)
	if result.views.Dashboard.Cursor != order[0] {
		t.Errorf("expected cursor=%d (1st displayed) after 'k', got %d", order[0], result.views.Dashboard.Cursor)
	}
}

func TestApp_KeyNavigation_DossierView(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewDossier {
		t.Errorf("expected ViewDossier after enter, got %d", result.view)
	}
}

func TestApp_KeyNavigation_HistoryView(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: 'H', Text: "H"})
	result := model.(App)
	if result.view != ViewHistory {
		t.Errorf("expected ViewHistory after 'H', got %d", result.view)
	}
}

func TestApp_MissionInput(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()

	model, _ := app.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	result := model.(App)
	model, _ = result.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	result = model.(App)

	if result.views.Mission.Input.Value() != "hi" {
		t.Errorf("expected mission input value='hi', got '%s'", result.views.Mission.Input.Value())
	}
}

func TestApp_MissionInput_EscReturns(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc in mission, got %d", result.view)
	}
}

func TestApp_MissionInput_Submit(t *testing.T) {
	app := NewApp()
	app.view = ViewMission
	app.boot.Ready = true
	app.views.Mission.Input.Focus()
	app.views.Mission.Input.SetValue("add pagination")

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := model.(App)
	if result.view != ViewPipeline {
		t.Errorf("expected ViewPipeline after mission submit, got %d", result.view)
	}
	if result.views.Pipeline.State == nil {
		t.Fatal("expected pipelineState to be set after mission submit")
	}
	if result.views.Pipeline.State.Task != "add pagination" {
		t.Errorf("expected pipeline task='add pagination', got '%s'", result.views.Pipeline.State.Task)
	}
}

func TestApp_QuitKey(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	_, cmd := app.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
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
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	_, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
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
	app.boot.Ready = false

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := model.(App)
	if !result.boot.Ready {
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
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	result := model.(App)
	if result.chrome.Focus != FocusMain {
		t.Errorf("expected focus=FocusMain after first tab, got %d", result.chrome.Focus)
	}

	model, _ = result.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	result = model.(App)
	if result.chrome.Focus != FocusRight {
		t.Errorf("expected focus=FocusRight after second tab, got %d", result.chrome.Focus)
	}

	model, _ = result.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	result = model.(App)
	if result.chrome.Focus != FocusSidebar {
		t.Errorf("expected focus=FocusSidebar after third tab (wrap), got %d", result.chrome.Focus)
	}
}

func TestApp_ThemeCycle(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false

	model, _ := app.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	result := model.(App)
	if result.theme.Name != "treehouse" {
		t.Errorf("expected theme=treehouse after first 'T', got %s", result.theme.Name)
	}

	model, _ = result.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	result = model.(App)
	if result.theme.Name != "classified" {
		t.Errorf("expected theme=classified after second 'T', got %s", result.theme.Name)
	}

	model, _ = result.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	result = model.(App)
	if result.theme.Name != "nerv" {
		t.Errorf("expected theme=nerv after third 'T', got %s", result.theme.Name)
	}

	model, _ = result.Update(tea.KeyPressMsg{Code: 'T', Text: "T"})
	result = model.(App)
	if result.theme.Name != "moonbase" {
		t.Errorf("expected theme=moonbase after fourth 'T' (wrap), got %s", result.theme.Name)
	}
}

func TestApp_PipelineEscAbort(t *testing.T) {
	app := NewApp()
	app.view = ViewPipeline
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.views.Pipeline.Running = true

	model, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result := model.(App)
	if !result.views.Pipeline.AbortPending {
		t.Error("expected abortPending=true after first esc during running pipeline")
	}

	// When pipeline is NOT running, esc returns to dashboard
	app2 := NewApp()
	app2.view = ViewPipeline
	app2.boot.Ready = true
	app2.views.Browser.Active = false
	app2.views.Terminal.Active = false
	app2.views.Pipeline.Running = false

	model, _ = app2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	result = model.(App)
	if result.view != ViewDashboard {
		t.Errorf("expected ViewDashboard after esc on idle pipeline, got %d", result.view)
	}
}

func TestApp_View_ReturnsInitializing(t *testing.T) {
	app := NewApp()
	app.boot.Ready = false

	output := app.renderFrame()
	if output != "  Initializing..." {
		t.Errorf("expected 'Initializing...' when not ready, got %q", output)
	}
}

func TestApp_CursorBoundsCheck(t *testing.T) {
	app := NewApp()
	app.view = ViewDashboard
	app.boot.Ready = true
	app.views.Browser.Active = false
	app.views.Terminal.Active = false
	app.views.Dashboard.Cursor = 0

	model, _ := app.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	result := model.(App)
	if result.views.Dashboard.Cursor != 0 {
		t.Errorf("expected cursor stays at 0, got %d", result.views.Dashboard.Cursor)
	}
}

// TestApp_FieldCountBounded ratchets the top-level App struct field budget so it
// can only shrink, never silently grow. App is now an aggregate root with three
// cohesive sub-models (env: EnvModel, theme: ThemeModel, views: ViewsModel) that
// group related state, reducing the original 47 fields to an aggregate root with
// 13 top-level fields.
//
// The views aggregate (ViewsModel) groups all per-view / panel / overlay state:
//   DashboardModel, PipelineModel, TerminalModel, BrowserModel, CommsModel,
//   MissionModel, SearchModel, SnippetPickerModel, ContextFileModel, DocsState,
//   ProjectsState.
//
// To LOWER this bound: extract another cohesive sub-model and decrement maxFields.
// Never RAISE it to accommodate a new loose field — extract a sub-model instead.
func TestApp_FieldCountBounded(t *testing.T) {
	count := reflect.TypeOf(App{}).NumField()
	// Ratchet: lower this only when an extraction reduces the count. Never raise it
	// to accommodate new loose fields — extract a sub-model instead.
	const maxFields = 13
	if count > maxFields {
		t.Errorf("App has %d fields, expected ≤ %d — did you add fields without extracting? See comment above.", count, maxFields)
	}
}
