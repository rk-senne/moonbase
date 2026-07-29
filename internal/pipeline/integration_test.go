//go:build integration

package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rk-senne/moonbase/internal/agents"
	"github.com/rk-senne/moonbase/internal/discovery"
)

// TestEndToEnd_FullPipeline simulates a complete pipeline run:
// 1. Load real agents from agents/ directory
// 2. Discover real project context (moonbase itself)
// 3. Run pipeline with mock backend
// 4. Verify risk gate, context accumulation, and phase routing
func TestEndToEnd_FullPipeline(t *testing.T) {
	// Find project root
	projectRoot := findProjectRoot(t)
	if projectRoot == "" {
		t.Skip("project root not found")
	}

	// 1. Load real agents
	agentsDir := filepath.Join(projectRoot, "agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		t.Skip("agents directory not found")
	}

	var agentList []agents.Agent
	files, _ := filepath.Glob(filepath.Join(agentsDir, "*.md"))
	for _, f := range files {
		a, err := agents.ParseAgentFile(f)
		if err != nil {
			t.Fatalf("failed to parse agent %s: %v", f, err)
		}
		agentList = append(agentList, *a)
	}

	if len(agentList) != 14 {
		t.Fatalf("expected 14 agents, got %d", len(agentList))
	}

	// 2. Discover project context
	ctx, err := discovery.Discover(projectRoot)
	if err != nil {
		t.Fatalf("discovery failed: %v", err)
	}

	if !ctx.HasSpecs() {
		t.Error("expected moonbase to have specs")
	}
	if !ctx.HasSteering() {
		t.Error("expected moonbase to have steering rules")
	}
	if ctx.Stack.Language != "go" {
		t.Errorf("expected Go stack, got: %s", ctx.Stack.Language)
	}

	// 3. Run pipeline
	p := New("Add user preferences feature to moonbase config")

	// Verify pipeline state
	if len(p.Phases) != 8 {
		t.Fatalf("expected 8 phases, got %d", len(p.Phases))
	}

	// Simulate Phase 1 (Analyst) output
	phase1Output := `# Numbuh 1 Mission Brief

## Mission Objective
Add user preferences (theme, default backend, agent order) to moonbase config.

## Acceptance Criteria
AC-1.1: WHEN user sets a preference THEN it persists across sessions
AC-1.2: WHEN config file is missing THEN defaults are used
AC-1.3: WHEN config has invalid values THEN a warning is shown

## Scope
### In Scope
- Config file structure
- Theme preference
- Default backend preference
### Out of Scope
- UI for editing preferences (future mission)

## Risks
- Config file location varies by OS

## Handoff
NEXT_AGENT: numbuh-2
RISK: LOW`

	p.Context.RecordPhase(1, phase1Output)
	p.Advance()

	// Verify context accumulation
	phase2Input := p.Context.ForPhase(2)
	if !strings.Contains(phase2Input, "AC-1.1") {
		t.Error("phase 2 input should contain ACs from phase 1")
	}
	if !strings.Contains(phase2Input, "user preferences") {
		t.Error("phase 2 input should contain the original task")
	}

	// Simulate Phase 2 (Architect) output
	phase2Output := `# Numbuh 2 Design Blueprint

## Proposed Approach
Add a config struct with YAML serialization. Store at ~/.config/moonbase/preferences.yaml.

## File / Module Impact
- internal/config/config.go (modify)
- internal/config/preferences.go (new)
- internal/config/preferences_test.go (new)

## Implementation Steps
1. Define preferences struct
2. Add load/save functions
3. Wire into TUI app initialization

## Handoff to Numbuh 3
Implement preferences.go with Load/Save + tests.`

	p.Context.RecordPhase(2, phase2Output)
	p.Advance()

	// Verify files extracted (from phase 3 output which has cleaner paths)
	phase3Input := p.Context.ForPhase(3)
	if !strings.Contains(phase3Input, "Design") {
		t.Error("phase 3 input should contain design from phase 2")
	}

	// Simulate Phase 3 (Implementer) output
	phase3Output := `# Numbuh 3 Implementation Report

## What Changed
Added preferences support with YAML persistence.

## Files Updated
- internal/config/preferences.go
- internal/config/preferences_test.go
- internal/config/config.go

## Tests / Build Run
go test ./internal/config/... — PASS (3 new tests)
go build ./... — PASS

## Handoff to Numbuh 4
Ready for QA verification.`

	p.Context.RecordPhase(3, phase3Output)
	p.Advance()

	// Simulate Phase 4 (QA) output — LOW risk
	phase4Output := `# Numbuh 4 QA Risk Report

## Verdict
LOW

## What I Tested
- Config loading with valid YAML
- Config loading with missing file (defaults)
- Config loading with invalid YAML (warning shown)

## Acceptance Criteria Check
- [x] AC-1.1: Preferences persist
- [x] AC-1.2: Defaults used when missing
- [x] AC-1.3: Warning on invalid

## Route
→ Numbuh 5 (review)`

	// Apply risk gate
	routing, err := p.ApplyRiskGate(phase4Output)
	if err != nil {
		t.Fatalf("risk gate error: %v", err)
	}
	if routing.Level != RiskLow {
		t.Errorf("expected LOW risk, got: %s", routing.Level)
	}
	if routing.TargetPhase != 5 {
		t.Errorf("expected target phase 5, got: %d", routing.TargetPhase)
	}

	// Record QA and advance to review
	p.Context.RecordPhase(4, phase4Output)
	p.Advance()

	// Verify reviewer gets full context
	phase5Input := p.Context.ForPhase(5)
	if !strings.Contains(phase5Input, "user preferences") {
		t.Error("reviewer should see original task")
	}
	if !strings.Contains(phase5Input, "AC-1.1") {
		t.Error("reviewer should see requirements")
	}
	if !strings.Contains(phase5Input, "preferences.go") {
		t.Error("reviewer should see implementation details")
	}
	if !strings.Contains(phase5Input, "LOW") {
		t.Error("reviewer should see QA verdict")
	}

	// 4. Test conditional phase triggers
	// With only 3 files changed, Numbuh 0 should NOT trigger
	trigger0 := EvaluateTrigger(p.Phases[5].TriggerSpec, p.Context)
	if trigger0.Invoke {
		t.Error("expected Numbuh 0 NOT to trigger with only 3 files changed")
	}

	// Security trigger should NOT fire (no auth content)
	trigger274 := EvaluateTrigger(p.Phases[6].TriggerSpec, p.Context)
	if trigger274.Invoke {
		t.Error("expected Numbuh 274 NOT to trigger for config-only change")
	}

	// 5. Test prompt composition
	var analyst *agents.Agent
	for i := range agentList {
		if agentList[i].Name == "numbuh-1" {
			analyst = &agentList[i]
			break
		}
	}
	if analyst == nil {
		t.Fatal("numbuh-1 agent not found")
	}

	composed := discovery.ComposePrompt(analyst.Prompt, ctx, "Add user preferences")
	if !strings.Contains(composed, "Numbuh 1") {
		t.Error("composed prompt should contain agent identity")
	}
	if !strings.Contains(composed, "dev-rules") {
		t.Error("composed prompt should contain steering rules")
	}
	if !strings.Contains(composed, "Add user preferences") {
		t.Error("composed prompt should contain task")
	}
	if len(composed) > 100 {
		// Just verify it's substantial
		t.Logf("Composed prompt length: %d chars", len(composed))
	}
}

