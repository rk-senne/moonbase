package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// executeAndRecordPhase runs a single phase and records results to flywheel.
// Returns the output string, usage info (nil if unavailable), and any error.
func executeAndRecordPhase(
	missionCtx context.Context,
	p *pipeline.Pipeline,
	phase *pipeline.Phase,
	agent *agents.Agent,
	ctx *discovery.ProjectContext,
	flywheel *pipeline.FlywheelLog,
	task string,
	pricing map[string]pipeline.ModelPrice,
) (string, *backend.UsageInfo, error) {
	phase.StartPhase()

	phaseInput := p.Context.ForPhase(phase.Number)

	// File injection for Phase 3 (Implementation)
	if phase.Number == 3 {
		if fileCtx := injectFileContext(p.Context); fileCtx != "" {
			phaseInput += fileCtx
		}
	}
	// Diff injection for Phase 4 (QA)
	if phase.Number == 4 && p.Context.Diff != "" {
		phaseInput += fmt.Sprintf("\n\n## Actual Changes (git diff)\n\n```diff\n%s\n```", p.Context.Diff)
	}

	composed := discovery.ComposePrompt(agent.Prompt, ctx, phaseInput)
	output, usage, err := backend.DeployComposed(missionCtx, composed, phaseInput, p.PhaseTimeout)

	if err != nil {
		phase.Status = pipeline.StatusFailed
		flywheel.Append(pipeline.FlywheelEntry{
			Timestamp:  time.Now().UTC(),
			TraceID:    p.TraceID,
			Phase:      phase.Number,
			Agent:      phase.AgentName,
			Task:       task,
			Outcome:    "failed",
			DurationMs: time.Since(phase.StartedAt).Milliseconds(),
			OutputSize: 0,
		})
		return "", nil, err
	}

	p.Context.RecordPhase(phase.Number, output)
	phase.CompletePhase()

	// Build flywheel entry with token/cost data if available.
	entry := pipeline.FlywheelEntry{
		Timestamp:   time.Now().UTC(),
		TraceID:     p.TraceID,
		Phase:       phase.Number,
		Agent:       phase.AgentName,
		Task:        task,
		Outcome:     "complete",
		RiskLevel:   p.Context.RiskLevel,
		DurationMs:  phase.ElapsedTime().Milliseconds(),
		OutputSize:  len(output),
		ReworkCount: p.Context.ReworkCount,
	}
	if usage != nil {
		entry.PromptTokens = usage.PromptTokens
		entry.CompletionTokens = usage.CompletionTokens
		entry.TotalTokens = usage.TotalTokens
		entry.Model = usage.Model
		if pricing != nil {
			entry.EstimatedCostUSD = pipeline.EstimateCost(usage.Model, usage.PromptTokens, usage.CompletionTokens, pricing)
		}
	}
	flywheel.Append(entry)

	// Capture git diff after Phase 3
	if phase.Number == 3 {
		if diffOutput, dErr := exec.Command("git", "diff").Output(); dErr == nil && len(diffOutput) > 0 {
			p.Context.Diff = string(diffOutput)
		}
	}

	// Parse structured meta
	if meta := pipeline.ParseMeta(output); meta != nil {
		if len(meta.FilesChanged) > 0 {
			p.Context.FilesChanged = append(p.Context.FilesChanged, meta.FilesChanged...)
		}
		if len(meta.Decisions) > 0 {
			p.Context.Decisions = append(p.Context.Decisions, meta.Decisions...)
		}
	}

	return output, usage, nil
}

