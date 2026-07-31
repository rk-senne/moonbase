package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// --- RunSpecialists tests ---

func TestRunSpecialists_AllSucceed(t *testing.T) {
	phases := []Phase{
		{Number: 7, AgentName: "numbuh-274", Conditional: true},
		{Number: 6, AgentName: "numbuh-0", Conditional: true},
		{Number: 8, AgentName: "numbuh-362", Conditional: true},
	}

	execute := func(ctx context.Context, p Phase) (string, error) {
		return fmt.Sprintf("output-phase-%d", p.Number), nil
	}

	cfg := FanOutConfig{MaxConcurrency: 4, TraceID: "test-trace"}
	results := RunSpecialists(context.Background(), phases, execute, cfg)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify sorted by phase number.
	for i, r := range results {
		expectedPhase := []int{6, 7, 8}[i]
		if r.Phase != expectedPhase {
			t.Errorf("result[%d]: expected phase %d, got %d", i, expectedPhase, r.Phase)
		}
		if r.Status != StatusComplete {
			t.Errorf("result[%d]: expected StatusComplete, got %d", i, r.Status)
		}
		if r.Err != nil {
			t.Errorf("result[%d]: unexpected error: %v", i, r.Err)
		}
		if r.Output != fmt.Sprintf("output-phase-%d", expectedPhase) {
			t.Errorf("result[%d]: unexpected output: %s", i, r.Output)
		}
		if r.Duration <= 0 {
			t.Errorf("result[%d]: expected positive duration, got %v", i, r.Duration)
		}
	}
}

func TestRunSpecialists_OneOfThreeFails(t *testing.T) {
	phases := []Phase{
		{Number: 6, AgentName: "numbuh-0", Conditional: true},
		{Number: 7, AgentName: "numbuh-274", Conditional: true},
		{Number: 8, AgentName: "numbuh-362", Conditional: true},
	}

	failErr := errors.New("backend timeout")
	execute := func(ctx context.Context, p Phase) (string, error) {
		if p.Number == 7 {
			return "", failErr
		}
		return fmt.Sprintf("output-%d", p.Number), nil
	}

	cfg := FanOutConfig{MaxConcurrency: 4, TraceID: "test-partial"}
	results := RunSpecialists(context.Background(), phases, execute, cfg)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Phase 6: success
	if results[0].Phase != 6 || results[0].Status != StatusComplete {
		t.Errorf("phase 6: expected complete, got phase=%d status=%d", results[0].Phase, results[0].Status)
	}

	// Phase 7: failed (no sibling cancellation)
	if results[1].Phase != 7 || results[1].Status != StatusFailed {
		t.Errorf("phase 7: expected failed, got phase=%d status=%d", results[1].Phase, results[1].Status)
	}
	if results[1].Err == nil {
		t.Error("phase 7: expected non-nil error")
	}

	// Phase 8: success (not cancelled by sibling failure)
	if results[2].Phase != 8 || results[2].Status != StatusComplete {
		t.Errorf("phase 8: expected complete, got phase=%d status=%d", results[2].Phase, results[2].Status)
	}
}

func TestRunSpecialists_AllFail(t *testing.T) {
	phases := []Phase{
		{Number: 6, AgentName: "numbuh-0", Conditional: true},
		{Number: 7, AgentName: "numbuh-274", Conditional: true},
		{Number: 8, AgentName: "numbuh-362", Conditional: true},
	}

	execute := func(ctx context.Context, p Phase) (string, error) {
		return "", fmt.Errorf("fail-phase-%d", p.Number)
	}

	cfg := FanOutConfig{MaxConcurrency: 4, TraceID: "test-all-fail"}
	results := RunSpecialists(context.Background(), phases, execute, cfg)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != StatusFailed {
			t.Errorf("phase %d: expected StatusFailed, got %d", r.Phase, r.Status)
		}
		if r.Err == nil {
			t.Errorf("phase %d: expected non-nil error", r.Phase)
		}
	}
}

