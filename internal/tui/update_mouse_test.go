package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rk-senne/moonbase/internal/docs"
	"github.com/rk-senne/moonbase/internal/projects"
)

// wheel builds a mouse-wheel event for the given button.
func wheel(btn tea.MouseButton) tea.MouseMsg {
	return tea.MouseMsg{Action: tea.MouseActionPress, Button: btn}
}

func TestMouse_BootSkip(t *testing.T) {
	app := NewApp()
	app.view = ViewBoot
	m, _ := app.Update(wheel(tea.MouseButtonWheelDown))
	if got := m.(App); got.view != ViewDashboard {
		t.Errorf("expected mouse on boot to skip to Dashboard, got view %v", got.view)
	}
}

func TestMouse_DashboardWheelMovesCursor(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.registry = newTestRegistry()
	app.views.Browser.Active = false // not in file-browser sub-mode
	app.views.Terminal.Active = false
	app.views.Dashboard.Cursor = 0

	if app.registry.Count() < 2 {
		t.Skip("need >=2 agents for cursor movement")
	}

	m, _ := app.Update(wheel(tea.MouseButtonWheelDown))
	down := m.(App)
	if down.views.Dashboard.Cursor != 1 {
		t.Errorf("wheel down: expected cursor 1, got %d", down.views.Dashboard.Cursor)
	}
	if down.views.Dashboard.Selected != 1 {
		t.Errorf("wheel down: expected Selected to track cursor (1), got %d", down.views.Dashboard.Selected)
	}

	m2, _ := down.Update(wheel(tea.MouseButtonWheelUp))
	up := m2.(App)
	if up.views.Dashboard.Cursor != 0 {
		t.Errorf("wheel up: expected cursor 0, got %d", up.views.Dashboard.Cursor)
	}
}

func TestMouse_DashboardWheelIgnoredInBrowserMode(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDashboard
	app.registry = newTestRegistry()
	app.views.Browser.Active = true // file-browser owns the panel
	app.views.Dashboard.Cursor = 0

	m, _ := app.Update(wheel(tea.MouseButtonWheelDown))
	if got := m.(App); got.views.Dashboard.Cursor != 0 {
		t.Errorf("expected roster cursor unchanged in browser mode, got %d", got.views.Dashboard.Cursor)
	}
}

func TestMouse_ProjectsWheelMovesCursor(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.view = ViewProjects
	app.views.ProjectNav = &ProjectsState{
		list:   []projects.Project{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		cursor: 0,
	}

	m, _ := app.Update(wheel(tea.MouseButtonWheelDown))
	if got := m.(App); got.views.ProjectNav.cursor != 1 {
		t.Errorf("wheel down: expected projects cursor 1, got %d", got.views.ProjectNav.cursor)
	}
}

func TestMouse_CommsWheelScrollsViewport(t *testing.T) {
	vp := viewport.New(40, 5)
	vp.SetContent(strings.Repeat("transmission line\n", 20))

	app := NewApp()
	app.boot.Ready = true
	app.view = ViewComms
	app.views.Comms.State = &CommsState{viewport: vp}

	m, _ := app.Update(wheel(tea.MouseButtonWheelDown))
	down := m.(App)
	if down.views.Comms.State.viewport.YOffset != mouseWheelLines {
		t.Errorf("wheel down: expected YOffset %d, got %d", mouseWheelLines, down.views.Comms.State.viewport.YOffset)
	}

	m2, _ := down.Update(wheel(tea.MouseButtonWheelUp))
	if up := m2.(App); up.views.Comms.State.viewport.YOffset != 0 {
		t.Errorf("wheel up: expected YOffset 0, got %d", up.views.Comms.State.viewport.YOffset)
	}
}

func TestMouse_DocsWheelScrollsViewport(t *testing.T) {
	vp := viewport.New(50, 5)
	vp.SetContent(strings.Repeat("doc line\n", 20))

	app := NewApp()
	app.boot.Ready = true
	app.view = ViewDocs
	app.views.Docs = &DocsState{
		files:    []docs.Doc{{Name: "readme.md", Path: "/tmp/x.md"}},
		viewport: vp,
		loaded:   true,
		content:  strings.Repeat("doc line\n", 20),
	}

	m, _ := app.Update(wheel(tea.MouseButtonWheelDown))
	if got := m.(App); got.views.Docs.viewport.YOffset != mouseWheelLines {
		t.Errorf("wheel down: expected docs YOffset %d, got %d", mouseWheelLines, got.views.Docs.viewport.YOffset)
	}
}

// TestMouse_NilStatesNoPanic ensures wheel events on views whose lazy state is
// not yet initialised are safe no-ops.
func TestMouse_NilStatesNoPanic(t *testing.T) {
	for _, v := range []View{ViewComms, ViewDocs, ViewProjects} {
		app := NewApp()
		app.boot.Ready = true
		app.view = v
		app.views.Comms.State = nil
		app.views.Docs = nil
		app.views.ProjectNav = nil
		// Should not panic.
		_, _ = app.Update(wheel(tea.MouseButtonWheelDown))
		_, _ = app.Update(wheel(tea.MouseButtonWheelUp))
	}
}
