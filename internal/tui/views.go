package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/pipeline"
)

func (a App) renderBoot() string {
	var b strings.Builder

	// Data cascade effect for first 3 frames
	if a.bootStep < 3 {
		cascadeH := a.height / 3
		cascade := GenerateCascade(a.width, cascadeH, a.bootStep)
		cascadeStyled := lipgloss.NewStyle().Foreground(ColorActive).Render(cascade)
		b.WriteString(cascadeStyled)
		b.WriteString("\n")
	}

	// Center vertically (less padding during cascade)
	padding := (a.height - 20) / 2
	if a.bootStep < 3 {
		padding = 2
	}
	for i := 0; i < padding; i++ {
		b.WriteString("\n")
	}

	logo := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render(kndLogo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, logo))
	b.WriteString("\n")

	subtitle := lipgloss.NewStyle().Foreground(ColorInfo).Render(moonbaseLogo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, subtitle))
	b.WriteString("\n\n")

	for i := 0; i < a.bootStep && i < len(bootMessages); i++ {
		style := lipgloss.NewStyle().Foreground(ColorActive)
		if i == a.bootStep-1 {
			msg := bootMessages[i]
			// Typewriter on the final message
			if i == len(bootMessages)-1 {
				revealed := a.anim.TypewriterText(msg)
				line := fmt.Sprintf("  %s %s", a.spinner.View(), revealed)
				b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, style.Render(line)))
			} else {
				line := fmt.Sprintf("  %s %s", a.spinner.View(), msg)
				b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, style.Render(line)))
			}
		} else {
			line := fmt.Sprintf("  ✓ %s", bootMessages[i])
			b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, lipgloss.NewStyle().Foreground(ColorDim).Render(line)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

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

func (a App) renderDossier() string {
	sidebarWidth := 24
	bodyH := a.height - 3

	agent := a.registry.Get(a.selected)

	header := a.renderHeader("Dossier › " + agent.Designation)
	sidebar := a.renderSidebar(sidebarWidth, bodyH)

	// Right column: portrait + stats
	portraitW := 20
	mainWidth := a.width - sidebarWidth - portraitW - 5

	var d strings.Builder
	nameStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorInfo)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)

	d.WriteString(nameStyle.Render(fmt.Sprintf("  %s", strings.ToUpper(agent.Name))) + "\n")
	d.WriteString(dimStyle.Render(fmt.Sprintf("  %s", agent.Description)) + "\n\n")

	d.WriteString(labelStyle.Render("  ─── CAPABILITIES ────────────────────────") + "\n")
	d.WriteString(fmt.Sprintf("  Tools:     %s\n", strings.Join(agent.Tools, ", ")))
	d.WriteString(fmt.Sprintf("  Auto:      %s\n", strings.Join(agent.AutoTools, ", ")))
	if agent.Shortcut != "" {
		d.WriteString(fmt.Sprintf("  Shortcut:  %s\n", agent.Shortcut))
	}
	if agent.Hooks != nil && len(agent.Hooks.OnActivate) > 0 {
		cmd := agent.Hooks.OnActivate[0].Command
		if len(cmd) > mainWidth-12 {
			cmd = cmd[:mainWidth-12] + "..."
		}
		d.WriteString(fmt.Sprintf("  Spawn:     %s\n", cmd))
	}
	d.WriteString("\n")

	d.WriteString(labelStyle.Render("  ─── PERSONALITY ─────────────────────────") + "\n")
	personality := extractPersonality(agent.Prompt)
	if personality != "" {
		wrapped := wordWrap(personality, mainWidth-6)
		for _, line := range strings.Split(wrapped, "\n") {
			d.WriteString(fmt.Sprintf("  %s\n", dimStyle.Render(line)))
		}
	}
	d.WriteString("\n")

	d.WriteString(labelStyle.Render("  ─── ACTIONS ─────────────────────────────") + "\n")
	d.WriteString("  [enter] Deploy    [c] Copy prompt\n")
	d.WriteString("  [t] Spawn hook    [esc] Back\n")

	dossierPanel := StylePanel.Width(mainWidth).Render(d.String())

	// Portrait panel
	var p strings.Builder
	portrait := PortraitFor(agent.Name)
	portraitStyled := lipgloss.NewStyle().Foreground(ColorBrand).Render(portrait)
	p.WriteString(labelStyle.Render("  ╭─ PORTRAIT ─╮") + "\n")
	p.WriteString(portraitStyled + "\n")
	p.WriteString(labelStyle.Render("  ╰────────────╯") + "\n")

	portraitPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim).
		Padding(0, 1).
		Width(portraitW).
		Render(p.String())

	mainBody := lipgloss.JoinHorizontal(lipgloss.Top, dossierPanel, " ", portraitPanel)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", mainBody)
	statusBar := a.renderStatusBar("[enter] DEPLOY  [c] COPY  [t] SPAWN HOOK  [esc] BACK  [↑↓] NAV")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

