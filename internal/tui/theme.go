package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Theme is an immutable value representing a complete colour palette.
// Themes are never mutated after construction — cycling swaps the whole value.
type Theme struct {
	Name    string
	Active  lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color
	Brand   lipgloss.Color
	Text    lipgloss.Color
	Muted   lipgloss.Color
	Dim     lipgloss.Color
	Bg      lipgloss.Color
	Header  lipgloss.Color // header accent (often == Brand)
}

// Styles holds pre-computed lipgloss styles derived from a Theme.
// Construct via NewStyles — never modify individual fields after creation.
type Styles struct {
	Header    lipgloss.Style
	Sidebar   lipgloss.Style
	Panel     lipgloss.Style
	Modal     lipgloss.Style
	Active    lipgloss.Style
	Inactive  lipgloss.Style
	StatusBar lipgloss.Style
}

// NewStyles is a pure function: same Theme in → equal Styles out.
// If the NO_COLOR environment variable is set, returns a degraded (uncoloured) palette.
func NewStyles(t Theme) Styles {
	if os.Getenv("NO_COLOR") != "" {
		return newDegradedStyles()
	}
	return Styles{
		Header: lipgloss.NewStyle().
			Background(t.Bg).
			Foreground(t.Brand).
			Bold(true).
			Padding(0, 1),
		Sidebar: lipgloss.NewStyle().
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(t.Dim).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(0).
			PaddingRight(1),
		Panel: lipgloss.NewStyle().
			Padding(0, 1),
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.Info).
			Padding(1, 2),
		Active: lipgloss.NewStyle().
			Foreground(t.Active).
			Bold(true),
		Inactive: lipgloss.NewStyle().
			Foreground(t.Dim),
		StatusBar: lipgloss.NewStyle().
			Background(t.Bg).
			Foreground(t.Muted).
			Padding(0, 1),
	}
}

// newDegradedStyles returns styles with no colour — used when NO_COLOR is set.
func newDegradedStyles() Styles {
	return Styles{
		Header:    lipgloss.NewStyle().Bold(true).Padding(0, 1),
		Sidebar:   lipgloss.NewStyle().BorderRight(true).BorderStyle(lipgloss.NormalBorder()).PaddingTop(1).PaddingBottom(1).PaddingLeft(0).PaddingRight(1),
		Panel:     lipgloss.NewStyle().Padding(0, 1),
		Modal:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2),
		Active:    lipgloss.NewStyle().Bold(true),
		Inactive:  lipgloss.NewStyle(),
		StatusBar: lipgloss.NewStyle().Padding(0, 1),
	}
}

// themeRegistry is the authoritative list of available themes.
// It is never mutated at runtime — append here to add themes (OCP).
var themeRegistry = []Theme{
	moonbaseTheme,
	treehouseTheme,
	classifiedTheme,
	nervTheme,
}

// The four built-in themes — colours pulled verbatim from the original cycleTheme().
var moonbaseTheme = Theme{
	Name:    "moonbase",
	Active:  lipgloss.Color("#5AF78E"),
	Warning: lipgloss.Color("#F3C14B"),
	Error:   lipgloss.Color("#FF6B6B"),
	Info:    lipgloss.Color("#7EC8E3"),
	Brand:   lipgloss.Color("#FFD700"),
	Text:    lipgloss.Color("#E4E4E7"),
	Muted:   lipgloss.Color("#9CA3AF"),
	Dim:     lipgloss.Color("#6B7280"),
	Bg:      lipgloss.Color("#1a1a2e"),
	Header:  lipgloss.Color("#FFD700"),
}

var treehouseTheme = Theme{
	Name:    "treehouse",
	Active:  lipgloss.Color("#33CC33"),
	Warning: lipgloss.Color("#F3C14B"),
	Error:   lipgloss.Color("#FF6B6B"),
	Info:    lipgloss.Color("#8B4513"),
	Brand:   lipgloss.Color("#228B22"),
	Text:    lipgloss.Color("#E4E4E7"),
	Muted:   lipgloss.Color("#9CA3AF"),
	Dim:     lipgloss.Color("#6B7280"),
	Bg:      lipgloss.Color("#1a1a2e"),
	Header:  lipgloss.Color("#006400"),
}

var classifiedTheme = Theme{
	Name:    "classified",
	Active:  lipgloss.Color("#FF0000"),
	Warning: lipgloss.Color("#F3C14B"),
	Error:   lipgloss.Color("#FF6B6B"),
	Info:    lipgloss.Color("#CC0000"),
	Brand:   lipgloss.Color("#FF3333"),
	Text:    lipgloss.Color("#E4E4E7"),
	Muted:   lipgloss.Color("#9CA3AF"),
	Dim:     lipgloss.Color("#6B7280"),
	Bg:      lipgloss.Color("#1a1a2e"),
	Header:  lipgloss.Color("#990000"),
}

var nervTheme = Theme{
	Name:    "nerv",
	Active:  lipgloss.Color("#FF6600"),
	Warning: lipgloss.Color("#F3C14B"),
	Error:   lipgloss.Color("#FF6B6B"),
	Info:    lipgloss.Color("#FF3399"),
	Brand:   lipgloss.Color("#9900CC"),
	Text:    lipgloss.Color("#E4E4E7"),
	Muted:   lipgloss.Color("#9CA3AF"),
	Dim:     lipgloss.Color("#6B7280"),
	Bg:      lipgloss.Color("#1a1a2e"),
	Header:  lipgloss.Color("#FF6600"),
}

// NextTheme returns the theme after the given one in the registry (wraps around).
func NextTheme(current Theme) Theme {
	for i, t := range themeRegistry {
		if t.Name == current.Name {
			return themeRegistry[(i+1)%len(themeRegistry)]
		}
	}
	// If not found (shouldn't happen), return first theme.
	return themeRegistry[0]
}

// ThemeByName looks up a theme by name. Returns moonbaseTheme if not found.
func ThemeByName(name string) Theme {
	for _, t := range themeRegistry {
		if t.Name == name {
			return t
		}
	}
	return moonbaseTheme
}

// ThemeCount returns the number of registered themes.
func ThemeCount() int {
	return len(themeRegistry)
}
