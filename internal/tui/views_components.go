package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

func (a App) renderHeader(breadcrumb string) string {
	// Left: brand + breadcrumb
	left := "🌙 MOONBASE › " + breadcrumb

	// Right: backend + stack + spec indicator
	var rightParts []string
	if a.env.Backend.Active != nil && a.env.Backend.Active.Name() != "" {
		rightParts = append(rightParts, a.env.Backend.Active.Name())
	}
	if a.projectCtx != nil && a.projectCtx.Stack.Language != "" {
		rightParts = append(rightParts, a.projectCtx.Stack.Language+"/"+a.projectCtx.Stack.BuildTool)
	}
	if a.projectCtx != nil && a.projectCtx.HasSpecs() {
		rightParts = append(rightParts, "◆")
	}

	// Mission-in-progress indicator: always visible from every view except the
	// pipeline view itself (which already dedicates the screen to the mission),
	// so the operator never loses sight of a running mission.
	if a.view != ViewPipeline {
		if seg, ok := a.missionIndicator(); ok {
			seg = truncateToWidth(seg, a.width/3)
			rightParts = append([]string{seg}, rightParts...)
		}
	}

	right := strings.Join(rightParts, " │ ")

	// Time
	clock := a.chrome.Clock
	if clock != "" {
		right = clock + "  " + right
	}

	// Calculate gap using visual width (handles emoji + unicode)
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := a.width - leftW - rightW - 4
	if gap < 1 {
		// Truncate breadcrumb if header too narrow
		maxLeft := a.width - rightW - 6
		if maxLeft < 4 {
			maxLeft = 4
		}
		if leftW > maxLeft {
			left = truncateToWidth(left, maxLeft)
		}
		gap = 1
	}

	full := left + strings.Repeat(" ", gap) + right

	return a.theme.Styles.Header.Width(a.width).Render(full)
}

