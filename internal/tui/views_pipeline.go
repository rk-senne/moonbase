package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/pipeline"
)

func (a App) renderPipeline() string {
	header := a.renderHeader("Pipeline")
	sidebarWidth := 24
	mainWidth := a.width - sidebarWidth - 1 // 1 space separator

	var phases strings.Builder
	phases.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Brand).Bold(true).Render("◆ PIPELINE") + "\n")
	phases.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Dim).Render("──────────────") + "\n\n")

	if a.pipeline.State == nil {
		phases.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Dim).Render(" No active mission") + "\n")
	} else {
		// Show TraceID subtly at the top
		if a.pipeline.State.TraceID != "" {
			traceStyle := lipgloss.NewStyle().Foreground(a.themeData.Dim)
			phases.WriteString(traceStyle.Render(fmt.Sprintf(" trace: %s", a.pipeline.State.TraceID)) + "\n\n")
		}

		// Show phase progress indicator (X/Y for mandatory phases)
		mandatoryCount := 0
		completedMandatory := 0
		for _, ph := range a.pipeline.State.Phases {
			if !ph.Conditional {
				mandatoryCount++
				if ph.Status == pipeline.StatusComplete || ph.Status == pipeline.StatusSkipped {
					completedMandatory++
				}
			}
		}
		progressStyle := lipgloss.NewStyle().Foreground(a.themeData.Info)
		phases.WriteString(progressStyle.Render(fmt.Sprintf(" Phase %d/%d", completedMandatory, mandatoryCount)) + "\n\n")

		for _, phase := range a.pipeline.State.Phases {
			badge := BadgeWaiting
			style := a.styles.Inactive
			timing := ""
			switch phase.Status {
			case pipeline.StatusComplete:
				badge = BadgePass
				style = lipgloss.NewStyle().Foreground(a.themeData.Active)
				elapsed := phase.ElapsedTime()
				if elapsed > 0 {
					timing = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
				}
			case pipeline.StatusRunning:
				badge = a.spinner.View()
				style = a.styles.Active
				elapsed := phase.ElapsedTime()
				if elapsed > 0 {
					timing = fmt.Sprintf(" [%ds]", int(elapsed.Seconds()))
				}
			case pipeline.StatusFailed:
				badge = BadgeFail
				style = lipgloss.NewStyle().Foreground(a.themeData.Error)
			case pipeline.StatusSkipped:
				badge = "⊘"
			case pipeline.StatusRework:
				badge = "🔁"
				style = lipgloss.NewStyle().Foreground(a.themeData.Warning)
			}
			cond := ""
			if phase.Conditional {
				cond = " ⚡"
			}
			line := fmt.Sprintf(" %s %d. %s%s%s", badge, phase.Number, phase.Name, cond, timing)
			phases.WriteString(style.Render(line) + "\n")
		}

		// Risk gate status
		if a.pipeline.State.Context != nil && a.pipeline.State.Context.RiskLevel != "" {
			phases.WriteString("\n")
			phases.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Dim).Render("──────────────") + "\n")
			riskStyle := lipgloss.NewStyle().Bold(true)
			switch a.pipeline.State.Context.RiskLevel {
			case "LOW":
				riskStyle = riskStyle.Foreground(a.themeData.Active)
			case "MEDIUM":
				riskStyle = riskStyle.Foreground(a.themeData.Warning)
			case "HIGH", "CRITICAL":
				riskStyle = riskStyle.Foreground(a.themeData.Error)
			}
			phases.WriteString(riskStyle.Render(fmt.Sprintf(" Risk: %s", a.pipeline.State.Context.RiskLevel)) + "\n")
			if a.pipeline.State.Context.ReworkCount > 0 {
				phases.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Warning).Render(
					fmt.Sprintf(" Rework: %d/%d", a.pipeline.State.Context.ReworkCount, a.pipeline.State.MaxRework)) + "\n")
			}
		}
	}

	phaseSidebar := a.styles.Sidebar.Width(sidebarWidth).Render(phases.String())

	var main strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(a.themeData.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.themeData.Dim)
	pipeStyle := lipgloss.NewStyle().Foreground(a.themeData.Muted)

	if a.pipeline.State != nil && a.pipeline.State.Task != "" {
		main.WriteString(titleStyle.Render(fmt.Sprintf("━━━ MISSION: %s ━━━", a.pipeline.State.Task)) + "\n\n")
	} else {
		main.WriteString(dimStyle.Render("━━━ AWAITING MISSION ━━━") + "\n\n")
	}

	// Render pipeline chat with phase sections
	maxLines := a.height - 12
	if maxLines < 5 {
		maxLines = 5
	}
	start := 0
	if len(a.pipeline.Chat) > maxLines {
		start = len(a.pipeline.Chat) - maxLines
	}
	for i := start; i < len(a.pipeline.Chat); i++ {
		msg := a.pipeline.Chat[i]
		if msg.Agent == "" {
			// System message — phase headers, risk gates, dividers
			content := msg.Content
			if strings.HasPrefix(content, "────") || strings.HasPrefix(content, "━━━") {
				// Phase header or divider — use brand colour
				main.WriteString(titleStyle.Render(content) + "\n")
			} else if strings.Contains(content, "Risk Gate") || strings.Contains(content, "🎯") {
				// Risk gate — use appropriate colour
				if strings.Contains(content, "LOW") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Active).Bold(true).Render("  "+content) + "\n")
				} else if strings.Contains(content, "MEDIUM") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Warning).Bold(true).Render("  "+content) + "\n")
				} else if strings.Contains(content, "HIGH") || strings.Contains(content, "CRITICAL") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Error).Bold(true).Render("  "+content) + "\n")
				} else {
					main.WriteString(dimStyle.Render("  "+content) + "\n")
				}
			} else if strings.HasPrefix(content, "└──") {
				// Phase completion footer
				if strings.Contains(content, "✅") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Active).Render(content) + "\n")
				} else if strings.Contains(content, "❌") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.themeData.Error).Render(content) + "\n")
				} else {
					main.WriteString(dimStyle.Render(content) + "\n")
				}
			} else if strings.Contains(content, "⏭️") || strings.Contains(content, "⚡") {
				// Conditional trigger/skip
				main.WriteString(dimStyle.Render("  "+content) + "\n")
			} else {
				main.WriteString(dimStyle.Render(" "+content) + "\n")
			}
		} else {
			// Agent output — render through glamour
			rendered := renderMarkdown(msg.Content, mainWidth-4)
			lines := strings.Split(rendered, "\n")
			for _, line := range lines {
				main.WriteString(pipeStyle.Render("│ ") + line + "\n")
			}
		}
	}

	mainPanel := a.styles.Panel.Width(mainWidth).Render(main.String())
	body := lipgloss.JoinHorizontal(lipgloss.Top, phaseSidebar, " ", mainPanel)

	var statusBar string
	if a.pipeline.AbortPending && time.Since(a.pipeline.AbortAt) < 3*time.Second {
		statusBar = a.renderStatusBar("⚠️ Press [esc] again to abort mission. Any other key to cancel.")
	} else {
		statusBar = a.renderContextualStatusBar()
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}
