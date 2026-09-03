package main

// Token-budget enforcement for a completed pipeline phase (AC-6).

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// phaseBudget bundles what the token-budget gate needs to evaluate one completed
// phase.
//
// Grouped into a struct rather than passed as nine parameters: a helper whose
// signature is as complicated as its body is a shallow module, and this one is
// called from a single place with a fixed set of inputs.
type phaseBudget struct {
	Pipeline *pipeline.Pipeline
	Phase    *pipeline.Phase
	Flywheel *pipeline.FlywheelLog
	Task     string
	Output   string
	Usage    *backend.UsageInfo

	// Consumed is the running mission total, updated in place so successive
	// phases accumulate against the same budget.
	Consumed *int
	Max      int
	WarnPct  int
}

// exceeded adds this phase's usage to the mission total and reports whether the
// budget is now blown. When it returns true the caller must stop the pipeline;
// the budget_exceeded outcome has already been recorded and a checkpoint saved so
// the run can be inspected or resumed.
//
// A zero Max means "unlimited" and short-circuits the whole check, which is why
// nil usage and unset budgets are both treated as "carry on".
func (b phaseBudget) exceeded() bool {
	if b.Usage == nil || b.Max <= 0 {
		return false
	}

	*b.Consumed += b.Usage.TotalTokens

	if *b.Consumed > b.Max {
		fmt.Printf("   🛑 Token budget exceeded (%dK / %dK). Pipeline stopped.\n",
			*b.Consumed/1000, b.Max/1000)
		b.Flywheel.Append(pipeline.FlywheelEntry{
			Timestamp:        time.Now().UTC(),
			TraceID:          b.Pipeline.TraceID,
			Phase:            b.Phase.Number,
			Agent:            b.Phase.AgentName,
			Task:             b.Task,
			Outcome:          "budget_exceeded",
			DurationMs:       b.Phase.ElapsedTime().Milliseconds(),
			OutputSize:       len(b.Output),
			PromptTokens:     b.Usage.PromptTokens,
			CompletionTokens: b.Usage.CompletionTokens,
			TotalTokens:      b.Usage.TotalTokens,
			Model:            b.Usage.Model,
		})
		checkpointDir := filepath.Join(mustUserHomeDir(), ".moonbase", "checkpoints")
		pipeline.SaveCheckpoint(b.Pipeline, checkpointDir)
		return true
	}

	if pct := (*b.Consumed * 100) / b.Max; pct >= b.WarnPct {
		fmt.Printf("   ⚠️  Token budget: %d%% used (%dK / %dK)\n",
			pct, *b.Consumed/1000, b.Max/1000)
	}
	return false
}