func (a App) renderSidebar(width int, maxH int) string {
	var s strings.Builder

	groups := buildSidebarGroups(a.registry)

	showHints := a.view == ViewDashboard
	roleWidth := 10
	if width < 28 {
		roleWidth = 0
	}

	// Build all roster lines (agents + group headers) to compute the visible
	// window. Each entry is tagged with its registry index (-1 for non-agent rows
	// like group headers and blank lines) so we can apply scroll-offset logic.
	type rosterLine struct {
		text       string
		agentIndex int // -1 for non-agent lines (headers, blanks)
	}

	var rosterLines []rosterLine

	for _, group := range groups {
		// Section header
		headerText := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true).Render("◆ " + group.title)
		rosterLines = append(rosterLines, rosterLine{text: headerText, agentIndex: -1})

		for _, entry := range group.entries {
			isSelected := entry.index == a.views.Dashboard.Cursor

			// Build the line
			prefix := " "
			badge := BadgeInactive
			nameStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Text)
			roleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Muted)

			if isSelected {
				prefix = "▸"
				badge = BadgeActive
				nameStyle = a.theme.Styles.Active
				roleStyle = lipgloss.NewStyle().Foreground(a.theme.Data.Active)
			}

			// Key hint
			hint := ""
			if showHints {
				hint = lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("[" + entry.key + "]")
			}

			// Name (truncate if needed)
			name := entry.name
			maxName := width - 8
			if roleWidth > 0 {
				maxName = width - roleWidth - 6
			}
			if lipgloss.Width(name) > maxName {
				runes := []rune(name)
				for len(runes) > 0 && lipgloss.Width(string(runes)) > maxName {
					runes = runes[:len(runes)-1]
				}
				name = string(runes)
			}

			// Role (right-aligned)
			role := ""
			if roleWidth > 0 {
				r := entry.role
				if len(r) > roleWidth {
					r = r[:roleWidth]
				}
				usedW := lipgloss.Width(prefix) + lipgloss.Width(badge) + 1 + lipgloss.Width(name) + lipgloss.Width(r) + 4
				if hint != "" {
					usedW += 3
				}
				gap := width - usedW
				if gap < 1 {
					gap = 1
				}
				role = roleStyle.Render(strings.Repeat(" ", gap) + r)
			}

			var line string
			if hint != "" {
				line = fmt.Sprintf("%s%s%s %s%s", prefix, hint, badge, nameStyle.Render(name), role)
			} else {
				line = fmt.Sprintf("%s%s %s%s", prefix, badge, nameStyle.Render(name), role)
			}
			rosterLines = append(rosterLines, rosterLine{text: line, agentIndex: entry.index})
		}
		// Trailing blank line after group
		rosterLines = append(rosterLines, rosterLine{text: "", agentIndex: -1})
	}

	// Determine the line index where the cursor is so we can scroll to it.
	cursorLineIdx := 0
	for i, rl := range rosterLines {
		if rl.agentIndex == a.views.Dashboard.Cursor {
			cursorLineIdx = i
			break
		}
	}

	// Reserve lines for tools + backend sections (shown when space permits).
	toolsLines := 3 + 7                              // header + separator + 7 tools
	backendLines := 3 + len(a.env.Backend.Available) // header + separator + backends
	extraLines := 2 + toolsLines + 1 + backendLines  // blanks + tools + blank + backend

	// Available height for the roster portion (leave room for extras if possible).
	var rosterMaxH int
	showExtras := false
	if maxH > len(rosterLines)+extraLines {
		// Enough room to show everything without scrolling.
		rosterMaxH = len(rosterLines)
		showExtras = true
	} else if maxH > len(rosterLines) {
		// Roster fits entirely but extras may be partial — show roster, skip extras.
		rosterMaxH = len(rosterLines)
	} else {
		// Roster doesn't fit — dedicate all space to it minus a scroll indicator line.
		rosterMaxH = maxH - 2
		if rosterMaxH < 5 {
			rosterMaxH = 5
		}
	}

	// Apply scroll offset to keep cursor visible within the roster window.
	offset := a.views.Dashboard.ScrollOffset
	if cursorLineIdx < offset {
		offset = cursorLineIdx
	}
	if cursorLineIdx >= offset+rosterMaxH {
		offset = cursorLineIdx - rosterMaxH + 1
	}
	// Clamp offset
	if offset < 0 {
		offset = 0
	}
	maxOffset := len(rosterLines) - rosterMaxH
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}

	// Render the visible portion of the roster.
	end := offset + rosterMaxH
	if end > len(rosterLines) {
		end = len(rosterLines)
	}

	// Scroll-up indicator
	if offset > 0 {
		s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("  ▲ more") + "\n")
	}

	for i := offset; i < end; i++ {
		s.WriteString(rosterLines[i].text + "\n")
	}

	// Scroll-down indicator
	if end < len(rosterLines) {
		s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("  ▼ more") + "\n")
	}

	// Tools section (only when space permits)
	if showExtras {
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true).Render("◆ TOOLS") + "\n")
		s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("──────────────────") + "\n")
		tools := []string{"lazygit", "docker", "btop", "nvim", "cmux", "tmux", "fish"}
		for _, tool := range tools {
			mark := "✗"
			st := a.theme.Styles.Inactive
			if a.isToolAvailable(tool) {
				mark = "✓"
				st = lipgloss.NewStyle().Foreground(a.theme.Data.Active)
			}
			s.WriteString(st.Render(fmt.Sprintf(" %s %s", mark, tool)) + "\n")
		}

		// Backend section
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true).Render("◆ BACKEND") + "\n")
		s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("──────────────────") + "\n")
		for _, b := range a.env.Backend.Available {
			mark := "✗"
			st := a.theme.Styles.Inactive
			if b.Available() {
				mark = "●"
				st = lipgloss.NewStyle().Foreground(a.theme.Data.Active)
				if a.chrome.Blink {
					mark = "◉"
				}
			}
			s.WriteString(st.Render(fmt.Sprintf(" %s %s", mark, b.Name())) + "\n")
		}
	}

	return a.theme.Styles.Sidebar.
		Width(width).
		Height(maxH).
		Render(s.String())
}

