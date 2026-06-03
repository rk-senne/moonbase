package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a App) renderBoot() string {
	var b strings.Builder

	// Center vertically
	padding := (a.height - 20) / 2
	for i := 0; i < padding; i++ {
		b.WriteString("\n")
	}

	// Logo
	logo := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render(kndLogo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, logo))
	b.WriteString("\n")

	subtitle := lipgloss.NewStyle().Foreground(ColorInfo).Render(moonbaseLogo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, subtitle))
	b.WriteString("\n\n")

	// Boot messages with progress
	for i := 0; i < a.bootStep && i < len(bootMessages); i++ {
		style := lipgloss.NewStyle().Foreground(ColorActive)
		if i == a.bootStep-1 {
			line := fmt.Sprintf("  %s %s", a.spinner.View(), bootMessages[i])
			b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, style.Render(line)))
		} else {
			line := fmt.Sprintf("  ✓ %s", bootMessages[i])
			b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, lipgloss.NewStyle().Foreground(ColorDim).Render(line)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (a App) renderDashboard() string {
	sidebarWidth := 26
	mainWidth := a.width - sidebarWidth - 4

	// Header
	header := a.renderHeader("ONLINE")

	// Sidebar
	sidebar := a.renderSidebar(sidebarWidth)

	// Main panels
	intelPanel := a.renderIntelFeed(mainWidth)
	statusPanel := a.renderSystemStatus(mainWidth)
	mainContent := lipgloss.JoinVertical(lipgloss.Left, intelPanel, statusPanel)

	// Join sidebar + main
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", mainContent)

	// Status bar
	var statusBar string
	if a.searching {
		statusBar = a.renderStatusBar("/ " + a.searchInput.View() + "  [enter] SELECT  [esc] CANCEL")
	} else {
		statusBar = a.renderStatusBar("[?] HELP  [↑↓] SELECT  [enter] DOSSIER  [m] MISSION  [T] THEME  [q] DISCONNECT")
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

func (a App) renderDossier() string {
	sidebarWidth := 26
	mainWidth := a.width - sidebarWidth - 4

	header := a.renderHeader("OPERATIVE DOSSIER")
	sidebar := a.renderSidebar(sidebarWidth)

	agent := a.registry.Get(a.selected)

	// Dossier content
	var d strings.Builder
	nameStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorInfo)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)

	d.WriteString(nameStyle.Render(fmt.Sprintf("  %s", strings.ToUpper(agent.Name))) + "\n")
	d.WriteString(dimStyle.Render(fmt.Sprintf("  %s", agent.Description)) + "\n\n")

	d.WriteString(labelStyle.Render("  ─── CAPABILITIES ────────────────────────") + "\n")
	d.WriteString(fmt.Sprintf("  Tools:     %s\n", strings.Join(agent.Tools, ", ")))
	d.WriteString(fmt.Sprintf("  Allowed:   %s\n", strings.Join(agent.AllowedTools, ", ")))
	if agent.KeyboardShortcut != "" {
		d.WriteString(fmt.Sprintf("  Shortcut:  %s\n", agent.KeyboardShortcut))
	}

	// Spawn hook
	if hooks, ok := agent.Hooks["agentSpawn"]; ok && len(hooks) > 0 {
		d.WriteString(fmt.Sprintf("  Spawn:     %s\n", hooks[0].Command))
	}
	d.WriteString("\n")

	// Personality (extract from prompt - last paragraph)
	d.WriteString(labelStyle.Render("  ─── PERSONALITY ─────────────────────────") + "\n")
	personality := extractPersonality(agent.Prompt)
	if personality != "" {
		// Word wrap
		wrapped := wordWrap(personality, mainWidth-6)
		for _, line := range strings.Split(wrapped, "\n") {
			d.WriteString(fmt.Sprintf("  %s\n", dimStyle.Render(line)))
		}
	}
	d.WriteString("\n")

	d.WriteString(labelStyle.Render("  ─── ACTIONS ─────────────────────────────") + "\n")
	d.WriteString("  [enter] Deploy operative    [c] Copy prompt\n")
	d.WriteString("  [esc] Back to dashboard\n")

	dossierPanel := StylePanel.Width(mainWidth).Render(d.String())
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", dossierPanel)
	statusBar := a.renderStatusBar("[enter] DEPLOY  [c] COPY PROMPT  [esc] BACK  [↑↓] NAVIGATE")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

func (a App) renderPipeline() string {
	header := a.renderHeader("MISSION ACTIVE")
	sidebarWidth := 26
	mainWidth := a.width - sidebarWidth - 4

	// Phase list sidebar
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
			case 2: // Complete
				badge = BadgePass
				style = lipgloss.NewStyle().Foreground(ColorActive)
			case 1: // Running
				badge = a.spinner.View()
				style = StyleActive
			case 4: // Failed
				badge = BadgeFail
				style = lipgloss.NewStyle().Foreground(ColorError)
			case 3: // Skipped
				badge = "⊘"
				style = StyleInactive
			}
			line := fmt.Sprintf(" %s %d. %s", badge, phase.Number, phase.Name)
			phases.WriteString(style.Render(line) + "\n")
			if phase.Duration != "" {
				phases.WriteString(StyleInactive.Render(fmt.Sprintf("      %s", phase.Duration)) + "\n")
			}
		}
	}

	phaseSidebar := StyleSidebar.Width(sidebarWidth).Render(phases.String())

	// Main panel: live output
	var main strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(ColorInfo).Bold(true)

	if a.pipelineState != nil && a.pipelineState.Task != "" {
		main.WriteString(titleStyle.Render(fmt.Sprintf("─ MISSION: %s ", a.pipelineState.Task)))
		main.WriteString(strings.Repeat("─", mainWidth-len(a.pipelineState.Task)-14) + "\n\n")
	} else {
		main.WriteString(titleStyle.Render("─ AWAITING MISSION ") + strings.Repeat("─", mainWidth-22) + "\n\n")
	}

	// Streaming output lines
	maxLines := a.height - 12
	if maxLines < 5 {
		maxLines = 5
	}
	start := 0
	if len(a.pipelineOutput) > maxLines {
		start = len(a.pipelineOutput) - maxLines
	}
	for i := start; i < len(a.pipelineOutput); i++ {
		line := a.pipelineOutput[i]
		if len(line) > mainWidth-4 {
			line = line[:mainWidth-4]
		}
		main.WriteString(" " + line + "\n")
	}

	if len(a.pipelineOutput) == 0 {
		main.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" Press [m] to brief a new mission.\n"))
		main.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" The KND Council will execute the full pipeline.\n"))
	}

	mainPanel := StylePanel.Width(mainWidth).Render(main.String())
	body := lipgloss.JoinHorizontal(lipgloss.Top, phaseSidebar, "  ", mainPanel)

	statusBar := a.renderStatusBar("[n] NEXT  [r] RETRY  [s] SKIP  [m] NEW MISSION  [esc] BACK")
	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

