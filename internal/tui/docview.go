package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/docs"
)

// DocsState holds the document viewer state
type DocsState struct {
	files    []docs.Doc
	cursor   int
	viewport viewport.Model
	content  string
	loaded   bool
}

func NewDocsState(width, height int) *DocsState {
	files := docs.Discover()
	vp := viewport.New(width-28, height-4)

	ds := &DocsState{
		files:    files,
		viewport: vp,
	}

	// Auto-load first doc
	if len(files) > 0 {
		ds.loadDoc(0, width-30)
	}
	return ds
}

func (d *DocsState) loadDoc(idx int, width int) {
	if idx < 0 || idx >= len(d.files) {
		return
	}
	d.cursor = idx
	rendered, err := docs.Render(d.files[idx].Path, width)
	if err != nil {
		d.content = fmt.Sprintf("Error: %v", err)
	} else {
		d.content = rendered
	}
	d.viewport.SetContent(d.content)
	d.viewport.GotoTop()
	d.loaded = true
}

func (a App) renderDocs() string {
	header := a.renderHeader("Documentation")

	if a.docs == nil || len(a.docs.files) == 0 {
		body := StylePanel.Width(a.width - 4).Render(
			lipgloss.NewStyle().Foreground(ColorDim).Render("\n  No documentation found.\n\n  Place .md files in: docs/, wiki/, spec/, or project root.\n"))
		statusBar := a.renderStatusBar("[esc] BACK")
		return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
	}

	sideW := 24
	mainW := a.width - sideW - 3

	// Sidebar: file list
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Foreground(ColorBrand).Bold(true).Render("◆ DOCS") + "\n")
	s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render("──────────────────") + "\n")

	for i, f := range a.docs.files {
		name := f.Name
		if len(name) > sideW-4 {
			name = name[:sideW-4]
		}
		if i == a.docs.cursor {
			s.WriteString(lipgloss.NewStyle().Foreground(ColorActive).Bold(true).Render(fmt.Sprintf(" ▸ %s", name)) + "\n")
		} else {
			s.WriteString(lipgloss.NewStyle().Foreground(ColorDim).Render(fmt.Sprintf("   %s", name)) + "\n")
		}
	}

	sidebar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorDim).
		Padding(0, 1).
		Width(sideW).
		Height(a.height - 3).
		Render(s.String())

	// Main: rendered markdown viewport
	mainPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorInfo).
		Width(mainW).
		Height(a.height - 3).
		Render(a.docs.viewport.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", mainPanel)
	statusBar := a.renderStatusBar("[↑↓] NAV  [enter] OPEN  [pgup/pgdn] SCROLL  [esc] BACK")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}
