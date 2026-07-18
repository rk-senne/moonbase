package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/f5508037/moonbase/internal/agents"
	"github.com/f5508037/moonbase/internal/backend"
	clip "github.com/f5508037/moonbase/internal/clipboard"
	"github.com/f5508037/moonbase/internal/discovery"
	"github.com/f5508037/moonbase/internal/pipeline"
)

// runMission executes the full KND Council pipeline from the CLI.
// It deploys agents sequentially, accumulates context, and applies risk gates.
func runMission(task string) {
	fmt.Println("🌙 KND Council — Mission Pipeline")
	fmt.Printf("   Task: %s\n\n", task)

	// Load agents
	reg := loadAgentRegistry()

	// Discover project context
	cwd, _ := os.Getwd()
	ctx, _ := discovery.Discover(cwd)
	if ctx != nil && (ctx.HasSpecs() || ctx.HasSteering()) {
		fmt.Printf("   Project: %s\n\n", ctx.Summary())
	}

	// Create pipeline
	p := pipeline.New(task)

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
		phase.StartPhase()

		if missionTrace {
			fmt.Printf("   [trace] Phase %d started at %s\n", phase.Number, phase.StartedAt.Format(time.RFC3339))
		}

		// Enhancement 3: Pre-flight file injection for Phase 3 (Implementation)
		phaseInput := p.Context.ForPhase(phase.Number)
		if phase.Number == 3 {
			fileContext := injectFileContext(p.Context)
			if fileContext != "" {
				phaseInput += fileContext
			}
		}

		// Enhancement 5: Inject git diff for Phase 4 (QA)
		if phase.Number == 4 && p.Context.Diff != "" {
			phaseInput += fmt.Sprintf("\n\n## Actual Changes (git diff)\n\n```diff\n%s\n```", p.Context.Diff)
		}

		// Compose full prompt: steering + agent + context
		composed := discovery.ComposePrompt(agent.Prompt, ctx, phaseInput)

		// Deploy to backend
		output, err := deployToBackend(agent, composed, phaseInput)
		if err != nil {
			handlePhaseFailure(p, phase, err)
			break
		}

		// Record output
		p.Context.RecordPhase(phase.Number, output)
		phase.CompletePhase()

		if missionTrace {
			fmt.Printf("   [trace] Phase %d completed at %s (elapsed: %s)\n", phase.Number, phase.CompletedAt.Format(time.RFC3339), phase.ElapsedTime().Round(time.Millisecond))
			fmt.Printf("   [trace] Phase %d output size: %d bytes\n", phase.Number, len(output))
		}

		// Enhancement 4: Parse structured meta from agent output
		if meta := pipeline.ParseMeta(output); meta != nil {
			if len(meta.FilesChanged) > 0 {
				p.Context.FilesChanged = append(p.Context.FilesChanged, meta.FilesChanged...)
			}
			if len(meta.Decisions) > 0 {
				p.Context.Decisions = append(p.Context.Decisions, meta.Decisions...)
			}
		}

		// Enhancement 5: Capture git diff after Phase 3 completes
		if phase.Number == 3 {
			if diffOutput, dErr := exec.Command("git", "diff").Output(); dErr == nil && len(diffOutput) > 0 {
				p.Context.Diff = string(diffOutput)
			}
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
				// Loop back — adjust i to re-run from the target phase
				i = targetIdx - 1 // -1 because loop will increment
				continue
			}
		}
	}

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
	cwd, _ := os.Getwd()
	ctx, _ := discovery.Discover(cwd)
	if ctx != nil && (ctx.HasSpecs() || ctx.HasSteering()) {
		fmt.Printf("   Project: %s\n\n", ctx.Summary())
	}

	// Fast pipeline: only Phase 3 (Implementation) and Phase 4 (QA)
	fastPhases := []pipeline.Phase{
		{Number: 3, Name: "Implementation", Operative: "Numbuh 3", AgentName: "numbuh-3", Status: pipeline.StatusPending},
		{Number: 4, Name: "QA", Operative: "Numbuh 4", AgentName: "numbuh-4", Status: pipeline.StatusPending},
	}

	p := pipeline.New(task)

	// Trace: print TraceID at start
	if missionTrace {
		fmt.Printf("   [trace] TraceID: %s\n\n", p.TraceID)
	}

	// Skip phases 1, 2, 5 and all conditionals
	for i := range p.Phases {
		p.Phases[i].Status = pipeline.StatusSkipped
	}

	for _, phase := range fastPhases {
		agent := reg.GetByName(phase.AgentName)
		if agent == nil {
			fmt.Printf("   ⚠️  Phase %d: agent %s not found\n", phase.Number, phase.AgentName)
			continue
		}

		fmt.Printf("   🔄 Phase %d: %s (%s)...\n", phase.Number, phase.Name, agent.Designation)
		phase.StartPhase()

		if missionTrace {
			fmt.Printf("   [trace] Phase %d started at %s\n", phase.Number, phase.StartedAt.Format(time.RFC3339))
		}

		phaseInput := p.Context.ForPhase(phase.Number)

		// Enhancement 5: Inject diff for QA
		if phase.Number == 4 && p.Context.Diff != "" {
			phaseInput += fmt.Sprintf("\n\n## Actual Changes (git diff)\n\n```diff\n%s\n```", p.Context.Diff)
		}

		composed := discovery.ComposePrompt(agent.Prompt, ctx, phaseInput)
		output, err := deployToBackend(agent, composed, phaseInput)
		if err != nil {
			fmt.Printf("   ❌ Phase %d failed: %v\n", phase.Number, err)
			break
		}

		p.Context.RecordPhase(phase.Number, output)
		phase.CompletePhase()

		if missionTrace {
			fmt.Printf("   [trace] Phase %d completed at %s (elapsed: %s)\n", phase.Number, phase.CompletedAt.Format(time.RFC3339), phase.ElapsedTime().Round(time.Millisecond))
			fmt.Printf("   [trace] Phase %d output size: %d bytes\n", phase.Number, len(output))
		}

		// Capture diff after implementation
		if phase.Number == 3 {
			if diffOutput, dErr := exec.Command("git", "diff").Output(); dErr == nil && len(diffOutput) > 0 {
				p.Context.Diff = string(diffOutput)
			}
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
			output, err := deployToBackend(agent, composed, phaseInput)
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
// Uses the backend package's retry wrapper (exponential backoff with jitter) and
// the Kiro.DeployRaw method to avoid double-composing the prompt.
//
// Falls back to clipboard + stdin if no AI backend is available.
// SECURITY: All subprocess execution uses SafeEnv() via the backend package.
func deployToBackend(agent *agents.Agent, composed string, task string) (string, error) {
	kiroBackend := &backend.Kiro{TrustTools: true}

	if kiroBackend.Available() {
		// Use a timeout context for the entire retry sequence (120s per attempt × 3 attempts).
		ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
		defer cancel()

		output, err := backend.WithRetryCtx(ctx, func() (string, error) {
			// Per-attempt timeout: 120s
			attemptCtx, attemptCancel := context.WithTimeout(ctx, 120*time.Second)
			defer attemptCancel()

			result, deployErr := kiroBackend.DeployRaw(composed, task)
			if deployErr != nil {
				// Check if the attempt timed out
				if attemptCtx.Err() == context.DeadlineExceeded {
					return "", fmt.Errorf("backend timed out after 120s: %w", deployErr)
				}
				return "", deployErr
			}

			// Respect attempt-level context
			if attemptCtx.Err() != nil {
				return "", fmt.Errorf("deploy cancelled: %w", attemptCtx.Err())
			}
			return result, nil
		}, backend.DefaultMaxAttempts)

		if err != nil {
			return "", fmt.Errorf("kiro-cli failed after %d attempts: %w", backend.DefaultMaxAttempts, err)
		}
		return output, nil
	}

	// No kiro-cli available — fall back to clipboard/stdin
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
