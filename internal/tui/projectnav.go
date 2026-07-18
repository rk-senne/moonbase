package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/f5508037/moonbase/internal/projects"
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

	if a.projectNav == nil || len(a.projectNav.list) == 0 {
		body := StylePanel.Width(a.width - 4).Render(
			lipgloss.NewStyle().Foreground(ColorDim).Render("\n  No projects found in ~/Workspace/Personal or ~/Workspace/Projects.\n"))
		statusBar := a.renderStatusBar("[esc] BACK")
		return lipgloss.JoinVertical(lipgloss.Left, header, "\n"+body, statusBar)
	}

	var s strings.Builder
	titleStyle := lipgloss.NewStyle().Foreground(ColorBrand).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)
	activeStyle := lipgloss.NewStyle().Foreground(ColorActive).Bold(true)

	s.WriteString(titleStyle.Render("  ◆ SELECT PROJECT") + "\n")
	s.WriteString(dimStyle.Render("  Navigate to a project to view its docs.") + "\n\n")

	typeIcons := map[string]string{
		"go":   "🔹",
		"node": "🟢",
		"java": "☕",
		"rust": "🦀",
		"git":  "📁",
	}

	for i, p := range a.projectNav.list {
		icon := typeIcons[p.Type]
		if icon == "" {
			icon = "📁"
		}
		name := p.Name
		path := dimStyle.Render(p.Path)

		if i == a.projectNav.cursor {
			s.WriteString(activeStyle.Render(fmt.Sprintf("  ▸ %s %s", icon, name)) + "\n")
			s.WriteString(fmt.Sprintf("      %s\n", path))
		} else {
			s.WriteString(dimStyle.Render(fmt.Sprintf("    %s %s", icon, name)) + "\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(dimStyle.Render(fmt.Sprintf("  %d projects found", len(a.projectNav.list))) + "\n")

	body := StylePanel.Width(a.width - 4).Height(a.height - 3).Render(s.String())
	statusBar := a.renderStatusBar("[↑↓] NAV  [enter] OPEN DOCS  [M] TMUX  [F] FISH  [esc] BACK")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

// selectProject cds into the project and opens its docs
func (a *App) selectProject() {
	if a.projectNav == nil || len(a.projectNav.list) == 0 {
		return
	}
	p := a.projectNav.list[a.projectNav.cursor]
	os.Chdir(p.Path)
	a.addIntel("Navigated to: %s", p.Name)
	// Refresh docs for this project
	a.docs = newDocsState(a.width, a.height)
	a.view = ViewDocs
}
