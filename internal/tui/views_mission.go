package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a App) renderMission() string {
	header := a.renderHeader("New Mission")

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(a.themeData.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.themeData.Dim)

	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("  ◆ NEW MISSION BRIEFING") + "\n\n")
	b.WriteString(dimStyle.Render("  Describe the objective. The KND Council will execute the") + "\n")
	b.WriteString(dimStyle.Render("  full pipeline: Analyst → Architect → Implement → QA → Review") + "\n\n")
	b.WriteString("  Mission: " + a.mission.Input.View() + "\n\n")
	b.WriteString(dimStyle.Render("  [enter] Deploy council    [esc] Cancel") + "\n")

	body := a.styles.Panel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderContextualStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body+"\n", statusBar)
}