// TestEndToEnd_ReworkLoop tests a pipeline that goes through a rework cycle.
func TestEndToEnd_ReworkLoop(t *testing.T) {
	p := New("fix authentication bypass")

	// Fast forward through phases 1-3
	p.Context.RecordPhase(1, "Requirements: fix auth bypass")
	p.Advance()
	p.Context.RecordPhase(2, "Design: add input validation to auth middleware")
	p.Advance()
	p.Context.RecordPhase(3, "Implementation: added validation")
	p.Advance()

	// QA returns MEDIUM — back to implementation
	qaOutput := `# QA Report

## Verdict
MEDIUM

## Findings
Missing test for empty token case. Edge case not covered.

## Route
→ Back to Numbuh 3`

	routing, err := p.ApplyRiskGate(qaOutput)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing.Level != RiskMedium {
		t.Errorf("expected MEDIUM, got: %s", routing.Level)
	}
	if p.Context.ReworkCount != 1 {
		t.Errorf("expected rework count 1, got %d", p.Context.ReworkCount)
	}

	// Implementer gets the QA feedback
	p.Context.RecordPhase(4, qaOutput)
	phase3Input := p.Context.ForPhase(3)
	if !strings.Contains(phase3Input, "REWORK REQUIRED") {
		t.Error("implementer should see rework indication")
	}
	if !strings.Contains(phase3Input, "empty token") {
		t.Error("implementer should see specific QA feedback")
	}

	// Second implementation pass
	p.Context.RecordPhase(3, "Fixed: added empty token test, edge case covered")
	p.Advance()

	// QA passes this time
	qaPass := "## Verdict\nLOW\n\nAll cases covered now."
	routing2, err := p.ApplyRiskGate(qaPass)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing2.Level != RiskLow {
		t.Errorf("expected LOW on second pass, got: %s", routing2.Level)
	}
}

// TestEndToEnd_MaxReworkEscalation tests that the pipeline stops after max rework loops.
func TestEndToEnd_MaxReworkEscalation(t *testing.T) {
	p := New("fix the thing")
	p.MaxRework = 2

	// Simulate 2 rework loops
	p.Context.RecordPhase(1, "reqs")
	p.Advance()
	p.Context.RecordPhase(2, "design")
	p.Advance()
	p.Context.RecordPhase(3, "code")
	p.Advance()

	// First rework
	_, err := p.ApplyRiskGate("## Verdict\nMEDIUM\n\nNope.")
	if err != nil {
		t.Fatalf("first rework should not error: %v", err)
	}

	// Back to implementation, code again
	p.Context.RecordPhase(3, "code v2")
	p.Advance()

	// Second rework
	_, err = p.ApplyRiskGate("## Verdict\nMEDIUM\n\nStill nope.")
	if err != nil {
		t.Fatalf("second rework should not error: %v", err)
	}

	// Back to implementation, code again
	p.Context.RecordPhase(3, "code v3")
	p.Advance()

	// Third attempt — should fail (max exceeded)
	_, err = p.ApplyRiskGate("## Verdict\nMEDIUM\n\nNever gonna work.")
	if err == nil {
		t.Fatal("expected error on third rework (max 2)")
	}
	if !strings.Contains(err.Error(), "max rework") {
		t.Errorf("expected max rework error, got: %v", err)
	}
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	for dir := wd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "agents")); err == nil {
				return dir
			}
		}
	}
	return ""
}
