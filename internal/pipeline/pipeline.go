// Package pipeline implements the KND Council's multi-phase execution engine.
// It orchestrates agents through mandatory and conditional phases, applies risk
// gates after QA, and manages rework loops when quality thresholds are not met.
package pipeline

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// PhaseStatus represents the current state of a pipeline phase.
type PhaseStatus int

const (
	StatusPending PhaseStatus = iota
	StatusRunning
	StatusComplete
	StatusSkipped
	StatusFailed
	StatusRework
)

// Phase represents a single phase in the KND Council pipeline.
type Phase struct {
	Number      int         // Phase number (1-8), determines execution order
	Name        string      // Human-readable phase name (e.g., "Analysis", "QA")
	Operative   string      // Display name of the operative (e.g., "Numbuh 1")
	AgentName   string      // Agent file name without extension (e.g., "numbuh-1")
	Status      PhaseStatus // Current execution status
	Duration    string      // Execution time (set after completion)
	Summary     string      // Brief outcome or error message
	Conditional bool        // If true, phase only runs when trigger conditions are met
	TriggerSpec string      // Trigger conditions for conditional phases (from agent frontmatter)
	StartedAt   time.Time   // When the phase began execution
	CompletedAt time.Time   // When the phase finished execution
}

// StartPhase marks the phase as running and records the start time.
func (ph *Phase) StartPhase() {
	ph.Status = StatusRunning
	ph.StartedAt = time.Now()
	ph.CompletedAt = time.Time{}
}

// CompletePhase marks the phase as complete and records duration.
func (ph *Phase) CompletePhase() {
	ph.CompletedAt = time.Now()
	ph.Status = StatusComplete
	ph.Duration = ph.CompletedAt.Sub(ph.StartedAt).Round(time.Millisecond).String()
}

// ElapsedTime returns the elapsed time since the phase started.
// Returns zero duration if the phase hasn't started yet.
func (ph *Phase) ElapsedTime() time.Duration {
	if ph.StartedAt.IsZero() {
		return 0
	}
	if !ph.CompletedAt.IsZero() {
		return ph.CompletedAt.Sub(ph.StartedAt)
	}
	return time.Since(ph.StartedAt)
}

// IsTimedOut returns true if the phase has been running longer than the given timeout.
func (ph *Phase) IsTimedOut(timeout time.Duration) bool {
	if ph.Status != StatusRunning || ph.StartedAt.IsZero() {
		return false
	}
	return time.Since(ph.StartedAt) > timeout
}

// Pipeline manages the full council execution flow.
type Pipeline struct {
	Task          string           // The original task/mission description
	Phases        []Phase          // All phases (mandatory + conditional)
	Current       int              // Index of the currently active phase
	Active        bool             // True while the pipeline is still executing
	Context       *PipelineContext // Accumulated state across phases
	MaxRework     int              // Maximum rework loops before escalation (default 2)
	TraceID       string           // Unique identifier for observability and checkpointing
	PhaseTimeout  time.Duration    // Maximum allowed duration per phase (default 5 minutes)
	MaxOutputSize int              // Maximum allowed output size in bytes (default 100KB)
	MaxRetries    int              // Maximum retries per phase (default 1)
	Retries       map[int]int      // Phase number → retry count
}

// New creates a new pipeline for a given task.
func New(task string) *Pipeline {
	return &Pipeline{
		Task:          task,
		Active:        true,
		MaxRework:     2,
		TraceID:       generateTraceID(),
		PhaseTimeout:  5 * time.Minute,
		MaxOutputSize: 100000,
		MaxRetries:    1,
		Retries:       make(map[int]int),
		Context:       NewPipelineContext(task),
		Phases: []Phase{
			{Number: 1, Name: "Analysis", Operative: "Numbuh 1", AgentName: "numbuh-1", Status: StatusPending},
			{Number: 2, Name: "Architecture", Operative: "Numbuh 2", AgentName: "numbuh-2", Status: StatusPending},
			{Number: 3, Name: "Implementation", Operative: "Numbuh 3", AgentName: "numbuh-3", Status: StatusPending},
			{Number: 4, Name: "QA", Operative: "Numbuh 4", AgentName: "numbuh-4", Status: StatusPending},
			{Number: 5, Name: "Review", Operative: "Numbuh 5", AgentName: "numbuh-5", Status: StatusPending},
			{Number: 6, Name: "Oversight", Operative: "Numbuh 0", AgentName: "numbuh-0", Status: StatusPending, Conditional: true, TriggerSpec: ">5 files changed, core logic changed, orchestration/pipeline changed"},
			{Number: 7, Name: "Security", Operative: "Numbuh 274", AgentName: "numbuh-274", Status: StatusPending, Conditional: true, TriggerSpec: "Auth/secrets/permissions changed, new endpoints, dependency CVEs"},
			{Number: 8, Name: "Deploy Prep", Operative: "Numbuh 362", AgentName: "numbuh-362", Status: StatusPending, Conditional: true, TriggerSpec: "CI/CD changed, Docker/infra touched, new env vars, deployment config"},
		},
	}
}

// NewFast creates a pipeline with only implementation (phase 3) and QA (phase 4) active.
// All other phases are pre-skipped. Used for trivial/well-specified tasks.
func NewFast(task string) *Pipeline {
	p := New(task)
	for i := range p.Phases {
		if p.Phases[i].Number != 3 && p.Phases[i].Number != 4 {
			p.Phases[i].Status = StatusSkipped
		}
	}
	return p
}

// generateTraceID creates a unique trace identifier using timestamp + random suffix.
func generateTraceID() string {
	ts := time.Now().UTC().Format("20060102T150405")
	suffix := make([]byte, 4)
	rand.Read(suffix)
	return fmt.Sprintf("%s-%s", ts, hex.EncodeToString(suffix))
}

