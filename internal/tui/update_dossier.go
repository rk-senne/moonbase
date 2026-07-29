package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleProjectsKeys handles key messages when the view is ViewProjects.
func (a App) handleProjectsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.projectNav == nil {
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
	case key.Matches(msg, a.keys.Up):
		if a.projectNav.cursor > 0 {
			a.projectNav.cursor--
		}
	case key.Matches(msg, a.keys.Down):
		if a.projectNav.cursor < len(a.projectNav.list)-1 {
			a.projectNav.cursor++
		}
	case key.Matches(msg, a.keys.Enter):
		a.selectProject()
	case key.Matches(msg, a.keys.LaunchCmux):
		return a, a.launchCmux()
	case key.Matches(msg, a.keys.LaunchFish):
		return a, a.launchTool("fish")
	}
	return a, nil
}

// handleDocsKeys handles key messages when the view is ViewDocs.
func (a App) handleDocsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.docs == nil {
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
	case key.Matches(msg, a.keys.Up):
		if a.docs.cursor > 0 {
			a.docs.cursor--
		}
	case key.Matches(msg, a.keys.Down):
		if a.docs.cursor < len(a.docs.files)-1 {
			a.docs.cursor++
		}
	case key.Matches(msg, a.keys.Enter):
		a.docs.loadDoc(a.docs.cursor, a.width-30)
	case key.Matches(msg, a.keys.DocsPageDown):
		a.docs.viewport.HalfViewDown()
	case key.Matches(msg, a.keys.DocsPageUp):
		a.docs.viewport.HalfViewUp()
	}
	return a, nil
}
