package main

import (
	"fmt"

	"github.com/rk-senne/moonbase/internal/pipeline"
)

// runMissionDryRun creates a pipeline, evaluates triggers, and prints the plan
// without invoking any backends. Useful for previewing what a mission would do.
func runMissionDryRun(task string) {
	fmt.Println("🌙 KND Council — Mission Dry Run")
	fmt.Printf("   Task: %s\n\n", task)

	// Determine depth
	var depth pipeline.Depth
	var reason string
	switch {
	case missionFast:
		depth = "override:fast"
		reason = "explicit --fast flag"
	case missionFull:
		depth = pipeline.DepthComplex
		reason = "explicit --full flag"
	case missionDepth != "":
		depth = validateDepthFlag(missionDepth)
		if depth == "" {
			return
		}
		reason = "override:" + missionDepth
	default:
		classification := pipeline.ClassifyTask(task)
		depth = classification.Depth
		reason = classification.Reason
	}

	fmt.Printf("   DEPTH CLASSIFICATION\n")
	fmt.Printf("   ─────────────────────────────────────\n")
	fmt.Printf("   Depth: %s (%s)\n\n", depth, reason)

	var p *pipeline.Pipeline
	if missionFast || depth == "override:fast" {
		p = pipeline.NewFast(task)
	} else if depth == pipeline.DepthTrivial || depth == pipeline.DepthSimple || depth == pipeline.DepthComplex {
		p = pipeline.NewAdaptive(task, depth, reason)
	} else {
		p = pipeline.New(task)
	}

	// Apply sequential override if requested.
	if missionSequential {
		p.ParallelSpecialists = false
	}

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
		} else if phase.Status == pipeline.StatusSkipped {
			fmt.Printf("   ⏭️  Phase %d: %s (%s) — skipped (depth: %s)\n",
				phase.Number, phase.Name, phase.Operative, depth)
		} else {
			fmt.Printf("   ▶️  Phase %d: %s (%s)\n",
				phase.Number, phase.Name, phase.Operative)
		}
	}

	fmt.Println()
	fmt.Println("   SPECIALISTS")
	fmt.Println("   ─────────────────────────────────────")
	if p.ParallelSpecialists {
		fmt.Printf("   Mode: parallel (concurrency: %d)\n", p.MaxSpecialistConcurrency)
	} else {
		fmt.Println("   Mode: sequential (override)")
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
	if !missionFast && depth != "override:fast" {
		fmt.Println("   Escalation: enabled (shallow depths promote on MEDIUM/HIGH risk)")
	}
	fmt.Println()
	fmt.Println("   ℹ️  No backends will be invoked. Use 'moonbase mission' without --dry-run to execute.")
}
