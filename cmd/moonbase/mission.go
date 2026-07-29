package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	clip "github.com/rk-senne/moonbase/internal/clipboard"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// executeAndRecordPhase runs a single phase and records results to flywheel.
// Returns the output string and any error.
func executeAndRecordPhase(
	p *pipeline.Pipeline,
	phase *pipeline.Phase,
	agent *agents.Agent,
	ctx *discovery.ProjectContext,
	flywheel *pipeline.FlywheelLog,
	task string,
) (string, error) {
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
	output, err := deployToBackend(agent, composed, phaseInput, p.PhaseTimeout)

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
		return "", err
	}

	p.Context.RecordPhase(phase.Number, output)
	phase.CompletePhase()

	flywheel.Append(pipeline.FlywheelEntry{
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
	})

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

	return output, nil
}

// runMission executes the full KND Council pipeline from the CLI.
// It deploys agents sequentially, accumulates context, and applies risk gates.
func runMission(task string) {
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

	// Create flywheel logger
	flywheel := pipeline.NewFlywheelLog()

	// Trace: print TraceID at start
	if missionTrace {
		fmt.Printf("   [trace] TraceID: %s\n", p.TraceID)
		fmt.Printf("   [trace] PhaseTimeout: %s, MaxOutputSize: %d\n\n", p.PhaseTimeout, p.MaxOutputSize)
	}

	// Execute mandatory phases (1-5)
	for i := 0; i < len(p.Phases); i++ {
		phase := &p.Phases[i]

		// Skip conditional phases if not triggered
		if phase.Conditional {
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
			fmt.Printf("   ⚠️  Phase %d: agent %s not found, skipping\n", phase.Number, phase.AgentName)
			phase.Status = pipeline.StatusSkipped
			continue
		}

		fmt.Printf("   🔄 Phase %d: %s (%s)...\n", phase.Number, phase.Name, agent.Designation)

		if missionTrace {
			fmt.Printf("   [trace] Phase %d started at %s\n", phase.Number, time.Now().Format(time.RFC3339))
		}

		output, err := executeAndRecordPhase(p, phase, agent, ctx, flywheel, task)
		if err != nil {
			handlePhaseFailure(p, phase, err)
			break
		}

		if missionTrace {
			fmt.Printf("   [trace] Phase %d completed at %s (elapsed: %s)\n", phase.Number, phase.CompletedAt.Format(time.RFC3339), phase.ElapsedTime().Round(time.Millisecond))
			fmt.Printf("   [trace] Phase %d output size: %d bytes\n", phase.Number, len(output))
		}

		fmt.Printf("   ✅ Phase %d complete (%d chars)\n", phase.Number, len(output))

		// Advance pipeline state to keep Current in sync
		p.Advance()

		// Apply risk gate after QA (phase 4)
		if phase.Number == 4 {
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
		}
	}

	// Save checkpoint after pipeline execution
	home := mustUserHomeDir()
	checkpointDir := filepath.Join(home, ".moonbase", "checkpoints")
	pipeline.SaveCheckpoint(p, checkpointDir)

	// Enhancement 6: Run conditional phases in parallel
	runConditionalPhasesParallel(p, reg, ctx)

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

	for i := range p.Phases {
		phase := &p.Phases[i]

		// Only run non-skipped phases (3 and 4)
		if phase.Status == pipeline.StatusSkipped {
			continue
		}

		agent := reg.GetByName(phase.AgentName)
		if agent == nil {
			fmt.Printf("   ⚠️  Phase %d: agent %s not found\n", phase.Number, phase.AgentName)
			continue
		}

		fmt.Printf("   🔄 Phase %d: %s (%s)...\n", phase.Number, phase.Name, agent.Designation)

		if missionTrace {
			fmt.Printf("   [trace] Phase %d started at %s\n", phase.Number, time.Now().Format(time.RFC3339))
		}

		output, err := executeAndRecordPhase(p, phase, agent, ctx, flywheel, task)
		if err != nil {
			fmt.Printf("   ❌ Phase %d failed: %v\n", phase.Number, err)
			break
		}

		if missionTrace {
			fmt.Printf("   [trace] Phase %d completed at %s (elapsed: %s)\n", phase.Number, phase.CompletedAt.Format(time.RFC3339), phase.ElapsedTime().Round(time.Millisecond))
			fmt.Printf("   [trace] Phase %d output size: %d bytes\n", phase.Number, len(output))
		}

		fmt.Printf("   ✅ Phase %d complete (%d chars)\n", phase.Number, len(output))

		// Apply risk gate after QA
		if phase.Number == 4 {
			routing, _ := p.ApplyRiskGate(output)
			fmt.Printf("   🎯 Risk Gate: %s — %s\n", routing.Level, routing.Action)
			if routing.Level == pipeline.RiskCritical || routing.Level == pipeline.RiskHigh {
				fmt.Println("\n   ⚠️  High risk on fast mission — consider running full pipeline.")
			}
		}
	}

	// Save checkpoint after pipeline execution
	home := mustUserHomeDir()
	checkpointDir := filepath.Join(home, ".moonbase", "checkpoints")
	pipeline.SaveCheckpoint(p, checkpointDir)

	fmt.Println("\n   ✅ Fast mission complete.")
}

