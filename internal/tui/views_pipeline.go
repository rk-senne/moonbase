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
	sidebarWidth := pipelineSidebarWidth
	mainWidth := a.pipelineMainWidth()

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

	// Build the whole conversation as styled lines, then show a scrollable
	// window into it (ChatScroll = lines up from the bottom; 0 = follow latest).
	lines := a.pipelineChatLines(mainWidth)
	chatH := a.pipelineChatHeight()
	total := len(lines)
	scroll := a.views.Pipeline.ChatScroll
	if maxScroll := total - chatH; scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	end := total - scroll
	begin := end - chatH
	if begin < 0 {
		begin = 0
	}
	if begin > 0 {
		main.WriteString(dimStyle.Render(fmt.Sprintf("  ▲ %d more above (↑/pgup to scroll)", begin)) + "\n")
	}
	for _, ln := range lines[begin:end] {
		main.WriteString(ln + "\n")
	}
	if end < total {
		main.WriteString(dimStyle.Render(fmt.Sprintf("  ▼ %d below · press End to follow", total-end)) + "\n")
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

// pipelineSidebarWidth is the fixed width of the phase sidebar column.
const pipelineSidebarWidth = 24

// pipelineMainWidth is the width of the conversation panel.
func (a App) pipelineMainWidth() int {
	w := a.width - pipelineSidebarWidth - 1
	if w < 20 {
		w = 20
	}
	return w
}

// pipelineChatHeight is the number of conversation lines visible at once. Used
// by both the renderer and the scroll handler so they agree on the window.
func (a App) pipelineChatHeight() int {
	h := a.height - 14 // header + mission title + indicators + status bar
	if h < 4 {
		h = 4
	}
	return h
}

// pipelineChatLines builds the entire conversation as styled display lines:
// completed messages (system lines + per-agent persona blocks) followed by the
// live streaming block. The live block is prefixed with a persona header so the
// operator can always see WHICH operative is currently speaking — making agent
// handoffs visible even mid-stream. Both the renderer and the scroll handler
// call this so the scrollable window is computed from the same content.
func (a App) pipelineChatLines(mainWidth int) []string {
	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)

	var lines []string
	prevAgent := ""
	for _, msg := range a.views.Pipeline.Chat {
		if msg.Agent == "" {
			prevAgent = ""
			content := msg.Content
			switch {
			case strings.HasPrefix(content, "────") || strings.HasPrefix(content, "━━━"):
				lines = append(lines, titleStyle.Render(content))
			case strings.Contains(content, "Risk Gate") || strings.Contains(content, "🎯"):
				st := dimStyle
				if strings.Contains(content, "LOW") {
					st = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)
				} else if strings.Contains(content, "MEDIUM") {
					st = lipgloss.NewStyle().Foreground(a.theme.Data.Warning).Bold(true)
				} else if strings.Contains(content, "HIGH") || strings.Contains(content, "CRITICAL") {
					st = lipgloss.NewStyle().Foreground(a.theme.Data.Error).Bold(true)
				}
				lines = append(lines, st.Render("  "+content))
			case strings.Contains(content, "⏭️") || strings.Contains(content, "⚡"):
				lines = append(lines, dimStyle.Render("  "+content))
			case strings.HasPrefix(content, "└──"):
				st := dimStyle
				if strings.Contains(content, "✅") {
					st = lipgloss.NewStyle().Foreground(a.theme.Data.Active)
				} else if strings.Contains(content, "❌") {
					st = lipgloss.NewStyle().Foreground(a.theme.Data.Error)
				}
				lines = append(lines, st.Render(content))
			default:
				lines = append(lines, dimStyle.Render(" "+content))
			}
			continue
		}

		// Agent block: persona header once per contiguous speaker, then content.
		persona := personaFor(msg.Agent)
		if msg.Agent != prevAgent {
			lines = append(lines, personaHeaderLine(persona, mainWidth, ""))
			if persona.Personality != "" {
				lines = append(lines, lipgloss.NewStyle().Foreground(persona.Color).Italic(true).Render("  "+persona.Personality))
			}
		}
		prevAgent = msg.Agent
		gutter := lipgloss.NewStyle().Foreground(persona.Color)
		for _, line := range strings.Split(renderMarkdown(msg.Content, mainWidth-4), "\n") {
			lines = append(lines, gutter.Render("│ ")+line)
		}
	}

	// Live streaming block — always labelled with the current operative so the
	// handoff is visible while output is still arriving.
	if a.views.Pipeline.Running && a.views.Pipeline.LiveBuf != "" {
		persona := personaFor(a.views.Pipeline.LiveAgent)
		lines = append(lines, personaHeaderLine(persona, mainWidth, a.spinner.View()+" streaming…"))
		// Bound the work: wrap only the tail of the buffer, show the last lines.
		raw := a.views.Pipeline.LiveBuf
		if len(raw) > 4000 {
			raw = raw[len(raw)-4000:]
		}
		wrapped := strings.Split(wordWrap(raw, mainWidth-4), "\n")
		const liveTail = 40
		if len(wrapped) > liveTail {
			wrapped = wrapped[len(wrapped)-liveTail:]
		}
		gutter := lipgloss.NewStyle().Foreground(persona.Color)
		muted := lipgloss.NewStyle().Foreground(a.theme.Data.Muted)
		for _, line := range wrapped {
			lines = append(lines, gutter.Render("│ ")+muted.Render(line))
		}
	}

	return lines
}

// personaHeaderLine renders an operative's header ("▸ Numbuh 2 · Hoagie Gilligan"),
// optionally with a trailing status (e.g. a spinner + "streaming…").
func personaHeaderLine(persona AgentPersona, mainWidth int, status string) string {
	header := "▸ " + persona.Operative
	if persona.Designation != "" && persona.Designation != persona.Operative {
		header += " · " + persona.Designation
	}
	if status != "" {
		header += "  " + status
	}
	return lipgloss.NewStyle().Foreground(persona.Color).Bold(true).Render(truncateToWidth(header, mainWidth-2))
}
