package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (a App) renderDashboard() string {
	headerH := 1
	statusH := 1
	bodyH := a.height - headerH - statusH - 1

	header := a.renderHeader("Dashboard")

	var body string
	if a.width >= 140 {
		body = a.render3Col(bodyH)
	} else if a.width >= 80 {
		body = a.render2Col(bodyH)
	} else {
		body = a.render1Col(bodyH)
	}

	var statusBar string
	if a.searching {
		statusBar = a.renderStatusBar("/ " + a.searchInput.View() + "  [enter] SELECT  [esc] CANCEL")
	} else if a.browsing {
		statusBar = a.renderStatusBar("[↑↓] NAV  [enter] OPEN  [backspace] BACK  [e] EDIT  [`] TERMINAL  [esc] EXIT")
	} else if a.termActive {
		statusBar = a.renderStatusBar("[enter] RUN  [`] FILE BROWSER  [esc] EXIT")
	} else {
		statusBar = a.renderStatusBar("[?] HELP  [↑↓] NAV  [enter] DOSSIER  [m] MISSION  [p] PROJECTS  [W] DOCS  [`] KND  [q] QUIT")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

func (a App) render3Col(h int) string {
	sideW := 24
	rightW := 30
	mainW := a.width - sideW - rightW - 6
	if mainW < 20 {
		// Fall back to 2col if not enough space
		return a.render2Col(h)
	}

	sidebar := a.renderSidebar(sideW, h)
	main := a.renderMainPanel(mainW, h)
	right := a.renderRightPanel(rightW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", main, " ", right)
}

func (a App) render2Col(h int) string {
	sideW := 24
	mainW := a.width - sideW - 3
	if mainW < 20 {
		mainW = 20
	}

	sidebar := a.renderSidebar(sideW, h)
	main := a.renderMainPanel(mainW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", main)
}

func (a App) render1Col(h int) string {
	return a.renderMainPanel(a.width-2, h)
}
