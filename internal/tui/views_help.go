package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// helpCategoryNames maps each FullHelp group index to its category label.
var helpCategoryNames = []string{
	"NAVIGATION",
	"MISSIONS",
	"VIEWS",
	"TOOLS",
	"COMMS",
	"SYSTEM",
}

// renderHelpCategory renders a single category of key bindings as a formatted block.
func renderHelpCategory(name string, bindings []key.Binding, keyStyle, descStyle lipgloss.Style) string {
	header := fmt.Sprintf("  ◆ %s", name)
	divider := "  " + strings.Repeat("─", 21)

	var lines []string
	lines = append(lines, header)
	lines = append(lines, divider)

	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		h := b.Help()
		line := fmt.Sprintf("  %-12s %s", keyStyle.Render(h.Key), descStyle.Render(h.Desc))
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (a App) renderHelp() string {
	header := a.renderHeader("Operations Manual")

	groups := a.keys.FullHelp()

	keyStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Text).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Info)

	// Render categories in pairs (left + right) to match the two-column layout
	var sections []string
	for i := 0; i < len(groups); i += 2 {
		leftName := "KEYS"
		if i < len(helpCategoryNames) {
			leftName = helpCategoryNames[i]
		}
		left := renderHelpCategory(leftName, groups[i], keyStyle, descStyle)

		if i+1 < len(groups) {
			rightName := "KEYS"
			if i+1 < len(helpCategoryNames) {
				rightName = helpCategoryNames[i+1]
			}
			right := renderHelpCategory(rightName, groups[i+1], keyStyle, descStyle)
			// Join left and right columns horizontally
			pair := lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Width(38).Render(left),
				lipgloss.NewStyle().Width(38).Render(right),
			)
			sections = append(sections, pair)
		} else {
			sections = append(sections, left)
		}
	}

	keyTable := strings.Join(sections, "\n\n")

	// KND-flavoured brand prose (kept as brand, not key data)
	brandFooter := lipgloss.NewStyle().Foreground(a.theme.Data.Info).Render(`
  ◆ THE KND WAY
  ──────────────────────
  Sector V = core pipeline.  Specialists = cross-cutting.
  Council = full lifecycle.  "We fight for kids everywhere."`)

	body := keyTable + "\n" + brandFooter

	statusBar := a.renderContextualStatusBar()
	return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body+"\n", statusBar)
}

// newHelpModel creates a help.Model configured for the TUI footer rendering.
func newHelpModel(width int, t Theme) help.Model {
	h := help.New()
	h.Width = width
	h.ShortSeparator = " • "
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.Text).Bold(true)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.Muted)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.Dim)
	return h
}
