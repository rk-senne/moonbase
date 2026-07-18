package pipeline

import (
	"strings"
	"testing"
	"time"
)

// === ValidateOutput tests ===

func TestPipeline_ValidateOutput_WithinLimit(t *testing.T) {
	tests := []struct {
		name   string
		output string
		max    int
	}{
		{"empty output", "", 100},
		{"short output", "hello", 100},
		{"exactly at limit", strings.Repeat("x", 100), 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New("test")
			p.MaxOutputSize = tt.max
			if err := p.ValidateOutput(tt.output); err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestPipeline_ValidateOutput_ExceedsLimit(t *testing.T) {
	tests := []struct {
		name   string
		output string
		max    int
	}{
		{"one over", strings.Repeat("x", 101), 100},
		{"way over", strings.Repeat("x", 500000), 100000},
		{"one char over default", strings.Repeat("a", 100001), 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New("test")
			p.MaxOutputSize = tt.max
			err := p.ValidateOutput(tt.output)
			if err == nil {
				t.Fatal("expected error for exceeding output limit")
			}
			if !strings.Contains(err.Error(), "exceeds maximum") {
				t.Errorf("expected 'exceeds maximum' in error, got: %v", err)
			}
		})
	}
}

func TestPipeline_ValidateOutput_DefaultLimit(t *testing.T) {
	p := New("test")
	// Default MaxOutputSize is 100000
	if p.MaxOutputSize != 100000 {
		t.Fatalf("unexpected default MaxOutputSize: %d", p.MaxOutputSize)
	}

	// Just under limit should pass
	if err := p.ValidateOutput(strings.Repeat("x", 99999)); err != nil {
		t.Errorf("expected no error for under-limit output: %v", err)
	}

	// Over limit should fail
	if err := p.ValidateOutput(strings.Repeat("x", 100001)); err == nil {
		t.Error("expected error for over-limit output")
	}
}

// === RetryPhase tests ===

func TestPipeline_RetryPhase_Success(t *testing.T) {
	p := New("test")
	p.Current = 0
	p.Phases[0].Status = StatusFailed

	err := p.RetryPhase()
	if err != nil {
		t.Fatalf("unexpected error on first retry: %v", err)
	}

	// Should have incremented retry count
	if p.Retries[1] != 1 { // Phase number 1
		t.Errorf("expected retry count 1, got %d", p.Retries[1])
	}

	// Phase should be restarted
	if p.Phases[0].Status != StatusRunning {
		t.Errorf("expected StatusRunning, got %d", p.Phases[0].Status)
	}
	if p.Phases[0].StartedAt.IsZero() {
		t.Error("expected StartedAt to be set after retry")
	}
}

func TestPipeline_RetryPhase_MaxRetriesExceeded(t *testing.T) {
	p := New("test")
	p.Current = 2
	p.MaxRetries = 1
	p.Retries[3] = 1 // Phase 3 (Implementation) already retried once

	err := p.RetryPhase()
	if err == nil {
		t.Fatal("expected error when max retries exceeded")
	}
	if !strings.Contains(err.Error(), "max retries") {
		t.Errorf("expected 'max retries' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "phase 3") {
		t.Errorf("expected 'phase 3' in error, got: %v", err)
	}
}

func TestPipeline_RetryPhase_NoActivePhase(t *testing.T) {
	p := New("test")
	p.Current = 99 // Out of bounds

	err := p.RetryPhase()
	if err == nil {
		t.Fatal("expected error when no active phase")
	}
	if !strings.Contains(err.Error(), "no active phase") {
		t.Errorf("expected 'no active phase' in error, got: %v", err)
	}
}

func TestPipeline_RetryPhase_MultipleRetries(t *testing.T) {
	p := New("test")
	p.Current = 0
	p.MaxRetries = 3

	// First retry
	if err := p.RetryPhase(); err != nil {
		t.Fatalf("retry 1 failed: %v", err)
	}
	// Second retry
	if err := p.RetryPhase(); err != nil {
		t.Fatalf("retry 2 failed: %v", err)
	}
	// Third retry
	if err := p.RetryPhase(); err != nil {
		t.Fatalf("retry 3 failed: %v", err)
	}
	// Fourth should fail
	if err := p.RetryPhase(); err == nil {
		t.Fatal("expected error on 4th retry with MaxRetries=3")
	}
}

// === Phase.StartPhase / CompletePhase / ElapsedTime / IsTimedOut ===

func TestPhase_StartPhase(t *testing.T) {
	ph := &Phase{
		Number: 1,
		Name:   "Analysis",
		Status: StatusPending,
	}

	ph.StartPhase()

	if ph.Status != StatusRunning {
		t.Errorf("expected StatusRunning, got %d", ph.Status)
	}
	if ph.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
	if !ph.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be zero after StartPhase")
	}
}

func TestPhase_StartPhase_ResetsCompletedAt(t *testing.T) {
	ph := &Phase{
		Number:      1,
		Name:        "Analysis",
		Status:      StatusComplete,
		CompletedAt: time.Now(),
	}

	ph.StartPhase()

	if !ph.CompletedAt.IsZero() {
		t.Error("StartPhase should reset CompletedAt to zero")
	}
}

func TestPhase_CompletePhase(t *testing.T) {
	ph := &Phase{
		Number:    1,
		Name:      "Analysis",
		Status:    StatusRunning,
		StartedAt: time.Now().Add(-2 * time.Second),
	}

	ph.CompletePhase()

	if ph.Status != StatusComplete {
		t.Errorf("expected StatusComplete, got %d", ph.Status)
	}
	if ph.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
	if ph.Duration == "" {
		t.Error("expected Duration to be set")
	}
	// Duration should be approximately 2 seconds
	if !strings.Contains(ph.Duration, "s") {
		t.Errorf("expected duration to contain 's', got: %s", ph.Duration)
	}
}

func TestPhase_ElapsedTime_NotStarted(t *testing.T) {
	ph := &Phase{Number: 1, Name: "Analysis"}

	elapsed := ph.ElapsedTime()
	if elapsed != 0 {
		t.Errorf("expected 0 elapsed for unstarted phase, got %v", elapsed)
	}
}

func TestPhase_ElapsedTime_Running(t *testing.T) {
	ph := &Phase{
		Number:    1,
		Name:      "Analysis",
		Status:    StatusRunning,
		StartedAt: time.Now().Add(-100 * time.Millisecond),
	}

	elapsed := ph.ElapsedTime()
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected elapsed > 50ms for running phase, got %v", elapsed)
	}
}

func TestPhase_ElapsedTime_Completed(t *testing.T) {
	start := time.Now().Add(-5 * time.Second)
	end := start.Add(3 * time.Second)

	ph := &Phase{
		Number:      1,
		Name:        "Analysis",
		Status:      StatusComplete,
		StartedAt:   start,
		CompletedAt: end,
	}

	elapsed := ph.ElapsedTime()
	if elapsed != 3*time.Second {
		t.Errorf("expected 3s elapsed, got %v", elapsed)
	}
}

func TestPhase_IsTimedOut_NotRunning(t *testing.T) {
	tests := []struct {
		name   string
		status PhaseStatus
	}{
		{"pending", StatusPending},
		{"complete", StatusComplete},
		{"failed", StatusFailed},
		{"skipped", StatusSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ph := &Phase{
				Number:    1,
				Status:    tt.status,
				StartedAt: time.Now().Add(-10 * time.Minute),
			}
			if ph.IsTimedOut(1 * time.Second) {
				t.Error("non-running phases should never be timed out")
			}
		})
	}
}

