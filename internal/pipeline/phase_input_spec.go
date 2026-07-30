package pipeline

// PhaseInputSpec declares what prior-phase context a given phase requires.
// The pipeline context assembler uses these specs to compose input for each
// phase without hard-coding phase-ordering knowledge in a switch statement.
type PhaseInputSpec struct {
	// RequiresPhases lists the prior phase numbers whose output is included.
	// Order matters: outputs are appended in this order.
	RequiresPhases []int

	// MaxPerPhase maps a phase number to the maximum character length for
	// that phase's output. A value of 0 means no truncation (include full output).
	MaxPerPhase map[int]int

	// HeaderFormat maps a phase number to the markdown header used when
	// including that phase's output. The format string receives the phase
	// number as an argument if it contains a %d verb.
	HeaderFormat map[int]string

	// IncludeRework indicates that this phase should include QA feedback
	// from Phase 4 when the pipeline is in a rework loop (ReworkCount > 0).
	IncludeRework bool
}

// phaseInputSpecs defines the canonical context requirements for each pipeline
// phase. These encode the same dependencies previously expressed in ForPhase's
// switch statement, making context assembly data-driven.
var phaseInputSpecs = map[int]PhaseInputSpec{
	1: {
		// Analyst: just the task — no prior phase outputs.
		RequiresPhases: nil,
		MaxPerPhase:    nil,
		HeaderFormat:   nil,
	},
	2: {
		// Architect: task + full requirements from phase 1.
		RequiresPhases: []int{1},
		MaxPerPhase:    map[int]int{1: 0},
		HeaderFormat:   map[int]string{1: "## Requirements (from Phase 1 — Analyst)"},
	},
	3: {
		// Implementer: task + requirements (summarized) + full design.
		// Also includes QA rework feedback when in rework loop.
		RequiresPhases: []int{1, 2},
		MaxPerPhase:    map[int]int{1: 3000, 2: 0},
		HeaderFormat: map[int]string{
			1: "## Requirements (from Phase 1 — Analyst)",
			2: "## Design (from Phase 2 — Architect)",
		},
		IncludeRework: true,
	},
	4: {
		// QA: task + requirements (summarized) + full implementation output.
		RequiresPhases: []int{1, 3},
		MaxPerPhase:    map[int]int{1: 2000, 3: 0},
		HeaderFormat: map[int]string{
			1: "## Requirements (from Phase 1 — Analyst)",
			3: "## Implementation (from Phase 3 — Implementer)",
		},
	},
	5: {
		// Reviewer: summary of everything including specialist outputs.
		RequiresPhases: []int{1, 2, 3, 4, 6, 7, 8},
		MaxPerPhase:    map[int]int{1: 1500, 2: 1500, 3: 2000, 4: 0, 6: 2000, 7: 2000, 8: 2000},
		HeaderFormat: map[int]string{
			1: "## Requirements (from Phase 1)",
			2: "## Design (from Phase 2)",
			3: "## Implementation (from Phase 3)",
			4: "## QA Report (from Phase 4)",
			6: "## Oversight Analysis (from Phase 6 — Numbuh 0)",
			7: "## Security Analysis (from Phase 7 — Numbuh 274)",
			8: "## Deploy Prep (from Phase 8 — Numbuh 362)",
		},
	},
}

// defaultPhaseInputSpec returns the spec used for phases not explicitly listed
// (specialists, oversight, etc.). Includes all core phases (1–5) summarized.
func defaultPhaseInputSpec() PhaseInputSpec {
	return PhaseInputSpec{
		RequiresPhases: []int{1, 2, 3, 4, 5},
		MaxPerPhase:    map[int]int{1: 1000, 2: 1000, 3: 1000, 4: 1000, 5: 1000},
		HeaderFormat: map[int]string{
			1: "## Phase 1 Output",
			2: "## Phase 2 Output",
			3: "## Phase 3 Output",
			4: "## Phase 4 Output",
			5: "## Phase 5 Output",
		},
	}
}

// lookupPhaseInputSpec returns the PhaseInputSpec for the given phase number.
// If no explicit spec is defined, the default specialist spec is returned.
func lookupPhaseInputSpec(phase int) PhaseInputSpec {
	if spec, ok := phaseInputSpecs[phase]; ok {
		return spec
	}
	return defaultPhaseInputSpec()
}
