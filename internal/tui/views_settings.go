package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rk-senne/moonbase/internal/tools"
)

// osDisplayName maps a GOOS value to a human label for the Settings sections.
func osDisplayName(goos string) string {
	switch goos {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return goos
	}
}

func (a App) renderSettings() string {
	header := a.renderHeader("Settings")

	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	okStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Active)
	warnStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Warning)
	sectionStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Info).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Text)
	curStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)

	sm := a.views.Settings
	var b strings.Builder

	b.WriteString(titleStyle.Render("  ⚙ SETTINGS") + "\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  moonbase %s", buildVersion)) + "  ")
	b.WriteString(okStyle.Render(fmt.Sprintf("Current OS: %s ✓", osDisplayName(sm.CurrentOS))) + "\n\n")

	// --- Actions: reboot ---
	b.WriteString(sectionStyle.Render("  MOONBASE") + "\n")
	rebootLabel := "⟳ Reboot & update moonbase"
	if sm.Cursor == settingsActionReboot {
		b.WriteString("  " + curStyle.Render("▸ "+rebootLabel) + "\n")
	} else {
		b.WriteString("    " + textStyle.Render(rebootLabel) + "\n")
	}
	b.WriteString(dimStyle.Render("    Pulls the latest moonbase, reinstalls it, and relaunches this TUI.") + "\n\n")

	// --- Dev environment: one section per OS ---
	mgrLabel := "none"
	if mgr, ok := tools.DetectManager(); ok {
		mgrLabel = mgr.Name
	}
	b.WriteString(sectionStyle.Render("  DEV ENVIRONMENT") + "  ")
	b.WriteString(dimStyle.Render(fmt.Sprintf("(installer: %s · confirmation required)", mgrLabel)) + "\n")

	for i, r := range sm.Rows {
		switch r.Kind {
		case rowInstallAll:
			a.renderOSSectionHeader(&b, r.OS, sm.CurrentOS)
			a.renderInstallAllRow(&b, sm, i, r.OS)
		case rowTool:
			a.renderToolRow(&b, sm, i, r)
		}
	}

	// Detail line: why the focused tool is useful (only when a tool row is selected).
	if row := sm.current(); row.Kind == rowTool && row.Tool.Why != "" {
		b.WriteString("\n  " + curStyle.Render("💡 "+row.Tool.Display) + dimStyle.Render("  "+row.Tool.Why) + "\n")
	}

	// --- Confirmation / result / hint ---
	b.WriteString("\n")
	switch {
	case sm.Confirm == "reboot":
		b.WriteString(warnStyle.Bold(true).Render("  Reboot & reinstall moonbase now? The TUI will close and reopen.") + "\n")
		b.WriteString(dimStyle.Render("  [y] yes    [n] cancel") + "\n")
	case strings.HasPrefix(sm.Confirm, "installall:"):
		goos := strings.TrimPrefix(sm.Confirm, "installall:")
		cmdStr, n := installAllPreview(goos)
		b.WriteString(warnStyle.Bold(true).Render(fmt.Sprintf("  Install all %d missing %s tools?  Runs: %s", n, osDisplayName(goos), cmdStr)) + "\n")
		b.WriteString(dimStyle.Render("  [y] yes    [n] cancel") + "\n")
	case sm.Confirm != "":
		if tool, ok := sm.toolByID(sm.Confirm); ok {
			cmdStr := installCommandPreview(tool)
			b.WriteString(warnStyle.Bold(true).Render(fmt.Sprintf("  Install %s?  Runs: %s", tool.Display, cmdStr)) + "\n")
			b.WriteString(dimStyle.Render("  [y] yes    [n] cancel") + "\n")
		}
	case sm.Result != "":
		b.WriteString(dimStyle.Render("  "+sm.Result) + "\n")
	default:
		b.WriteString(dimStyle.Render("  [↑/↓] move   [enter] select   [esc] back") + "\n")
	}

	body := a.theme.Styles.Panel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderContextualStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
}

// renderOSSectionHeader writes the "macOS"/"Linux" section header, marking the
// running OS active (✓) and the other as view-only (grayed).
func (a App) renderOSSectionHeader(b *strings.Builder, os, currentOS string) {
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	name := osDisplayName(os)
	b.WriteString("\n")
	if os == currentOS {
		active := lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)
		b.WriteString("  " + active.Render(name+" ✓") + "  " + dimStyle.Render("· this machine") + "\n")
	} else {
		b.WriteString("  " + dimStyle.Render(name+" · not this machine — view only") + "\n")
	}
}

// renderInstallAllRow writes the per-OS "Install all" action. It is actionable
// (and cursor-highlightable) only for the running OS; the other OS is grayed.
func (a App) renderInstallAllRow(b *strings.Builder, sm SettingsModel, rowIdx int, os string) {
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	if os != sm.CurrentOS {
		b.WriteString("    " + dimStyle.Render("⬇ Install all (available on this OS only)") + "\n")
		return
	}
	_, n := installAllPreview(os)
	label := fmt.Sprintf("⬇ Install all %s tools", osDisplayName(os))
	if n == 0 {
		label += " (all present ✓)"
	} else {
		label += fmt.Sprintf(" (%d missing)", n)
	}
	if sm.Cursor == rowIdx {
		cur := lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)
		b.WriteString("  " + cur.Render("▸ "+label) + "\n")
	} else {
		b.WriteString("    " + lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Render(label) + "\n")
	}
}

// renderToolRow writes a single dev-tool row. In the running-OS section it shows
// live ✓/✗ install status and is cursor-highlightable; in the other OS section
// it is grayed and shows a neutral bullet (host PATH can't reflect that OS).
func (a App) renderToolRow(b *strings.Builder, sm SettingsModel, rowIdx int, r settingsRow) {
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	okStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Active)
	warnStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Warning)
	tool := r.Tool

	if r.OS != sm.CurrentOS {
		name := fmt.Sprintf("%-12s", tool.Display)
		b.WriteString("    " + dimStyle.Render("· "+name+"  "+tool.Description) + "\n")
		return
	}

	mark := warnStyle.Render("✗")
	if tools.IsInstalled(tool.ID) {
		mark = okStyle.Render("✓")
	}
	nameStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Text)
	prefix := "  "
	if sm.Cursor == rowIdx {
		prefix = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true).Render("▸ ")
		nameStyle = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)
	}
	name := nameStyle.Render(fmt.Sprintf("%-12s", tool.Display))
	b.WriteString(fmt.Sprintf("  %s%s %s  %s\n", prefix, mark, name, dimStyle.Render(tool.Description)))
}

// installAllPreview returns the batch install command preview and the count of
// missing installable tools for the given OS.
func installAllPreview(goos string) (string, int) {
	mgr, ok := tools.DetectManager()
	if !ok {
		return "(no package manager)", 0
	}
	list := tools.ToolsForOS(goos)
	n := 0
	for _, t := range list {
		if !tools.IsInstalled(t.ID) && t.InstallableWith(mgr) {
			n++
		}
	}
	plan, _, ok := tools.InstallAllPlan(list, mgr)
	if !ok {
		return "(nothing to install)", n
	}
	return plan.Display(), n
}

// installCommandPreview returns the exact command that installing tool would run.
func installCommandPreview(tool tools.Tool) string {
	if tool.Bootstrap {
		return tools.HomebrewInstallPlan().Display()
	}
	if mgr, ok := tools.DetectManager(); ok {
		if plan, ok, _ := tools.BuildInstall(tool, mgr); ok {
			return plan.Display()
		}
	}
	return "(manual)"
}
