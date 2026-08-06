package tui

import (
	"fmt"
	"runtime"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/rk-senne/moonbase/internal/tools"
)

func (a App) renderTools() string {
	header := a.renderHeader("Tools")

	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	okStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Active)
	warnStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Warning)

	var b strings.Builder
	b.WriteString(titleStyle.Render("  ◆ TOOLS ARSENAL") + "\n")

	// Detected package manager line — transparency about what will run.
	mgrLabel := "none detected"
	if mgr, ok := tools.DetectManager(); ok {
		mgrLabel = mgr.Name
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("  Package manager: %s · %s · moonbase asks before installing.", mgrLabel, runtime.GOOS)) + "\n\n")

	tm := a.views.Tools
	renderRow := func(idx int, tool tools.Tool) {
		installed := tools.IsInstalled(tool.ID)
		mark := warnStyle.Render("✗")
		if installed {
			mark = okStyle.Render("✓")
		}

		cursor := "  "
		nameStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Text)
		if idx == tm.Cursor {
			cursor = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true).Render("▸ ")
			nameStyle = lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)
		}

		name := nameStyle.Render(fmt.Sprintf("%-12s", tool.Display))
		desc := dimStyle.Render(tool.Description)
		b.WriteString(fmt.Sprintf("  %s%s %s  %s\n", cursor, mark, name, desc))
	}

	// Critical section, then cool section, preserving catalog order + indices.
	b.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Info).Bold(true).Render("  CRITICAL") + "\n")
	for i, tool := range tm.Catalog {
		if tool.Category == tools.Critical {
			renderRow(i, tool)
		}
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(a.theme.Data.Info).Bold(true).Render("  COOL & STABLE") + "\n")
	for i, tool := range tm.Catalog {
		if tool.Category == tools.Cool {
			renderRow(i, tool)
		}
	}

	// Confirmation prompt (shows the exact command that will run).
	b.WriteString("\n")
	if tm.Confirm != "" {
		if tool, ok := tm.toolByID(tm.Confirm); ok {
			cmdStr := "(manual)"
			if mgr, ok := tools.DetectManager(); ok {
				if plan, ok, _ := tools.BuildInstall(tool, mgr); ok {
					cmdStr = plan.Display()
				}
			}
			prompt := fmt.Sprintf("  Install %s?  Runs: %s", tool.Display, cmdStr)
			b.WriteString(warnStyle.Bold(true).Render(prompt) + "\n")
			b.WriteString(dimStyle.Render("  [y] yes    [n] cancel") + "\n")
		}
	} else if tm.Result != "" {
		b.WriteString(dimStyle.Render("  "+tm.Result) + "\n")
	} else {
		b.WriteString(dimStyle.Render("  [↑/↓] move   [enter] install selected   [esc] back") + "\n")
	}

	body := a.theme.Styles.Panel.Width(a.width - 4).Render(b.String())
	statusBar := a.renderContextualStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
}
