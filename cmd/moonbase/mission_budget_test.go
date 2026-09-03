package main

import (
	"testing"

	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// The token-budget gate became unit-testable once it was extracted from the
// 268-line pipeline loop — previously it could only be reached through a full
// mission run.

func newBudgetFixture(t *testing.T, consumed *int, max, warnPct int, total int) phaseBudget {
	t.Helper()
	// Keep flywheel writes inside the test's temp dir.
	t.Setenv("HOME", t.TempDir())

	return phaseBudget{
		Pipeline: &pipeline.Pipeline{
			TraceID: "trace-test",
			Task:    "the task",
			Context: &pipeline.PipelineContext{},
		},
		Phase:    &pipeline.Phase{Number: 3, AgentName: "numbuh-3"},
		Flywheel: pipeline.NewFlywheelLog(),
		Task:     "the task",
		Output:   "some output",
		Usage: &backend.UsageInfo{
			PromptTokens:     total / 2,
			CompletionTokens: total - total/2,
			TotalTokens:      total,
			Model:            "test-model",
		},
		Consumed: consumed,
		Max:      max,
		WarnPct:  warnPct,
	}
}

func TestPhaseBudget_NilUsageIsNotAViolation(t *testing.T) {
	consumed := 0
	b := newBudgetFixture(t, &consumed, 1000, 80, 0)
	b.Usage = nil

	if b.exceeded() {
		t.Error("nil usage must not trip the budget")
	}
	if consumed != 0 {
		t.Errorf("consumed = %d, want unchanged", consumed)
	}
}

// A zero or negative budget means unlimited and must short-circuit entirely —
// including the division that computes the warning percentage.
func TestPhaseBudget_UnlimitedBudgetShortCircuits(t *testing.T) {
	for _, max := range []int{0, -1} {
		consumed := 0
		b := newBudgetFixture(t, &consumed, max, 80, 5000)
		if b.exceeded() {
			t.Errorf("max=%d must be treated as unlimited", max)
		}
		if consumed != 0 {
			t.Errorf("max=%d: consumed = %d, want unchanged", max, consumed)
		}
	}
}

func TestPhaseBudget_AccumulatesAcrossPhases(t *testing.T) {
	consumed := 0
	for i := 1; i <= 3; i++ {
		b := newBudgetFixture(t, &consumed, 10_000, 80, 1000)
		if b.exceeded() {
			t.Fatalf("phase %d should be within budget", i)
		}
	}
	if consumed != 3000 {
		t.Errorf("consumed = %d, want 3000 accumulated across three phases", consumed)
	}
}

func TestPhaseBudget_StopsWhenExceeded(t *testing.T) {
	consumed := 900
	b := newBudgetFixture(t, &consumed, 1000, 80, 200)

	if !b.exceeded() {
		t.Fatal("expected the budget to be exceeded at 1100 of 1000")
	}
	if consumed != 1100 {
		t.Errorf("consumed = %d, want 1100 recorded before stopping", consumed)
	}
}

// Exactly at the limit is not over it.
func TestPhaseBudget_ExactlyAtLimitIsAllowed(t *testing.T) {
	consumed := 900
	b := newBudgetFixture(t, &consumed, 1000, 80, 100)

	if b.exceeded() {
		t.Error("consuming exactly the budget must not stop the pipeline")
	}
	if consumed != 1000 {
		t.Errorf("consumed = %d, want 1000", consumed)
	}
}

func TestPhaseBudget_WarnThresholdDoesNotStop(t *testing.T) {
	consumed := 0
	b := newBudgetFixture(t, &consumed, 1000, 50, 600) // 60% — past the warning

	if b.exceeded() {
		t.Error("crossing the warning threshold must not stop the pipeline")
	}
}
