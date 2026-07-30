package pipeline

import (
	"fmt"
	"strings"
)

// PipelineContext accumulates state as the pipeline executes phase by phase.
// Each phase's output is recorded and made available to subsequent phases as
// structured input, enabling agents to build on prior work.
type PipelineContext struct {
	Task         string         // The original mission task description
	PhaseOutputs map[int]string // Phase number → agent output text
	FilesChanged []string       // Accumulated file paths touched across all phases
	Decisions    []string       // Key decisions recorded during pipeline execution
	RiskLevel    string         // Current risk assessment from QA (LOW/MEDIUM/HIGH/CRITICAL)
	ReworkCount  int            // Number of times the pipeline has looped back for fixes
	Diff         string         // Git diff captured after Phase 3 (Enhancement 5)
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
// It includes the original task plus relevant prior phase outputs,
// driven by the declarative PhaseInputSpec for the target phase.
func (pc *PipelineContext) ForPhase(phase int) string {
	var sections []string

	// Always include the original task
	sections = append(sections, fmt.Sprintf("## Original Task\n\n%s", pc.Task))

	// Look up the spec for this phase
	spec := lookupPhaseInputSpec(phase)

	// Assemble prior-phase outputs according to the spec
	for _, reqPhase := range spec.RequiresPhases {
		out, ok := pc.PhaseOutputs[reqPhase]
		if !ok {
			continue
		}

		// Apply truncation if specified
		if maxLen, hasLimit := spec.MaxPerPhase[reqPhase]; hasLimit && maxLen > 0 {
			out = summarize(out, maxLen)
		}

		// Use the configured header
		header := spec.HeaderFormat[reqPhase]
		sections = append(sections, fmt.Sprintf("%s\n\n%s", header, out))
	}

	// Include rework feedback from QA (Phase 4) when spec declares it
	if spec.IncludeRework && pc.ReworkCount > 0 {
		if out, ok := pc.PhaseOutputs[4]; ok {
			sections = append(sections, fmt.Sprintf("## QA Feedback (REWORK REQUIRED — attempt %d)\n\n%s", pc.ReworkCount, out))
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
