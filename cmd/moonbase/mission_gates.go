package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// Phase failure, risk gate, and interrupt handling.

// handlePhaseFailure prints the failure message, marks the phase as failed, and stops the pipeline.
// Centralizes phase failure handling for consistent error reporting across mission types.
func handlePhaseFailure(p *pipeline.Pipeline, phase *pipeline.Phase, err error) {
	fmt.Printf("   ❌ Phase %d failed: %v\n", phase.Number, err)
	phase.Status = pipeline.StatusFailed
	p.Stop(err.Error())
}

// handleRiskGate applies the QA risk assessment and prints the routing decision.
// Returns (shouldContinue, targetPhaseIndex) where:
//   - shouldContinue=false means the pipeline should stop (CRITICAL risk or max rework exceeded)
//   - targetPhaseIndex >= 0 means the pipeline should loop back to that index (MEDIUM/HIGH risk)
//   - targetPhaseIndex < 0 means the pipeline should proceed normally (LOW risk)
func handleRiskGate(p *pipeline.Pipeline, output string) (shouldContinue bool, targetPhaseIndex int) {
	routing, rErr := p.ApplyRiskGate(output)
	fmt.Printf("   🎯 Risk Gate: %s — %s\n", routing.Level, routing.Action)

	if routing.Level == pipeline.RiskCritical {
		fmt.Println("\n   🛑 CRITICAL RISK — Pipeline stopped. Escalating to human.")
		return false, -1
	}
	if rErr != nil {
		fmt.Printf("\n   🛑 %v\n", rErr)
		return false, -1
	}
	if routing.Level == pipeline.RiskMedium || routing.Level == pipeline.RiskHigh {
		// Find the target phase index to loop back to
		for j, ph := range p.Phases {
			if ph.Number == routing.TargetPhase {
				return true, j
			}
		}
	}

	// LOW risk or unknown — proceed normally
	return true, -1
}

// handleMissionInterrupt handles graceful shutdown when SIGINT/SIGTERM is received.
// It marks the current phase as failed, logs an "interrupted" flywheel entry, saves
// a checkpoint for later resume, and prints a clear message to the user.
func handleMissionInterrupt(p *pipeline.Pipeline, phase *pipeline.Phase, flywheel *pipeline.FlywheelLog, task string) {
	phase.Status = pipeline.StatusFailed

	flywheel.Append(pipeline.FlywheelEntry{
		Timestamp:  time.Now().UTC(),
		TraceID:    p.TraceID,
		Phase:      phase.Number,
		Agent:      phase.AgentName,
		Task:       task,
		Outcome:    "interrupted",
		DurationMs: time.Since(phase.StartedAt).Milliseconds(),
		OutputSize: 0,
	})

	home := mustUserHomeDir()
	checkpointDir := filepath.Join(home, ".moonbase", "checkpoints")
	pipeline.SaveCheckpoint(p, checkpointDir)

	fmt.Printf("\n🛑 Mission interrupted — checkpoint saved (trace %s). Resume with 'moonbase replay %s'.\n", p.TraceID, p.TraceID)
}

// effectivePricing merges default model pricing with user overrides from config.
// Config prices take precedence over defaults for the same model name.
func effectivePricing(cfg config.Config) map[string]pipeline.ModelPrice {
	merged := pipeline.DefaultPricing()
	for model, price := range cfg.ModelPricing {
		merged[model] = price
	}
	return merged
}
