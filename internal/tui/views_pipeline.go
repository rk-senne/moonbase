package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

func (a App) renderPipeline() string {
	header := a.renderHeader("Pipeline")
	sidebarWidth := 24
	mainWidth := a.width - sidebarWidth - 1 // 1 space separator

	var phases strings.Builder
	phases.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true).Render("◆ PIPELINE") + "\n")
	phases.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("──────────────") + "\n\n")

	if a.views.Pipeline.State == nil {
		phases.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render(" No active mission") + "\n")
	} else {
		// Show TraceID subtly at the top
		if a.views.Pipeline.State.TraceID != "" {
			traceStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
			phases.WriteString(traceStyle.Render(fmt.Sprintf(" trace: %s", a.views.Pipeline.State.TraceID)) + "\n\n")
		}

		// Show phase progress indicator (X/Y for mandatory phases)
		mandatoryCount := 0
		completedMandatory := 0
		for _, ph := range a.views.Pipeline.State.Phases {
			if !ph.Conditional {
				mandatoryCount++
				if ph.Status == pipeline.StatusComplete || ph.Status == pipeline.StatusSkipped {
					completedMandatory++
				}
			}
		}
		progressStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Info)
		phases.WriteString(progressStyle.Render(fmt.Sprintf(" Phase %d/%d", completedMandatory, mandatoryCount)) + "\n\n")

		for i, phase := range a.views.Pipeline.State.Phases {
			// Insert fan-out group header before first conditional when parallel is enabled.
			if phase.Conditional && a.views.Pipeline.State.ParallelSpecialists {
				// Check if this is the first conditional in the list.
				isFirst := true
				for j := 0; j < i; j++ {
					if a.views.Pipeline.State.Phases[j].Conditional {
						isFirst = false
						break
					}
				}
				if isFirst {
					fanoutStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Info)
					phases.WriteString(fanoutStyle.Render(" ⚡ Fan-Out") + "\n")
				}
			}

			badge := BadgeWaiting
			style := a.theme.Styles.Inactive
			timing := ""
			switch phase.Status {
			case pipeline.StatusComplete:
				badge = BadgePass
				style = lipgloss.NewStyle().Foreground(a.theme.Data.Active)
				elapsed := phase.ElapsedTime()
				if elapsed > 0 {
					timing = fmt.Sprintf(" (%.1fs)", elapsed.Seconds())
				}
			case pipeline.StatusRunning:
				badge = a.spinner.View()
				style = a.theme.Styles.Active
				elapsed := phase.ElapsedTime()
				if elapsed > 0 {
					timing = fmt.Sprintf(" [%ds]", int(elapsed.Seconds()))
				}
			case pipeline.StatusFailed:
				badge = BadgeFail
				style = lipgloss.NewStyle().Foreground(a.theme.Data.Error)
			case pipeline.StatusSkipped:
				badge = "⊘"
			case pipeline.StatusRework:
				badge = "🔁"
				style = lipgloss.NewStyle().Foreground(a.theme.Data.Warning)
			}
			cond := ""
			if phase.Conditional {
				cond = " ⚡"
			}
			line := fmt.Sprintf(" %s %d. %s%s%s", badge, phase.Number, phase.Name, cond, timing)
			phases.WriteString(style.Render(line) + "\n")
		}

		// Risk gate status
		if a.views.Pipeline.State.Context != nil && a.views.Pipeline.State.Context.RiskLevel != "" {
			phases.WriteString("\n")
			phases.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("──────────────") + "\n")
			riskStyle := lipgloss.NewStyle().Bold(true)
			switch a.views.Pipeline.State.Context.RiskLevel {
			case "LOW":
				riskStyle = riskStyle.Foreground(a.theme.Data.Active)
			case "MEDIUM":
				riskStyle = riskStyle.Foreground(a.theme.Data.Warning)
			case "HIGH", "CRITICAL":
				riskStyle = riskStyle.Foreground(a.theme.Data.Error)
			}
			phases.WriteString(riskStyle.Render(fmt.Sprintf(" Risk: %s", a.views.Pipeline.State.Context.RiskLevel)) + "\n")
			if a.views.Pipeline.State.Context.ReworkCount > 0 {
				phases.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Warning).Render(
					fmt.Sprintf(" Rework: %d/%d", a.views.Pipeline.State.Context.ReworkCount, a.views.Pipeline.State.MaxRework)) + "\n")
			}
		}
	}

	phaseSidebar := a.theme.Styles.Sidebar.Width(sidebarWidth).Render(phases.String())

	var main strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)

	if a.views.Pipeline.State != nil && a.views.Pipeline.State.Task != "" {
		main.WriteString(titleStyle.Render(fmt.Sprintf("━━━ MISSION: %s ━━━", a.views.Pipeline.State.Task)) + "\n\n")
	} else {
		main.WriteString(dimStyle.Render("━━━ AWAITING MISSION ━━━") + "\n\n")
	}

	// Render pipeline chat with phase sections
	maxLines := a.height - 12
	if maxLines < 5 {
		maxLines = 5
	}

	// If a phase is actively streaming, render the live buffer as plain
	// wrapped text (no glamour) so high-frequency chunks don't trigger
	// expensive re-renders (AC-1.1/1.2). The live buffer is flushed into a
	// normal PipelineMsg on phase completion.
	streamTail := ""
	if a.views.Pipeline.Running && a.views.Pipeline.LiveBuf != "" {
		raw := a.views.Pipeline.LiveBuf
		wrapped := wordWrap(raw, mainWidth-4)
		lines := strings.Split(wrapped, "\n")
		// Show last 12 lines of streaming output for better visibility.
		tailCount := 12
		if len(lines) < tailCount {
			tailCount = len(lines)
		}
		tail := lines[len(lines)-tailCount:]
		streamTail = strings.Join(tail, "\n")
	}

	start := 0
	if len(a.views.Pipeline.Chat) > maxLines {
		start = len(a.views.Pipeline.Chat) - maxLines
	}
	prevAgent := ""
	for i := start; i < len(a.views.Pipeline.Chat); i++ {
		msg := a.views.Pipeline.Chat[i]
		if msg.Agent == "" {
			prevAgent = ""
			// System message — phase headers, risk gates, dividers
			content := msg.Content
			if strings.HasPrefix(content, "────") || strings.HasPrefix(content, "━━━") {
				// Phase header or divider — use brand colour
				main.WriteString(titleStyle.Render(content) + "\n")
			} else if strings.Contains(content, "Risk Gate") || strings.Contains(content, "🎯") {
				// Risk gate — use appropriate colour
				if strings.Contains(content, "LOW") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true).Render("  "+content) + "\n")
				} else if strings.Contains(content, "MEDIUM") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Warning).Bold(true).Render("  "+content) + "\n")
				} else if strings.Contains(content, "HIGH") || strings.Contains(content, "CRITICAL") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Error).Bold(true).Render("  "+content) + "\n")
				} else {
					main.WriteString(dimStyle.Render("  "+content) + "\n")
				}
			} else if strings.HasPrefix(content, "└──") {
				// Phase completion footer
				if strings.Contains(content, "✅") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Active).Render(content) + "\n")
				} else if strings.Contains(content, "❌") {
					main.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Error).Render(content) + "\n")
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
			// Agent output — show a persona header (who + personality) the first
			// time this operative speaks in a contiguous block, then the content
			// with a persona-coloured gutter so each voice is visually distinct.
			persona := personaFor(msg.Agent)
			if msg.Agent != prevAgent {
				headerStyle := lipgloss.NewStyle().Foreground(persona.Color).Bold(true)
				header := "▸ " + persona.Operative
				if persona.Designation != "" && persona.Designation != persona.Operative {
					header += " · " + persona.Designation
				}
				header = truncateToWidth(header, mainWidth-2)
				main.WriteString(headerStyle.Render(header) + "\n")
				if persona.Personality != "" {
					main.WriteString(lipgloss.NewStyle().Foreground(persona.Color).Italic(true).Render("  "+persona.Personality) + "\n")
				}
			}
			prevAgent = msg.Agent
			gutter := lipgloss.NewStyle().Foreground(persona.Color)
			rendered := renderMarkdown(msg.Content, mainWidth-4)
			lines := strings.Split(rendered, "\n")
			for _, line := range lines {
				main.WriteString(gutter.Render("│ ") + line + "\n")
			}
		}
	}

	// Render streaming tail if active (plain wrapped text, persona-coloured gutter)
	if streamTail != "" {
		persona := personaFor(a.views.Pipeline.LiveAgent)
		gutter := lipgloss.NewStyle().Foreground(persona.Color)
		streamLines := strings.Split(streamTail, "\n")
		for _, line := range streamLines {
			main.WriteString(gutter.Render("│ ") + lipgloss.NewStyle().Foreground(a.theme.Data.Muted).Render(line) + "\n")
		}
	}

	mainPanel := a.theme.Styles.Panel.Width(mainWidth).Render(main.String())

	body := lipgloss.JoinHorizontal(lipgloss.Top, phaseSidebar, " ", mainPanel)

	var statusBar string
	if a.views.Pipeline.AbortPending && time.Since(a.views.Pipeline.AbortAt) < 3*time.Second {
		statusBar = a.renderStatusBar("⚠️ Press [esc] again to abort mission. Any other key to cancel.")
	} else {
		statusBar = a.renderContextualStatusBar()
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}
