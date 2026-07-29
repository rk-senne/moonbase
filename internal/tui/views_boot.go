package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a App) renderBoot() string {
	var b strings.Builder

	// Data cascade effect for first 3 frames
	if a.bootStep < 3 {
		cascadeH := a.height / 3
		cascade := generateCascade(a.width, cascadeH, a.bootStep)
		cascadeStyled := lipgloss.NewStyle().Foreground(a.themeData.Active).Render(cascade)
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

	logo := lipgloss.NewStyle().Foreground(a.themeData.Brand).Bold(true).Render(kndLogo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, logo))
	b.WriteString("\n")

	subtitle := lipgloss.NewStyle().Foreground(a.themeData.Info).Render(moonbaseLogo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, subtitle))
	b.WriteString("\n\n")

	for i := 0; i < a.bootStep && i < len(bootMessages); i++ {
		style := lipgloss.NewStyle().Foreground(a.themeData.Active)
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
			b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, lipgloss.NewStyle().Foreground(a.themeData.Dim).Render(line)))
		}
		b.WriteString("\n")
	}

	return b.String()
}
