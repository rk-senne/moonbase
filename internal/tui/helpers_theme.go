package tui

import "github.com/charmbracelet/lipgloss"

func (a *App) cycleTheme() {
	switch a.theme {
	case "moonbase":
		a.theme = "treehouse"
		ColorActive = lipgloss.Color("#33CC33")
		ColorInfo = lipgloss.Color("#8B4513")
		ColorBrand = lipgloss.Color("#228B22")
		ColorHeader = lipgloss.Color("#006400")
	case "treehouse":
		a.theme = "classified"
		ColorActive = lipgloss.Color("#FF0000")
		ColorInfo = lipgloss.Color("#CC0000")
		ColorBrand = lipgloss.Color("#FF3333")
		ColorHeader = lipgloss.Color("#990000")
	case "classified":
		a.theme = "nerv"
		ColorActive = lipgloss.Color("#FF6600")
		ColorInfo = lipgloss.Color("#FF3399")
		ColorBrand = lipgloss.Color("#9900CC")
		ColorHeader = lipgloss.Color("#FF6600")
	default:
		a.theme = "moonbase"
		ColorActive = lipgloss.Color("#00FF88")
		ColorInfo = lipgloss.Color("#00BBFF")
		ColorBrand = lipgloss.Color("#FFD700")
		ColorHeader = lipgloss.Color("#FF6600")
	}
}
