package pipeline

import (
	"encoding/json"
	"testing"
)

func TestClassifyTask(t *testing.T) {
	tests := []struct {
		name      string
		task      string
		wantDepth Depth
	}{
		// Trivial tasks
		{
			name:      "trivial - fix typo in README",
			task:      "fix typo in README",
			wantDepth: DepthTrivial,
		},
		{
			name:      "trivial - rename getUserName to getUsername",
			task:      "rename getUserName to getUsername",
			wantDepth: DepthTrivial,
		},
		{
			name:      "trivial - remove unused import",
			task:      "remove unused import in main.go",
			wantDepth: DepthTrivial,
		},
		{
			name:      "trivial - fix spelling error",
			task:      "fix spelling error in config",
			wantDepth: DepthTrivial,
		},
		{
			name:      "trivial - update comment",
			task:      "update comment in parser",
			wantDepth: DepthTrivial,
		},
		// Complex tasks
		{
			name:      "complex - rate limiting with Redis",
			task:      "add rate limiting to the API with per-user quotas and Redis backing",
			wantDepth: DepthComplex,
		},
		{
			name:      "complex - pagination with cursor and caching",
			task:      "implement pagination for /users endpoint with cursor-based navigation and Redis caching",
			wantDepth: DepthComplex,
		},
		{
			name:      "complex - very long task (>200 chars)",
			task:      "refactor the entire authentication system to support OAuth2 with multiple providers including Google, GitHub, and Azure AD, add rate limiting per provider, implement token refresh with rotation, add audit logging for all auth events, and create comprehensive integration tests covering all flows",
			wantDepth: DepthComplex,
		},
		{
			name:      "complex - multiple file paths (>=3)",
			task:      "update internal/pipeline/context.go, internal/pipeline/riskgate.go, and internal/agents/parser.go to support new metadata format",
			wantDepth: DepthComplex,
		},
		{
			name:      "complex - many complexity signals",
			task:      "implement new endpoint with authentication and rate limiting and pagination",
			wantDepth: DepthComplex,
		},
		// Simple tasks (default / ambiguous)
		{
			name:      "simple - fix the auth check",
			task:      "fix the auth check",
			wantDepth: DepthSimple,
		},
		{
			name:      "simple - update error message in login",
			task:      "update the error message in login handler",
			wantDepth: DepthSimple,
		},
		{
			name:      "simple - add a test for the parser",
			task:      "add a test for the parser edge case",
			wantDepth: DepthSimple,
		},
		{
			name:      "simple - empty string defaults to simple",
			task:      "",
			wantDepth: DepthSimple,
		},
		{
			name:      "simple - 81 chars no signals",
			task:      "this is a task description that is exactly eighty-one characters long without kw!",
			wantDepth: DepthSimple,
		},
		{
			name:      "simple - moderate length with weak signal",
			task:      "fix the build issue in the authentication module when running locally",
			wantDepth: DepthSimple,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyTask(tt.task)
			if result.Depth != tt.wantDepth {
				t.Errorf("ClassifyTask(%q) = %s, want %s (reason: %s)",
					tt.task, result.Depth, tt.wantDepth, result.Reason)
			}
			// Reason should never be empty
			if result.Reason == "" {
				t.Errorf("ClassifyTask(%q) returned empty reason", tt.task)
			}
		})
	}
}

func TestClassifyTask_AmbiguityResolvesToSimple(t *testing.T) {
	// Tasks with some complexity signal but short — should resolve to simple, not trivial
	ambiguous := []string{
		"add a test for the login flow",       // has "add" signal
		"create a helper function for parsing", // has "create" signal
		"build the error handler for timeouts", // has "build" signal
	}
	for _, task := range ambiguous {
		result := ClassifyTask(task)
		if result.Depth == DepthTrivial {
			t.Errorf("ClassifyTask(%q) = trivial, but ambiguity should resolve to simple (reason: %s)", task, result.Reason)
		}
	}
}

func TestClassifyTask_ExactBoundary80Chars(t *testing.T) {
	// Exactly 80 characters with no signals -> trivial
	task := "fix formatting in the utils file so that it passes the linter check correctly"
	if len(task) > 80 {
		// Trim to exactly 80
		task = task[:80]
	}
	// Verify it's <=80 and has a trivial keyword
	result := ClassifyTask(task)
	if result.Depth != DepthTrivial {
		t.Errorf("80-char task with trivial keyword ('formatting') classified as %s, expected trivial (reason: %s)",
			result.Depth, result.Reason)
	}
}

