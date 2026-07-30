package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/rk-senne/moonbase/internal/docs"
)

// DocsState holds the document viewer state
type DocsState struct {
	files    []docs.Doc
	cursor   int
	viewport viewport.Model
	content  string
	loaded   bool
}

func newDocsState(width, height int) *DocsState {
	files := docs.Discover()
	vp := viewport.New(viewport.WithWidth(width-28), viewport.WithHeight(height-4))

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

	if a.views.Docs == nil || len(a.views.Docs.files) == 0 {
		body := a.theme.Styles.Panel.Width(a.width - 4).Render(
			lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("\n  No documentation found.\n\n  Place .md files in: docs/, wiki/, spec/, or project root.\n"))
		statusBar := a.renderContextualStatusBar()
		return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
	}

	sideW := 24
	mainW := a.width - sideW - 1 // 1 space separator

	// Sidebar: file list
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true).Render("◆ DOCS") + "\n")
	s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("──────────────────") + "\n")

	for i, f := range a.views.Docs.files {
		name := f.Name
		if len(name) > sideW-4 {
			name = name[:sideW-4]
		}
		if i == a.views.Docs.cursor {
			s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true).Render(fmt.Sprintf(" ▸ %s", name)) + "\n")
		} else {
			s.WriteString(lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render(fmt.Sprintf("   %s", name)) + "\n")
		}
	}

	sidebar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(a.theme.Data.Dim).
		Padding(0, 1).
		Width(sideW).
		Height(a.height - 3).
		Render(s.String())

	// Main: rendered markdown viewport
	mainPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Data.Info).
		Width(mainW).
		Height(a.height - 3).
		Render(a.views.Docs.viewport.View())

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", mainPanel)
	statusBar := a.renderContextualStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}
