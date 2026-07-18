package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
		if maxLeft < 4 {
			maxLeft = 4
		}
		if len(left) > maxLeft {
			left = left[:maxLeft] + "…"
		}
		gap = 1
	}

	full := left + strings.Repeat(" ", gap) + right

	return StyleHeader.Width(a.width).Render(full)
}

func (a App) renderSidebar(width int, maxH int) string {
	var s strings.Builder

	groups := buildSidebarGroups(a.registry)

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
	tools := []string{"lazygit", "docker", "btop", "nvim", "cmux", "tmux", "fish"}
	for _, tool := range tools {
		mark := "✗"
		st := StyleInactive
		if a.isToolAvailable(tool) {
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
		if width > 12 && len(msg) > width-10 {
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
		if width > 6 && len(line) > width-4 {
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