// ValidateOutput checks if the output size is within the allowed limit.
// Returns an error if the output exceeds MaxOutputSize bytes.
func (p *Pipeline) ValidateOutput(output string) error {
	if len(output) > p.MaxOutputSize {
		return fmt.Errorf("output size %d bytes exceeds maximum %d bytes", len(output), p.MaxOutputSize)
	}
	return nil
}

// RetryPhase increments the retry count for the current phase and re-runs it.
// Returns an error if the maximum retry count has been exceeded.
func (p *Pipeline) RetryPhase() error {
	phase := p.CurrentPhase()
	if phase == nil {
		return fmt.Errorf("no active phase to retry")
	}
	p.Retries[phase.Number]++
	if p.Retries[phase.Number] > p.MaxRetries {
		return fmt.Errorf("max retries (%d) exceeded for phase %d (%s)", p.MaxRetries, phase.Number, phase.Name)
	}
	phase.StartPhase()
	return nil
}

// CurrentPhase returns the current phase.
func (p *Pipeline) CurrentPhase() *Phase {
	if p.Current >= 0 && p.Current < len(p.Phases) {
		return &p.Phases[p.Current]
	}
	return nil
}

// Advance moves to the next phase after completing the current one.
func (p *Pipeline) Advance() {
	if p.Current < len(p.Phases)-1 {
		p.Phases[p.Current].Status = StatusComplete
		p.Current++
		p.Phases[p.Current].Status = StatusRunning
	} else {
		p.Phases[p.Current].Status = StatusComplete
		p.Active = false
	}
}

// Retry re-runs the current phase.
func (p *Pipeline) Retry() {
	p.Phases[p.Current].Status = StatusRunning
}

// Skip marks the current phase as skipped and advances.
func (p *Pipeline) Skip() {
	p.Phases[p.Current].Status = StatusSkipped
	if p.Current < len(p.Phases)-1 {
		p.Current++
		p.Phases[p.Current].Status = StatusRunning
	} else {
		p.Active = false
	}
}

// RouteToPhase sends the pipeline back to a specific phase (for rework loops).
// Returns an error if max rework count is exceeded.
func (p *Pipeline) RouteToPhase(targetPhase int) error {
	p.Context.ReworkCount++
	if p.Context.ReworkCount > p.MaxRework {
		return fmt.Errorf("max rework loops (%d) exceeded — escalating to human for review", p.MaxRework)
	}

	// Mark current phase as rework
	p.Phases[p.Current].Status = StatusRework

	// Find target phase index
	for i, phase := range p.Phases {
		if phase.Number == targetPhase {
			p.Current = i
			p.Phases[i].Status = StatusRunning
			return nil
		}
	}

	return fmt.Errorf("target phase %d not found in pipeline — check pipeline phase configuration", targetPhase)
}

// Stop halts the pipeline (for CRITICAL risk).
func (p *Pipeline) Stop(reason string) {
	if p.Current >= 0 && p.Current < len(p.Phases) {
		p.Phases[p.Current].Status = StatusFailed
		p.Phases[p.Current].Summary = reason
	}
	p.Active = false
}

// ApplyRiskGate applies the QA risk assessment and routes accordingly.
// Returns the routing decision and any error (e.g., max rework exceeded).
func (p *Pipeline) ApplyRiskGate(qaOutput string) (RiskRouting, error) {
	routing := ParseRiskGate(qaOutput)
	p.Context.RiskLevel = string(routing.Level)

	switch routing.Level {
	case RiskLow:
		// Proceed to review
		return routing, nil
	case RiskMedium:
		// Route back to implementation
		err := p.RouteToPhase(routing.TargetPhase)
		return routing, err
	case RiskHigh:
		// Route back to design
		err := p.RouteToPhase(routing.TargetPhase)
		return routing, err
	case RiskCritical:
		// Stop the pipeline
		p.Stop("CRITICAL risk — pipeline stopped")
		return routing, nil
	default:
		// Unknown risk — treat as medium
		err := p.RouteToPhase(3)
		return routing, err
	}
}

// ShouldInvokeConditional checks if a conditional phase should run.
func (p *Pipeline) ShouldInvokeConditional(phase *Phase) TriggerResult {
	if !phase.Conditional {
		return TriggerResult{Invoke: true, Reason: "mandatory phase"}
	}
	return EvaluateTrigger(phase.TriggerSpec, p.Context)
}

// IsComplete returns true when all phases are done.
func (p *Pipeline) IsComplete() bool {
	return !p.Active
}

// StatusSummary returns a brief status of all phases.
func (p *Pipeline) StatusSummary() string {
	var lines []string
	for _, phase := range p.Phases {
		icon := statusIcon(phase.Status)
		conditional := ""
		if phase.Conditional {
			conditional = " (conditional)"
		}
		lines = append(lines, fmt.Sprintf("%s Phase %d: %s — %s%s",
			icon, phase.Number, phase.Name, phase.Operative, conditional))
	}
	return fmt.Sprintf("Pipeline: %s\n%s", p.Task, joinLines(lines))
}

// statusIcon maps a PhaseStatus to a display emoji.
// NOTE: This is a PRESENTATION concern (maps domain state to UI representation).
// If the pipeline package needs to become framework-independent, move this function
// and StatusSummary to the TUI layer (internal/tui) or a dedicated "view" package.
func statusIcon(s PhaseStatus) string {
	switch s {
	case StatusPending:
		return "⏳"
	case StatusRunning:
		return "🔄"
	case StatusComplete:
		return "✅"
	case StatusSkipped:
		return "⏭️"
	case StatusFailed:
		return "❌"
	case StatusRework:
		return "🔁"
	default:
		return "?"
	}
}

func joinLines(lines []string) string {
	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}