func (a App) renderPipeline() string {
	header := a.renderHeader("Pipeline")
	sidebarWidth := 24
	mainWidth := a.width - sidebarWidth - 3

	var phases strings.Builder
	phases.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ PIPELINE") + "\n")
	phases.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────") + "\n\n")

	if a.pipelineState == nil {
		phases.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" No active mission") + "\n")
	} else {
		for _, phase := range a.pipelineState.Phases {
			badge := BadgeWaiting
			style := StyleInactive
			switch phase.Status {
			case pipeline.StatusComplete:
				badge = BadgePass
				style = lipgloss.NewStyle().Foreground(ColorActive)
			case pipeline.StatusRunning:
				badge = a.spinner.View()
				style = StyleActive
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
			line := fmt.Sprintf(" %s %d. %s%s", badge, phase.Number, phase.Name, cond)
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
			// Agent output — indented with pipe
			lines := strings.Split(msg.Content, "\n")
			maxOutput := 8
			for j, line := range lines {
				if j >= maxOutput && len(lines) > maxOutput {
					main.WriteString(pipeStyle.Render(fmt.Sprintf("│ [+%d more lines]", len(lines)-maxOutput)) + "\n")
					break
				}
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
		a.abortPending = false
	}
	statusBar := a.renderStatusBar(statusHint)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

func (a App) renderMission() string {
	header := a.renderHeader("New Mission")

	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)

	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("  ◆ NEW MISSION BRIEFING") + "\n\n")
	b.WriteString(dimStyle.Render("  Describe the objective. The KND Council will execute the") + "\n")
	b.WriteString(dimStyle.Render("  full pipeline: Analyst → Architect → Implement → QA → Review") + "\n\n")
	b.WriteString("  Mission: " + a.missionInput.View() + "\n\n")
	b.WriteString(dimStyle.Render("  [enter] Deploy council    [esc] Cancel") + "\n")

	body := StylePanel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderStatusBar("[enter] DEPLOY COUNCIL  [esc] CANCEL")
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body+"\n", statusBar)
}

func (a App) renderHelp() string {
	header := a.renderHeader("Operations Manual")

	help := `
  ◆ NAVIGATION                          ◆ MISSIONS
  ─────────────────────                  ──────────────────────
  ↑↓ / jk    Navigate roster            m         New mission
  0-9        Jump to operative           n         Next phase
  enter      Open dossier / deploy       r         Retry phase
  esc        Back / close                s         Skip phase
  /          Search operatives           H         Mission history
  tab        Cycle panel focus
  q          Disconnect

  ◆ VIEWS                               ◆ TOOLS
  ─────────────────────                  ──────────────────────
  p          Project navigator           L         lazygit
  W          Document viewer             B         btop
  C          Open COMMS (chat)           V         nvim
  ?          This manual                 tab       TERMINAL (in main)
  T          Cycle theme

  ◆ COMMS (CHAT)                        ◆ SYSTEM
  ─────────────────────                  ──────────────────────
  enter      Send message                P         Create PR (personal)
  ctrl+f     Attach file                 d         Git diff
  ctrl+s     Snippet picker              g         Git status
  @name      Switch agent                w         Toggle file watcher
  >name      Relay last response to agent
  >>name msg Relay custom message to agent
  esc        Back to dossier

  ◆ THE KND WAY
  ──────────────────────
  Sector V = core pipeline.  Specialists = cross-cutting.
  Council = full lifecycle.  "We fight for kids everywhere."`

	body := lipgloss.NewStyle().Foreground(ColorInfo).Render(help)
	statusBar := a.renderStatusBar("[esc] CLOSE MANUAL")
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body+"\n", statusBar)
}

