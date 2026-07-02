package pipeline

import (
	"strings"
	"testing"
)

// --- PipelineContext tests ---

func TestNewPipelineContext(t *testing.T) {
	ctx := NewPipelineContext("build the widget")
	if ctx.Task != "build the widget" {
		t.Errorf("expected task 'build the widget', got: %s", ctx.Task)
	}
	if len(ctx.PhaseOutputs) != 0 {
		t.Error("expected empty phase outputs")
	}
}

func TestPipelineContext_RecordPhase(t *testing.T) {
	ctx := NewPipelineContext("test")
	ctx.RecordPhase(1, "## Requirements\n\nAC-1.1: Do the thing.")
	ctx.RecordPhase(2, "## Design\n\nFiles:\n- internal/agents/parser.go\n- internal/agents/registry.go")

	if len(ctx.PhaseOutputs) != 2 {
		t.Errorf("expected 2 phase outputs, got %d", len(ctx.PhaseOutputs))
	}
	if !strings.Contains(ctx.PhaseOutputs[1], "AC-1.1") {
		t.Error("expected phase 1 output to contain AC-1.1")
	}
	// Should extract files from phase 2 output
	if len(ctx.FilesChanged) != 2 {
		t.Errorf("expected 2 files changed, got %d: %v", len(ctx.FilesChanged), ctx.FilesChanged)
	}
}

func TestPipelineContext_ForPhase_Analyst(t *testing.T) {
	ctx := NewPipelineContext("add pagination")
	input := ctx.ForPhase(1)

	if !strings.Contains(input, "add pagination") {
		t.Error("expected task in phase 1 input")
	}
}

func TestPipelineContext_ForPhase_Architect(t *testing.T) {
	ctx := NewPipelineContext("add pagination")
	ctx.RecordPhase(1, "## Requirements\n\nAC-1.1: Support page/pageSize params")

	input := ctx.ForPhase(2)
	if !strings.Contains(input, "add pagination") {
		t.Error("expected task in phase 2 input")
	}
	if !strings.Contains(input, "AC-1.1") {
		t.Error("expected requirements in phase 2 input")
	}
}

func TestPipelineContext_ForPhase_Implementer_Rework(t *testing.T) {
	ctx := NewPipelineContext("add pagination")
	ctx.RecordPhase(1, "Requirements here")
	ctx.RecordPhase(2, "Design here")
	ctx.RecordPhase(4, "## Verdict\nMEDIUM\n\nMissing edge case test.")
	ctx.ReworkCount = 1

	input := ctx.ForPhase(3)
	if !strings.Contains(input, "REWORK REQUIRED") {
		t.Error("expected rework indication in phase 3 input after rework")
	}
	if !strings.Contains(input, "Missing edge case test") {
		t.Error("expected QA feedback in rework input")
	}
}

// --- Risk Gate tests ---

func TestParseRiskGate_Low(t *testing.T) {
	output := "# Numbuh 4 QA Risk Report\n\n## Verdict\nLOW\n\n## What I Tested\nEverything."
	routing := ParseRiskGate(output)

	if routing.Level != RiskLow {
		t.Errorf("expected LOW, got: %s", routing.Level)
	}
	if routing.TargetPhase != 5 {
		t.Errorf("expected target phase 5, got: %d", routing.TargetPhase)
	}
}

func TestParseRiskGate_Medium(t *testing.T) {
	output := "# QA Report\n\n## Verdict\n\nMEDIUM\n\nMissing test for edge case."
	routing := ParseRiskGate(output)

	if routing.Level != RiskMedium {
		t.Errorf("expected MEDIUM, got: %s", routing.Level)
	}
	if routing.TargetPhase != 3 {
		t.Errorf("expected target phase 3, got: %d", routing.TargetPhase)
	}
}

func TestParseRiskGate_High(t *testing.T) {
	output := "## Verdict\nHIGH\n\nDesign is fundamentally flawed."
	routing := ParseRiskGate(output)

	if routing.Level != RiskHigh {
		t.Errorf("expected HIGH, got: %s", routing.Level)
	}
	if routing.TargetPhase != 2 {
		t.Errorf("expected target phase 2, got: %d", routing.TargetPhase)
	}
}

func TestParseRiskGate_Critical(t *testing.T) {
	output := "# Critical Stop\n\n## Verdict\nCRITICAL\n\nCredentials exposed in log output."
	routing := ParseRiskGate(output)

	if routing.Level != RiskCritical {
		t.Errorf("expected CRITICAL, got: %s", routing.Level)
	}
	if routing.TargetPhase != 0 {
		t.Errorf("expected target phase 0 (stop), got: %d", routing.TargetPhase)
	}
}

