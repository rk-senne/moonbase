package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (a App) renderDashboard() string {
	headerH := 1
	statusH := 1
	bodyH := a.height - headerH - statusH - 2 // extra line for spacing

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
	// The sidebar has a right border (1 char) + right padding (1 char) = renders wider
	// The main panel has rounded border (1 char each side) + padding (1 char each side) = +4
	// The right panel has rounded border (1 char each side) + padding (1 char each side) = +4
	// Two separating spaces between panels = 2
	// Total overhead: sidebar border/pad accounted in Width(), same for main/right
	// But lipgloss Width() constrains content, so actual render = Width param
	// Separators between panels: 2 spaces (1 + 1)
	mainW := a.width - sideW - rightW - 2
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
	// One space separator between sidebar and main
	mainW := a.width - sideW - 1
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