// === COMPONENTS ===

func (a App) renderHeader(breadcrumb string) string {
	// Left: brand + breadcrumb
	left := "🌙 MOONBASE › " + breadcrumb

	// Right: backend + stack + spec indicator
	var rightParts []string
	if a.activeBackend != nil && a.activeBackend.Name() != "" {
		rightParts = append(rightParts, a.activeBackend.Name())
	}
	if a.projectCtx != nil && a.projectCtx.Stack.Language != "" {
		rightParts = append(rightParts, a.projectCtx.Stack.Language+"/"+a.projectCtx.Stack.BuildTool)
	}
	if a.projectCtx != nil && a.projectCtx.HasSpecs() {
		rightParts = append(rightParts, "◆")
	}
	right := strings.Join(rightParts, " │ ")

	// Time
	clock := a.clock
	if clock != "" {
		right = clock + "  " + right
	}

	// Calculate gap
	gap := a.width - len(left) - len(right) - 4
	if gap < 1 {
		// Truncate breadcrumb if header too narrow
		maxLeft := a.width - len(right) - 6
		if maxLeft > 10 && len(left) > maxLeft {
			left = left[:maxLeft] + "…"
		}
		gap = 1
	}

	full := left + strings.Repeat(" ", gap) + right

	return StyleHeader.Width(a.width).Render(full)
}

func (a App) renderSidebar(width int, maxH int) string {
	var s strings.Builder

	// Agent roster with grouping
	type agentEntry struct {
		key   string
		name  string
		role  string
		index int
	}
	type agentGroup struct {
		title   string
		entries []agentEntry
	}

	groups := []agentGroup{
		{"SECTOR V", []agentEntry{
			{"0", "Numbuh 0", "Overseer", -1},
			{"1", "Numbuh 1", "Analyst", -1},
			{"2", "Numbuh 2", "Architect", -1},
			{"3", "Numbuh 3", "Implement", -1},
			{"4", "Numbuh 4", "QA", -1},
			{"5", "Numbuh 5", "Reviewer", -1},
		}},
		{"SPECIALISTS", []agentEntry{
			{"6", "Numbuh 362", "DevOps", -1},
			{"7", "Numbuh 274", "Security", -1},
			{"8", "Numbuh 86", "Cleanup", -1},
			{"9", "Numbuh 999", "Docs", -1},
			{"F", "Numbuh 13", "Chaos", -1},
		}},
		{"META", []agentEntry{
			{"K", "Council", "Pipeline", -1},
			{"Z", "Sector Z", "Legacy", -1},
			{"M", "Numbuh 9", "Migration", -1},
		}},
	}

	// Resolve actual registry indices
	allAgents := a.registry.All()
	for gi := range groups {
		for ei := range groups[gi].entries {
			for ai, agent := range allAgents {
				matchName := groups[gi].entries[ei].name
				if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(matchName)) ||
					(matchName == "Council" && agent.Name == "knd-council") ||
					(matchName == "Sector Z" && agent.Name == "sector-z") {
					groups[gi].entries[ei].index = ai
					break
				}
			}
		}
	}

	showHints := a.view == ViewDashboard
	roleWidth := 10
	if width < 28 {
		roleWidth = 0
	}

	for _, group := range groups {
		// Section header
		s.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ "+group.title) + "\n")

		for _, entry := range group.entries {
			isSelected := entry.index == a.cursor

			// Build the line
			prefix := " "
			badge := BadgeInactive
			nameStyle := lipgloss.NewStyle().Foreground(ColorText)
			roleStyle := lipgloss.NewStyle().Foreground(ColorMuted)

			if isSelected {
				prefix = "▸"
				badge = BadgeActive
				nameStyle = StyleActive
				roleStyle = lipgloss.NewStyle().Foreground(ColorActive)
			}

			// Key hint
			hint := ""
			if showHints {
				hint = lipgloss.NewStyle().Foreground(ColorDim).Render("["+entry.key+"]")
			}

			// Name (truncate if needed)
			name := entry.name
			maxName := width - 8
			if roleWidth > 0 {
				maxName = width - roleWidth - 6
			}
			if len(name) > maxName {
				name = name[:maxName]
			}

			// Role (right-aligned)
			role := ""
			if roleWidth > 0 {
				r := entry.role
				if len(r) > roleWidth {
					r = r[:roleWidth]
				}
				gap := width - len(prefix) - len(badge) - 1 - len(name) - len(r) - 4
				if hint != "" {
					gap -= 3
				}
				if gap < 1 {
					gap = 1
				}
				role = roleStyle.Render(strings.Repeat(" ", gap) + r)
			}

			if hint != "" {
				s.WriteString(fmt.Sprintf("%s%s%s %s%s\n", prefix, hint, badge, nameStyle.Render(name), role))
			} else {
				s.WriteString(fmt.Sprintf("%s%s %s%s\n", prefix, badge, nameStyle.Render(name), role))
			}
		}
		s.WriteString("\n")
	}

	// Tools section
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ TOOLS") + "\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────────") + "\n")
	tools := []string{"lazygit", "docker", "btop", "nvim", "tmux", "fish"}
	for _, tool := range tools {
		mark := "✗"
		st := StyleInactive
		if _, err := exec.LookPath(tool); err == nil {
			mark = "✓"
			st = lipgloss.NewStyle().Foreground(ColorActive)
		}
		s.WriteString(st.Render(fmt.Sprintf(" %s %s", mark, tool)) + "\n")
	}

	// Backend section
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ BACKEND") + "\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────────") + "\n")
	for _, b := range a.backends {
		mark := "✗"
		st := StyleInactive
		if b.Available() {
			mark = "●"
			st = lipgloss.NewStyle().Foreground(ColorActive)
			if a.blink {
				mark = "◉"
			}
		}
		s.WriteString(st.Render(fmt.Sprintf(" %s %s", mark, b.Name())) + "\n")
	}

	return StyleSidebar.
		Width(width).
		Height(maxH).
		Render(s.String())
}

