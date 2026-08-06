package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/rk-senne/moonbase/internal/agents"
)

// mouseWheelLines is how many lines a single mouse-wheel notch scrolls a viewport.
const mouseWheelLines = 3

// Sidebar hit-testing geometry. The dashboard renders header (1 line) then the body;
// the sidebar panel has only a right border and PaddingTop(1), so its first content
// row sits at absolute Y=2. The sidebar column width matches render2Col/render3Col.
const (
	sidebarContentTopY = 2
	sidebarWidth       = 24
)

// handleMouseWheel routes mouse wheel events. Mouse support is strictly
// additive: every action here has a keyboard equivalent, so the TUI remains fully
// usable without a mouse.
func (a App) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	// Boot view: any interaction skips the boot animation, mirroring key handling.
	if a.view == ViewBoot {
		a.view = ViewDashboard
		a.addIntel("Boot skipped by operative.")
		return a, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		return a.mouseScroll(-1), nil
	case tea.MouseWheelDown:
		return a.mouseScroll(1), nil
	}
	return a, nil
}

// handleMouseClick routes mouse click events.
func (a App) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	// Boot view: any interaction skips the boot animation, mirroring key handling.
	if a.view == ViewBoot {
		a.view = ViewDashboard
		a.addIntel("Boot skipped by operative.")
		return a, nil
	}

	if msg.Button == tea.MouseLeft {
		return a.mouseClick(msg), nil
	}
	return a, nil
}

// mouseScroll routes a wheel notch to whatever is scrollable in the current view:
// dir < 0 scrolls up, dir > 0 scrolls down. Viewport-backed views (COMMS, Docs)
// scroll their content; list views (Dashboard roster, Projects) move the cursor,
// mirroring the j/k keys.
func (a App) mouseScroll(dir int) App {
	switch a.view {
	case ViewComms:
		if a.views.Comms.State != nil {
			if dir < 0 {
				a.views.Comms.State.viewport.ScrollUp(mouseWheelLines)
			} else {
				a.views.Comms.State.viewport.ScrollDown(mouseWheelLines)
			}
		}
	case ViewDocs:
		if a.views.Docs != nil {
			if dir < 0 {
				a.views.Docs.viewport.ScrollUp(mouseWheelLines)
			} else {
				a.views.Docs.viewport.ScrollDown(mouseWheelLines)
			}
		}
	case ViewProjects:
		if a.views.ProjectNav != nil {
			if dir < 0 && a.views.ProjectNav.cursor > 0 {
				a.views.ProjectNav.cursor--
			} else if dir > 0 && a.views.ProjectNav.cursor < len(a.views.ProjectNav.list)-1 {
				a.views.ProjectNav.cursor++
			}
		}
	case ViewDashboard:
		// Roster selection — only when not in a sub-mode that owns the panel.
		// Uses display-order navigation so the wheel never skips grouped agents.
		if !a.views.Terminal.Active && !a.views.Browser.Active && a.registry != nil {
			a.moveSidebarCursor(dir)
		}
	}
	return a
}

// mouseClick handles a left-click. On the dashboard it selects the operative whose
// roster row was clicked and opens its dossier — the mouse equivalent of arrowing to
// an operative and pressing enter. The sidebar is only present in the 2-col and
// 3-col layouts (width >= 80).
func (a App) mouseClick(msg tea.MouseClickMsg) App {
	if a.view != ViewDashboard || a.width < 80 || a.registry == nil {
		return a
	}
	// Must be within the sidebar column.
	if msg.X < 0 || msg.X > sidebarWidth+1 {
		return a
	}
	relRow := msg.Y - sidebarContentTopY
	if relRow < 0 {
		return a
	}
	idx, ok := sidebarRowToIndex(a.registry)[relRow]
	if !ok {
		return a // clicked a header, blank line, or a non-operative section
	}
	a.views.Dashboard.Cursor = idx
	a.views.Dashboard.Selected = idx
	a.view = ViewDossier
	return a
}

// sidebarRowToIndex maps a sidebar content row (0-based, relative to the sidebar's
// first rendered content line) to the registry index of the operative on that row.
// It walks the exact same group structure renderSidebar renders — each group emits a
// header line, one line per entry, then a trailing blank line — so the hit-map and
// the rendered sidebar cannot drift out of sync.
func sidebarRowToIndex(reg *agents.Registry) map[int]int {
	m := make(map[int]int)
	row := 0
	for _, group := range buildSidebarGroups(reg) {
		row++ // group header line ("◆ TITLE")
		for _, entry := range group.entries {
			if entry.index >= 0 { // skip fallback placeholder entries
				m[row] = entry.index
			}
			row++
		}
		row++ // trailing blank line after the group
	}
	return m
}
