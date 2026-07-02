package pipeline

import (
	"fmt"
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
	Number      int
	Name        string
	Operative   string
	AgentName   string      // agent file name (e.g., "numbuh-1")
	Status      PhaseStatus
	Duration    string
	Summary     string
	Conditional bool
	TriggerSpec string // trigger conditions for conditional phases
}

// Pipeline manages the full council execution flow.
type Pipeline struct {
	Task        string
	Phases      []Phase
	Current     int
	Active      bool
	Context     *PipelineContext
	MaxRework   int // maximum rework loops (default 2)
}

// Backend is the interface the pipeline uses to deploy agents.
// This is satisfied by the backend package implementations.
type Backend interface {
	Deploy(agentName string, prompt string, input string) (string, error)
}

// New creates a new pipeline for a given task.
func New(task string) *Pipeline {
	return &Pipeline{
		Task:      task,
		Active:    true,
		MaxRework: 2,
		Context:   NewPipelineContext(task),
		Phases: []Phase{
			{1, "Analysis", "Numbuh 1", "numbuh-1", StatusPending, "", "", false, ""},
			{2, "Architecture", "Numbuh 2", "numbuh-2", StatusPending, "", "", false, ""},
			{3, "Implementation", "Numbuh 3", "numbuh-3", StatusPending, "", "", false, ""},
			{4, "QA", "Numbuh 4", "numbuh-4", StatusPending, "", "", false, ""},
			{5, "Review", "Numbuh 5", "numbuh-5", StatusPending, "", "", false, ""},
			{6, "Oversight", "Numbuh 0", "numbuh-0", StatusPending, "", "", true, ">5 files changed, core logic changed, orchestration/pipeline changed"},
			{7, "Security", "Numbuh 274", "numbuh-274", StatusPending, "", "", true, "Auth/secrets/permissions changed, new endpoints, dependency CVEs"},
			{8, "Deploy Prep", "Numbuh 362", "numbuh-362", StatusPending, "", "", true, "CI/CD changed, Docker/infra touched, new env vars, deployment config"},
		},
	}
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
		return fmt.Errorf("max rework loops (%d) exceeded — escalating to human", p.MaxRework)
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

	return fmt.Errorf("target phase %d not found in pipeline", targetPhase)
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
