package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// handleProjectsKeys handles key messages when the view is ViewProjects.
func (a App) handleProjectsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.views.ProjectNav == nil {
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
	case key.Matches(msg, a.keys.Up):
		if a.views.ProjectNav.cursor > 0 {
			a.views.ProjectNav.cursor--
		}
	case key.Matches(msg, a.keys.Down):
		if a.views.ProjectNav.cursor < len(a.views.ProjectNav.list)-1 {
			a.views.ProjectNav.cursor++
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
func (a App) handleDocsKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if a.views.Docs == nil {
		return a, nil
	}
	switch {
	case key.Matches(msg, a.keys.Back):
		a.view = ViewDashboard
	case key.Matches(msg, a.keys.Up):
		if a.views.Docs.cursor > 0 {
			a.views.Docs.cursor--
		}
	case key.Matches(msg, a.keys.Down):
		if a.views.Docs.cursor < len(a.views.Docs.files)-1 {
			a.views.Docs.cursor++
		}
	case key.Matches(msg, a.keys.Enter):
		a.views.Docs.loadDoc(a.views.Docs.cursor, a.width-30)
	case key.Matches(msg, a.keys.DocsPageDown):
		a.views.Docs.viewport.HalfPageDown()
	case key.Matches(msg, a.keys.DocsPageUp):
		a.views.Docs.viewport.HalfPageUp()
	}
	return a, nil
}