// runMission executes the full KND Council pipeline from the CLI.
// It deploys agents sequentially, accumulates context, and applies risk gates.
func runMission(task string) {
	// Acquire WIP lock
	release, lockErr := acquireMissionLock(missionForce)
	if lockErr != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", lockErr)
		return
	}
	defer release()

	// Set up graceful shutdown via SIGINT/SIGTERM
	missionCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("🌙 KND Council — Mission Pipeline")
	fmt.Printf("   Task: %s\n\n", task)

	// Load agents
	reg := loadAgentRegistry()

	// Discover project context
	cwd := mustGetwd()
	ctx := discovery.Discover(cwd)
	if ctx.HasSpecs() || ctx.HasSteering() {
		fmt.Printf("   Project: %s\n\n", ctx.Summary())
	}

	// Create pipeline
	p := pipeline.New(task)

	// Apply parallel specialist configuration from config and CLI flags.
	cfg := config.Load()
	p.ParallelSpecialists = cfg.ParallelSpecialists
	p.MaxSpecialistConcurrency = cfg.MaxSpecialistConcurrency
	if missionSequential {
		p.ParallelSpecialists = false
	}

	// Create flywheel logger
	flywheel := pipeline.NewFlywheelLog()

	// Trace: print TraceID at start
	if missionTrace {
		fmt.Printf("   [trace] TraceID: %s\n", p.TraceID)
		fmt.Printf("   [trace] PhaseTimeout: %s, MaxOutputSize: %d\n\n", p.PhaseTimeout, p.MaxOutputSize)
	}

	// Execute phases via the shared pipeline loop
	runPipelineLoop(missionCtx, p, reg, ctx, flywheel, task, pipelineLoopOptions{
		handleConditionals:   true,
		advanceAfterPhase:    true,
		reworkOnRisk:         true,
		runConditionalParallel: true,
	})

	// Enhancement 6: Run conditional phases in parallel
	runConditionalPhasesParallel(p, reg, ctx, missionCtx)

	// Final summary
	fmt.Println()
	if p.IsComplete() || p.Context.RiskLevel == string(pipeline.RiskLow) {
		fmt.Println("   ✅ Mission pipeline complete.")
	}
	if len(p.Context.FilesChanged) > 0 {
		fmt.Printf("   Files touched: %s\n", strings.Join(p.Context.FilesChanged, ", "))
	}
}

// runMissionFast executes a collapsed pipeline: Implementation → QA only.
// Skips Analysis and Architecture for trivial/well-specified tasks.
func runMissionFast(task string) {
	// Acquire WIP lock
	release, lockErr := acquireMissionLock(missionForce)
	if lockErr != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", lockErr)
		return
	}
	defer release()

	// Set up graceful shutdown via SIGINT/SIGTERM
	missionCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("🌙 KND Council — Fast Mission (Implementation → QA)")
	fmt.Printf("   Task: %s\n\n", task)

	// Load agents
	reg := loadAgentRegistry()

	// Discover project context
	cwd := mustGetwd()
	ctx := discovery.Discover(cwd)
	if ctx.HasSpecs() || ctx.HasSteering() {
		fmt.Printf("   Project: %s\n\n", ctx.Summary())
	}

	// Fast pipeline: only Phase 3 (Implementation) and Phase 4 (QA) active
	p := pipeline.NewFast(task)

	// Create flywheel logger
	flywheel := pipeline.NewFlywheelLog()

	// Trace: print TraceID at start
	if missionTrace {
		fmt.Printf("   [trace] TraceID: %s\n\n", p.TraceID)
	}

	// Execute phases via the shared pipeline loop
	runPipelineLoop(missionCtx, p, reg, ctx, flywheel, task, pipelineLoopOptions{
		handleConditionals:   false,
		advanceAfterPhase:    false,
		reworkOnRisk:         false,
		runConditionalParallel: false,
	})

	fmt.Println("\n   ✅ Fast mission complete.")
}

// pipelineLoopOptions captures the behavioural differences between full and fast
// mission modes within the shared phase iteration loop.
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
}

// runPipelineLoop iterates a pipeline's phases, executing each via
// executeAndRecordPhase and applying risk gates, interrupt handling, and failure
// semantics based on the provided options.
//
// This is the shared core of both runMission (full) and runMissionFast (fast).
func runPipelineLoop(
	missionCtx context.Context,
	p *pipeline.Pipeline,
	reg *agents.Registry,
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

		// Handle phase skipping
		if opts.handleConditionals {
			// Full mode: evaluate conditional trigger
			if phase.Conditional {
				trigger := p.ShouldInvokeConditional(phase)
				if !trigger.Invoke {
					fmt.Printf("   ⏭️  Phase %d: %s — skipped (%s)\n", phase.Number, phase.Name, trigger.Reason)
					phase.Status = pipeline.StatusSkipped
					continue
				}
				fmt.Printf("   ⚡ Phase %d: %s — triggered (%s)\n", phase.Number, phase.Name, trigger.Reason)
			}
		} else {
			// Fast mode: skip already-skipped phases silently
			if phase.Status == pipeline.StatusSkipped {
				continue
			}
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
			if opts.reworkOnRisk {
				handlePhaseFailure(p, phase, err)
			} else {
				fmt.Printf("   ❌ Phase %d failed: %v\n", phase.Number, err)
			}
			break
		}

		// Budget enforcement (AC-6): check cumulative tokens after each phase.
		if usage != nil && budgetMax > 0 {
			missionTokens += usage.TotalTokens
			if missionTokens > budgetMax {
				fmt.Printf("   🛑 Token budget exceeded (%dK / %dK). Pipeline stopped.\n",
					missionTokens/1000, budgetMax/1000)
				flywheel.Append(pipeline.FlywheelEntry{
					Timestamp:        time.Now().UTC(),
					TraceID:          p.TraceID,
					Phase:            phase.Number,
					Agent:            phase.AgentName,
					Task:             task,
					Outcome:          "budget_exceeded",
					DurationMs:       phase.ElapsedTime().Milliseconds(),
					OutputSize:       len(output),
					PromptTokens:     usage.PromptTokens,
					CompletionTokens: usage.CompletionTokens,
					TotalTokens:      usage.TotalTokens,
					Model:            usage.Model,
				})
				home := mustUserHomeDir()
				checkpointDir := filepath.Join(home, ".moonbase", "checkpoints")
				pipeline.SaveCheckpoint(p, checkpointDir)
				return
			}
			pct := (missionTokens * 100) / budgetMax
			if pct >= warnPct {
				fmt.Printf("   ⚠️  Token budget: %d%% used (%dK / %dK)\n",
					pct, missionTokens/1000, budgetMax/1000)
			}
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
						DurationMs:  phase.ElapsedTime().Milliseconds(),
						OutputSize:  len(output),
						ReworkCount: p.Context.ReworkCount,
					})
					// Loop back — adjust i to re-run from the target phase
					i = targetIdx - 1 // -1 because loop will increment
					continue
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

