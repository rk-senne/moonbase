package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// The shared pipeline loop driving every mission depth.

type pipelineLoopOptions struct {
	// handleConditionals enables conditional-phase trigger checking and skip messages.
	// When false, phases with StatusSkipped are silently skipped.
	handleConditionals bool
	// advanceAfterPhase calls p.Advance() after each successful phase to keep
	// the pipeline's Current pointer in sync.
	advanceAfterPhase bool
	// reworkOnRisk enables the full risk gate with rework looping (MEDIUM/HIGH
	// routes back, CRITICAL stops). When false, the risk gate is informational only.
	reworkOnRisk bool
	// runConditionalParallel is reserved for future use. Currently conditional
	// parallel execution is triggered separately after the loop.
	runConditionalParallel bool
	// allowEscalation enables mid-pipeline depth promotion when QA signals
	// insufficient analysis on a shallow pipeline (MEDIUM/HIGH risk).
	allowEscalation bool
}

// runPipelineLoop iterates a pipeline's phases, executing each via
// executeAndRecordPhase and applying risk gates, interrupt handling, and failure
// semantics based on the provided options.
//
// This is the shared core of both runMission (full) and runMissionFast (fast).
func runPipelineLoop(
	missionCtx context.Context,
	p *pipeline.Pipeline,
	reg agentLookup,
	ctx *discovery.ProjectContext,
	flywheel *pipeline.FlywheelLog,
	task string,
	opts pipelineLoopOptions,
) {
	// Load config for pricing and budget.
	cfg := config.Load()
	pricing := effectivePricing(cfg)
	budgetMax := cfg.TokenBudget.MaxTokensPerMission
	warnPct := cfg.TokenBudget.WarnThresholdPct
	if warnPct <= 0 {
		warnPct = 80 // default warn at 80%
	}

	var missionTokens int

	for i := 0; i < len(p.Phases); i++ {
		// Check for interrupt before starting next phase
		if missionCtx.Err() != nil {
			handleMissionInterrupt(p, &p.Phases[i], flywheel, task)
			return
		}

		phase := &p.Phases[i]

		// Silently skip phases pre-skipped by the depth profile (adaptive/full mode:
		// only non-conditional phases such as Analysis/Architecture/Review at a shallow
		// depth) or by fast mode (any already-skipped phase).
		if shouldSilentlySkip(*phase, opts.handleConditionals) {
			continue
		}

		// Conditional trigger evaluation (full/adaptive mode only).
		if opts.handleConditionals && phase.Conditional {
			trigger := p.ShouldInvokeConditional(phase)
			if !trigger.Invoke {
				fmt.Printf("   ⏭️  Phase %d: %s — skipped (%s)\n", phase.Number, phase.Name, trigger.Reason)
				phase.Status = pipeline.StatusSkipped
				continue
			}
			fmt.Printf("   ⚡ Phase %d: %s — triggered (%s)\n", phase.Number, phase.Name, trigger.Reason)
		}

		// Resolve agent
		agent := reg.GetByName(phase.AgentName)
		if agent == nil {
			if opts.handleConditionals {
				fmt.Printf("   ⚠️  Phase %d: agent %s not found, skipping\n", phase.Number, phase.AgentName)
				phase.Status = pipeline.StatusSkipped
			} else {
				fmt.Printf("   ⚠️  Phase %d: agent %s not found\n", phase.Number, phase.AgentName)
			}
			continue
		}

		fmt.Printf("   🔄 Phase %d: %s (%s)...\n", phase.Number, phase.Name, agent.Designation)

		if missionTrace {
			fmt.Printf("   [trace] Phase %d started at %s\n", phase.Number, time.Now().Format(time.RFC3339))
		}

		output, usage, err := executeAndRecordPhase(missionCtx, p, phase, agent, ctx, flywheel, task, pricing)
		if err != nil {
			// Check if the error was due to interrupt
			if missionCtx.Err() != nil {
				handleMissionInterrupt(p, phase, flywheel, task)
				return
			}

			// Auto-retry with exponential backoff before giving up.
			retried := false
			maxRetries := cfg.MaxRetries
			if maxRetries <= 0 {
				maxRetries = 3
			}
			backoffBase := cfg.RetryBackoffBase
			if backoffBase <= 0 {
				backoffBase = 1000
			}
			for attempt := 1; attempt <= maxRetries; attempt++ {
				backoff := time.Duration(backoffBase*(1<<(attempt-1))) * time.Millisecond
				fmt.Printf("   ⚠️ Phase %d failed: %v\n", phase.Number, err)
				fmt.Printf("   🔄 Auto-retrying (%d/%d) in %s...\n", attempt, maxRetries, backoff.Round(time.Millisecond))

				// Log the retry to flywheel
				flywheel.Append(pipeline.FlywheelEntry{
					Timestamp:   time.Now().UTC(),
					TraceID:     p.TraceID,
					Phase:       phase.Number,
					Agent:       phase.AgentName,
					Task:        task,
					Outcome:     "retried",
					DurationMs:  phase.ElapsedTime().Milliseconds(),
					OutputSize:  0,
					ReworkCount: p.Context.ReworkCount,
				})

				// Wait for backoff (or interrupt)
				select {
				case <-missionCtx.Done():
					handleMissionInterrupt(p, phase, flywheel, task)
					return
				case <-time.After(backoff):
				}

				// Re-execute the phase
				output, usage, err = executeAndRecordPhase(missionCtx, p, phase, agent, ctx, flywheel, task, pricing)
				if err == nil {
					retried = true
					break
				}
				if missionCtx.Err() != nil {
					handleMissionInterrupt(p, phase, flywheel, task)
					return
				}
			}

			if !retried {
				if opts.reworkOnRisk {
					handlePhaseFailure(p, phase, err)
				} else {
					fmt.Printf("   ❌ Phase %d failed after %d retries: %v\n", phase.Number, maxRetries, err)
				}
				break
			}
		}

		// Budget enforcement (AC-6): check cumulative tokens after each phase.
		budget := phaseBudget{
			Pipeline: p, Phase: phase, Flywheel: flywheel,
			Task: task, Output: output, Usage: usage,
			Consumed: &missionTokens, Max: budgetMax, WarnPct: warnPct,
		}
		if budget.exceeded() {
			return
		}

		if missionTrace {
			fmt.Printf("   [trace] Phase %d completed at %s (elapsed: %s)\n", phase.Number, phase.CompletedAt.Format(time.RFC3339), phase.ElapsedTime().Round(time.Millisecond))
			fmt.Printf("   [trace] Phase %d output size: %d bytes\n", phase.Number, len(output))
		}

		fmt.Printf("   ✅ Phase %d complete (%d chars)\n", phase.Number, len(output))

		// Advance pipeline state to keep Current in sync
		if opts.advanceAfterPhase {
			p.Advance()
		}

		// Apply risk gate after QA (phase 4)
		if phase.Number == 4 {
			if opts.reworkOnRisk {
				// Parse risk level without side effects to check for escalation first.
				routing := pipeline.ParseRiskGate(output)

				// Escalation check: if depth is shallow and risk is non-LOW/non-CRITICAL, escalate.
				// CRITICAL always stops first — escalation cannot override CRITICAL.
				if opts.allowEscalation && routing.Level != pipeline.RiskLow && routing.Level != pipeline.RiskCritical && p.Depth != pipeline.DepthComplex {
					targetDepth := pipeline.EscalationTarget(p.Depth, routing.Level)
					if targetDepth != p.Depth {
						fmt.Printf("   ⬆️  Escalating: %s → %s (QA risk: %s)\n", p.Depth, targetDepth, routing.Level)
						p.Escalate(targetDepth)

						// Log escalation to flywheel
						flywheel.Append(pipeline.FlywheelEntry{
							Timestamp:     time.Now().UTC(),
							TraceID:       p.TraceID,
							Phase:         phase.Number,
							Agent:         phase.AgentName,
							Task:          task,
							Outcome:       "escalated",
							RiskLevel:     string(routing.Level),
							Depth:         string(targetDepth),
							DepthReason:   p.DepthReason,
							EscalatedFrom: string(p.OrigDepth),
							EscalatedTo:   string(targetDepth),
							DurationMs:    phase.ElapsedTime().Milliseconds(),
							OutputSize:    len(output),
						})

						// Route to the earliest pending non-conditional phase
						targetIdx := -1
						for j, ph := range p.Phases {
							if ph.Status == pipeline.StatusPending && !ph.Conditional {
								targetIdx = j
								break
							}
						}
						if targetIdx >= 0 {
							i = targetIdx - 1 // -1 because loop will increment
							continue
						}
					}
				}

				// Normal risk gate handling (unchanged path).
				shouldContinue, targetIdx := handleRiskGate(p, output)
				if !shouldContinue {
					break
				}
				if targetIdx >= 0 {
					// Log rework to flywheel
					flywheel.Append(pipeline.FlywheelEntry{
						Timestamp:   time.Now().UTC(),
						TraceID:     p.TraceID,
						Phase:       phase.Number,
						Agent:       phase.AgentName,
						Task:        task,
						Outcome:     "rework",
						RiskLevel:   p.Context.RiskLevel,
						Depth:       string(p.Depth),
						DepthReason: p.DepthReason,
						DurationMs:  phase.ElapsedTime().Milliseconds(),
						OutputSize:  len(output),
						ReworkCount: p.Context.ReworkCount,
					})
					// Loop back — adjust i to re-run from the target phase
					i = targetIdx - 1 // -1 because loop will increment
					continue
				}

				// Ensure Review (Phase 5) runs on LOW risk when the depth includes it:
				// escalated pipelines and non-escalated simple depth both proceed to Review.
				// Trivial (non-escalated) intentionally skips Review on LOW.
				if routing.Level == pipeline.RiskLow && (p.Escalated || p.Depth == pipeline.DepthSimple) {
					p.UnskipPhase(5)
				}
			} else {
				// Fast mode: informational risk gate only
				routing, _ := p.ApplyRiskGate(output)
				fmt.Printf("   🎯 Risk Gate: %s — %s\n", routing.Level, routing.Action)
				if routing.Level == pipeline.RiskCritical || routing.Level == pipeline.RiskHigh {
					fmt.Println("\n   ⚠️  High risk on fast mission — consider running full pipeline.")
				}
			}
		}
	}

	// Save checkpoint after pipeline execution
	home := mustUserHomeDir()
	checkpointDir := filepath.Join(home, ".moonbase", "checkpoints")
	pipeline.SaveCheckpoint(p, checkpointDir)
}
