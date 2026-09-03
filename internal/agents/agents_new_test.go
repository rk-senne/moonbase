package agents

import (
	"testing"
)

// === HasGuardrails tests ===

func TestAgent_HasGuardrails_Nil(t *testing.T) {
	a := &Agent{}
	if a.HasGuardrails() {
		t.Error("expected HasGuardrails=false when Guardrails is nil")
	}
}

func TestAgent_HasGuardrails_Set(t *testing.T) {
	a := &Agent{
		Guardrails: &GuardrailsConfig{
			MaxTurns:  10,
			MaxOutput: 5000,
		},
	}
	if !a.HasGuardrails() {
		t.Error("expected HasGuardrails=true when Guardrails is set")
	}
}

func TestAgent_HasGuardrails_EmptyConfig(t *testing.T) {
	a := &Agent{
		Guardrails: &GuardrailsConfig{},
	}
	// Even empty config is non-nil, so HasGuardrails should be true
	if !a.HasGuardrails() {
		t.Error("expected HasGuardrails=true even with empty GuardrailsConfig")
	}
}

// === MaxTurnsLimit tests ===

func TestAgent_MaxTurnsLimit_NoGuardrails(t *testing.T) {
	a := &Agent{}
	if a.MaxTurnsLimit() != 0 {
		t.Errorf("expected 0 when Guardrails is nil, got %d", a.MaxTurnsLimit())
	}
}

func TestAgent_MaxTurnsLimit_Set(t *testing.T) {
	tests := []struct {
		name     string
		maxTurns int
		expected int
	}{
		{"zero means unlimited", 0, 0},
		{"ten turns", 10, 10},
		{"large value", 1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{
				Guardrails: &GuardrailsConfig{MaxTurns: tt.maxTurns},
			}
			if a.MaxTurnsLimit() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, a.MaxTurnsLimit())
			}
		})
	}
}

// === MaxOutputLimit tests ===

func TestAgent_MaxOutputLimit_NoGuardrails(t *testing.T) {
	a := &Agent{}
	if a.MaxOutputLimit() != 0 {
		t.Errorf("expected 0 when Guardrails is nil, got %d", a.MaxOutputLimit())
	}
}

func TestAgent_MaxOutputLimit_Set(t *testing.T) {
	tests := []struct {
		name      string
		maxOutput int
		expected  int
	}{
		{"zero means unlimited", 0, 0},
		{"five thousand chars", 5000, 5000},
		{"large value", 100000, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{
				Guardrails: &GuardrailsConfig{MaxOutput: tt.maxOutput},
			}
			if a.MaxOutputLimit() != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, a.MaxOutputLimit())
			}
		})
	}
}

// === HasPreToolHooks tests ===

func TestAgent_HasPreToolHooks_NoHooks(t *testing.T) {
	a := &Agent{}
	if a.HasPreToolHooks() {
		t.Error("expected HasPreToolHooks=false when Hooks is nil")
	}
}

func TestAgent_HasPreToolHooks_EmptyHooks(t *testing.T) {
	a := &Agent{
		Hooks: &HooksConfig{},
	}
	if a.HasPreToolHooks() {
		t.Error("expected HasPreToolHooks=false when PreToolUse is empty")
	}
}

func TestAgent_HasPreToolHooks_WithHooks(t *testing.T) {
	a := &Agent{
		Hooks: &HooksConfig{
			PreToolUse: []Hook{
				{Command: "validate-input.sh", TimeoutMs: 1000},
			},
		},
	}
	if !a.HasPreToolHooks() {
		t.Error("expected HasPreToolHooks=true when PreToolUse has entries")
	}
}

func TestAgent_HasPreToolHooks_OnlyOtherHooks(t *testing.T) {
	a := &Agent{
		Hooks: &HooksConfig{
			OnActivate: []Hook{{Command: "setup.sh"}},
			OnComplete: []Hook{{Command: "cleanup.sh"}},
			// No PreToolUse
		},
	}
	if a.HasPreToolHooks() {
		t.Error("expected HasPreToolHooks=false when only other hooks are set")
	}
}

// === HasPostToolHooks tests ===

func TestAgent_HasPostToolHooks_NoHooks(t *testing.T) {
	a := &Agent{}
	if a.HasPostToolHooks() {
		t.Error("expected HasPostToolHooks=false when Hooks is nil")
	}
}

func TestAgent_HasPostToolHooks_EmptyHooks(t *testing.T) {
	a := &Agent{
		Hooks: &HooksConfig{},
	}
	if a.HasPostToolHooks() {
		t.Error("expected HasPostToolHooks=false when PostToolUse is empty")
	}
}

func TestAgent_HasPostToolHooks_WithHooks(t *testing.T) {
	a := &Agent{
		Hooks: &HooksConfig{
			PostToolUse: []Hook{
				{Command: "log-tool-use.sh", TimeoutMs: 500},
				{Command: "audit.sh", TimeoutMs: 2000},
			},
		},
	}
	if !a.HasPostToolHooks() {
		t.Error("expected HasPostToolHooks=true when PostToolUse has entries")
	}
}

func TestAgent_HasPostToolHooks_OnlyOtherHooks(t *testing.T) {
	a := &Agent{
		Hooks: &HooksConfig{
			OnActivate: []Hook{{Command: "setup.sh"}},
			PreToolUse: []Hook{{Command: "validate.sh"}},
			// No PostToolUse
		},
	}
	if a.HasPostToolHooks() {
		t.Error("expected HasPostToolHooks=false when only other hooks are set")
	}
}

// === Combined guardrails + hooks tests ===

func TestAgent_FullGuardrailsConfig(t *testing.T) {
	a := &Agent{
		Name: "numbuh-4",
		Role: "QA",
		Guardrails: &GuardrailsConfig{
			MaxTurns:    25,
			MaxOutput:   50000,
			InputRules:  []string{`^(?!.*DROP TABLE)`},
			OutputRules: []string{`## Verdict`},
			StopWords:   []string{"ABORT", "EMERGENCY"},
		},
		Hooks: &HooksConfig{
			OnActivate:  []Hook{{Command: "echo starting", TimeoutMs: 100}},
			PreToolUse:  []Hook{{Command: "check-safety.sh", TimeoutMs: 2000}},
			PostToolUse: []Hook{{Command: "log-usage.sh", TimeoutMs: 1000}},
			OnComplete:  []Hook{{Command: "echo done", TimeoutMs: 100}},
		},
	}

	if !a.HasGuardrails() {
		t.Error("expected HasGuardrails=true")
	}
	if a.MaxTurnsLimit() != 25 {
		t.Errorf("expected MaxTurnsLimit=25, got %d", a.MaxTurnsLimit())
	}
	if a.MaxOutputLimit() != 50000 {
		t.Errorf("expected MaxOutputLimit=50000, got %d", a.MaxOutputLimit())
	}
	if !a.HasPreToolHooks() {
		t.Error("expected HasPreToolHooks=true")
	}
	if !a.HasPostToolHooks() {
		t.Error("expected HasPostToolHooks=true")
	}

	// Verify guardrails fields are accessible
	if len(a.Guardrails.InputRules) != 1 {
		t.Errorf("expected 1 input rule, got %d", len(a.Guardrails.InputRules))
	}
	if len(a.Guardrails.OutputRules) != 1 {
		t.Errorf("expected 1 output rule, got %d", len(a.Guardrails.OutputRules))
	}
	if len(a.Guardrails.StopWords) != 2 {
		t.Errorf("expected 2 stop words, got %d", len(a.Guardrails.StopWords))
	}
}