func TestParseRiskGate_InlineVerdict(t *testing.T) {
	output := "Some text\nVerdict: LOW\nMore text"
	routing := ParseRiskGate(output)

	if routing.Level != RiskLow {
		t.Errorf("expected LOW from inline verdict, got: %s", routing.Level)
	}
}

func TestParseRiskGate_Unknown(t *testing.T) {
	output := "I tested stuff. Looks okay I guess."
	routing := ParseRiskGate(output)

	// Unknown defaults to MEDIUM (cautious)
	if routing.Level != RiskMedium {
		t.Errorf("expected MEDIUM (default for unknown), got: %s", routing.Level)
	}
}

func TestParseRiskGate_MixedCase(t *testing.T) {
	output := "## Verdict\n\n**Low**\n\nAll good."
	routing := ParseRiskGate(output)

	if routing.Level != RiskLow {
		t.Errorf("expected LOW from mixed case, got: %s", routing.Level)
	}
}

// --- Pipeline orchestration tests ---

func TestPipeline_New(t *testing.T) {
	p := New("build the feature")

	if len(p.Phases) != 8 {
		t.Errorf("expected 8 phases, got %d", len(p.Phases))
	}
	if !p.Active {
		t.Error("expected pipeline to be active")
	}
	if p.Context.Task != "build the feature" {
		t.Error("expected context task to match")
	}
}

func TestPipeline_Advance(t *testing.T) {
	p := New("test")
	p.Phases[0].Status = StatusRunning

	p.Advance()

	if p.Phases[0].Status != StatusComplete {
		t.Error("expected phase 0 to be complete")
	}
	if p.Phases[1].Status != StatusRunning {
		t.Error("expected phase 1 to be running")
	}
	if p.Current != 1 {
		t.Errorf("expected current=1, got %d", p.Current)
	}
}

func TestPipeline_Skip(t *testing.T) {
	p := New("test")
	p.Current = 5 // Oversight (conditional)
	p.Phases[5].Status = StatusRunning

	p.Skip()

	if p.Phases[5].Status != StatusSkipped {
		t.Error("expected phase 5 to be skipped")
	}
	if p.Current != 6 {
		t.Errorf("expected current=6, got %d", p.Current)
	}
}

func TestPipeline_RouteToPhase(t *testing.T) {
	p := New("test")
	p.Current = 3 // QA

	err := p.RouteToPhase(3) // Back to implementation
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Current != 2 { // Phase 3 is index 2
		t.Errorf("expected current=2 (implementation), got %d", p.Current)
	}
	if p.Phases[2].Status != StatusRunning {
		t.Error("expected implementation phase to be running")
	}
	if p.Context.ReworkCount != 1 {
		t.Errorf("expected rework count 1, got %d", p.Context.ReworkCount)
	}
}

func TestPipeline_RouteToPhase_MaxRework(t *testing.T) {
	p := New("test")
	p.Current = 3
	p.Context.ReworkCount = 2 // Already at max

	err := p.RouteToPhase(3)
	if err == nil {
		t.Error("expected error when max rework exceeded")
	}
	if !strings.Contains(err.Error(), "max rework") {
		t.Errorf("expected max rework error, got: %v", err)
	}
}

func TestPipeline_ApplyRiskGate_Low(t *testing.T) {
	p := New("test")
	p.Current = 3 // QA phase (index 3, phase number 4)

	routing, err := p.ApplyRiskGate("## Verdict\nLOW\n\nAll clear.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing.Level != RiskLow {
		t.Errorf("expected LOW, got: %s", routing.Level)
	}
	// Pipeline should NOT have rerouted
	if p.Current != 3 {
		t.Errorf("expected current to stay at 3 for LOW risk, got %d", p.Current)
	}
}

func TestPipeline_ApplyRiskGate_Medium(t *testing.T) {
	p := New("test")
	p.Current = 3

	routing, err := p.ApplyRiskGate("## Verdict\nMEDIUM\n\nMissing tests.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing.Level != RiskMedium {
		t.Errorf("expected MEDIUM, got: %s", routing.Level)
	}
	// Should route back to phase 3 (implementation, index 2)
	if p.Current != 2 {
		t.Errorf("expected reroute to index 2, got %d", p.Current)
	}
}