func TestRunSpecialists_ConcurrencyCap1_Sequential(t *testing.T) {
	phases := []Phase{
		{Number: 6, AgentName: "numbuh-0", Conditional: true},
		{Number: 7, AgentName: "numbuh-274", Conditional: true},
		{Number: 8, AgentName: "numbuh-362", Conditional: true},
	}

	// Track execution order using timestamps.
	type execRecord struct {
		phase int
		start time.Time
		end   time.Time
	}
	var mu atomic.Int64
	records := make([]execRecord, 3)

	execute := func(ctx context.Context, p Phase) (string, error) {
		idx := mu.Add(1) - 1
		start := time.Now()
		time.Sleep(10 * time.Millisecond) // ensure measurable non-overlap
		end := time.Now()
		records[idx] = execRecord{phase: p.Number, start: start, end: end}
		return fmt.Sprintf("output-%d", p.Number), nil
	}

	cfg := FanOutConfig{MaxConcurrency: 1, TraceID: "test-cap1"}
	results := RunSpecialists(context.Background(), phases, execute, cfg)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify no overlap: each start must be >= previous end.
	for i := 1; i < len(records); i++ {
		if records[i].start.Before(records[i-1].end) {
			t.Errorf("concurrency violation: record[%d] started at %v before record[%d] ended at %v",
				i, records[i].start, i-1, records[i-1].end)
		}
	}
}

func TestRunSpecialists_ContextCancellation(t *testing.T) {
	phases := []Phase{
		{Number: 6, AgentName: "numbuh-0", Conditional: true},
		{Number: 7, AgentName: "numbuh-274", Conditional: true},
		{Number: 8, AgentName: "numbuh-362", Conditional: true},
	}

	ctx, cancel := context.WithCancel(context.Background())

	execute := func(innerCtx context.Context, p Phase) (string, error) {
		// Cancel after first specialist starts
		if p.Number == 6 {
			cancel()
		}
		// Simulate work that respects context
		select {
		case <-innerCtx.Done():
			return "", innerCtx.Err()
		case <-time.After(100 * time.Millisecond):
			return fmt.Sprintf("output-%d", p.Number), nil
		}
	}

	cfg := FanOutConfig{MaxConcurrency: 4, TraceID: "test-cancel"}
	results := RunSpecialists(ctx, phases, execute, cfg)

	// All should be present (either complete or failed due to cancellation).
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// At least one should have a context cancellation error.
	var cancelledCount int
	for _, r := range results {
		if r.Err != nil && errors.Is(r.Err, context.Canceled) {
			cancelledCount++
		}
	}
	if cancelledCount == 0 {
		t.Error("expected at least one cancelled specialist")
	}
}

func TestRunSpecialists_EmptyPhases(t *testing.T) {
	results := RunSpecialists(context.Background(), nil, nil, FanOutConfig{})
	if results != nil {
		t.Errorf("expected nil results for empty phases, got %v", results)
	}
}

func TestRunSpecialists_Determinism(t *testing.T) {
	phases := []Phase{
		{Number: 8, AgentName: "numbuh-362", Conditional: true},
		{Number: 6, AgentName: "numbuh-0", Conditional: true},
		{Number: 7, AgentName: "numbuh-274", Conditional: true},
	}

	execute := func(ctx context.Context, p Phase) (string, error) {
		// Random-ish delay to stress ordering.
		time.Sleep(time.Duration(p.Number) * time.Millisecond)
		return fmt.Sprintf("output-%d", p.Number), nil
	}

	cfg := FanOutConfig{MaxConcurrency: 4, TraceID: "test-determinism"}

	// Run 100 iterations, verify output order never varies.
	for i := 0; i < 100; i++ {
		results := RunSpecialists(context.Background(), phases, execute, cfg)
		if len(results) != 3 {
			t.Fatalf("iteration %d: expected 3 results, got %d", i, len(results))
		}
		if results[0].Phase != 6 || results[1].Phase != 7 || results[2].Phase != 8 {
			t.Fatalf("iteration %d: order violation: phases [%d, %d, %d]",
				i, results[0].Phase, results[1].Phase, results[2].Phase)
		}
	}
}

// --- TriggeredSpecialists tests ---