func (a App) renderMission() string {
	header := a.renderHeader("MISSION BRIEFING")

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
	header := a.renderHeader("OPERATIONS MANUAL")

	help := `
  ◆ NAVIGATION                          ◆ MISSIONS
  ─────────────────────                  ──────────────────────
  ↑↓ / jk    Navigate roster            m         New mission
  0-9        Jump to operative           n         Next phase
  enter      Open dossier / deploy       r         Retry phase
  esc        Back / close                s         Skip phase
  /          Search operatives           l         View log
  q          Disconnect from Moonbase

  ◆ OPERATIVE ACTIONS                   ◆ AI BACKENDS
  ─────────────────────                  ──────────────────────
  enter      Deploy with current AI      Supported:
  c          Copy prompt to clipboard    • kiro-cli (native)
  t          Run spawn hook              • codex (OpenAI CLI)
                                         • openai / anthropic (API)
  ◆ SYSTEM                               • ollama (local)
  ─────────────────────                  • clipboard (fallback)
  d          Git diff
  g          Git status                 ◆ THE KND WAY
  T          Cycle theme                ──────────────────────
  ?          This manual                Sector V = core pipeline.
                                        Specialists = cross-cutting.
                                        Council = full lifecycle.

                                        "We fight for kids everywhere."`

	body := lipgloss.NewStyle().Foreground(ColorInfo).Render(help)
	statusBar := a.renderStatusBar("[esc] CLOSE MANUAL")
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body+"\n", statusBar)
}

