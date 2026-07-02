package pipeline

import (
	"fmt"
	"strings"
)

// PipelineContext accumulates state as the pipeline executes phase by phase.
type PipelineContext struct {
	Task         string
	PhaseOutputs map[int]string   // phase number → agent output
	FilesChanged []string         // accumulated files touched across phases
	Decisions    []string         // decisions recorded during pipeline
	RiskLevel    string           // current risk assessment from QA
	ReworkCount  int              // how many times the pipeline has looped back
}

// NewPipelineContext creates a fresh context for a pipeline run.
func NewPipelineContext(task string) *PipelineContext {
	return &PipelineContext{
		Task:         task,
		PhaseOutputs: make(map[int]string),
	}
}

// RecordPhase stores the output of a completed phase.
func (pc *PipelineContext) RecordPhase(phase int, output string) {
	pc.PhaseOutputs[phase] = output

	// Extract files changed from output (look for common patterns)
	files := extractFilesChanged(output)
	for _, f := range files {
		if !contains(pc.FilesChanged, f) {
			pc.FilesChanged = append(pc.FilesChanged, f)
		}
	}
}

// ForPhase composes the input context for a given phase.
// It includes the original task plus relevant prior phase outputs.
func (pc *PipelineContext) ForPhase(phase int) string {
	var sections []string

	// Always include the original task
	sections = append(sections, fmt.Sprintf("## Original Task\n\n%s", pc.Task))

	// Include relevant prior outputs based on what the current phase needs
	switch {
	case phase == 1:
		// Analyst gets just the task (no prior context)
	case phase == 2:
		// Architect gets the requirements from phase 1
		if out, ok := pc.PhaseOutputs[1]; ok {
			sections = append(sections, fmt.Sprintf("## Requirements (from Phase 1 — Analyst)\n\n%s", out))
		}
	case phase == 3:
		// Implementer gets requirements + design
		if out, ok := pc.PhaseOutputs[1]; ok {
			sections = append(sections, fmt.Sprintf("## Requirements (from Phase 1 — Analyst)\n\n%s", summarize(out, 3000)))
		}
		if out, ok := pc.PhaseOutputs[2]; ok {
			sections = append(sections, fmt.Sprintf("## Design (from Phase 2 — Architect)\n\n%s", out))
		}
		// If this is a rework, include QA feedback
		if pc.ReworkCount > 0 {
			if out, ok := pc.PhaseOutputs[4]; ok {
				sections = append(sections, fmt.Sprintf("## QA Feedback (REWORK REQUIRED — attempt %d)\n\n%s", pc.ReworkCount, out))
			}
		}
	case phase == 4:
		// QA gets requirements + what was implemented
		if out, ok := pc.PhaseOutputs[1]; ok {
			sections = append(sections, fmt.Sprintf("## Requirements (from Phase 1 — Analyst)\n\n%s", summarize(out, 2000)))
		}
		if out, ok := pc.PhaseOutputs[3]; ok {
			sections = append(sections, fmt.Sprintf("## Implementation (from Phase 3 — Implementer)\n\n%s", out))
		}
	case phase == 5:
		// Reviewer gets a summary of everything
		if out, ok := pc.PhaseOutputs[1]; ok {
			sections = append(sections, fmt.Sprintf("## Requirements (from Phase 1)\n\n%s", summarize(out, 1500)))
		}
		if out, ok := pc.PhaseOutputs[2]; ok {
			sections = append(sections, fmt.Sprintf("## Design (from Phase 2)\n\n%s", summarize(out, 1500)))
		}
		if out, ok := pc.PhaseOutputs[3]; ok {
			sections = append(sections, fmt.Sprintf("## Implementation (from Phase 3)\n\n%s", summarize(out, 2000)))
		}
		if out, ok := pc.PhaseOutputs[4]; ok {
			sections = append(sections, fmt.Sprintf("## QA Report (from Phase 4)\n\n%s", out))
		}
	default:
		// Specialists/oversight get everything summarized
		for i := 1; i <= 5; i++ {
			if out, ok := pc.PhaseOutputs[i]; ok {
				sections = append(sections, fmt.Sprintf("## Phase %d Output\n\n%s", i, summarize(out, 1000)))
			}
		}
	}

	// Include files changed if any
	if len(pc.FilesChanged) > 0 {
		sections = append(sections, fmt.Sprintf("## Files Changed\n\n%s", strings.Join(pc.FilesChanged, "\n")))
	}

	return strings.Join(sections, "\n\n---\n\n")
}

// summarize truncates output to maxLen characters.
func summarize(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n\n...(truncated for context efficiency)"
}

// extractFilesChanged looks for file paths in agent output.
// Heuristic: lines that look like file paths (contain / and end with common extensions).
func extractFilesChanged(output string) []string {
	var files []string
	lines := strings.Split(output, "\n")

	extensions := []string{".go", ".java", ".js", ".ts", ".py", ".rs", ".md", ".yaml", ".yml", ".json", ".toml", ".xml", ".css", ".html", ".jsx", ".tsx"}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip lines that are too long (probably not file paths)
		if len(line) > 200 || len(line) < 3 {
			continue
		}
		// Strip common prefixes from diff output
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "+ ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "`")
		line = strings.TrimSuffix(line, "`")
		line = strings.TrimSpace(line)

		for _, ext := range extensions {
			if strings.HasSuffix(line, ext) && (strings.Contains(line, "/") || strings.Contains(line, "\\")) {
				// Basic sanity: no spaces in path (unless quoted)
				if !strings.Contains(line, " ") || (strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "\"")) {
					files = append(files, line)
					break
				}
			}
		}
	}

	return files
}

// contains checks if a string slice contains a value.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