func TestTriggeredSpecialists_FiltersCorrectly(t *testing.T) {
	phases := []Phase{
		{Number: 5, Name: "Review", Conditional: false, TriggerSpec: ""},
		{Number: 6, Name: "Oversight", Conditional: true, TriggerSpec: ">5 files changed, core logic changed"},
		{Number: 7, Name: "Security", Conditional: true, TriggerSpec: "Auth/secrets/permissions changed"},
		{Number: 8, Name: "Deploy", Conditional: true, TriggerSpec: "CI/CD changed, Docker/infra touched"},
	}

	pctx := NewPipelineContext("test")
	// Trigger security (auth keyword in output) but not others.
	pctx.PhaseOutputs[3] = "implemented authentication middleware"

	triggered := TriggeredSpecialists(phases, pctx)

	// Only phase 7 (Security) should trigger.
	if len(triggered) != 1 {
		t.Fatalf("expected 1 triggered specialist, got %d", len(triggered))
	}
	if triggered[0].Number != 7 {
		t.Errorf("expected phase 7 triggered, got phase %d", triggered[0].Number)
	}
}

func TestTriggeredSpecialists_NonConditionalExcluded(t *testing.T) {
	phases := []Phase{
		{Number: 3, Name: "Implementation", Conditional: false},
	}
	pctx := NewPipelineContext("test")
	triggered := TriggeredSpecialists(phases, pctx)
	if len(triggered) != 0 {
		t.Errorf("expected 0 triggered, got %d", len(triggered))
	}
}

// --- IsIndependentSpecialist tests ---

func TestIsIndependentSpecialist_ReadOnly(t *testing.T) {
	readOnly := true
	phase := Phase{Conditional: true}
	tools := []string{"read", "shell", "grep"}

	if !IsIndependentSpecialist(phase, tools, &readOnly) {
		t.Error("expected independent when shell.read_only=true")
	}
}

func TestIsIndependentSpecialist_NoWriteTool(t *testing.T) {
	phase := Phase{Conditional: true}
	tools := []string{"read", "shell", "grep", "glob", "code", "knowledge"}

	if !IsIndependentSpecialist(phase, tools, nil) {
		t.Error("expected independent when no write in tools and no shell config")
	}
}

func TestIsIndependentSpecialist_HasWriteTool(t *testing.T) {
	phase := Phase{Conditional: true}
	tools := []string{"read", "write", "shell"}

	if IsIndependentSpecialist(phase, tools, nil) {
		t.Error("expected NOT independent when write is in tools")
	}
}

func TestIsIndependentSpecialist_ReadOnlyFalse_WithWrite(t *testing.T) {
	readOnly := false
	phase := Phase{Conditional: true}
	tools := []string{"read", "write", "shell"}

	if IsIndependentSpecialist(phase, tools, &readOnly) {
		t.Error("expected NOT independent when read_only=false and write in tools")
	}
}

func TestIsIndependentSpecialist_ReadOnlyFalse_NoWrite(t *testing.T) {
	readOnly := false
	phase := Phase{Conditional: true}
	tools := []string{"read", "shell", "grep"}

	// read_only=false but no write tool → independent (no write tool is the check)
	if !IsIndependentSpecialist(phase, tools, &readOnly) {
		t.Error("expected independent when read_only=false but no write in tools")
	}
}

func TestIsIndependentSpecialist_NotConditional(t *testing.T) {
	readOnly := true
	phase := Phase{Conditional: false}
	tools := []string{"read", "grep"}

	if IsIndependentSpecialist(phase, tools, &readOnly) {
		t.Error("expected NOT independent when phase is not conditional")
	}
}

// --- clampConcurrency tests ---

