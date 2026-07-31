package pipeline

import (
	"fmt"
	"strings"
)

// Depth represents the pipeline execution depth.
type Depth string

const (
	DepthTrivial Depth = "trivial"
	DepthSimple  Depth = "simple"
	DepthComplex Depth = "complex"
)

// DepthClassification holds the result of task complexity analysis.
type DepthClassification struct {
	Depth  Depth  // The classified depth
	Reason string // Human-readable explanation for flywheel/CLI output
}

// ClassifyTask analyzes a task description and returns the recommended
// pipeline depth. Reuses the reasoning-protocol task-scaling ladder:
//   - trivial: fix directly, verify builds (short, no complexity signals)
//   - simple:  read context → implement → test (moderate, some signals)
//   - complex: full protocol (long, multiple signals, multi-scope)
//
// Ambiguity resolves to simple — never under-estimates to trivial.
func ClassifyTask(task string) DepthClassification {
	signals := countComplexitySignals(task)
	length := len(task)
	paths := countFilePaths(task)

	// Empty or whitespace-only: default to simple (can't classify nothing)
	if len(strings.TrimSpace(task)) == 0 {
		return DepthClassification{
			Depth:  DepthSimple,
			Reason: "default classification",
		}
	}

	// Complex: long tasks, many signals, multi-scope
	if length > 200 || signals >= 3 || paths >= 3 {
		return DepthClassification{
			Depth:  DepthComplex,
			Reason: complexReason(length, signals, paths),
		}
	}

	// Trivial: short tasks with zero complexity signals AND has trivial indicators
	if length <= 80 && signals == 0 && paths <= 1 && hasTrivialIndicator(task) {
		return DepthClassification{
			Depth:  DepthTrivial,
			Reason: "short task, no complexity signals",
		}
	}

	// Default: simple
	return DepthClassification{
		Depth:  DepthSimple,
		Reason: simpleReason(length, signals, paths),
	}
}

// complexityKeywords are words/phrases that indicate non-trivial work.
// Drawn from the reasoning-protocol task-scaling vocabulary.
var complexityKeywords = []string{
	// Feature addition
	"implement", "add", "create", "build", "introduce", "new endpoint",
	// Structural change
	"refactor", "redesign", "migrate", "restructure", "architecture",
	// System concerns
	"rate limit", "pagination", "authentication", "authorization",
	"caching", "concurrency", "performance", "quota",
	// Security
	"security", "permission",
	// Multi-step
	"and then", "followed by", "across all", "every",
}

// trivialKeywords are words that suggest minimal scope.
// Their presence cancels complexity signals but cannot make a task trivial on their own.
var trivialKeywords = []string{
	"fix typo", "rename", "remove unused", "update comment",
	"fix import", "formatting", "whitespace", "spelling",
}

// trivialIndicators are phrases that positively identify a task as trivial.
// A task is only classified as trivial if it contains at least one of these.
var trivialIndicators = []string{
	"fix typo", "rename", "remove unused", "update comment",
	"fix import", "formatting", "whitespace", "spelling",
	"fix spelling", "typo", "unused import",
}

// countComplexitySignals counts complexity indicators in a task description.
// Each complexity keyword match adds +1; each trivial keyword match subtracts 1.
// The result is floored at 0.
func countComplexitySignals(task string) int {
	lower := strings.ToLower(task)
	signals := 0

	for _, kw := range complexityKeywords {
		if strings.Contains(lower, kw) {
			signals++
		}
	}

	for _, kw := range trivialKeywords {
		if strings.Contains(lower, kw) {
			signals--
		}
	}

	if signals < 0 {
		signals = 0
	}
	return signals
}

// hasTrivialIndicator returns true if the task contains a positive trivial signal.
// Without a trivial indicator, short tasks default to simple (ambiguity → simple).
func hasTrivialIndicator(task string) bool {
	lower := strings.ToLower(task)
	for _, indicator := range trivialIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}

// countFilePaths counts distinct file paths referenced in the task description.
// Uses word-level scanning to detect paths embedded in prose (e.g.,
// "update internal/foo/bar.go and src/baz.ts").
func countFilePaths(task string) int {
	extensions := []string{".go", ".java", ".js", ".ts", ".py", ".rs", ".md", ".yaml", ".yml", ".json", ".toml", ".xml", ".css", ".html", ".jsx", ".tsx"}
	words := strings.Fields(task)
	count := 0
	seen := make(map[string]bool)

	for _, word := range words {
		// Strip common punctuation
		word = strings.Trim(word, "`,\"'()[]{}:")
		if len(word) < 4 || !strings.Contains(word, "/") {
			continue
		}
		for _, ext := range extensions {
			if strings.HasSuffix(word, ext) && !seen[word] {
				seen[word] = true
				count++
				break
			}
		}
	}
	return count
}

