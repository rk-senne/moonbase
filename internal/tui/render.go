package tui

import (
	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders markdown text using glamour with auto-styling and word wrap.
// Falls back to raw text if rendering fails.
func RenderMarkdown(md string, width int) string {
	if width <= 0 {
		width = 80
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return md
	}

	rendered, err := r.Render(md)
	if err != nil {
		return md
	}

	return rendered
}
