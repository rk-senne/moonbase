package tui

import "github.com/charmbracelet/lipgloss"

// Moonbase color palette
var (
	ColorActive  = lipgloss.Color("#00FF88")
	ColorWarning = lipgloss.Color("#FFAA00")
	ColorError   = lipgloss.Color("#FF4444")
	ColorInfo    = lipgloss.Color("#00BBFF")
	ColorDim     = lipgloss.Color("#555555")
	ColorHeader  = lipgloss.Color("#FF6600")
	ColorBrand   = lipgloss.Color("#FFD700")
)

// Styles
var (
	StyleHeader = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a2e")).
			Foreground(ColorBrand).
			Bold(true).
			Padding(0, 1)

	StyleSidebar = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorDim).
			Padding(1, 1)

	StylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorInfo).
			Padding(0, 1)

	StyleActive = lipgloss.NewStyle().
			Foreground(ColorActive).
			Bold(true)

	StyleInactive = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a2e")).
			Foreground(ColorDim).
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