func TestPhase_IsTimedOut_NotStarted(t *testing.T) {
	ph := &Phase{
		Number: 1,
		Status: StatusRunning,
		// StartedAt is zero
	}
	if ph.IsTimedOut(1 * time.Second) {
		t.Error("phase with zero StartedAt should not be timed out")
	}
}

func TestPhase_IsTimedOut_WithinTimeout(t *testing.T) {
	ph := &Phase{
		Number:    1,
		Status:    StatusRunning,
		StartedAt: time.Now(),
	}
	if ph.IsTimedOut(5 * time.Minute) {
		t.Error("just-started phase should not be timed out")
	}
}

func TestPhase_IsTimedOut_Exceeded(t *testing.T) {
	ph := &Phase{
		Number:    1,
		Status:    StatusRunning,
		StartedAt: time.Now().Add(-10 * time.Minute),
	}
	if !ph.IsTimedOut(5 * time.Minute) {
		t.Error("phase running for 10 min should be timed out with 5 min timeout")
	}
}

// === generateTraceID tests ===

func TestGenerateTraceID_NonEmpty(t *testing.T) {
	id := generateTraceID()
	if id == "" {
		t.Fatal("expected non-empty trace ID")
	}
}

func TestGenerateTraceID_HasTimestamp(t *testing.T) {
	id := generateTraceID()
	// Format: 20060102T150405-<hex>
	if !strings.Contains(id, "T") {
		t.Errorf("expected trace ID to contain timestamp separator 'T', got: %s", id)
	}
	if !strings.Contains(id, "-") {
		t.Errorf("expected trace ID to contain '-' separator, got: %s", id)
	}
}

