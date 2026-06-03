package pipeline

// Phase represents a single phase in the KND Council pipeline
type Phase struct {
	Number      int
	Name        string
	Operative   string
	Status      PhaseStatus
	Duration    string
	Summary     string
	Conditional bool
}

type PhaseStatus int

const (
	StatusPending PhaseStatus = iota
	StatusRunning
	StatusComplete
	StatusSkipped
	StatusFailed
)

// Pipeline manages the full council execution flow
type Pipeline struct {
	Task    string
	Phases  []Phase
	Current int
	Active  bool
}

func New(task string) *Pipeline {
	return &Pipeline{
		Task:   task,
		Active: true,
		Phases: []Phase{
			{1, "Analysis", "Numbuh 1", StatusPending, "", "", false},
			{2, "Architecture", "Numbuh 2", StatusPending, "", "", false},
			{3, "Implementation", "Numbuh 3", StatusPending, "", "", false},
			{4, "QA", "Numbuh 4", StatusPending, "", "", false},
			{5, "Review", "Numbuh 5", StatusPending, "", "", false},
			{6, "Oversight", "Numbuh 0", StatusPending, "", "", true},
			{7, "Security", "Numbuh 274", StatusPending, "", "", true},
			{8, "Deploy", "Numbuh 362", StatusPending, "", "", true},
		},
	}
}

func (p *Pipeline) Advance() {
	if p.Current < len(p.Phases)-1 {
		p.Phases[p.Current].Status = StatusComplete
		p.Current++
		p.Phases[p.Current].Status = StatusRunning
	}
}

func (p *Pipeline) Retry() {
	p.Phases[p.Current].Status = StatusRunning
}

func (p *Pipeline) Skip() {
	p.Phases[p.Current].Status = StatusSkipped
	p.Current++
	if p.Current < len(p.Phases) {
		p.Phases[p.Current].Status = StatusRunning
	}
}
