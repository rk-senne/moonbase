package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
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
		Depth:       string(p.Depth),
		DepthReason: p.DepthReason,
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
