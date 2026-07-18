package tui

import (
	"strings"
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