func TestNewAdaptive_Trivial(t *testing.T) {
	p := NewAdaptive("fix typo", DepthTrivial, "short task, no complexity signals")

	// Phase 4 (QA) must NEVER be skipped — core invariant
	for _, phase := range p.Phases {
		if phase.Number == 4 && phase.Status == StatusSkipped {
			t.Fatal("INVARIANT VIOLATED: Phase 4 (QA) is skipped at trivial depth")
		}
	}

	// Phases 1, 2, 5 should be skipped
	for _, phase := range p.Phases {
		switch phase.Number {
		case 1, 2, 5:
			if phase.Status != StatusSkipped {
				t.Errorf("Phase %d (%s) should be skipped at trivial depth, got %d", phase.Number, phase.Name, phase.Status)
			}
		case 3:
			if phase.Status != StatusPending {
				t.Errorf("Phase 3 (Implementation) should be pending at trivial depth, got %d", phase.Status)
			}
		case 4:
			if phase.Status != StatusPending {
				t.Errorf("Phase 4 (QA) should be pending at trivial depth, got %d", phase.Status)
			}
		case 6, 7, 8:
			// Conditional phases stay pending (their Conditional flag handles them)
			if phase.Status != StatusPending {
				t.Errorf("Phase %d (conditional) should remain pending, got %d", phase.Number, phase.Status)
			}
		}
	}

	if p.Depth != DepthTrivial {
		t.Errorf("Depth: got %s, want trivial", p.Depth)
	}
	if p.DepthReason != "short task, no complexity signals" {
		t.Errorf("DepthReason: got %q", p.DepthReason)
	}
}

func TestNewAdaptive_Simple(t *testing.T) {
	p := NewAdaptive("fix auth check", DepthSimple, "1 complexity signal(s)")

	// Phase 4 (QA) must NEVER be skipped — core invariant
	for _, phase := range p.Phases {
		if phase.Number == 4 && phase.Status == StatusSkipped {
			t.Fatal("INVARIANT VIOLATED: Phase 4 (QA) is skipped at simple depth")
		}
	}

	// Phases 2, 5 should be skipped
	for _, phase := range p.Phases {
		switch phase.Number {
		case 1:
			if phase.Status != StatusPending {
				t.Errorf("Phase 1 (Analysis) should be pending at simple depth, got %d", phase.Status)
			}
		case 2, 5:
			if phase.Status != StatusSkipped {
				t.Errorf("Phase %d (%s) should be skipped at simple depth, got %d", phase.Number, phase.Name, phase.Status)
			}
		case 3, 4:
			if phase.Status != StatusPending {
				t.Errorf("Phase %d (%s) should be pending at simple depth, got %d", phase.Number, phase.Name, phase.Status)
			}
		}
	}

	if p.Depth != DepthSimple {
		t.Errorf("Depth: got %s, want simple", p.Depth)
	}
}

func TestNewAdaptive_Complex(t *testing.T) {
	p := NewAdaptive("implement full feature", DepthComplex, "3 complexity signals")

	// Phase 4 (QA) must NEVER be skipped — core invariant
	for _, phase := range p.Phases {
		if phase.Number == 4 && phase.Status == StatusSkipped {
			t.Fatal("INVARIANT VIOLATED: Phase 4 (QA) is skipped at complex depth")
		}
	}

	// All mandatory phases should be pending
	for _, phase := range p.Phases {
		if phase.Number >= 1 && phase.Number <= 5 {
			if phase.Status != StatusPending {
				t.Errorf("Phase %d (%s) should be pending at complex depth, got %d", phase.Number, phase.Name, phase.Status)
			}
		}
	}

	if p.Depth != DepthComplex {
		t.Errorf("Depth: got %s, want complex", p.Depth)
	}
}

func TestNewAdaptive_Phase4NeverSkipped_AllDepths(t *testing.T) {
	// Explicit invariant test: Phase 4 (QA) MUST be pending at every depth
	depths := []Depth{DepthTrivial, DepthSimple, DepthComplex}
	for _, depth := range depths {
		p := NewAdaptive("test task", depth, "test")
		for _, phase := range p.Phases {
			if phase.Number == 4 && phase.Status == StatusSkipped {
				t.Fatalf("INVARIANT VIOLATED: Phase 4 (QA) is StatusSkipped at depth %s", depth)
			}
		}
		// Also verify Phase 3 (Implementation) is never skipped
		for _, phase := range p.Phases {
			if phase.Number == 3 && phase.Status == StatusSkipped {
				t.Fatalf("Phase 3 (Implementation) should never be skipped at depth %s", depth)
			}
		}
	}
}

