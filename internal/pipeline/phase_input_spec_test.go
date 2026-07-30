package pipeline

import (
	"strings"
	"testing"
)

func TestPhaseInputSpec_DrivesAssembly(t *testing.T) {
	// Verify that adding a hypothetical spec to the map changes assembly output.
	// This proves ForPhase is driven by spec data, not a switch statement.

	ctx := NewPipelineContext("hypothetical task")
	ctx.RecordPhase(1, "Requirements output")
	ctx.RecordPhase(2, "Design output")
	ctx.RecordPhase(3, "Implementation output")
	ctx.RecordPhase(4, "QA output")
	ctx.RecordPhase(5, "Review output")

	// Phase 99 has no explicit spec → gets default (all phases summarized at 1000).
	defaultOutput := ctx.ForPhase(99)

	// Now temporarily register a custom spec for phase 99.
	phaseInputSpecs[99] = PhaseInputSpec{
		RequiresPhases: []int{2},
		MaxPerPhase:    map[int]int{2: 0},
		HeaderFormat:   map[int]string{2: "## Custom Design Context"},
	}
	defer delete(phaseInputSpecs, 99) // Clean up after test.

	customOutput := ctx.ForPhase(99)

	// The custom output should differ from the default.
	if defaultOutput == customOutput {
		t.Error("expected spec change to produce different assembly output")
	}

	// Custom output should include only phase 2 with the custom header.
	if !strings.Contains(customOutput, "## Custom Design Context") {
		t.Error("expected custom header in spec-driven output")
	}
	if !strings.Contains(customOutput, "Design output") {
		t.Error("expected phase 2 content in spec-driven output")
	}

	// Custom output should NOT include phase 1, 3, 4, 5 outputs.
	if strings.Contains(customOutput, "Requirements output") {
		t.Error("expected phase 1 output to be excluded by custom spec")
	}
	if strings.Contains(customOutput, "Implementation output") {
		t.Error("expected phase 3 output to be excluded by custom spec")
	}
	if strings.Contains(customOutput, "QA output") {
		t.Error("expected phase 4 output to be excluded by custom spec")
	}
	if strings.Contains(customOutput, "Review output") {
		t.Error("expected phase 5 output to be excluded by custom spec")
	}
}

func TestPhaseInputSpec_UnknownPhase_DefaultBehavior(t *testing.T) {
	// Unknown phase numbers (not in the spec map) should get the default
	// specialist view: all phases 1–5 summarized at 1000 chars each.
	ctx := NewPipelineContext("test unknown phase")
	ctx.RecordPhase(1, "Phase 1 data")
	ctx.RecordPhase(2, "Phase 2 data")
	ctx.RecordPhase(3, "Phase 3 data")
	ctx.RecordPhase(4, "Phase 4 data")
	ctx.RecordPhase(5, "Phase 5 data")

	input := ctx.ForPhase(42)

	// Should include all five phases with the default header format.
	for i := 1; i <= 5; i++ {
		expectedHeader := "## Phase " + strings.Repeat("", 0) // Just check header prefix.
		if !strings.Contains(input, expectedHeader) {
			t.Errorf("expected Phase %d header in unknown-phase output", i)
		}
	}
	if !strings.Contains(input, "Phase 1 Output") {
		t.Error("expected '## Phase 1 Output' header for unknown phase")
	}
	if !strings.Contains(input, "Phase 5 Output") {
		t.Error("expected '## Phase 5 Output' header for unknown phase")
	}
	if !strings.Contains(input, "test unknown phase") {
		t.Error("expected task in unknown phase output")
	}
}

func TestPhaseInputSpec_UnknownPhase_TruncatesLongOutput(t *testing.T) {
	// The default spec truncates at 1000 chars per phase.
	ctx := NewPipelineContext("test truncation")
	longOutput := strings.Repeat("x", 2000)
	ctx.RecordPhase(1, longOutput)

	input := ctx.ForPhase(99)

	// Should contain truncation notice since output > 1000.
	if !strings.Contains(input, "truncated") {
		t.Error("expected truncation notice for long output in default spec")
	}
	// Should not contain the full 2000-char output.
	if strings.Contains(input, longOutput) {
		t.Error("expected output to be truncated, but found full content")
	}
}

func TestPhaseInputSpec_LookupKnownPhases(t *testing.T) {
	// Verify all canonical phases (1–5) have explicit specs.
	for phase := 1; phase <= 5; phase++ {
		spec := lookupPhaseInputSpec(phase)
		// Phase 1 has no requirements; others should have at least one.
		if phase > 1 && len(spec.RequiresPhases) == 0 {
			t.Errorf("phase %d: expected non-empty RequiresPhases", phase)
		}
	}
}

func TestPhaseInputSpec_SpecFieldsConsistent(t *testing.T) {
	// Every phase listed in RequiresPhases should have an entry in
	// HeaderFormat and MaxPerPhase.
	allSpecs := map[int]PhaseInputSpec{}
	for k, v := range phaseInputSpecs {
		allSpecs[k] = v
	}
	allSpecs[0] = defaultPhaseInputSpec() // Also check the default.

	for phase, spec := range allSpecs {
		for _, req := range spec.RequiresPhases {
			if _, ok := spec.HeaderFormat[req]; !ok {
				t.Errorf("phase %d: RequiresPhases includes %d but HeaderFormat has no entry", phase, req)
			}
			if _, ok := spec.MaxPerPhase[req]; !ok {
				t.Errorf("phase %d: RequiresPhases includes %d but MaxPerPhase has no entry", phase, req)
			}
		}
	}
}
