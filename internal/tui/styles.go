package tui

import "github.com/charmbracelet/lipgloss"

// Moonbase colour system — weight-first hierarchy, colour only for status
var (
	// Status colours (used ONLY for pass/fail/warning indicators)
	ColorActive  = lipgloss.Color("#5AF78E") // green  — success, complete, selected
	ColorWarning = lipgloss.Color("#F3C14B") // amber  — in progress, caution
	ColorError   = lipgloss.Color("#FF6B6B") // red    — fail, critical, danger
	ColorInfo    = lipgloss.Color("#7EC8E3") // cyan   — interactive, links

	// Structural colours (text hierarchy via weight)
	ColorBrand = lipgloss.Color("#FFD700") // gold   — headings and brand marks only
	ColorText  = lipgloss.Color("#E4E4E7") // light  — primary text
	ColorMuted = lipgloss.Color("#9CA3AF") // gray   — secondary, labels, roles
	ColorDim   = lipgloss.Color("#6B7280") // dark   — disabled, hints, dividers
	ColorBg    = lipgloss.Color("#1a1a2e") // navy   — header/statusbar background

	// Legacy aliases (for compatibility with existing code referencing these)
	ColorHeader = ColorBrand
)

// Styles — minimal borders, weight-driven hierarchy
var (
	StyleHeader = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorBrand).
			Bold(true).
			Padding(0, 1)

	StyleSidebar = lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorDim).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(0).
			PaddingRight(1)

	StylePanel = lipgloss.NewStyle().
			Padding(0, 1) // No border — content breathes

	StyleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorInfo).
			Padding(1, 2)

	StyleActive = lipgloss.NewStyle().
			Foreground(ColorActive).
			Bold(true)

	StyleInactive = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleStatusBar = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorMuted).
			Padding(0, 1)
)

// Badges
const (
	BadgeActive   = "◉"
	BadgeInactive = "○"
	BadgePass     = "✅"
	BadgeRunning  = "🔄"
	BadgeWaiting  = "⏳"
	BadgeFail     = "❌"
)
