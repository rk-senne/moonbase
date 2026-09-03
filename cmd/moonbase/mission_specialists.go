package main

import (
	"context"
	"fmt"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/backend"
	"github.com/rk-senne/moonbase/internal/discovery"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

// Parallel fan-out of triggered conditional specialists.

// runConditionalPhasesParallel executes phases 6, 7, 8 concurrently.
// Enhancement 6: These phases are independent and can run in parallel.
func runConditionalPhasesParallel(p *pipeline.Pipeline, reg agentLookup, ctx *discovery.ProjectContext, missionCtx context.Context) {
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