// complexReason builds a human-readable reason for complex classification.
func complexReason(length, signals, paths int) string {
	var parts []string
	if length > 200 {
		parts = append(parts, fmt.Sprintf("task length %d chars", length))
	}
	if signals >= 3 {
		parts = append(parts, fmt.Sprintf("%d complexity signals", signals))
	}
	if paths >= 3 {
		parts = append(parts, fmt.Sprintf("%d file paths", paths))
	}
	if len(parts) == 0 {
		return "complex task detected"
	}
	return strings.Join(parts, ", ")
}

// simpleReason builds a human-readable reason for simple classification.
func simpleReason(length, signals, paths int) string {
	var parts []string
	if signals > 0 {
		parts = append(parts, fmt.Sprintf("%d complexity signal(s)", signals))
	}
	if length > 80 {
		parts = append(parts, "moderate length")
	}
	if paths > 1 {
		parts = append(parts, fmt.Sprintf("%d file paths", paths))
	}
	if len(parts) == 0 {
		return "default classification"
	}
	return strings.Join(parts, ", ")
}

// NewAdaptive creates a pipeline with phases configured for the given depth.
// Phases not included in the depth profile are pre-skipped.
//
// Phase layout by depth:
//   - trivial: Phase 3 (Implementation) + Phase 4 (QA) only
//   - simple:  Phase 1 (Analysis) + Phase 3 + Phase 4
//   - complex: All 5 mandatory phases (same as New)
//
// Conditional phases (6, 7, 8) remain pending regardless of depth.
// Phase 4 (QA) is NEVER skipped at any depth — this is the core invariant.
func NewAdaptive(task string, depth Depth, reason string) *Pipeline {
	p := New(task)
	p.Depth = depth
	p.DepthReason = reason

	switch depth {
	case DepthTrivial:
		// Only Phase 3 + 4
		for i := range p.Phases {
			n := p.Phases[i].Number
			if n == 1 || n == 2 || n == 5 {
				p.Phases[i].Status = StatusSkipped
			}
		}
	case DepthSimple:
		// Phase 1 + 3 + 4 (skip Architecture and Review initially)
		for i := range p.Phases {
			n := p.Phases[i].Number
			if n == 2 || n == 5 {
				p.Phases[i].Status = StatusSkipped
			}
		}
	case DepthComplex:
		// All phases active — same as New(), no changes needed
	}

	return p
}

// Escalate promotes the pipeline to a deeper depth. It un-skips phases
// that the new depth requires and marks the pipeline as escalated.
// Returns an error if the pipeline is already at the target depth or deeper.
func (p *Pipeline) Escalate(targetDepth Depth) error {
	if p.Depth == targetDepth || p.Depth == DepthComplex {
		return fmt.Errorf("already at depth %s, cannot escalate to %s", p.Depth, targetDepth)
	}

	p.OrigDepth = p.Depth
	p.Depth = targetDepth
	p.Escalated = true

	switch targetDepth {
	case DepthSimple:
		// Un-skip Phase 1 (Analysis)
		p.unskipPhase(1)
		// Un-skip Phase 5 (Review) — escalated pipelines get full review
		p.unskipPhase(5)
	case DepthComplex:
		// Un-skip Phases 1, 2, and 5 (Analysis + Architecture + Review)
		p.unskipPhase(1)
		p.unskipPhase(2)
		p.unskipPhase(5)
	}

	return nil
}

// unskipPhase resets a skipped phase to pending so it can be executed.
func (p *Pipeline) unskipPhase(number int) {
	for i := range p.Phases {
		if p.Phases[i].Number == number && p.Phases[i].Status == StatusSkipped {
			p.Phases[i].Status = StatusPending
		}
	}
}

// UnskipPhase resets a skipped phase to pending so it can be executed.
// Exported for use by the mission loop after escalation.
func (p *Pipeline) UnskipPhase(number int) {
	p.unskipPhase(number)
}

// EscalationTarget determines the depth to escalate to based on current
// depth and risk level. CRITICAL is never handled here — it stops the pipeline.
func EscalationTarget(current Depth, risk RiskLevel) Depth {
	return escalationTarget(current, risk)
}

// escalationTarget determines the depth to escalate to based on current
// depth and risk level.
func escalationTarget(current Depth, risk RiskLevel) Depth {
	switch {
	case risk == RiskHigh || risk == RiskCritical:
		return DepthComplex
	case risk == RiskMedium && current == DepthTrivial:
		return DepthSimple
	case risk == RiskMedium && current == DepthSimple:
		return DepthComplex
	default:
		return current
	}
}