func TestPipeline_ApplyRiskGate_Critical(t *testing.T) {
	p := New("test")
	p.Current = 3

	routing, err := p.ApplyRiskGate("## Verdict\nCRITICAL\n\nSecrets exposed!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing.Level != RiskCritical {
		t.Errorf("expected CRITICAL, got: %s", routing.Level)
	}
	if p.Active {
		t.Error("expected pipeline to be stopped for CRITICAL")
	}
}

func TestPipeline_Stop(t *testing.T) {
	p := New("test")
	p.Current = 2
	p.Phases[2].Status = StatusRunning

	p.Stop("something went wrong")

	if p.Active {
		t.Error("expected pipeline to be inactive after stop")
	}
	if p.Phases[2].Status != StatusFailed {
		t.Error("expected current phase to be failed")
	}
}

// --- Trigger tests ---

func TestEvaluateTrigger_FileCount(t *testing.T) {
	ctx := NewPipelineContext("big refactor")
	ctx.FilesChanged = []string{
		"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go",
	}

	result := EvaluateTrigger(">5 files changed, core logic changed", ctx)
	if !result.Invoke {
		t.Error("expected trigger to fire with 7 files changed")
	}
	if !strings.Contains(result.Reason, "more than 5 files") {
		t.Errorf("expected file count reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_Security(t *testing.T) {
	ctx := NewPipelineContext("add login")
	ctx.RecordPhase(3, "Added authentication middleware and JWT token validation")

	result := EvaluateTrigger("Auth/secrets/permissions changed, new endpoints, dependency CVEs", ctx)
	if !result.Invoke {
		t.Error("expected security trigger to fire")
	}
	if !strings.Contains(result.Reason, "security") {
		t.Errorf("expected security reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_NotMet(t *testing.T) {
	ctx := NewPipelineContext("fix typo in readme")
	ctx.RecordPhase(3, "Fixed typo in README.md line 42")
	ctx.FilesChanged = []string{"README.md"}

	result := EvaluateTrigger(">5 files changed, core logic changed", ctx)
	if result.Invoke {
		t.Error("expected trigger NOT to fire for simple readme fix")
	}
}

func TestEvaluateTrigger_Empty(t *testing.T) {
	ctx := NewPipelineContext("test")
	result := EvaluateTrigger("", ctx)
	if result.Invoke {
		t.Error("expected empty trigger to not invoke")
	}
}

func TestEvaluateTrigger_DeploymentContent(t *testing.T) {
	ctx := NewPipelineContext("update CI")
	ctx.RecordPhase(3, "Modified the Dockerfile and docker-compose.yml")

	result := EvaluateTrigger("CI/CD changed, Docker/infra touched, new env vars", ctx)
	if !result.Invoke {
		t.Error("expected deployment trigger to fire")
	}
}

// --- extractFilesChanged tests ---

func TestExtractFilesChanged(t *testing.T) {
	output := `## Files Updated

- internal/agents/parser.go
- internal/agents/registry.go
- docs/README.md

Some other text here.`

	files := extractFilesChanged(output)
	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(files), files)
	}
}

func TestExtractFilesChanged_BacktickedPaths(t *testing.T) {
	output := "Files updated:\n`internal/pipeline/context.go`\n`internal/pipeline/riskgate.go`"
	files := extractFilesChanged(output)
	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(files), files)
	}
}

func TestExtractFilesChanged_NoFiles(t *testing.T) {
	output := "Everything looks good. No files were changed."
	files := extractFilesChanged(output)
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d: %v", len(files), files)
	}
}

// === Gap Coverage: StatusSummary, CurrentPhase, IsComplete, Advance end ===

func TestPipeline_StatusSummary(t *testing.T) {
	p := New("build the feature")
	p.Phases[0].Status = StatusComplete
	p.Phases[1].Status = StatusRunning
	p.Current = 1

	summary := p.StatusSummary()

	if !strings.Contains(summary, "build the feature") {
		t.Error("expected task in summary")
	}
	if !strings.Contains(summary, "✅") {
		t.Error("expected complete icon in summary")
	}
	if !strings.Contains(summary, "🔄") {
		t.Error("expected running icon in summary")
	}
	if !strings.Contains(summary, "⏳") {
		t.Error("expected pending icon in summary")
	}
	if !strings.Contains(summary, "Phase 1") {
		t.Error("expected Phase 1 in summary")
	}
	if !strings.Contains(summary, "Phase 8") {
		t.Error("expected Phase 8 in summary (8 phases total)")
	}
	if !strings.Contains(summary, "(conditional)") {
		t.Error("expected (conditional) tag for specialist phases")
	}
}

