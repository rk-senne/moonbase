package main

import (
	"testing"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

// TestShouldSilentlySkip is the regression test for the adaptive-depth showstopper:
// in adaptive/full mode the loop must silently skip NON-conditional phases the depth
// profile pre-skipped (otherwise depth classification has no runtime effect), while
// still evaluating conditional phases. In fast mode any skipped phase is skipped.
func TestShouldSilentlySkip(t *testing.T) {
	tests := []struct {
		name               string
		status             pipeline.PhaseStatus
		conditional        bool
		handleConditionals bool
		want               bool
	}{
		{"adaptive: pre-skipped non-conditional -> skip", pipeline.StatusSkipped, false, true, true},
		{"adaptive: pending non-conditional -> run", pipeline.StatusPending, false, true, false},
		{"adaptive: skipped conditional -> evaluate (not silent-skip)", pipeline.StatusSkipped, true, true, false},
		{"adaptive: pending conditional -> evaluate", pipeline.StatusPending, true, true, false},
		{"fast: skipped -> skip", pipeline.StatusSkipped, false, false, true},
		{"fast: pending -> run", pipeline.StatusPending, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ph := pipeline.Phase{Status: tt.status, Conditional: tt.conditional}
			if got := shouldSilentlySkip(ph, tt.handleConditionals); got != tt.want {
				t.Errorf("shouldSilentlySkip(status=%v, conditional=%v, handleConditionals=%v) = %v, want %v",
					tt.status, tt.conditional, tt.handleConditionals, got, tt.want)
			}
		})
	}
}

// TestNewAdaptive_SkipPredicate_ByDepth verifies that the loop's skip predicate,
// applied to a freshly constructed adaptive pipeline, skips exactly the phases the
// depth profile intends — and NEVER skips QA (Phase 4). This ties the constructor
// to the runtime skip logic (the gap that made the feature non-functional).
func TestNewAdaptive_SkipPredicate_ByDepth(t *testing.T) {
	cases := []struct {
		depth       pipeline.Depth
		wantSkipped map[int]bool // core phase number -> silently skipped in adaptive mode
	}{
		{pipeline.DepthTrivial, map[int]bool{1: true, 2: true, 3: false, 4: false, 5: true}},
		{pipeline.DepthSimple, map[int]bool{1: false, 2: true, 3: false, 4: false, 5: true}},
		{pipeline.DepthComplex, map[int]bool{1: false, 2: false, 3: false, 4: false, 5: false}},
	}
	for _, c := range cases {
		p := pipeline.NewAdaptive("task", c.depth, "test")
		for _, ph := range p.Phases {
			if ph.Number < 1 || ph.Number > 5 {
				continue // core phases only
			}
			got := shouldSilentlySkip(ph, true)
			if want := c.wantSkipped[ph.Number]; got != want {
				t.Errorf("depth=%s phase=%d: shouldSilentlySkip=%v, want %v", c.depth, ph.Number, got, want)
			}
			// HARD INVARIANT: QA (Phase 4) must NEVER be skipped at any depth.
			if ph.Number == 4 && got {
				t.Fatalf("INVARIANT VIOLATED: QA (Phase 4) would be skipped at depth %s", c.depth)
			}
		}
	}
}
