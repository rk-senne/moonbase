package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a App) renderDossier() string {
	sidebarWidth := 24
	bodyH := a.height - 3

	agent := a.registry.Get(a.selected)

	header := a.renderHeader("Dossier › " + agent.Designation)
	sidebar := a.renderSidebar(sidebarWidth, bodyH)

	// Right column: portrait + stats
	portraitW := 20
	mainWidth := a.width - sidebarWidth - portraitW - 2 // 2 space separators
	if mainWidth < 20 {
		mainWidth = 20
	}

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
		if mainWidth > 15 && len(cmd) > mainWidth-12 {
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
	portrait := portraitFor(agent.Name)
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
