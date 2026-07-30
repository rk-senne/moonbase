package tui

import tea "github.com/charmbracelet/bubbletea"

// mouseWheelLines is how many lines a single mouse-wheel notch scrolls a viewport.
const mouseWheelLines = 3

// handleMouse routes mouse events to the active view. Mouse support is strictly
// additive: every action here has a keyboard equivalent, so the TUI remains fully
// usable without a mouse. Enabled via tea.WithMouseCellMotion in the program.
func (a App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Boot view: any interaction skips the boot animation, mirroring key handling.
	if a.view == ViewBoot {
		a.view = ViewDashboard
		a.addIntel("Boot skipped by operative.")
		return a, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return a.mouseScroll(-1), nil
	case tea.MouseButtonWheelDown:
		return a.mouseScroll(1), nil
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
				a.views.Comms.State.viewport.LineUp(mouseWheelLines)
			} else {
				a.views.Comms.State.viewport.LineDown(mouseWheelLines)
			}
		}
	case ViewDocs:
		if a.views.Docs != nil {
			if dir < 0 {
				a.views.Docs.viewport.LineUp(mouseWheelLines)
			} else {
				a.views.Docs.viewport.LineDown(mouseWheelLines)
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
		if !a.views.Terminal.Active && !a.views.Browser.Active && a.registry != nil {
			if dir < 0 && a.views.Dashboard.Cursor > 0 {
				a.views.Dashboard.Cursor--
			} else if dir > 0 && a.views.Dashboard.Cursor < a.registry.Count()-1 {
				a.views.Dashboard.Cursor++
			}
			a.views.Dashboard.Selected = a.views.Dashboard.Cursor
		}
	}
	return a
}
