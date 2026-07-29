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

// === Additional coverage tests ===

func TestPipelineContext_ForPhase_QA(t *testing.T) {
	ctx := NewPipelineContext("build widget")
	ctx.RecordPhase(1, "Requirements output")
	ctx.RecordPhase(3, "Implementation output with changes to internal/widget/widget.go")

	input := ctx.ForPhase(4)
	if !strings.Contains(input, "Requirements") {
		t.Error("expected requirements in phase 4 (QA) input")
	}
	if !strings.Contains(input, "Implementation") {
		t.Error("expected implementation output in phase 4 (QA) input")
	}
}

func TestPipelineContext_ForPhase_Review(t *testing.T) {
	ctx := NewPipelineContext("build widget")
	ctx.RecordPhase(1, "Requirements output")
	ctx.RecordPhase(2, "Design output")
	ctx.RecordPhase(3, "Implementation output")
	ctx.RecordPhase(4, "QA output all good")

	input := ctx.ForPhase(5)
	if !strings.Contains(input, "Requirements (from Phase 1)") {
		t.Error("expected requirements summary in phase 5 input")
	}
	if !strings.Contains(input, "Design (from Phase 2)") {
		t.Error("expected design summary in phase 5 input")
	}
	if !strings.Contains(input, "Implementation (from Phase 3)") {
		t.Error("expected implementation summary in phase 5 input")
	}
	if !strings.Contains(input, "QA Report (from Phase 4)") {
		t.Error("expected QA report in phase 5 input")
	}
}

func TestPipelineContext_ForPhase_Specialist(t *testing.T) {
	ctx := NewPipelineContext("big refactor")
	ctx.RecordPhase(1, "Requirements")
	ctx.RecordPhase(2, "Design")
	ctx.RecordPhase(3, "Implementation")
	ctx.RecordPhase(4, "QA")
	ctx.RecordPhase(5, "Review")

	// Phase > 5 gets the "default" specialist view
	input := ctx.ForPhase(6)
	if !strings.Contains(input, "Phase 1 Output") {
		t.Error("expected Phase 1 Output in specialist input")
	}
	if !strings.Contains(input, "Phase 5 Output") {
		t.Error("expected Phase 5 Output in specialist input")
	}
}

func TestPipelineContext_ForPhase_FilesChanged(t *testing.T) {
	ctx := NewPipelineContext("test")
	ctx.FilesChanged = []string{"internal/foo/bar.go", "internal/baz/qux.go"}

	input := ctx.ForPhase(2)
	if !strings.Contains(input, "Files Changed") {
		t.Error("expected Files Changed section")
	}
	if !strings.Contains(input, "internal/foo/bar.go") {
		t.Error("expected file path in Files Changed section")
	}
}

func TestPipelineContext_ForPhase_Implementer_NoRework(t *testing.T) {
	ctx := NewPipelineContext("add feature")
	ctx.RecordPhase(1, "Requirements")
	ctx.RecordPhase(2, "Design")
	// No rework, reworkCount = 0

	input := ctx.ForPhase(3)
	if strings.Contains(input, "REWORK") {
		t.Error("should NOT contain REWORK when rework count is 0")
	}
	if !strings.Contains(input, "Requirements") {
		t.Error("expected requirements in phase 3 input")
	}
	if !strings.Contains(input, "Design") {
		t.Error("expected design in phase 3 input")
	}
}

func TestSummarize_ShortOutput(t *testing.T) {
	// When output is shorter than maxLen, should return as-is
	short := "hello world"
	result := summarize(short, 100)
	if result != short {
		t.Errorf("expected %q, got %q", short, result)
	}
}

func TestSummarize_LongOutput(t *testing.T) {
	long := strings.Repeat("x", 200)
	result := summarize(long, 50)
	if len(result) <= 50 {
		t.Error("expected truncated result to be longer than maxLen due to suffix")
	}
	if !strings.Contains(result, "truncated") {
		t.Error("expected truncation notice")
	}
	if !strings.HasPrefix(result, strings.Repeat("x", 50)) {
		t.Error("expected result to start with first 50 chars")
	}
}

func TestContains_Hit(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !contains(slice, "b") {
		t.Error("expected contains to find 'b'")
	}
}