func TestEscalate_TrivialToSimple(t *testing.T) {
	p := NewAdaptive("fix something", DepthTrivial, "short task")

	err := p.Escalate(DepthSimple)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Depth != DepthSimple {
		t.Errorf("Depth: got %s, want simple", p.Depth)
	}
	if p.OrigDepth != DepthTrivial {
		t.Errorf("OrigDepth: got %s, want trivial", p.OrigDepth)
	}
	if !p.Escalated {
		t.Error("Escalated should be true")
	}

	// Phase 1 should be unskipped (pending)
	for _, phase := range p.Phases {
		if phase.Number == 1 && phase.Status != StatusPending {
			t.Errorf("Phase 1 should be pending after escalation to simple, got %d", phase.Status)
		}
		// Phase 2 should remain skipped
		if phase.Number == 2 && phase.Status != StatusSkipped {
			t.Errorf("Phase 2 should remain skipped after trivial→simple escalation, got %d", phase.Status)
		}
		// Phase 5 (Review) should be unskipped
		if phase.Number == 5 && phase.Status != StatusPending {
			t.Errorf("Phase 5 should be pending after escalation (Review on escalated pipelines), got %d", phase.Status)
		}
	}
}

func TestEscalate_TrivialToComplex(t *testing.T) {
	p := NewAdaptive("fix something", DepthTrivial, "short task")

	err := p.Escalate(DepthComplex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Depth != DepthComplex {
		t.Errorf("Depth: got %s, want complex", p.Depth)
	}
	if p.OrigDepth != DepthTrivial {
		t.Errorf("OrigDepth: got %s, want trivial", p.OrigDepth)
	}
	if !p.Escalated {
		t.Error("Escalated should be true")
	}

	// Phases 1, 2, 5 should all be unskipped (pending)
	for _, phase := range p.Phases {
		switch phase.Number {
		case 1, 2, 5:
			if phase.Status != StatusPending {
				t.Errorf("Phase %d should be pending after escalation to complex, got %d", phase.Number, phase.Status)
			}
		}
	}
}

func TestEscalate_SimpleToComplex(t *testing.T) {
	p := NewAdaptive("moderate task", DepthSimple, "test")

	err := p.Escalate(DepthComplex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Depth != DepthComplex {
		t.Errorf("Depth: got %s, want complex", p.Depth)
	}
	if p.OrigDepth != DepthSimple {
		t.Errorf("OrigDepth: got %s, want simple", p.OrigDepth)
	}

	// Phase 2 should be unskipped
	for _, phase := range p.Phases {
		if phase.Number == 2 && phase.Status != StatusPending {
			t.Errorf("Phase 2 should be pending after simple→complex escalation, got %d", phase.Status)
		}
		// Phase 5 should be unskipped
		if phase.Number == 5 && phase.Status != StatusPending {
			t.Errorf("Phase 5 should be pending after escalation, got %d", phase.Status)
		}
	}
}

func TestEscalate_AlreadyComplex(t *testing.T) {
	p := NewAdaptive("big task", DepthComplex, "test")

	err := p.Escalate(DepthComplex)
	if err == nil {
		t.Fatal("expected error when escalating already-complex pipeline")
	}
}

func TestEscalate_ReviewUnskipped(t *testing.T) {
	// Any escalation should un-skip Phase 5 (Review)
	tests := []struct {
		name   string
		from   Depth
		to     Depth
	}{
		{"trivial to simple", DepthTrivial, DepthSimple},
		{"trivial to complex", DepthTrivial, DepthComplex},
		{"simple to complex", DepthSimple, DepthComplex},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewAdaptive("test", tt.from, "test")
			p.Escalate(tt.to)

			for _, phase := range p.Phases {
				if phase.Number == 5 && phase.Status != StatusPending {
					t.Errorf("Phase 5 (Review) should be pending after escalation %s→%s, got %d",
						tt.from, tt.to, phase.Status)
				}
			}
		})
	}
}

func TestEscalationTarget(t *testing.T) {
	tests := []struct {
		name     string
		current  Depth
		risk     RiskLevel
		expected Depth
	}{
		// MEDIUM risk escalations
		{"medium + trivial → simple", DepthTrivial, RiskMedium, DepthSimple},
		{"medium + simple → complex", DepthSimple, RiskMedium, DepthComplex},
		{"medium + complex → complex (no change)", DepthComplex, RiskMedium, DepthComplex},
		// HIGH risk escalations
		{"high + trivial → complex", DepthTrivial, RiskHigh, DepthComplex},
		{"high + simple → complex", DepthSimple, RiskHigh, DepthComplex},
		{"high + complex → complex (no change)", DepthComplex, RiskHigh, DepthComplex},
		// CRITICAL — returns complex but escalation logic never reaches here
		// (CRITICAL stops the pipeline before escalation check)
		{"critical + trivial → complex", DepthTrivial, RiskCritical, DepthComplex},
		{"critical + simple → complex", DepthSimple, RiskCritical, DepthComplex},
		// LOW risk — no escalation
		{"low + trivial → trivial (no change)", DepthTrivial, RiskLow, DepthTrivial},
		{"low + simple → simple (no change)", DepthSimple, RiskLow, DepthSimple},
		{"low + complex → complex (no change)", DepthComplex, RiskLow, DepthComplex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscalationTarget(tt.current, tt.risk)
			if result != tt.expected {
				t.Errorf("EscalationTarget(%s, %s) = %s, want %s",
					tt.current, tt.risk, result, tt.expected)
			}
		})
	}
}