func TestGenerateTraceID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateTraceID()
		if ids[id] {
			t.Fatalf("duplicate trace ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateTraceID_Format(t *testing.T) {
	id := generateTraceID()
	parts := strings.SplitN(id, "-", 2)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts (timestamp-hex), got %d: %s", len(parts), id)
	}
	// Timestamp part should be 15 chars (20060102T150405)
	if len(parts[0]) != 15 {
		t.Errorf("expected timestamp part to be 15 chars, got %d: %s", len(parts[0]), parts[0])
	}
	// Hex suffix should be 8 chars (4 bytes = 8 hex chars)
	if len(parts[1]) != 8 {
		t.Errorf("expected hex suffix to be 8 chars, got %d: %s", len(parts[1]), parts[1])
	}
}

// === ParseMeta additional table-driven tests ===

func TestParseMeta_TableDriven(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		expectNil    bool
		expectRisk   string
		expectFiles  int
		expectACKeys int
	}{
		{
			name:      "empty string",
			output:    "",
			expectNil: true,
		},
		{
			name:      "no marker at all",
			output:    "just regular text with no JSON",
			expectNil: true,
		},
		{
			name:      "marker but unclosed brace",
			output:    `{"__moonbase_meta": {"risk": "LOW"`,
			expectNil: true,
		},
		{
			name:       "valid with risk only",
			output:     `{"__moonbase_meta": {"risk": "HIGH"}}`,
			expectNil:  false,
			expectRisk: "HIGH",
		},
		{
			name:        "valid with files",
			output:      `{"__moonbase_meta": {"files_changed": ["a.go", "b.go", "c.go"]}}`,
			expectNil:   false,
			expectFiles: 3,
		},
		{
			name:         "valid with AC results",
			output:       `text before {"__moonbase_meta": {"ac_results": {"AC-1": "PASS", "AC-2": "FAIL"}}}`,
			expectNil:    false,
			expectACKeys: 2,
		},
		{
			name:      "double occurrence picks last",
			output:    `{"__moonbase_meta": {"risk": "LOW"}} more text {"__moonbase_meta": {"risk": "CRITICAL"}}`,
			expectNil: false,
			expectRisk: "CRITICAL",
		},
		{
			name:      "nested JSON before meta",
			output:    `{"config": {"nested": {"deep": true}}} and then {"__moonbase_meta": {"risk": "MEDIUM"}}`,
			expectNil: false,
			expectRisk: "MEDIUM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := ParseMeta(tt.output)
			if tt.expectNil {
				if meta != nil {
					t.Errorf("expected nil, got %+v", meta)
				}
				return
			}
			if meta == nil {
				t.Fatal("expected non-nil meta")
			}
			if tt.expectRisk != "" && meta.Risk != tt.expectRisk {
				t.Errorf("expected risk %q, got %q", tt.expectRisk, meta.Risk)
			}
			if tt.expectFiles > 0 && len(meta.FilesChanged) != tt.expectFiles {
				t.Errorf("expected %d files, got %d", tt.expectFiles, len(meta.FilesChanged))
			}
			if tt.expectACKeys > 0 && len(meta.ACResults) != tt.expectACKeys {
				t.Errorf("expected %d AC keys, got %d", tt.expectACKeys, len(meta.ACResults))
			}
		})
	}
}

// === Pipeline initialization tests ===

func TestPipeline_New_Defaults(t *testing.T) {
	p := New("my task")

	if p.TraceID == "" {
		t.Error("expected non-empty TraceID")
	}
	if p.PhaseTimeout != 5*time.Minute {
		t.Errorf("expected PhaseTimeout=5m, got %v", p.PhaseTimeout)
	}
	if p.MaxOutputSize != 100000 {
		t.Errorf("expected MaxOutputSize=100000, got %d", p.MaxOutputSize)
	}
	if p.MaxRetries != 1 {
		t.Errorf("expected MaxRetries=1, got %d", p.MaxRetries)
	}
	if p.MaxRework != 2 {
		t.Errorf("expected MaxRework=2, got %d", p.MaxRework)
	}
	if p.Retries == nil {
		t.Error("expected Retries map to be initialized")
	}
}
