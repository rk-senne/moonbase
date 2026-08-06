package tui

import (
	"fmt"
	"runtime"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rk-senne/moonbase/internal/tools"
)

func (a App) renderSettings() string {
	header := a.renderHeader("Settings")

	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	okStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Active)
	warnStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Warning)
	sectionStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Info).Bold(true)

	sm := a.views.Settings
	var b strings.Builder

	b.WriteString(titleStyle.Render("  ⚙ SETTINGS") + "\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("  moonbase %s · %s", buildVersion, runtime.GOOS)) + "\n\n")

	// --- Actions: reboot ---
	b.WriteString(sectionStyle.Render("  MOONBASE") + "\n")
	rebootLabel := "⟳ Reboot & update moonbase"
	if sm.Cursor == settingsActionReboot {
		cur := lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)
		b.WriteString("  " + cur.Render("▸ "+rebootLabel) + "\n")
	} else {
		b.WriteString("    " + lipgloss.NewStyle().Foreground(a.theme.Data.Text).Render(rebootLabel) + "\n")
	}
	b.WriteString(dimStyle.Render("    Pulls the latest moonbase, reinstalls it, and relaunches this TUI.") + "\n\n")

	// --- Dev environment catalog ---
	mgrLabel := "none"
	if mgr, ok := tools.DetectManager(); ok {
		mgrLabel = mgr.Name
	}
	b.WriteString(sectionStyle.Render("  DEV ENVIRONMENT") + "  ")
	b.WriteString(dimStyle.Render(fmt.Sprintf("(installer: %s · confirmation required)", mgrLabel)) + "\n")

	renderRow := func(catalogIdx int, tool tools.Tool) {
		row := catalogIdx + 1 // cursor offset (0 = reboot)
		installed := tools.IsInstalled(tool.ID)
		mark := warnStyle.Render("✗")
		if installed {
			mark = okStyle.Render("✓")
		}
		nameStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Text)
		prefix := "  "
		if sm.Cursor == row {
			prefix = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true).Render("▸ ")
			nameStyle = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)
		}
		name := nameStyle.Render(fmt.Sprintf("%-12s", tool.Display))
		b.WriteString(fmt.Sprintf("  %s%s %s  %s\n", prefix, mark, name, dimStyle.Render(tool.Description)))
	}

	// Group by category in catalog order: Runtime, Critical, Cool.
	for _, cat := range []tools.Category{tools.Runtime, tools.Critical, tools.Cool} {
		first := true
		for i, tool := range sm.Catalog {
			if tool.Category != cat {
				continue
			}
			if first {
				b.WriteString(dimStyle.Render("  "+strings.ToUpper(cat.String())) + "\n")
				first = false
			}
			renderRow(i, tool)
		}
	}

	// --- Confirmation / result / hint ---
	b.WriteString("\n")
	switch {
	case sm.Confirm == "reboot":
		b.WriteString(warnStyle.Bold(true).Render("  Reboot & reinstall moonbase now? The TUI will close and reopen.") + "\n")
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
