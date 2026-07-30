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

func TestFlywheelEntry_TokenFieldsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fl := &FlywheelLog{path: filepath.Join(dir, "flywheel.jsonl")}

	entry := FlywheelEntry{
		Timestamp:        time.Now().UTC(),
		TraceID:          "20260730T120000-token-test",
		Phase:            3,
		Agent:            "numbuh-3",
		Task:             "implement feature",
		Outcome:          "complete",
		DurationMs:       5000,
		OutputSize:       8000,
		PromptTokens:     45000,
		CompletionTokens: 12000,
		TotalTokens:      57000,
		Model:            "gpt-4o",
		EstimatedCostUSD: 0.2325,
	}

	if err := fl.Append(entry); err != nil {
		t.Fatalf("failed to append entry: %v", err)
	}

	// Read back and verify token fields
	file, err := os.Open(fl.Path())
	if err != nil {
		t.Fatalf("failed to open flywheel log: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan()
	var decoded FlywheelEntry
	if err := json.Unmarshal(scanner.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.PromptTokens != 45000 {
		t.Errorf("prompt tokens: got %d, want 45000", decoded.PromptTokens)
	}
	if decoded.CompletionTokens != 12000 {
		t.Errorf("completion tokens: got %d, want 12000", decoded.CompletionTokens)
	}
	if decoded.TotalTokens != 57000 {
		t.Errorf("total tokens: got %d, want 57000", decoded.TotalTokens)
	}
	if decoded.Model != "gpt-4o" {
		t.Errorf("model: got %q, want %q", decoded.Model, "gpt-4o")
	}
	if decoded.EstimatedCostUSD != 0.2325 {
		t.Errorf("estimated cost: got %f, want 0.2325", decoded.EstimatedCostUSD)
	}
	// Existing fields still intact
	if decoded.Phase != 3 {
		t.Errorf("phase: got %d, want 3", decoded.Phase)
	}
	if decoded.Agent != "numbuh-3" {
		t.Errorf("agent: got %q, want %q", decoded.Agent, "numbuh-3")
	}
}

func TestFlywheelEntry_LegacyWithoutTokenFieldsReadsAsZero(t *testing.T) {
	// Simulate reading a legacy entry that was written before token fields existed.
	// The JSON has no token fields — they should unmarshal as zero values.
	legacyJSON := `{"v":1,"timestamp":"2026-07-20T12:00:00Z","trace_id":"legacy-trace","phase":1,"agent":"numbuh-1","task":"old task","outcome":"complete","duration_ms":1200,"output_size":4500,"rework_count":0}`

	var entry FlywheelEntry
	if err := json.Unmarshal([]byte(legacyJSON), &entry); err != nil {
		t.Fatalf("failed to unmarshal legacy entry: %v", err)
	}

	if entry.PromptTokens != 0 {
		t.Errorf("expected 0 prompt tokens for legacy entry, got %d", entry.PromptTokens)
	}
	if entry.CompletionTokens != 0 {
		t.Errorf("expected 0 completion tokens for legacy entry, got %d", entry.CompletionTokens)
	}
	if entry.TotalTokens != 0 {
		t.Errorf("expected 0 total tokens for legacy entry, got %d", entry.TotalTokens)
	}
	if entry.Model != "" {
		t.Errorf("expected empty model for legacy entry, got %q", entry.Model)
	}
	if entry.EstimatedCostUSD != 0 {
		t.Errorf("expected 0 cost for legacy entry, got %f", entry.EstimatedCostUSD)
	}

	// Existing fields should still be correct
	if entry.SchemaVersion != 1 {
		t.Errorf("schema version: got %d, want 1", entry.SchemaVersion)
	}
	if entry.Phase != 1 {
		t.Errorf("phase: got %d, want 1", entry.Phase)
	}
	if entry.Agent != "numbuh-1" {
		t.Errorf("agent: got %q, want %q", entry.Agent, "numbuh-1")
	}
}

func TestFlywheelEntry_V0LegacyReadsAsZero(t *testing.T) {
	// v=0 entries (written before schema versioning) should also tolerate missing fields.
	v0JSON := `{"v":0,"timestamp":"2026-07-01T10:00:00Z","trace_id":"v0-trace","phase":2,"agent":"numbuh-2","task":"v0 task","outcome":"rework","risk_level":"MEDIUM","duration_ms":3000,"output_size":2000,"rework_count":1}`

	var entry FlywheelEntry
	if err := json.Unmarshal([]byte(v0JSON), &entry); err != nil {
		t.Fatalf("failed to unmarshal v0 entry: %v", err)
	}

	if entry.SchemaVersion != 0 {
		t.Errorf("schema version: got %d, want 0", entry.SchemaVersion)
	}
	if entry.PromptTokens != 0 {
		t.Errorf("expected 0 prompt tokens for v0 entry, got %d", entry.PromptTokens)
	}
	if entry.TotalTokens != 0 {
		t.Errorf("expected 0 total tokens for v0 entry, got %d", entry.TotalTokens)
	}
	if entry.Model != "" {
		t.Errorf("expected empty model for v0 entry, got %q", entry.Model)
	}
}

func TestFlywheelEntry_TokenFieldsOmitEmptyWhenZero(t *testing.T) {
	// Ensure omitempty means zero-value token fields are not serialized.
	entry := FlywheelEntry{
		SchemaVersion: 1,
		Timestamp:     time.Now().UTC(),
		TraceID:       "omitempty-test",
		Phase:         1,
		Agent:         "numbuh-1",
		Task:          "test",
		Outcome:       "complete",
		DurationMs:    100,
		OutputSize:    50,
		// Token fields are zero — should not appear in JSON
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)
	// Zero token fields should NOT appear in JSON (omitempty)
	if contains := "prompt_tokens"; len(jsonStr) > 0 {
		var m map[string]interface{}
		json.Unmarshal(data, &m)
		if _, exists := m["prompt_tokens"]; exists {
			t.Error("expected prompt_tokens to be omitted when zero")
		}
		if _, exists := m["completion_tokens"]; exists {
			t.Error("expected completion_tokens to be omitted when zero")
		}
		if _, exists := m["total_tokens"]; exists {
			t.Error("expected total_tokens to be omitted when zero")
		}
		if _, exists := m["model"]; exists {
			t.Error("expected model to be omitted when empty")
		}
		if _, exists := m["estimated_cost_usd"]; exists {
			t.Error("expected estimated_cost_usd to be omitted when zero")
		}
		_ = contains
	}
}
