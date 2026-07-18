package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Checkpoint save/load roundtrip tests ---

func TestCheckpoint_SaveLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	p := New("add user preferences")
	p.Context.RecordPhase(1, "Requirements: AC-1.1 support preferences")
	p.Advance()
	p.Context.RecordPhase(2, "Design: YAML config at ~/.config/moonbase/prefs.yaml")
	p.Advance()
	p.Context.RecordPhase(3, "Implementation: added preferences.go")
	p.Advance()
	p.Context.ReworkCount = 1
	p.Context.RiskLevel = "MEDIUM"

	// Save checkpoint
	err := SaveCheckpoint(p, dir)
	if err != nil {
		t.Fatalf("failed to save checkpoint: %v", err)
	}

	// Verify file exists with correct permissions
	cpPath := filepath.Join(dir, p.TraceID+".json")
	info, err := os.Stat(cpPath)
	if err != nil {
		t.Fatalf("checkpoint file not found: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}

	// Load checkpoint
	cp, err := LoadCheckpoint(dir, p.TraceID)
	if err != nil {
		t.Fatalf("failed to load checkpoint: %v", err)
	}

	// Verify fields roundtrip correctly
	if cp.TraceID != p.TraceID {
		t.Errorf("trace ID mismatch: got %s, want %s", cp.TraceID, p.TraceID)
	}
	if cp.Task != "add user preferences" {
		t.Errorf("task mismatch: got %s", cp.Task)
	}
	if cp.Current != p.Current {
		t.Errorf("current mismatch: got %d, want %d", cp.Current, p.Current)
	}
	if cp.ReworkCount != 1 {
		t.Errorf("rework count mismatch: got %d, want 1", cp.ReworkCount)
	}
	if cp.RiskLevel != "MEDIUM" {
		t.Errorf("risk level mismatch: got %s, want MEDIUM", cp.RiskLevel)
	}
}

func TestCheckpoint_SaveLoad_PhaseStatuses(t *testing.T) {
	dir := t.TempDir()

	p := New("test statuses")
	p.Phases[0].Status = StatusComplete
	p.Phases[1].Status = StatusComplete
	p.Phases[2].Status = StatusRunning
	p.Phases[3].Status = StatusPending
	p.Phases[4].Status = StatusPending
	p.Phases[5].Status = StatusSkipped
	p.Current = 2

	err := SaveCheckpoint(p, dir)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	cp, err := LoadCheckpoint(dir, p.TraceID)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Verify all phase statuses persisted
	expected := map[int]string{
		1: "complete",
		2: "complete",
		3: "running",
		4: "pending",
		5: "pending",
		6: "skipped",
		7: "pending",
		8: "pending",
	}

	for phase, wantStatus := range expected {
		gotStatus, ok := cp.PhaseStatuses[phase]
		if !ok {
			t.Errorf("phase %d missing from checkpoint statuses", phase)
			continue
		}
		if gotStatus != wantStatus {
			t.Errorf("phase %d status: got %s, want %s", phase, gotStatus, wantStatus)
		}
	}
}

func TestCheckpoint_SaveLoad_PhaseOutputs(t *testing.T) {
	dir := t.TempDir()

	p := New("test outputs")
	p.Context.RecordPhase(1, "Phase 1 output with AC-1.1")
	p.Context.RecordPhase(2, "Phase 2 output with design for internal/foo/bar.go")
	p.Context.RecordPhase(3, "Phase 3 output: all tests pass")
	p.Advance()
	p.Advance()
	p.Advance()

	err := SaveCheckpoint(p, dir)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	cp, err := LoadCheckpoint(dir, p.TraceID)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Verify all phase outputs persisted
	if cp.PhaseOutputs[1] != "Phase 1 output with AC-1.1" {
		t.Errorf("phase 1 output mismatch: %s", cp.PhaseOutputs[1])
	}
	if cp.PhaseOutputs[2] != "Phase 2 output with design for internal/foo/bar.go" {
		t.Errorf("phase 2 output mismatch: %s", cp.PhaseOutputs[2])
	}
	if cp.PhaseOutputs[3] != "Phase 3 output: all tests pass" {
		t.Errorf("phase 3 output mismatch: %s", cp.PhaseOutputs[3])
	}
}

func TestCheckpoint_Load_NotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadCheckpoint(dir, "nonexistent-trace-id")
	if err == nil {
		t.Error("expected error loading nonexistent checkpoint")
	}
}

func TestCheckpoint_Save_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep", "checkpoints")

	p := New("test dir creation")
	err := SaveCheckpoint(p, dir)
	if err != nil {
		t.Fatalf("save should create nested directories: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestCheckpoint_Roundtrip_AfterRework(t *testing.T) {
	dir := t.TempDir()

	p := New("fix auth bug")
	p.Context.RecordPhase(1, "Requirements")
	p.Advance()
	p.Context.RecordPhase(2, "Design")
	p.Advance()
	p.Context.RecordPhase(3, "Implementation v1")
	p.Advance()

	// Simulate rework
	_, _ = p.ApplyRiskGate("## Verdict\nMEDIUM\n\nMissing tests.")
	p.Context.RecordPhase(3, "Implementation v2 with tests")

	err := SaveCheckpoint(p, dir)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	cp, err := LoadCheckpoint(dir, p.TraceID)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if cp.ReworkCount != 1 {
		t.Errorf("expected rework count 1, got %d", cp.ReworkCount)
	}
	if cp.RiskLevel != "MEDIUM" {
		t.Errorf("expected MEDIUM risk level, got %s", cp.RiskLevel)
	}
	// Phase 3 output should be the latest (v2)
	if cp.PhaseOutputs[3] != "Implementation v2 with tests" {
		t.Errorf("expected latest implementation output, got: %s", cp.PhaseOutputs[3])
	}
}