// --- Components ---

func (a App) renderHeader(status string) string {
	left := "░░░ K.N.D. MOONBASE ░░░ TACTICAL OPERATIONS NETWORK"
	right := fmt.Sprintf("░░░ %s ░░░", status)
	gap := a.width - len(left) - len(right)
	if gap < 0 {
		gap = 1
	}
	full := left + strings.Repeat("░", gap) + right

	return StyleHeader.Width(a.width).Render(full)
}

func (a App) renderSidebar(width int) string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ OPERATIVES") + "\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────") + "\n\n")

	s.WriteString(lipgloss.NewStyle().Foreground(ColorInfo).Render("SECTOR V") + "\n")

	allAgents := a.registry.All()
	for i, agent := range allAgents {
		badge := BadgeInactive
		style := StyleInactive
		if i == a.cursor {
			badge = BadgeActive
			style = StyleActive
		}

		// Shorten name for sidebar
		shortName := agent.Name
		if len(shortName) > 14 {
			shortName = shortName[:14]
		}

		line := fmt.Sprintf(" %s %s", badge, shortName)
		s.WriteString(style.Render(line) + "\n")

		// Add specialist header after sector v
		if i == 5 {
			s.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorInfo).Render("SPECIALISTS") + "\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ AI BACKEND") + "\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────") + "\n")
	for _, b := range a.backends {
		mark := "✗"
		style := StyleInactive
		if b.Available() {
			mark = "✓"
			style = lipgloss.NewStyle().Foreground(ColorActive)
		}
		s.WriteString(style.Render(fmt.Sprintf(" %s %s", mark, b.Name())) + "\n")
	}

	return StyleSidebar.Width(width).Render(s.String())
}

func (a App) renderIntelFeed(width int) string {
	var s strings.Builder
	title := lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("─ INTEL FEED ")
	s.WriteString(title + strings.Repeat("─", width-16) + "\n")

	// Show last entries that fit
	maxEntries := 8
	start := 0
	if len(a.intel) > maxEntries {
		start = len(a.intel) - maxEntries
	}

	if len(a.intel) == 0 {
		s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(" Awaiting intel...") + "\n")
	}

	for i := start; i < len(a.intel); i++ {
		entry := a.intel[i]
		timeStyle := lipgloss.NewStyle().Foreground(ColorDim)
		s.WriteString(fmt.Sprintf(" %s  %s\n", timeStyle.Render(entry.Time), entry.Message))
	}

	return StylePanel.Width(width).Render(s.String())
}

func (a App) renderSystemStatus(width int) string {
	var s strings.Builder
	title := lipgloss.NewStyle().Foreground(ColorInfo).Bold(true).Render("─ SYSTEM STATUS ")
	s.WriteString(title + strings.Repeat("─", width-19) + "\n")

	gitStyle := lipgloss.NewStyle().Foreground(ColorActive)
	if !a.gitClean {
		gitStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	}

	s.WriteString(fmt.Sprintf(" GIT     %s %s\n", gitStyle.Render(a.gitBranch), a.gitStatus()))

	dockerStatus := lipgloss.NewStyle().Foreground(ColorDim).Render("not running")
	if a.dockerCount > 0 {
		dockerStatus = lipgloss.NewStyle().Foreground(ColorActive).Render(fmt.Sprintf("● %d containers", a.dockerCount))
	}
	s.WriteString(fmt.Sprintf(" DOCKER  %s\n", dockerStatus))
	s.WriteString(fmt.Sprintf(" AGENTS  %d loaded\n", a.registry.Count()))

	return StylePanel.Width(width).Render(s.String())
}

func (a App) renderStatusBar(keys string) string {
	return StyleStatusBar.Width(a.width).Render("  " + keys)
}

// --- Utilities ---

func extractPersonality(prompt string) string {
	lines := strings.Split(prompt, "\\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Personality:") {
			return line
		}
	}
	// Try actual newlines
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