func TestContains_Miss(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if contains(slice, "z") {
		t.Error("expected contains to NOT find 'z'")
	}
}

func TestContains_Empty(t *testing.T) {
	if contains(nil, "a") {
		t.Error("expected contains to return false for nil slice")
	}
}

func TestPipeline_ApplyRiskGate_High(t *testing.T) {
	p := New("test")
	p.Current = 3

	routing, err := p.ApplyRiskGate("## Verdict\nHIGH\n\nNeeds redesign.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing.Level != RiskHigh {
		t.Errorf("expected HIGH, got: %s", routing.Level)
	}
	// Should route back to phase 2 (design, index 1)
	if p.Current != 1 {
		t.Errorf("expected reroute to index 1 (design), got %d", p.Current)
	}
}

func TestPipeline_ApplyRiskGate_Unknown(t *testing.T) {
	p := New("test")
	p.Current = 3

	routing, err := p.ApplyRiskGate("some random text without any risk keywords at all that is also very long to avoid the short-line heuristic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routing.Level != RiskMedium {
		t.Errorf("expected MEDIUM (from unknown default), got: %s", routing.Level)
	}
}

func TestPipeline_Skip_LastPhase(t *testing.T) {
	p := New("test")
	p.Current = len(p.Phases) - 1
	p.Phases[p.Current].Status = StatusRunning

	p.Skip()

	if p.Active {
		t.Error("expected pipeline to be inactive after skipping last phase")
	}
	if p.Phases[len(p.Phases)-1].Status != StatusSkipped {
		t.Error("expected last phase to be skipped")
	}
}

func TestStatusIcon_Default(t *testing.T) {
	// Use an invalid status to trigger the default case
	result := statusIcon(PhaseStatus(99))
	if result != "?" {
		t.Errorf("expected '?' for unknown status, got: %s", result)
	}
}

// === Additional trigger category tests ===