func (a App) renderMainPanel(width int, maxH int) string {
	// KND file browser mode
	if a.browsing && a.fileBrowser != nil {
		return a.renderFileBrowser(width, maxH)
	}

	// Terminal/Intel mode
	borderColor := ColorInfo
	if a.focus == FocusMain {
		borderColor = ColorActive
	}

	var s strings.Builder

	cwdShort := a.cwd
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(cwdShort, home) {
		cwdShort = "~" + cwdShort[len(home):]
	}
	title := lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("─ TERMINAL ")
	cwd := lipgloss.NewStyle().Foreground(ColorDim).Render(cwdShort + " ")
	s.WriteString(title + cwd + strings.Repeat("─", max(1, width-len(cwdShort)-14)) + "\n")

	maxLines := maxH - 4
	if maxLines < 4 {
		maxLines = 4
	}

	var lines []string
	for _, entry := range a.intel {
		timeStyle := lipgloss.NewStyle().Foreground(ColorDim)
		msg := entry.Message
		if len(msg) > width-10 {
			msg = msg[:width-10]
		}
		lines = append(lines, fmt.Sprintf(" %s  %s", timeStyle.Render(entry.Time), msg))
	}
	for _, line := range a.termOutput {
		lines = append(lines, " "+line)
	}

	if len(lines) == 0 {
		s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" Type commands below. [`] for file browser.") + "\n")
	}

	start := 0
	if len(lines) > maxLines {
		start = len(lines) - maxLines
	}
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if len(line) > width-4 {
			line = line[:width-4]
		}
		s.WriteString(line + "\n")
	}

	if a.termActive {
		prompt := lipgloss.NewStyle().Foreground(ColorActive).Bold(true).Render(" $ ")
		s.WriteString(prompt + a.termInput.View())
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Height(maxH).
		Render(s.String())
}

