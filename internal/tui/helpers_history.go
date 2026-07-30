package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rk-senne/moonbase/internal/history"
)

// --- Mission History View ---

func (a App) renderHistory() string {
	header := a.renderHeader("Mission History")

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	labelStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Info)

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ◆ MISSION LOG") + "\n")
	b.WriteString(labelStyle.Render("  ─────────────────────────────────────────────") + "\n\n")

	missions := history.Load()
	if len(missions) == 0 {
		b.WriteString(dimStyle.Render("  No missions logged yet.") + "\n")
		b.WriteString(dimStyle.Render("  Run a pipeline to start recording history.") + "\n")
	} else {
		b.WriteString(fmt.Sprintf("  %-4s %-30s %-10s %s\n", "ID", "TASK", "OUTCOME", "DURATION"))
		b.WriteString(labelStyle.Render("  ─────────────────────────────────────────────") + "\n")
		for i := len(missions) - 1; i >= 0 && i >= len(missions)-20; i-- {
			m := missions[i]
			status := "✅"
			if m.Outcome == "aborted" {
				status = "❌"
			}
			task := m.Task
			if len(task) > 28 {
				task = task[:28] + ".."
			}
			b.WriteString(fmt.Sprintf("  %-4d %-30s %s %-10s %s\n",
				m.ID, task, status, m.Outcome, m.Duration))
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  [esc] Back") + "\n")

	body := a.theme.Styles.Panel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderContextualStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
}