func TestEvaluateTrigger_Migration(t *testing.T) {
	ctx := NewPipelineContext("upgrade framework")
	ctx.RecordPhase(3, "Performed migration from v1 to v2 with breaking change in API")

	result := EvaluateTrigger("upgrade, migration, breaking change, deprecation, framework version", ctx)
	if !result.Invoke {
		t.Error("expected migration trigger to fire")
	}
	if !strings.Contains(result.Reason, "migration") {
		t.Errorf("expected migration reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_DeadCode(t *testing.T) {
	ctx := NewPipelineContext("cleanup")
	ctx.RecordPhase(3, "Removed dead code and unused imports, found zombie function")

	result := EvaluateTrigger("dead code, unused, stale, deprecated, zombie features", ctx)
	if !result.Invoke {
		t.Error("expected dead code trigger to fire")
	}
	if !strings.Contains(result.Reason, "tech debt") {
		t.Errorf("expected tech debt reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_EdgeCase(t *testing.T) {
	ctx := NewPipelineContext("improve parser")
	ctx.RecordPhase(3, "Added edge case handling for parser when input is empty or malformed")

	result := EvaluateTrigger("edge case, fragile, user-facing, parser, state machine", ctx)
	if !result.Invoke {
		t.Error("expected edge case trigger to fire")
	}
	if !strings.Contains(result.Reason, "edge case") {
		t.Errorf("expected edge case reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_Documentation(t *testing.T) {
	ctx := NewPipelineContext("add docs")
	ctx.RecordPhase(3, "Updated the README and added API documentation")

	result := EvaluateTrigger("readme, api doc, adr, changelog, onboarding", ctx)
	if !result.Invoke {
		t.Error("expected documentation trigger to fire")
	}
	if !strings.Contains(result.Reason, "documentation") {
		t.Errorf("expected documentation reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_Legacy(t *testing.T) {
	ctx := NewPipelineContext("fix old module")
	ctx.RecordPhase(3, "Touched legacy code that nobody knows how it works, used git blame")

	result := EvaluateTrigger("old, mysterious, undocumented, legacy, nobody-knows", ctx)
	if !result.Invoke {
		t.Error("expected legacy trigger to fire")
	}
	if !strings.Contains(result.Reason, "legacy") {
		t.Errorf("expected legacy reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_CoreLogic(t *testing.T) {
	ctx := NewPipelineContext("refactor")
	ctx.RecordPhase(3, "Refactored the state machine and introduced new design pattern for orchestration")

	result := EvaluateTrigger("core logic, orchestration, pipeline, architecture, new pattern", ctx)
	if !result.Invoke {
		t.Error("expected core logic trigger to fire")
	}
	if !strings.Contains(result.Reason, "core logic") {
		t.Errorf("expected core logic reason, got: %s", result.Reason)
	}
}

func TestEvaluateTrigger_MultipleReasons(t *testing.T) {
	ctx := NewPipelineContext("big change")
	ctx.FilesChanged = []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go"}
	ctx.RecordPhase(3, "Refactored core architecture with new design pattern for the state machine")

	result := EvaluateTrigger(">5 files changed, core logic changed, orchestration/pipeline changed", ctx)
	if !result.Invoke {
		t.Error("expected trigger to fire with multiple reasons")
	}
	// Should contain both reasons
	if !strings.Contains(result.Reason, "more than 5 files") {
		t.Errorf("expected file count reason, got: %s", result.Reason)
	}
	if !strings.Contains(result.Reason, "core logic") {
		t.Errorf("expected core logic reason, got: %s", result.Reason)
	}
}

// === F-10: Structured meta risk gate tests ===

func TestParseRiskGate_MetaProvided_WinsOverRegex(t *testing.T) {
	// Meta says HIGH, but the prose says "## Verdict\nLOW"
	// Meta should win because it's the primary structured path.
	output := `# QA Report

## Verdict
LOW

Everything looks fine on the surface.

{"__moonbase_meta": {"risk": "HIGH", "files_changed": ["main.go"]}}`

	routing := ParseRiskGate(output)
	if routing.Level != RiskHigh {
		t.Errorf("expected HIGH from meta (overriding regex LOW), got: %s", routing.Level)
	}
	if routing.TargetPhase != 2 {
		t.Errorf("expected target phase 2 for HIGH risk, got: %d", routing.TargetPhase)
	}
}

func TestParseRiskGate_MetaCritical(t *testing.T) {
	output := `Some QA output here.
{"__moonbase_meta": {"risk": "CRITICAL", "decisions": ["credentials leaked"]}}`

	routing := ParseRiskGate(output)
	if routing.Level != RiskCritical {
		t.Errorf("expected CRITICAL from meta, got: %s", routing.Level)
	}
	if routing.TargetPhase != 0 {
		t.Errorf("expected target phase 0 (stop) for CRITICAL, got: %d", routing.TargetPhase)
	}
}

func TestParseRiskGate_MetaMediumCaseInsensitive(t *testing.T) {
	output := `{"__moonbase_meta": {"risk": "medium"}}`
	routing := ParseRiskGate(output)
	if routing.Level != RiskMedium {
		t.Errorf("expected MEDIUM from meta (case-insensitive), got: %s", routing.Level)
	}
}

func TestParseRiskGate_MetaEmptyRisk_FallsBackToRegex(t *testing.T) {
	// Meta present but risk field empty — should fall back to regex.
	output := `## Verdict
LOW

{"__moonbase_meta": {"risk": "", "files_changed": ["foo.go"]}}`

	routing := ParseRiskGate(output)
	if routing.Level != RiskLow {
		t.Errorf("expected LOW from regex fallback (meta risk empty), got: %s", routing.Level)
	}
}

func TestParseRiskGate_NoMeta_FallsBackToRegex(t *testing.T) {
	// No meta block at all — pure regex path.
	output := "## Verdict\nMEDIUM\n\nMissing edge case coverage."
	routing := ParseRiskGate(output)
	if routing.Level != RiskMedium {
		t.Errorf("expected MEDIUM from regex fallback, got: %s", routing.Level)
	}
}

func TestParseRiskGate_MetaInvalidRisk_FallsBackToRegex(t *testing.T) {
	// Meta has a risk value that doesn't match any known level.
	output := `## Verdict
HIGH

{"__moonbase_meta": {"risk": "BANANA"}}`

	routing := ParseRiskGate(output)
	if routing.Level != RiskHigh {
		t.Errorf("expected HIGH from regex fallback (meta risk unrecognized), got: %s", routing.Level)
	}
}
