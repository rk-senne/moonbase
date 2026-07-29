package tui

import (
	"fmt"
	"strings"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

// StatusIcon maps a pipeline PhaseStatus to its display emoji.
// This is a presentation concern — domain state → UI representation.
func StatusIcon(s pipeline.PhaseStatus) string {
	switch s {
	case pipeline.StatusPending:
		return "⏳"
	case pipeline.StatusRunning:
		return "🔄"
	case pipeline.StatusComplete:
		return "✅"
	case pipeline.StatusSkipped:
		return "⏭️"
	case pipeline.StatusFailed:
		return "❌"
	case pipeline.StatusRework:
		return "🔁"
	default:
		return "?"
	}
}

// PipelineStatusSummary returns a brief human-readable status of all pipeline phases.
// This is a presentation helper for displaying pipeline state in the TUI and CLI.
func PipelineStatusSummary(p *pipeline.Pipeline) string {
	var lines []string
	for _, phase := range p.Phases {
		icon := StatusIcon(phase.Status)
		conditional := ""
		if phase.Conditional {
			conditional = " (conditional)"
		}
		lines = append(lines, fmt.Sprintf("%s Phase %d: %s — %s%s",
			icon, phase.Number, phase.Name, phase.Operative, conditional))
	}
	return fmt.Sprintf("Pipeline: %s\n%s", p.Task, strings.Join(lines, "\n")+"\n")
}