func TestPipeline_StatusSummary_AllStatuses(t *testing.T) {
	p := New("test all icons")
	p.Phases[0].Status = StatusComplete
	p.Phases[1].Status = StatusRunning
	p.Phases[2].Status = StatusSkipped
	p.Phases[3].Status = StatusFailed
	p.Phases[4].Status = StatusRework

	summary := p.StatusSummary()
	if !strings.Contains(summary, "✅") {
		t.Error("missing complete icon")
	}
	if !strings.Contains(summary, "🔄") {
		t.Error("missing running icon")
	}
	if !strings.Contains(summary, "⏭️") {
		t.Error("missing skipped icon")
	}
	if !strings.Contains(summary, "❌") {
		t.Error("missing failed icon")
	}
	if !strings.Contains(summary, "🔁") {
		t.Error("missing rework icon")
	}
}

func TestPipeline_CurrentPhase(t *testing.T) {
	p := New("test")
	p.Current = 0

	phase := p.CurrentPhase()
	if phase == nil {
		t.Fatal("expected non-nil current phase")
	}
	if phase.Name != "Analysis" {
		t.Errorf("expected 'Analysis', got: %s", phase.Name)
	}
	if phase.AgentName != "numbuh-1" {
		t.Errorf("expected agent 'numbuh-1', got: %s", phase.AgentName)
	}
}

func TestPipeline_CurrentPhase_OutOfBounds(t *testing.T) {
	p := New("test")
	p.Current = 99

	phase := p.CurrentPhase()
	if phase != nil {
		t.Error("expected nil for out-of-bounds current")
	}
}

func TestPipeline_CurrentPhase_Negative(t *testing.T) {
	p := New("test")
	p.Current = -1

	phase := p.CurrentPhase()
	if phase != nil {
		t.Error("expected nil for negative current")
	}
}

func TestPipeline_IsComplete_Active(t *testing.T) {
	p := New("test")
	if p.IsComplete() {
		t.Error("new pipeline should not be complete")
	}
}

func TestPipeline_IsComplete_Stopped(t *testing.T) {
	p := New("test")
	p.Stop("done")
	if !p.IsComplete() {
		t.Error("stopped pipeline should be complete")
	}
}

func TestPipeline_Advance_ToEnd(t *testing.T) {
	p := New("test")
	// Advance through all phases
	for i := 0; i < len(p.Phases); i++ {
		p.Phases[i].Status = StatusRunning
		p.Advance()
	}
	if p.Active {
		t.Error("pipeline should be inactive after advancing past all phases")
	}
	if p.Phases[len(p.Phases)-1].Status != StatusComplete {
		t.Error("last phase should be complete")
	}
}

func TestPipeline_Retry(t *testing.T) {
	p := New("test")
	p.Current = 2
	p.Phases[2].Status = StatusFailed

	p.Retry()

	if p.Phases[2].Status != StatusRunning {
		t.Error("expected retry to set phase back to running")
	}
	if p.Current != 2 {
		t.Error("expected current to stay the same on retry")
	}
}

func TestPipeline_ShouldInvokeConditional_Mandatory(t *testing.T) {
	p := New("test")
	// Phase 0 is mandatory (Analysis)
	result := p.ShouldInvokeConditional(&p.Phases[0])
	if !result.Invoke {
		t.Error("mandatory phase should always invoke")
	}
	if result.Reason != "mandatory phase" {
		t.Errorf("expected 'mandatory phase' reason, got: %s", result.Reason)
	}
}

func TestPipeline_ShouldInvokeConditional_NotTriggered(t *testing.T) {
	p := New("fix typo")
	p.Context.FilesChanged = []string{"README.md"}
	p.Context.RecordPhase(3, "Fixed typo in README")

	// Phase 5 is Oversight — ">5 files changed, core logic changed"
	result := p.ShouldInvokeConditional(&p.Phases[5])
	if result.Invoke {
		t.Error("oversight should NOT trigger for a typo fix")
	}
}

func TestPipeline_RouteToPhase_InvalidTarget(t *testing.T) {
	p := New("test")
	p.Current = 3

	err := p.RouteToPhase(99)
	if err == nil {
		t.Error("expected error for nonexistent target phase")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}