func (a App) renderMainPanel(width int, maxH int) string {
	// KND file browser mode
	if a.views.Browser.Active && a.views.Browser.FileBrowser != nil {
		return a.renderFileBrowser(width, maxH)
	}

	// Terminal/Intel mode
	borderColor := a.theme.Data.Info
	if a.chrome.Focus == FocusMain {
		borderColor = a.theme.Data.Active
	}

	var s strings.Builder

	cwdShort := a.views.Terminal.Cwd
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(cwdShort, home) {
		cwdShort = "~" + cwdShort[len(home):]
	}
	title := lipgloss.NewStyle().Foreground(a.theme.Data.Info).Bold(true).Render("─ TERMINAL ")
	cwd := lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render(cwdShort + " ")
	titleW := lipgloss.Width(title)
	cwdW := lipgloss.Width(cwd)
	fillW := max(1, width-titleW-cwdW)
	s.WriteString(title + cwd + strings.Repeat("─", fillW) + "\n")

	maxLines := maxH - 4
	if maxLines < 4 {
		maxLines = 4
	}

	var lines []string
	for _, entry := range a.intel {
		timeStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
		msg := entry.Message
		if width > 12 && lipgloss.Width(msg) > width-10 {
			// Truncate by runes
			runes := []rune(msg)
			for len(runes) > 0 && lipgloss.Width(string(runes)) > width-10 {
				runes = runes[:len(runes)-1]
			}
			msg = string(runes)
		}
		lines = append(lines, fmt.Sprintf(" %s  %s", timeStyle.Render(entry.Time), msg))
	}
	for _, line := range a.views.Terminal.Output {
		lines = append(lines, " "+line)
	}

	if len(lines) == 0 {
		s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render(" Type commands below. [`] for file browser.") + "\n")
	}

	start := 0
	if len(lines) > maxLines {
		start = len(lines) - maxLines
	}
	for i := start; i < len(lines); i++ {
		line := lines[i]
		if width > 6 && lipgloss.Width(line) > width-4 {
			runes := []rune(line)
			for len(runes) > 0 && lipgloss.Width(string(runes)) > width-4 {
				runes = runes[:len(runes)-1]
			}
			line = string(runes)
		}
		s.WriteString(line + "\n")
	}

	if a.views.Terminal.Active {
		prompt := lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true).Render(" $ ")
		s.WriteString(prompt + a.views.Terminal.Input.View())
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
	borderColor := a.theme.Data.Dim
	if a.chrome.Focus == FocusRight {
		borderColor = a.theme.Data.Active
	}

	var s strings.Builder
	labelStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Info).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)

	// System Status
	radar := lipgloss.NewStyle().Foreground(a.theme.Data.Active).Render(a.chrome.Anim.RenderRadar())
	sysLabel := labelStyle.Render("─ SYSTEM STATUS ")
	sysLabelW := lipgloss.Width(sysLabel)
	radarW := lipgloss.Width(radar)
	s.WriteString(sysLabel + radar + " " + strings.Repeat("─", max(1, width-sysLabelW-radarW-1)) + "\n")

	gitStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Active)
	if !a.env.System.Clean {
		gitStyle = lipgloss.NewStyle().Foreground(a.theme.Data.Warning)
	}
	s.WriteString(fmt.Sprintf(" GIT    %s %s\n", gitStyle.Render(a.env.System.Branch), a.gitStatus()))

	dockerStatus := dimStyle.Render("not running")
	if a.env.System.Docker > 0 {
		dockerStatus = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Render(fmt.Sprintf("● %d up", a.env.System.Docker))
	}
	s.WriteString(fmt.Sprintf(" DOCKER %s\n", dockerStatus))
	s.WriteString(fmt.Sprintf(" AGENTS %d loaded\n", a.registry.Count()))
	s.WriteString("\n")

	// Threat Level
	threatLabel := labelStyle.Render("─ THREAT LEVEL ")
	s.WriteString(threatLabel + strings.Repeat("─", max(1, width-lipgloss.Width(threatLabel))) + "\n")
	s.WriteString(" " + a.renderThreatGauge(width-4) + "\n")
	s.WriteString(" " + dimStyle.Render(computeThreat(a.threatSignals()).Reason) + "\n\n")

	// Mission History
	missionLabel := labelStyle.Render("─ MISSION HISTORY ")
	s.WriteString(missionLabel + strings.Repeat("─", max(1, width-lipgloss.Width(missionLabel))) + "\n")
	for i, m := range a.views.Mission.History {
		if i >= 5 {
			break
		}
		name := m.Name
		maxMissionW := width - 8
		if maxMissionW < 4 {
			maxMissionW = 4
		}
		if lipgloss.Width(name) > maxMissionW {
			runes := []rune(name)
			for len(runes) > 0 && lipgloss.Width(string(runes)) > maxMissionW {
				runes = runes[:len(runes)-1]
			}
			name = string(runes)
		}
		s.WriteString(fmt.Sprintf(" %s #%d %s\n", m.Status, len(a.views.Mission.History)-i, name))
	}
	if len(a.views.Mission.History) == 0 {
		s.WriteString(dimStyle.Render(" No missions yet.") + "\n")
	}

	// Recent Files from watcher
	s.WriteString("\n")
	recentLabel := labelStyle.Render("─ RECENT FILES ")
	s.WriteString(recentLabel + strings.Repeat("─", max(1, width-lipgloss.Width(recentLabel))) + "\n")
	if a.env.Infra.Watcher != nil && a.env.Infra.Watcher.Running() {
		recent := a.env.Infra.Watcher.Recent()
		if len(recent) == 0 {
			s.WriteString(dimStyle.Render(" watching...") + "\n")
		}
		for i := len(recent) - 1; i >= 0 && i >= len(recent)-5; i-- {
			f := recent[i]
			ts := f.Time.Format("15:04")
			name := f.Path
			if lipgloss.Width(name) > width-10 {
				runes := []rune(name)
				for len(runes) > 0 && lipgloss.Width(string(runes)) > width-10 {
					runes = runes[:len(runes)-1]
				}
				name = string(runes)
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

// threatSignals returns the current working-tree signals from detected state.
func (a App) threatSignals() ThreatSignals {
	return a.env.System.Signals()
}

func (a App) renderThreatGauge(width int) string {
	lvl := computeThreat(a.threatSignals())

	color := a.theme.Data.Active // LOW
	switch lvl.Name {
	case "MEDIUM":
		color = a.theme.Data.Warning
	case "HIGH", "CRITICAL":
		color = a.theme.Data.Error
	}

	// Fill the 10-segment bar from the 0..100 score; any non-zero risk shows
	// at least one segment.
	filled := lvl.Score / 10
	if filled == 0 && lvl.Score > 0 {
		filled = 1
	}
	if filled > 10 {
		filled = 10
	}

	barW := width - len(lvl.Name) - 2
	if barW < 5 {
		barW = 5
	}
	if filled > barW {
		filled = barW
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barW-filled)
	return lipgloss.NewStyle().Foreground(color).Render(bar + " " + lvl.Name)
}

func (a App) renderStatusBar(keys string) string {
	uptime := fmt.Sprintf("▲ %s", a.uptime())
	keysW := lipgloss.Width(keys)
	uptimeW := lipgloss.Width(uptime)
	gap := a.width - keysW - uptimeW - 4
	if gap < 1 {
		return a.theme.Styles.StatusBar.Width(a.width).Render("  " + keys)
	}
	return a.theme.Styles.StatusBar.Width(a.width).Render("  " + keys + strings.Repeat(" ", gap) + uptime)
}

// renderContextualStatusBar renders a footer with only the keys valid for the current
// view and sub-mode, generated from the KeyMap (never hand-duplicated).
func (a App) renderContextualStatusBar() string {
	bindings := a.keys.keysFor(a.view, a.views.Search.Active, a.views.Terminal.Active, a.views.Browser.Active)
	h := newHelpModel(a.width-4, a.theme.Data)
	footer := h.ShortHelpView(bindings)
	return a.renderStatusBar(footer)
}