// runConditionalPhasesParallel executes phases 6, 7, 8 concurrently.
// Enhancement 6: These phases are independent and can run in parallel.
func runConditionalPhasesParallel(p *pipeline.Pipeline, reg *agents.Registry, ctx *discovery.ProjectContext, missionCtx context.Context) {
	// Collect conditional phases that should trigger
	type conditionalWork struct {
		phase *pipeline.Phase
		agent *agents.Agent
	}
	var work []conditionalWork

	for i := range p.Phases {
		phase := &p.Phases[i]
		if !phase.Conditional || phase.Status != pipeline.StatusPending {
			continue
		}
		trigger := p.ShouldInvokeConditional(phase)
		if !trigger.Invoke {
			phase.Status = pipeline.StatusSkipped
			continue
		}
		agent := reg.GetByName(phase.AgentName)
		if agent == nil {
			phase.Status = pipeline.StatusSkipped
			continue
		}
		work = append(work, conditionalWork{phase, agent})
	}

	if len(work) == 0 {
		return
	}

	fmt.Printf("\n   ⚡ Running %d conditional phase(s) in parallel...\n", len(work))

	type result struct {
		phase  int
		output string
		err    error
	}
	results := make(chan result, len(work))

	for _, w := range work {
		go func(phase *pipeline.Phase, agent *agents.Agent) {
			phaseInput := p.Context.ForPhase(phase.Number)
			composed := discovery.ComposePrompt(agent.Prompt, ctx, phaseInput)
			output, _, err := backend.DeployComposed(missionCtx, composed, phaseInput, p.PhaseTimeout)
			results <- result{phase.Number, output, err}
		}(w.phase, w.agent)
	}

	// Collect results
	for range work {
		r := <-results
		for i := range p.Phases {
			if p.Phases[i].Number == r.phase {
				if r.err != nil {
					p.Phases[i].Status = pipeline.StatusFailed
					fmt.Printf("   ❌ Phase %d failed: %v\n", r.phase, r.err)
				} else {
					p.Phases[i].Status = pipeline.StatusComplete
					p.Context.RecordPhase(r.phase, r.output)
					fmt.Printf("   ✅ Phase %d complete (%d chars)\n", r.phase, len(r.output))
				}
				break
			}
		}
	}
}

// injectFileContext reads files mentioned in the Architecture output and injects
// their contents into the prompt. Enhancement 3: Pre-flight file injection.
func injectFileContext(pCtx *pipeline.PipelineContext) string {
	if len(pCtx.FilesChanged) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n--- PRE-FLIGHT FILE CONTEXT ---\n")
	sb.WriteString("These files were identified in the design phase. Current contents:\n\n")

	totalSize := 0
	const maxFileSize = 8000
	const maxTotalSize = 32000

	for _, f := range pCtx.FilesChanged {
		if totalSize >= maxTotalSize {
			sb.WriteString("\n...(remaining files omitted for context budget)\n")
			break
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) > maxFileSize {
			content = content[:maxFileSize] + "\n...(truncated)"
		}
		sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", f, content))
		totalSize += len(content)
	}

	sb.WriteString("--- END PRE-FLIGHT FILE CONTEXT ---\n")
	return sb.String()
}



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
