package main

import (
	"fmt"

	"github.com/f5508037/moonbase/internal/pipeline"
)

// runMissionDryRun creates a pipeline, evaluates triggers, and prints the plan
// without invoking any backends. Useful for previewing what a mission would do.
func runMissionDryRun(task string) {
	fmt.Println("🌙 KND Council — Mission Dry Run")
	fmt.Printf("   Task: %s\n\n", task)

	p := pipeline.New(task)

	fmt.Println("   EXECUTION PLAN")
	fmt.Println("   ─────────────────────────────────────")

	for _, phase := range p.Phases {
		if phase.Conditional {
			trigger := p.ShouldInvokeConditional(&phase)
			if trigger.Invoke {
				fmt.Printf("   ⚡ Phase %d: %s (%s) — triggered (%s)\n",
					phase.Number, phase.Name, phase.Operative, trigger.Reason)
			} else {
				fmt.Printf("   ⏭️  Phase %d: %s (%s) — would skip (%s)\n",
					phase.Number, phase.Name, phase.Operative, trigger.Reason)
			}
		} else {
			fmt.Printf("   ▶️  Phase %d: %s (%s)\n",
				phase.Number, phase.Name, phase.Operative)
		}
	}

	fmt.Println()
	fmt.Println("   RISK GATE")
	fmt.Println("   ─────────────────────────────────────")
	fmt.Println("   After Phase 4 (QA), risk assessment determines routing:")
	fmt.Println("     LOW      → proceed to Review (Phase 5)")
	fmt.Println("     MEDIUM   → rework from Implementation (Phase 3)")
	fmt.Println("     HIGH     → redesign from Architecture (Phase 2)")
	fmt.Println("     CRITICAL → stop pipeline, escalate to human")
	fmt.Printf("   Max rework loops: %d\n", p.MaxRework)
	fmt.Println()
	fmt.Println("   ℹ️  No backends will be invoked. Use 'moonbase mission' without --dry-run to execute.")
}
