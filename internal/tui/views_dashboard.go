package tui

import (
	"charm.land/lipgloss/v2"
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
	if a.views.Search.Active {
		bindings := a.keys.keysFor(a.view, a.views.Search.Active, a.views.Terminal.Active, a.views.Browser.Active)
		h := newHelpModel(a.width-4, a.theme.Data)
		keyHints := h.ShortHelpView(bindings)
		statusBar = a.renderStatusBar("/ " + a.views.Search.Input.View() + "  " + keyHints)
	} else {
		statusBar = a.renderContextualStatusBar()
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