func (a App) renderRightPanel(width int, maxH int) string {
	borderColor := ColorDim
	if a.focus == FocusRight {
		borderColor = ColorActive
	}

	var s strings.Builder
	labelStyle := lipgloss.NewStyle().Foreground(ColorInfo).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)

	// System Status
	radar := lipgloss.NewStyle().Foreground(ColorActive).Render(a.anim.RenderRadar())
	s.WriteString(labelStyle.Render("─ SYSTEM STATUS ") + radar + " " + strings.Repeat("─", max(1, width-21)) + "\n")

	gitStyle := lipgloss.NewStyle().Foreground(ColorActive)
	if !a.gitClean {
		gitStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	}
	s.WriteString(fmt.Sprintf(" GIT    %s %s\n", gitStyle.Render(a.gitBranch), a.gitStatus()))

	dockerStatus := dimStyle.Render("not running")
	if a.dockerCount > 0 {
		dockerStatus = lipgloss.NewStyle().Foreground(ColorActive).Render(fmt.Sprintf("● %d up", a.dockerCount))
	}
	s.WriteString(fmt.Sprintf(" DOCKER %s\n", dockerStatus))
	s.WriteString(fmt.Sprintf(" AGENTS %d loaded\n", a.registry.Count()))
	s.WriteString("\n")

	// Threat Level
	s.WriteString(labelStyle.Render("─ THREAT LEVEL ") + strings.Repeat("─", max(1, width-18)) + "\n")
	s.WriteString(" " + a.renderThreatGauge(width-4) + "\n\n")

	// Mission History
	s.WriteString(labelStyle.Render("─ MISSION HISTORY ") + strings.Repeat("─", max(1, width-21)) + "\n")
	for i, m := range a.missions {
		if i >= 5 {
			break
		}
		s.WriteString(fmt.Sprintf(" %s #%d %s\n", m.Status, len(a.missions)-i, m.Name))
	}
	if len(a.missions) == 0 {
		s.WriteString(dimStyle.Render(" No missions yet.") + "\n")
	}

	// Recent Files from watcher
	s.WriteString("\n")
	s.WriteString(labelStyle.Render("─ RECENT FILES ") + strings.Repeat("─", max(1, width-18)) + "\n")
	if a.fileWatcher != nil && a.fileWatcher.Running() {
		recent := a.fileWatcher.Recent()
		if len(recent) == 0 {
			s.WriteString(dimStyle.Render(" watching...") + "\n")
		}
		for i := len(recent) - 1; i >= 0 && i >= len(recent)-5; i-- {
			f := recent[i]
			ts := f.Time.Format("15:04")
			name := f.Path
			if len(name) > width-10 {
				name = name[:width-10]
			}
			s.WriteString(fmt.Sprintf(" %s %s\n", dimStyle.Render(ts), name))
		}
	} else {
		s.WriteString(dimStyle.Render(" [w] to start") + "\n")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Height(maxH).
		Render(s.String())
}

func (a App) renderThreatGauge(width int) string {
	// Map git diff lines to threat level
	level := "LOW"
	color := ColorActive
	filled := 2

	switch {
	case a.gitDiffLines > 500:
		level = "CRITICAL"
		color = ColorError
		filled = 10
	case a.gitDiffLines > 200:
		level = "HIGH"
		color = ColorError
		filled = 8
	case a.gitDiffLines > 50:
		level = "MEDIUM"
		color = ColorWarning
		filled = 5
	case a.gitDiffLines > 10:
		level = "LOW"
		color = ColorActive
		filled = 3
	}

	barW := width - len(level) - 2
	if barW < 5 {
		barW = 5
	}
	if filled > barW {
		filled = barW
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	return lipgloss.NewStyle().Foreground(color).Render(bar+" "+level)
}

func (a App) renderStatusBar(keys string) string {
	uptime := fmt.Sprintf("▲ %s", a.uptime())
	gap := a.width - len(keys) - len(uptime) - 4
	if gap < 1 {
		return StyleStatusBar.Width(a.width).Render("  " + keys)
	}
	return StyleStatusBar.Width(a.width).Render("  " + keys + strings.Repeat(" ", gap) + uptime)
}

// === UTILITIES ===

func extractPersonality(prompt string) string {
	lines := strings.Split(prompt, "\\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Personality:") {
			return line
		}
	}
	lines = strings.Split(prompt, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Personality:") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		width = 60
	}
	words := strings.Fields(text)
	var lines []string
	var current string
	for _, word := range words {
		if len(current)+len(word)+1 > width {
			lines = append(lines, current)
			current = word
		} else {
			if current == "" {
				current = word
			} else {
				current += " " + word
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}
