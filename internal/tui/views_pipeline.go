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
	phases.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ PIPELINE") + "\n")
	phases.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────") + "\n\n")

	if a.pipelineState == nil {
		phases.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" No active mission") + "\n")
	} else {
		// Show TraceID subtly at the top
		if a.pipelineState.TraceID != "" {
			traceStyle := lipgloss.NewStyle().Foreground(ColorDim)
			phases.WriteString(traceStyle.Render(fmt.Sprintf(" trace: %s", a.pipelineState.TraceID)) + "\n\n")
		}

		// Show phase progress indicator (X/Y for mandatory phases)
		mandatoryCount := 0
		completedMandatory := 0
		for _, ph := range a.pipelineState.Phases {
			if !ph.Conditional {
				mandatoryCount++
				if ph.Status == pipeline.StatusComplete || ph.Status == pipeline.StatusSkipped {
					completedMandatory++
				}
			}
		}
		progressStyle := lipgloss.NewStyle().Foreground(ColorInfo)
		phases.WriteString(progressStyle.Render(fmt.Sprintf(" Phase %d/%d", completedMandatory, mandatoryCount)) + "\n\n")

		for _, phase := range a.pipelineState.Phases {
			badge := BadgeWaiting
			style := StyleInactive
			timing := ""
			switch phase.Status {
			case pipeline.StatusComplete:
				badge = BadgePass
				style = lipgloss.NewStyle().Foreground(ColorActive)
				elapsed := phase.ElapsedTime()
				if elapsed > 0 {
					timing = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
				}
			case pipeline.StatusRunning:
				badge = a.spinner.View()
				style = StyleActive
				elapsed := phase.ElapsedTime()
				if elapsed > 0 {
					timing = fmt.Sprintf(" [%ds]", int(elapsed.Seconds()))
				}
			case pipeline.StatusFailed:
				badge = BadgeFail
				style = lipgloss.NewStyle().Foreground(ColorError)
			case pipeline.StatusSkipped:
				badge = "⊘"
			case pipeline.StatusRework:
				badge = "🔁"
				style = lipgloss.NewStyle().Foreground(ColorWarning)
			}
			cond := ""
			if phase.Conditional {
				cond = " ⚡"
			}
			line := fmt.Sprintf(" %s %d. %s%s%s", badge, phase.Number, phase.Name, cond, timing)
			phases.WriteString(style.Render(line) + "\n")
		}

		// Risk gate status
		if a.pipelineState.Context != nil && a.pipelineState.Context.RiskLevel != "" {
			phases.WriteString("\n")
			phases.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────") + "\n")
			riskStyle := lipgloss.NewStyle().Bold(true)
			switch a.pipelineState.Context.RiskLevel {
			case "LOW":
				riskStyle = riskStyle.Foreground(ColorActive)
			case "MEDIUM":
				riskStyle = riskStyle.Foreground(ColorWarning)
			case "HIGH", "CRITICAL":
				riskStyle = riskStyle.Foreground(ColorError)
			}
			phases.WriteString(riskStyle.Render(fmt.Sprintf(" Risk: %s", a.pipelineState.Context.RiskLevel)) + "\n")
			if a.pipelineState.Context.ReworkCount > 0 {
				phases.WriteString(lipgloss.NewStyle().Foreground(ColorWarning).Render(
					fmt.Sprintf(" Rework: %d/%d", a.pipelineState.Context.ReworkCount, a.pipelineState.MaxRework)) + "\n")
			}
		}
	}

	phaseSidebar := StyleSidebar.Width(sidebarWidth).Render(phases.String())

	var main strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)
	pipeStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	if a.pipelineState != nil && a.pipelineState.Task != "" {
		main.WriteString(titleStyle.Render(fmt.Sprintf("━━━ MISSION: %s ━━━", a.pipelineState.Task)) + "\n\n")
	} else {
		main.WriteString(dimStyle.Render("━━━ AWAITING MISSION ━━━") + "\n\n")
	}

	// Render pipeline chat with phase sections
	maxLines := a.height - 12
	if maxLines < 5 {
		maxLines = 5
	}
	start := 0
	if len(a.pipelineChat) > maxLines {
		start = len(a.pipelineChat) - maxLines
	}
	for i := start; i < len(a.pipelineChat); i++ {
		msg := a.pipelineChat[i]
		if msg.Agent == "" {
			// System message — phase headers, risk gates, dividers
			content := msg.Content
			if strings.HasPrefix(content, "────") || strings.HasPrefix(content, "━━━") {
				// Phase header or divider — use brand colour
				main.WriteString(titleStyle.Render(content) + "\n")
			} else if strings.Contains(content, "Risk Gate") || strings.Contains(content, "🎯") {
				// Risk gate — use appropriate colour
				if strings.Contains(content, "LOW") {
					main.WriteString(lipgloss.NewStyle().Foreground(ColorActive).Bold(true).Render("  "+content) + "\n")
				} else if strings.Contains(content, "MEDIUM") {
					main.WriteString(lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("  "+content) + "\n")
				} else if strings.Contains(content, "HIGH") || strings.Contains(content, "CRITICAL") {
					main.WriteString(lipgloss.NewStyle().Foreground(ColorError).Bold(true).Render("  "+content) + "\n")
				} else {
					main.WriteString(dimStyle.Render("  "+content) + "\n")
				}
			} else if strings.HasPrefix(content, "└──") {
				// Phase completion footer
				if strings.Contains(content, "✅") {
					main.WriteString(lipgloss.NewStyle().Foreground(ColorActive).Render(content) + "\n")
				} else if strings.Contains(content, "❌") {
					main.WriteString(lipgloss.NewStyle().Foreground(ColorError).Render(content) + "\n")
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

	mainPanel := StylePanel.Width(mainWidth).Render(main.String())
	body := lipgloss.JoinHorizontal(lipgloss.Top, phaseSidebar, " ", mainPanel)

	var statusHint string
	if a.abortPending && time.Since(a.abortPendingAt) < 3*time.Second {
		statusHint = "⚠️ Press [esc] again to abort mission. Any other key to cancel."
	} else {
		statusHint = "[n] NEXT  [r] RETRY  [s] SKIP  [m] NEW MISSION  [esc] BACK"
	}
	statusBar := a.renderStatusBar(statusHint)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}
