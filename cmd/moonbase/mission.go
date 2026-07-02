package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	dir := agentsDir()
	reg := agents.NewRegistry(dir)
	loadCmd := reg.Load()
	msg := loadCmd()
	if loaded, ok := msg.(agents.AgentsLoadedMsg); ok && loaded.Err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load agents: %v\n", loaded.Err)
		os.Exit(1)
	}

	// Discover project context
	cwd, _ := os.Getwd()
	ctx, _ := discovery.Discover(cwd)
	if ctx != nil && (ctx.HasSpecs() || ctx.HasSteering()) {
		fmt.Printf("   Project: %s\n\n", ctx.Summary())
	}

	// Create pipeline
	p := pipeline.New(task)

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
		phase.Status = pipeline.StatusRunning

		// Compose input for this phase
		phaseInput := p.Context.ForPhase(phase.Number)

		// Compose full prompt: steering + agent + context
		composed := discovery.ComposePrompt(agent.Prompt, ctx, phaseInput)

		// Deploy to backend
		output, err := deployToBackend(agent, composed, phaseInput)
		if err != nil {
			fmt.Printf("   ❌ Phase %d failed: %v\n", phase.Number, err)
			phase.Status = pipeline.StatusFailed
			p.Stop(err.Error())
			break
		}

		// Record output
		p.Context.RecordPhase(phase.Number, output)
		phase.Status = pipeline.StatusComplete

		// Show summary (first 200 chars)
		summary := strings.TrimSpace(output)
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		fmt.Printf("   ✅ Phase %d complete (%d chars)\n", phase.Number, len(output))

		// Apply risk gate after QA (phase 4)
		if phase.Number == 4 {
			routing, rErr := p.ApplyRiskGate(output)
			fmt.Printf("   🎯 Risk Gate: %s — %s\n", routing.Level, routing.Action)

			if routing.Level == pipeline.RiskCritical {
				fmt.Println("\n   🛑 CRITICAL RISK — Pipeline stopped. Escalating to human.")
				break
			}
			if rErr != nil {
				fmt.Printf("\n   🛑 %v\n", rErr)
				break
			}
			if routing.Level == pipeline.RiskMedium || routing.Level == pipeline.RiskHigh {
				// Loop back — adjust i to re-run from the target phase
				for j, ph := range p.Phases {
					if ph.Number == routing.TargetPhase {
						i = j - 1 // -1 because loop will increment
						break
					}
				}
				continue
			}
		}
	}

	// Final summary
	fmt.Println()
	if p.IsComplete() || p.Context.RiskLevel == string(pipeline.RiskLow) {
		fmt.Println("   ✅ Mission pipeline complete.")
	}
	if len(p.Context.FilesChanged) > 0 {
		fmt.Printf("   Files touched: %s\n", strings.Join(p.Context.FilesChanged, ", "))
	}
}

// deployToBackend tries kiro-cli, then falls back to clipboard with manual input.
// SECURITY: kiro-cli subprocess uses SafeEnv() to prevent env var leakage.
func deployToBackend(agent *agents.Agent, composed string, task string) (string, error) {
	// Try kiro-cli
	if kiro, err := exec.LookPath("kiro-cli"); err == nil {
		tmpFile, tErr := os.CreateTemp("", "moonbase-mission-*.md")
		if tErr == nil {
			tmpFile.WriteString(composed)
			tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			cmd := exec.Command(kiro, "chat",
				"--system-prompt", tmpFile.Name(),
				"--message", task,
			)
			// SECURITY: Use SafeEnv to prevent leaking sensitive env vars to subprocess.
			cmd.Env = backend.SafeEnv()

			output, err := cmd.CombinedOutput()
			if err == nil {
				return string(output), nil
			}
			// If kiro-cli fails with bad flags, try simpler invocation
		}
	}

	// Fallback: copy to clipboard and ask user to paste result
	if err := clip.Copy(composed); err == nil {
		fmt.Printf("\n   📋 Prompt copied to clipboard (%d chars).\n", len(composed))
		fmt.Printf("   Paste into your AI tool. When done, paste the response below.\n")
		fmt.Printf("   (Type END on a line by itself to finish, or press Ctrl+C to abort)\n\n")

		// Read multi-line input until "END"
		var lines []string
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "END" {
				break
			}
			lines = append(lines, line)
		}
		return strings.Join(lines, "\n"), nil
	}

	return "", fmt.Errorf("no backend available — install kiro-cli or ensure clipboard is accessible")
}
