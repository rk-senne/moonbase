package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// Mission entry points: full, fast, and adaptive-depth pipeline runs.

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
	p.Depth = "override:full"

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
		handleConditionals:     true,
		advanceAfterPhase:      true,
		reworkOnRisk:           true,
		runConditionalParallel: true,
		allowEscalation:        false,
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
	p.Depth = "override:fast"

	// Create flywheel logger
	flywheel := pipeline.NewFlywheelLog()

	// Trace: print TraceID at start
	if missionTrace {
		fmt.Printf("   [trace] TraceID: %s\n\n", p.TraceID)
	}

	// Execute phases via the shared pipeline loop
	runPipelineLoop(missionCtx, p, reg, ctx, flywheel, task, pipelineLoopOptions{
		handleConditionals:     false,
		advanceAfterPhase:      false,
		reworkOnRisk:           false,
		runConditionalParallel: false,
		allowEscalation:        false,
	})

	fmt.Println("\n   ✅ Fast mission complete.")
}

// runMissionAdaptive executes the pipeline at the given adaptive depth.
// Enables mid-pipeline escalation when QA signals insufficient analysis.
func runMissionAdaptive(task string, depth pipeline.Depth, reason string) {
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

	fmt.Println("🌙 KND Council — Adaptive Mission")
	fmt.Printf("   Task: %s\n", task)
	fmt.Printf("   Depth: %s (%s)\n\n", depth, reason)

	// Load agents
	reg := loadAgentRegistry()

	// Discover project context
	cwd := mustGetwd()
	ctx := discovery.Discover(cwd)
	if ctx.HasSpecs() || ctx.HasSteering() {
		fmt.Printf("   Project: %s\n\n", ctx.Summary())
	}

	// Create adaptive pipeline
	p := pipeline.NewAdaptive(task, depth, reason)

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

	// Execute phases via the shared pipeline loop with escalation enabled
	runPipelineLoop(missionCtx, p, reg, ctx, flywheel, task, pipelineLoopOptions{
		handleConditionals:     true,
		advanceAfterPhase:      true,
		reworkOnRisk:           true,
		runConditionalParallel: true,
		allowEscalation:        true,
	})

	// Enhancement 6: Run conditional phases in parallel
	runConditionalPhasesParallel(p, reg, ctx, missionCtx)

	// Final summary
	fmt.Println()
	if p.IsComplete() || p.Context.RiskLevel == string(pipeline.RiskLow) {
		fmt.Println("   ✅ Adaptive mission complete.")
	}
	if p.Escalated {
		fmt.Printf("   📈 Escalated: %s → %s\n", p.OrigDepth, p.Depth)
	}
	if len(p.Context.FilesChanged) > 0 {
		fmt.Printf("   Files touched: %s\n", strings.Join(p.Context.FilesChanged, ", "))
	}
}

// pipelineLoopOptions captures the behavioural differences between full and fast
// mission modes within the shared phase iteration loop.
// shouldSilentlySkip reports whether a phase should be skipped without execution
// or messaging, given the loop mode. In adaptive/full mode (handleConditionals),
// only NON-conditional phases pre-skipped by the depth profile are silently skipped
// — conditional phases still have their triggers evaluated. In fast mode, any
// already-skipped phase is skipped. This is the runtime enforcement of adaptive depth.
func shouldSilentlySkip(phase pipeline.Phase, handleConditionals bool) bool {
	if handleConditionals {
		return phase.Status == pipeline.StatusSkipped && !phase.Conditional
	}
	return phase.Status == pipeline.StatusSkipped
}
