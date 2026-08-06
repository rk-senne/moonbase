package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func extractPersonality(prompt string) string {
	lines := strings.Split(prompt, "\\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Personality:") {
			return line
		}
	}
	lines = strings.Split(prompt, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Personality:") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// truncateToWidth truncates s so its visual width (as measured by
// lipgloss.Width, which handles emoji, CJK, and combining characters) does
// not exceed max cells. An ellipsis is appended when truncation occurs.
// Returns s unchanged if it already fits.
func truncateToWidth(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > max-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

func wordWrap(text string, width int) string {
	if width <= 0 {
		width = 60
	}
	words := strings.Fields(text)
	var lines []string
	var current string
	for _, word := range words {
		if len(current)+len(word)+1 > width {
			lines = append(lines, current)
			current = word
		} else {
			if current == "" {
				current = word
			} else {
				current += " " + word
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}