func TestCountComplexitySignals(t *testing.T) {
	tests := []struct {
		name     string
		task     string
		wantMin  int
		wantMax  int
	}{
		{"no signals", "fix typo in README", 0, 0},
		{"one signal - add", "add a test", 1, 1},
		{"trivial keyword cancels", "rename unused import", 0, 0},
		{"multiple signals", "implement rate limiting with authentication", 3, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countComplexitySignals(tt.task)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("countComplexitySignals(%q) = %d, want [%d, %d]",
					tt.task, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCountFilePaths(t *testing.T) {
	tests := []struct {
		name  string
		task  string
		want  int
	}{
		{"no paths", "fix the bug", 0},
		{"one path", "update internal/pipeline/context.go", 1},
		{"three paths", "change internal/a/b.go and internal/c/d.go and internal/e/f.go", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countFilePaths(tt.task)
			if got != tt.want {
				t.Errorf("countFilePaths(%q) = %d, want %d", tt.task, got, tt.want)
			}
		})
	}
}

func TestFlywheelEntry_DepthFieldsOmitEmpty(t *testing.T) {
	// When depth fields are empty, they should not appear in JSON
	entry := FlywheelEntry{
		SchemaVersion: 1,
		Phase:         1,
		Agent:         "numbuh-1",
		Task:          "test",
		Outcome:       "complete",
		DurationMs:    100,
		OutputSize:    50,
		// Depth fields intentionally left empty
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if _, exists := m["depth"]; exists {
		t.Error("expected 'depth' to be omitted when empty")
	}
	if _, exists := m["depth_reason"]; exists {
		t.Error("expected 'depth_reason' to be omitted when empty")
	}
	if _, exists := m["escalated_from"]; exists {
		t.Error("expected 'escalated_from' to be omitted when empty")
	}
	if _, exists := m["escalated_to"]; exists {
		t.Error("expected 'escalated_to' to be omitted when empty")
	}
}

func TestFlywheelEntry_DepthFieldsPresent(t *testing.T) {
	// When depth fields are set, they should appear in JSON
	entry := FlywheelEntry{
		SchemaVersion: 1,
		Phase:         4,
		Agent:         "numbuh-4",
		Task:          "test",
		Outcome:       "escalated",
		DurationMs:    200,
		OutputSize:    100,
		Depth:         "simple",
		DepthReason:   "1 complexity signal(s)",
		EscalatedFrom: "trivial",
		EscalatedTo:   "simple",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if v, ok := m["depth"]; !ok || v != "simple" {
		t.Errorf("expected depth='simple', got %v", v)
	}
	if v, ok := m["depth_reason"]; !ok || v != "1 complexity signal(s)" {
		t.Errorf("expected depth_reason set, got %v", v)
	}
	if v, ok := m["escalated_from"]; !ok || v != "trivial" {
		t.Errorf("expected escalated_from='trivial', got %v", v)
	}
	if v, ok := m["escalated_to"]; !ok || v != "simple" {
		t.Errorf("expected escalated_to='simple', got %v", v)
	}
}

func TestFlywheelEntry_LegacyWithoutDepthFieldsReadsAsEmpty(t *testing.T) {
	// Legacy entries without depth fields should unmarshal without error
	legacyJSON := `{"v":1,"timestamp":"2026-07-20T12:00:00Z","trace_id":"legacy","phase":1,"agent":"numbuh-1","task":"old","outcome":"complete","duration_ms":100,"output_size":50,"rework_count":0}`

	var entry FlywheelEntry
	if err := json.Unmarshal([]byte(legacyJSON), &entry); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if entry.Depth != "" {
		t.Errorf("expected empty depth for legacy entry, got %q", entry.Depth)
	}
	if entry.DepthReason != "" {
		t.Errorf("expected empty depth_reason for legacy entry, got %q", entry.DepthReason)
	}
	if entry.EscalatedFrom != "" {
		t.Errorf("expected empty escalated_from for legacy entry, got %q", entry.EscalatedFrom)
	}
	if entry.EscalatedTo != "" {
		t.Errorf("expected empty escalated_to for legacy entry, got %q", entry.EscalatedTo)
	}
}
