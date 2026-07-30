package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (a App) renderMission() string {
	header := a.renderHeader("New Mission")

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)

	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("  ◆ NEW MISSION BRIEFING") + "\n\n")
	b.WriteString(dimStyle.Render("  Describe the objective. The KND Council will execute the") + "\n")
	b.WriteString(dimStyle.Render("  full pipeline: Analyst → Architect → Implement → QA → Review") + "\n\n")
	b.WriteString("  Mission: " + a.views.Mission.Input.View() + "\n\n")
	b.WriteString(dimStyle.Render("  [enter] Deploy council    [esc] Cancel") + "\n")

	body := a.theme.Styles.Panel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderContextualStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body+"\n", statusBar)
}
