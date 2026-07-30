package tui

import (
	"sort"
	"strings"
	"testing"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/docs"
	"github.com/rk-senne/moonbase/internal/projects"
)

// wheelDown builds a mouse-wheel-down event.
func wheelDown() tea.MouseWheelMsg {
	return tea.MouseWheelMsg{Button: tea.MouseWheelDown}
}

// wheelUp builds a mouse-wheel-up event.
func wheelUp() tea.MouseWheelMsg {
	return tea.MouseWheelMsg{Button: tea.MouseWheelUp}
}

// firstRunes returns the first n runes of s (or all of s if shorter).
func firstRunes(s string, n int) string {
	r := []rune(s)
	if len(r) < n {
		return s
	}
	return string(r[:n])
}

func TestMouse_BootSkip(t *testing.T) {
	app := NewApp()
	app.view = ViewBoot
	m, _ := app.Update(wheelDown())
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

	m, _ := app.Update(wheelDown())
	down := m.(App)
	if down.views.Dashboard.Cursor != 1 {
		t.Errorf("wheel down: expected cursor 1, got %d", down.views.Dashboard.Cursor)
	}
	if down.views.Dashboard.Selected != 1 {
		t.Errorf("wheel down: expected Selected to track cursor (1), got %d", down.views.Dashboard.Selected)
	}

	m2, _ := down.Update(wheelUp())
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

	m, _ := app.Update(wheelDown())
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

	m, _ := app.Update(wheelDown())
	if got := m.(App); got.views.ProjectNav.cursor != 1 {
		t.Errorf("wheel down: expected projects cursor 1, got %d", got.views.ProjectNav.cursor)
	}
}

func TestMouse_CommsWheelScrollsViewport(t *testing.T) {
	vp := viewport.New(viewport.WithWidth(40), viewport.WithHeight(5))
	vp.SetContent(strings.Repeat("transmission line\n", 20))

	app := NewApp()
	app.boot.Ready = true
	app.view = ViewComms
	app.views.Comms.State = &CommsState{viewport: vp}

	m, _ := app.Update(wheelDown())
	down := m.(App)
	if down.views.Comms.State.viewport.YOffset() != mouseWheelLines {
		t.Errorf("wheel down: expected YOffset %d, got %d", mouseWheelLines, down.views.Comms.State.viewport.YOffset())
	}

	m2, _ := down.Update(wheelUp())
	if up := m2.(App); up.views.Comms.State.viewport.YOffset() != 0 {
		t.Errorf("wheel up: expected YOffset 0, got %d", up.views.Comms.State.viewport.YOffset())
	}
}

func TestMouse_DocsWheelScrollsViewport(t *testing.T) {
	vp := viewport.New(viewport.WithWidth(50), viewport.WithHeight(5))
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

	m, _ := app.Update(wheelDown())
	if got := m.(App); got.views.Docs.viewport.YOffset() != mouseWheelLines {
		t.Errorf("wheel down: expected docs YOffset %d, got %d", mouseWheelLines, got.views.Docs.viewport.YOffset())
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
		_, _ = app.Update(wheelDown())
		_, _ = app.Update(wheelUp())
	}
}

// TestSidebarRowToIndex_MatchesRender verifies the hit-map is in lock-step with the
// rendered sidebar: for every mapped row, the corresponding rendered line contains
// that operative's designation. This catches any drift between the hit-map's
// group-walk and renderSidebar.
func TestSidebarRowToIndex_MatchesRender(t *testing.T) {
	reg := newTestRegistry()
	app := NewApp()
	app.boot.Ready = true
	app.registry = reg

	m := sidebarRowToIndex(reg)
	if len(m) == 0 {
		t.Fatal("expected non-empty sidebar row map")
	}

	lines := strings.Split(app.renderSidebar(sidebarWidth, 60), "\n")
	all := reg.All()
	// renderSidebar's Sidebar style has PaddingTop(1), so its output prepends one
	// blank line: content row r appears at output line r+sidebarRenderPadTop. (The
	// full-screen offset also adds the 1-line header — see sidebarContentTopY.)
	const sidebarRenderPadTop = 1
	for row, idx := range m {
		line := row + sidebarRenderPadTop
		if line >= len(lines) {
			t.Errorf("row %d (line %d) beyond rendered sidebar (%d lines)", row, line, len(lines))
			continue
		}
		if idx < 0 || idx >= len(all) {
			t.Errorf("row %d -> out-of-range index %d", row, idx)
			continue
		}
		want := all[idx].Designation
		if want == "" {
			want = all[idx].Name
		}
		prefix := firstRunes(want, 4)
		if !strings.Contains(lines[line], prefix) {
			t.Errorf("row %d: rendered line %q missing operative %d prefix %q", row, lines[line], idx, prefix)
		}
	}
}

// TestMouse_ClickSelectsOperative is the end-to-end proof: it computes the screen Y
// of a roster row, validates that offset against the REAL full-screen render (the
// line at that Y must contain the operative's designation), then clicks there and
// asserts the operative is selected and its dossier opened.
func TestMouse_ClickSelectsOperative(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 160 // 3-col layout, sidebar visible
	app.height = 50
	app.view = ViewDashboard
	app.registry = newTestRegistry()

	m := sidebarRowToIndex(app.registry)
	if len(m) < 2 {
		t.Skip("need >=2 mapped operative rows")
	}
	rows := make([]int, 0, len(m))
	for r := range m {
		rows = append(rows, r)
	}
	sort.Ints(rows)
	row := rows[1] // a stable, non-edge entry
	idx := m[row]
	clickY := sidebarContentTopY + row

	all := app.registry.All()
	want := all[idx].Designation
	if want == "" {
		want = all[idx].Name
	}
	prefix := firstRunes(want, 4)

	// Offset validation against the actual render.
	vlines := strings.Split(app.renderFrame(), "\n")
	if clickY >= len(vlines) {
		t.Fatalf("clickY %d beyond rendered view (%d lines)", clickY, len(vlines))
	}
	if !strings.Contains(vlines[clickY], prefix) {
		t.Fatalf("offset check failed: full-view line %d = %q, expected to contain %q (operative %d)", clickY, vlines[clickY], prefix, idx)
	}

	// Click that roster row.
	res, _ := app.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 3, Y: clickY})
	got := res.(App)
	if got.views.Dashboard.Selected != idx {
		t.Errorf("expected Selected=%d after click, got %d", idx, got.views.Dashboard.Selected)
	}
	if got.view != ViewDossier {
		t.Errorf("expected view=Dossier after click, got %v", got.view)
	}
}

// TestMouse_ClickOutsideSidebarIgnored ensures clicks outside the sidebar column or
// on non-operative rows don't select anything or change the view.
func TestMouse_ClickOutsideSidebarIgnored(t *testing.T) {
	app := NewApp()
	app.boot.Ready = true
	app.width = 160
	app.height = 50
	app.view = ViewDashboard
	app.registry = newTestRegistry()

	// Click far to the right, in the main panel.
	res, _ := app.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 100, Y: 5})
	if got := res.(App); got.view != ViewDashboard {
		t.Errorf("click in main panel should not open dossier, got view %v", got.view)
	}

	// Click on the very top row (a group header, not an operative).
	res2, _ := app.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 3, Y: sidebarContentTopY})
	if got := res2.(App); got.view != ViewDashboard {
		t.Errorf("click on group header should not open dossier, got view %v", got.view)
	}
}