// runConditionalPhasesParallel executes phases 6, 7, 8 concurrently.
// Enhancement 6: These phases are independent and can run in parallel.
func runConditionalPhasesParallel(p *pipeline.Pipeline, reg *agents.Registry, ctx *discovery.ProjectContext) {
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
			output, err := deployToBackend(agent, composed, phaseInput, p.PhaseTimeout)
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

// deployToBackend deploys a pre-composed prompt via the preferred backend with retry.
// Uses the backend package's retry wrapper (exponential backoff with jitter).
//
// If the backend implements backend.RawDeployer, it sends the pre-composed prompt
// directly (avoiding double-composition). Otherwise, it passes the composed prompt
// as the task parameter with an empty agent, which is acceptable for backends that
// treat system+task as a single conversation turn.
//
// The phaseTimeout parameter bounds the total retry budget: the overall context
// deadline is set to phaseTimeout, and per-attempt timeouts are derived as
// phaseTimeout / DefaultMaxAttempts (capped at a sensible minimum of 30s).
//
// Falls back to clipboard + stdin if no AI backend is available.
// SECURITY: All subprocess execution uses SafeEnv() via the backend package.
func deployToBackend(agent *agents.Agent, composed string, task string, phaseTimeout time.Duration) (string, error) {
	be := backend.Preferred()

	if be.Name() != "clipboard" {
		// Derive per-attempt timeout from the phase timeout so the total retry
		// budget never exceeds the phase timeout.
		perAttempt := phaseTimeout / time.Duration(backend.DefaultMaxAttempts)
		if perAttempt < 30*time.Second {
			perAttempt = 30 * time.Second
		}

		ctx, cancel := context.WithTimeout(context.Background(), phaseTimeout)
		defer cancel()

		output, err := backend.WithRetryCtx(ctx, func() (string, error) {
			attemptCtx, attemptCancel := context.WithTimeout(ctx, perAttempt)
			defer attemptCancel()

			var result string
			var deployErr error

			// Use RawDeployer if available to avoid double-composing the prompt.
			if raw, ok := be.(backend.RawDeployer); ok {
				result, deployErr = raw.DeployRaw(composed, task)
			} else {
				// Fallback: pass composed prompt as task with the agent.
				// This causes a double-composition, but it's the best we can do
				// for backends that don't implement RawDeployer.
				result, deployErr = be.Deploy(*agent, nil, composed)
			}

			if deployErr != nil {
				if attemptCtx.Err() == context.DeadlineExceeded {
					return "", fmt.Errorf("backend timed out after %s: %w", perAttempt, deployErr)
				}
				return "", deployErr
			}

			if attemptCtx.Err() != nil {
				return "", fmt.Errorf("deploy cancelled: %w", attemptCtx.Err())
			}
			return result, nil
		}, backend.DefaultMaxAttempts)

		if err != nil {
			return "", fmt.Errorf("%s failed after %d attempts: %w", be.Name(), backend.DefaultMaxAttempts, err)
		}
		return output, nil
	}

	// Clipboard backend — fall back to clipboard/stdin
	return fallbackDeploy(composed, task)
}

// fallbackDeploy handles the case where no AI backend (kiro-cli) is available.
// It copies the composed prompt to the clipboard and reads the response from stdin.
// Returns the user-provided response or an error if no fallback mechanism is available.
func fallbackDeploy(composed, task string) (string, error) {
	if err := clip.Copy(composed); err == nil {
		fmt.Printf("\n   📋 Prompt copied to clipboard (%d chars).\n", len(composed))
		fmt.Printf("   Paste into your AI tool. When done, paste the response below.\n")
		fmt.Printf("   (Type END on a line by itself to finish, or press Ctrl+C to abort)\n\n")

		// Read multi-line input until "END" with size limit
		var lines []string
		totalSize := 0
		const maxInputSize = 1 << 20 // 1MB
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "END" {
				break
			}
			totalSize += len(line) + 1
			if totalSize > maxInputSize {
				fmt.Fprintf(os.Stderr, "   ⚠️  Input truncated at 1MB\n")
				break
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n"), nil
	}

	return "", fmt.Errorf("no backend available — install kiro-cli or ensure clipboard is accessible")
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
