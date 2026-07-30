package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rk-senne/moonbase/internal/projects"
)

type ProjectsState struct {
	list   []projects.Project
	cursor int
}

func newProjectsState() *ProjectsState {
	return &ProjectsState{
		list: projects.Discover(),
	}
}

func (a App) renderProjects() string {
	header := a.renderHeader("Projects")

	if a.views.ProjectNav == nil || len(a.views.ProjectNav.list) == 0 {
		body := a.theme.Styles.Panel.Width(a.width - 4).Render(
			lipgloss.NewStyle().Foreground(a.theme.Data.Dim).Render("\n  No projects found in ~/Workspace/Personal or ~/Workspace/Projects.\n"))
		statusBar := a.renderContextualStatusBar()
		return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
	}

	var s strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Brand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Dim)
	activeStyle := lipgloss.NewStyle().Foreground(a.theme.Data.Active).Bold(true)

	s.WriteString(titleStyle.Render("  ◆ SELECT PROJECT") + "\n")
	s.WriteString(dimStyle.Render("  Navigate to a project to view its docs.") + "\n\n")

	typeIcons := map[string]string{
		"go":   "🔹",
		"node": "🟢",
		"java": "☕",
		"rust": "🦀",
		"git":  "📁",
	}

	for i, p := range a.views.ProjectNav.list {
		icon := typeIcons[p.Type]
		if icon == "" {
			icon = "📁"
		}
		name := p.Name
		path := dimStyle.Render(p.Path)

		if i == a.views.ProjectNav.cursor {
			s.WriteString(activeStyle.Render(fmt.Sprintf("  ▸ %s %s", icon, name)) + "\n")
			s.WriteString(fmt.Sprintf("      %s\n", path))
		} else {
			s.WriteString(dimStyle.Render(fmt.Sprintf("    %s %s", icon, name)) + "\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(dimStyle.Render(fmt.Sprintf("  %d projects found", len(a.views.ProjectNav.list))) + "\n")

	body := a.theme.Styles.Panel.Width(a.width - 4).Height(a.height - 3).Render(s.String())
	statusBar := a.renderContextualStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

// selectProject cds into the project and opens its docs
func (a *App) selectProject() {
	if a.views.ProjectNav == nil || len(a.views.ProjectNav.list) == 0 {
		return
	}
	p := a.views.ProjectNav.list[a.views.ProjectNav.cursor]
	os.Chdir(p.Path)
	a.addIntel("Navigated to: %s", p.Name)
	// Refresh docs for this project
	a.views.Docs = newDocsState(a.width, a.height)
	a.view = ViewDocs
}
