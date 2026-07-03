package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleProjectsKeys handles key messages when the view is ViewProjects.
func (a App) handleProjectsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.projectNav == nil {
		return a, nil
	}
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

// handleDocsKeys handles key messages when the view is ViewDocs.
func (a App) handleDocsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.docs == nil {
		return a, nil
	}
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
