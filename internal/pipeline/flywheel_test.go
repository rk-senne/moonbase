package pipeline

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFlywheelLog_AppendAndRead(t *testing.T) {
	// Create a FlywheelLog with a temp directory
	dir := t.TempDir()
	fl := &FlywheelLog{path: filepath.Join(dir, "flywheel.jsonl")}

	entries := []FlywheelEntry{
		{
			Timestamp:   time.Now().UTC(),
			TraceID:     "20260718T120000-abc123",
			Phase:       1,
			Agent:       "numbuh-1",
			Task:        "add pagination",
			Outcome:     "complete",
			RiskLevel:   "",
			DurationMs:  1200,
			OutputSize:  4500,
			ReworkCount: 0,
		},
		{
			Timestamp:   time.Now().UTC(),
			TraceID:     "20260718T120000-abc123",
			Phase:       3,
			Agent:       "numbuh-3",
			Task:        "add pagination",
			Outcome:     "rework",
			RiskLevel:   "MEDIUM",
			DurationMs:  8500,
			OutputSize:  12000,
			ReworkCount: 1,
		},
		{
			Timestamp:   time.Now().UTC(),
			TraceID:     "20260718T130000-def456",
			Phase:       4,
			Agent:       "numbuh-4",
			Task:        "fix auth bypass",
			Outcome:     "failed",
			RiskLevel:   "CRITICAL",
			DurationMs:  3200,
			OutputSize:  800,
			ReworkCount: 0,
		},
	}

	// Append 3 entries with different outcomes
	for _, entry := range entries {
		if err := fl.Append(entry); err != nil {
			t.Fatalf("failed to append entry: %v", err)
		}
	}

	// Read back the file and verify it's valid JSONL
	file, err := os.Open(fl.Path())
	if err != nil {
		t.Fatalf("failed to open flywheel log: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var decoded []FlywheelEntry
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry FlywheelEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("line %d is not valid JSON: %v\nContent: %s", lineNum, err, string(line))
		}
		decoded = append(decoded, entry)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("error reading flywheel log: %v", err)
	}

	// Verify we got 3 entries back
	if len(decoded) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(decoded))
	}

	// Verify each entry deserialized correctly
	tests := []struct {
		name       string
		wantPhase  int
		wantAgent  string
		wantOutcome string
		wantRisk   string
		wantTask   string
	}{
		{
			name:       "entry 1 - complete",
			wantPhase:  1,
			wantAgent:  "numbuh-1",
			wantOutcome: "complete",
			wantRisk:   "",
			wantTask:   "add pagination",
		},
		{
			name:       "entry 2 - rework",
			wantPhase:  3,
			wantAgent:  "numbuh-3",
			wantOutcome: "rework",
			wantRisk:   "MEDIUM",
			wantTask:   "add pagination",
		},
		{
			name:       "entry 3 - failed",
			wantPhase:  4,
			wantAgent:  "numbuh-4",
			wantOutcome: "failed",
			wantRisk:   "CRITICAL",
			wantTask:   "fix auth bypass",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decoded[i]
			if got.Phase != tt.wantPhase {
				t.Errorf("phase: got %d, want %d", got.Phase, tt.wantPhase)
			}
			if got.Agent != tt.wantAgent {
				t.Errorf("agent: got %s, want %s", got.Agent, tt.wantAgent)
			}
			if got.Outcome != tt.wantOutcome {
				t.Errorf("outcome: got %s, want %s", got.Outcome, tt.wantOutcome)
			}
			if got.RiskLevel != tt.wantRisk {
				t.Errorf("risk level: got %s, want %s", got.RiskLevel, tt.wantRisk)
			}
			if got.Task != tt.wantTask {
				t.Errorf("task: got %s, want %s", got.Task, tt.wantTask)
			}
			if got.Timestamp.IsZero() {
				t.Error("timestamp should not be zero")
			}
		})
	}
}

func TestFlywheelLog_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deep")
	fl := &FlywheelLog{path: filepath.Join(dir, "flywheel.jsonl")}

	entry := FlywheelEntry{
		Timestamp:  time.Now().UTC(),
		TraceID:    "test-trace",
		Phase:      1,
		Agent:      "numbuh-1",
		Task:       "test",
		Outcome:    "complete",
		DurationMs: 100,
		OutputSize: 50,
	}

	if err := fl.Append(entry); err != nil {
		t.Fatalf("should create nested directories: %v", err)
	}

	if _, err := os.Stat(fl.Path()); err != nil {
		t.Fatalf("log file should exist: %v", err)
	}
}

func TestFlywheelLog_AppendsNotOverwrites(t *testing.T) {
	dir := t.TempDir()
	fl := &FlywheelLog{path: filepath.Join(dir, "flywheel.jsonl")}

	// Append first entry
	entry1 := FlywheelEntry{
		Timestamp: time.Now().UTC(),
		TraceID:   "trace-1",
		Phase:     1,
		Agent:     "numbuh-1",
		Task:      "first task",
		Outcome:   "complete",
	}
	if err := fl.Append(entry1); err != nil {
		t.Fatalf("first append failed: %v", err)
	}

	// Append second entry (simulating a new pipeline run)
	entry2 := FlywheelEntry{
		Timestamp: time.Now().UTC(),
		TraceID:   "trace-2",
		Phase:     1,
		Agent:     "numbuh-1",
		Task:      "second task",
		Outcome:   "complete",
	}
	if err := fl.Append(entry2); err != nil {
		t.Fatalf("second append failed: %v", err)
	}

	// Verify both entries exist
	data, err := os.ReadFile(fl.Path())
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	scanner := bufio.NewScanner(
		// Re-scan from bytes
		func() *os.File {
			f, _ := os.Open(fl.Path())
			return f
		}(),
	)

	count := 0
	for scanner.Scan() {
		if len(scanner.Bytes()) > 0 {
			count++
		}
	}
	_ = data // used above for reading

	if count != 2 {
		t.Errorf("expected 2 lines, got %d", count)
	}
}

func TestFlywheelLog_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	fl := &FlywheelLog{path: filepath.Join(dir, "flywheel.jsonl")}

	entry := FlywheelEntry{
		Timestamp: time.Now().UTC(),
		TraceID:   "test",
		Phase:     1,
		Agent:     "numbuh-1",
		Task:      "test",
		Outcome:   "complete",
	}
	if err := fl.Append(entry); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	info, err := os.Stat(fl.Path())
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestFlywheelLog_Path(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "flywheel.jsonl")
	fl := &FlywheelLog{path: path}

	if fl.Path() != path {
		t.Errorf("path mismatch: got %s, want %s", fl.Path(), path)
	}
}