func TestClampConcurrency(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 4},   // below minimum → default
		{-1, 4},  // negative → default
		{1, 1},   // minimum valid
		{4, 4},   // normal
		{16, 16}, // maximum valid
		{17, 16}, // above maximum → clamped
		{100, 16},
	}
	for _, tc := range tests {
		got := clampConcurrency(tc.input)
		if got != tc.expected {
			t.Errorf("clampConcurrency(%d) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

// --- MergeSpecialistResults tests ---

func TestMergeSpecialistResults_Success(t *testing.T) {
	pctx := NewPipelineContext("test merge")

	results := []FanOutResult{
		{Phase: 6, Agent: "numbuh-0", Output: "oversight analysis complete\ninternal/pipeline/fanout.go", Status: StatusComplete},
		{Phase: 7, Agent: "numbuh-274", Output: "security review: no issues\ninternal/config/config.go", Status: StatusComplete},
	}

	pctx.MergeSpecialistResults(results)

	if pctx.PhaseOutputs[6] != results[0].Output {
		t.Errorf("phase 6 output mismatch")
	}
	if pctx.PhaseOutputs[7] != results[1].Output {
		t.Errorf("phase 7 output mismatch")
	}

	// Files should be extracted.
	if len(pctx.FilesChanged) == 0 {
		t.Error("expected files to be extracted from outputs")
	}
}

func TestMergeSpecialistResults_WithFailure(t *testing.T) {
	pctx := NewPipelineContext("test merge failure")

	results := []FanOutResult{
		{Phase: 6, Agent: "numbuh-0", Output: "oversight done", Status: StatusComplete},
		{Phase: 7, Agent: "numbuh-274", Err: errors.New("timeout"), Status: StatusFailed},
	}

	pctx.MergeSpecialistResults(results)

	if pctx.PhaseOutputs[6] != "oversight done" {
		t.Errorf("phase 6: expected 'oversight done', got '%s'", pctx.PhaseOutputs[6])
	}
	if pctx.PhaseOutputs[7] != "[SPECIALIST FAILED: timeout]" {
		t.Errorf("phase 7: expected failure marker, got '%s'", pctx.PhaseOutputs[7])
	}
}

func TestMergeSpecialistResults_Deterministic(t *testing.T) {
	// Run multiple times — same input → same output.
	for i := 0; i < 50; i++ {
		pctx := NewPipelineContext("determinism test")
		results := []FanOutResult{
			{Phase: 6, Agent: "numbuh-0", Output: "analysis\ninternal/pipeline/pipeline.go", Status: StatusComplete},
			{Phase: 7, Agent: "numbuh-274", Output: "security\ninternal/config/config.go", Status: StatusComplete},
			{Phase: 8, Agent: "numbuh-362", Output: "deploy check", Status: StatusComplete},
		}
		pctx.MergeSpecialistResults(results)

		if pctx.PhaseOutputs[6] != "analysis\ninternal/pipeline/pipeline.go" {
			t.Fatalf("iteration %d: phase 6 output mismatch", i)
		}
		if pctx.PhaseOutputs[7] != "security\ninternal/config/config.go" {
			t.Fatalf("iteration %d: phase 7 output mismatch", i)
		}
	}
}

func TestMergeSpecialistResults_DeduplicatesFiles(t *testing.T) {
	pctx := NewPipelineContext("dedup test")
	// Pre-populate a file.
	pctx.FilesChanged = []string{"internal/pipeline/pipeline.go"}

	results := []FanOutResult{
		{Phase: 6, Agent: "numbuh-0", Output: "found issue in internal/pipeline/pipeline.go", Status: StatusComplete},
		{Phase: 7, Agent: "numbuh-274", Output: "checked internal/pipeline/pipeline.go", Status: StatusComplete},
	}

	pctx.MergeSpecialistResults(results)

	// Should not have duplicates.
	count := 0
	for _, f := range pctx.FilesChanged {
		if f == "internal/pipeline/pipeline.go" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("expected no duplicate files, got %d occurrences", count)
	}
}

// --- Checkpoint SpecialistResults tests ---

func TestCheckpoint_SpecialistResults_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	p := New("test specialist checkpoint")
	p.Context.RecordPhase(1, "reqs")
	p.Advance()

	// Save with no specialist results first (legacy compat).
	if err := SaveCheckpoint(p, dir); err != nil {
		t.Fatalf("save without specialist results: %v", err)
	}

	cp, err := LoadCheckpoint(dir, p.TraceID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cp.SpecialistResults != nil {
		t.Error("expected nil SpecialistResults for checkpoint without fan-out")
	}

	// Now save with specialist results.
	p2 := New("test specialist checkpoint v2")
	specResults := map[int]string{6: "complete", 7: "failed", 8: "complete"}
	// Write checkpoint manually to test the field.
	statuses := make(map[int]string)
	for _, phase := range p2.Phases {
		statuses[phase.Number] = statusName(phase.Status)
	}
	cpWithSpec := Checkpoint{
		SchemaVersion:     currentCheckpointVersion,
		TraceID:           p2.TraceID,
		Task:              p2.Task,
		Current:           p2.Current,
		PhaseStatuses:     statuses,
		PhaseOutputs:      p2.Context.PhaseOutputs,
		ReworkCount:       0,
		RiskLevel:         "",
		CreatedAt:         time.Now().UTC(),
		SpecialistResults: specResults,
	}

	data, err := json.MarshalIndent(cpWithSpec, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	path := filepath.Join(dir, p2.TraceID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	loaded, err := LoadCheckpoint(dir, p2.TraceID)
	if err != nil {
		t.Fatalf("load specialist checkpoint: %v", err)
	}
	if len(loaded.SpecialistResults) != 3 {
		t.Fatalf("expected 3 specialist results, got %d", len(loaded.SpecialistResults))
	}
	if loaded.SpecialistResults[6] != "complete" {
		t.Errorf("phase 6: expected 'complete', got '%s'", loaded.SpecialistResults[6])
	}
	if loaded.SpecialistResults[7] != "failed" {
		t.Errorf("phase 7: expected 'failed', got '%s'", loaded.SpecialistResults[7])
	}
}

// --- Flywheel ParallelGroup tests ---

func TestFlywheelEntry_ParallelGroup_Marshal(t *testing.T) {
	entry := FlywheelEntry{
		Timestamp:     time.Now().UTC(),
		TraceID:       "trace-123",
		Phase:         6,
		Agent:         "numbuh-0",
		Task:          "test task",
		Outcome:       "complete",
		DurationMs:    1500,
		OutputSize:    1000,
		ParallelGroup: "trace-123-fanout",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Should contain the parallel_group field.
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	if pg, ok := parsed["parallel_group"]; !ok || pg != "trace-123-fanout" {
		t.Errorf("expected parallel_group='trace-123-fanout', got %v", pg)
	}
}

func TestFlywheelEntry_ParallelGroup_OmitEmpty(t *testing.T) {
	entry := FlywheelEntry{
		Timestamp:  time.Now().UTC(),
		TraceID:    "trace-456",
		Phase:      1,
		Agent:      "numbuh-1",
		Task:       "test",
		Outcome:    "complete",
		DurationMs: 500,
		OutputSize: 200,
		// ParallelGroup intentionally empty.
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	if _, ok := parsed["parallel_group"]; ok {
		t.Error("expected parallel_group to be omitted when empty")
	}
}

func TestFlywheelEntry_ParallelGroup_SharedBatch(t *testing.T) {
	groupID := "trace-789-fanout"
	entries := []FlywheelEntry{
		{Phase: 6, Agent: "numbuh-0", ParallelGroup: groupID, Outcome: "complete"},
		{Phase: 7, Agent: "numbuh-274", ParallelGroup: groupID, Outcome: "failed"},
		{Phase: 8, Agent: "numbuh-362", ParallelGroup: groupID, Outcome: "complete"},
	}

	for _, e := range entries {
		if e.ParallelGroup != groupID {
			t.Errorf("phase %d: expected group '%s', got '%s'", e.Phase, groupID, e.ParallelGroup)
		}
	}
}

// --- Config parallel_specialists=false path test ---

func TestRunSpecialists_ConfigDisabled(t *testing.T) {
	// Simulates the config path where ParallelSpecialists=false.
	// The function itself always runs — the gating is at the call site.
	// When disabled, the caller simply doesn't call RunSpecialists.
	// This test verifies that a pipeline with ParallelSpecialists=false has the flag set.
	p := New("test disabled")
	p.ParallelSpecialists = false

	if p.ParallelSpecialists {
		t.Error("expected ParallelSpecialists=false after explicit set")
	}
}

func TestPipeline_DefaultParallelValues(t *testing.T) {
	p := New("test defaults")
	if !p.ParallelSpecialists {
		t.Error("expected ParallelSpecialists=true by default")
	}
	if p.MaxSpecialistConcurrency != 4 {
		t.Errorf("expected MaxSpecialistConcurrency=4, got %d", p.MaxSpecialistConcurrency)
	}
}

func TestNewFast_InheritsParallelDefaults(t *testing.T) {
	p := NewFast("fast task")
	if !p.ParallelSpecialists {
		t.Error("expected ParallelSpecialists=true on fast pipeline")
	}
	if p.MaxSpecialistConcurrency != 4 {
		t.Errorf("expected MaxSpecialistConcurrency=4, got %d", p.MaxSpecialistConcurrency)
	}
}
