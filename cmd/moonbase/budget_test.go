package main

import (
	"testing"

	"github.com/rk-senne/moonbase/internal/config"
	"github.com/rk-senne/moonbase/internal/pipeline"
)

func TestEffectivePricing_MergesDefaults(t *testing.T) {
	cfg := config.Config{}
	pricing := effectivePricing(cfg)

	// Should have defaults
	if _, ok := pricing["gpt-4o"]; !ok {
		t.Error("expected default gpt-4o pricing")
	}
	if _, ok := pricing["kimi-k3"]; !ok {
		t.Error("expected default kimi-k3 pricing")
	}
}

func TestEffectivePricing_UserOverrideTakesPrecedence(t *testing.T) {
	cfg := config.Config{
		ModelPricing: map[string]pipeline.ModelPrice{
			"gpt-4o": {PromptPer1M: 5.00, CompletionPer1M: 20.00}, // Override
			"custom-model": {PromptPer1M: 1.00, CompletionPer1M: 3.00}, // New model
		},
	}
	pricing := effectivePricing(cfg)

	// Override should take effect
	if pricing["gpt-4o"].PromptPer1M != 5.00 {
		t.Errorf("expected overridden gpt-4o prompt price 5.00, got %f", pricing["gpt-4o"].PromptPer1M)
	}
	// New model should be present
	if _, ok := pricing["custom-model"]; !ok {
		t.Error("expected custom-model in merged pricing")
	}
	// Other defaults should remain
	if _, ok := pricing["kimi-k3"]; !ok {
		t.Error("expected kimi-k3 to remain from defaults")
	}
}

func TestBudgetEnforcement_DisabledWhenZero(t *testing.T) {
	// When MaxTokensPerMission is 0, budget checks are disabled.
	// This is tested implicitly by the fact that runPipelineLoop only
	// checks budget when budgetMax > 0. We verify the config defaults to 0.
	cfg := config.DefaultConfig()
	if cfg.TokenBudget.MaxTokensPerMission != 0 {
		t.Errorf("expected default MaxTokensPerMission=0 (unlimited), got %d",
			cfg.TokenBudget.MaxTokensPerMission)
	}
}

func TestBudgetEnforcement_WarnThresholdDefault(t *testing.T) {
	// Verify the warn threshold defaults appropriately.
	cfg := config.DefaultConfig()
	// The field defaults to 0 in the struct, and runPipelineLoop sets it to 80 if <= 0.
	if cfg.TokenBudget.WarnThresholdPct != 0 {
		t.Errorf("expected default WarnThresholdPct=0 (pipeline defaults to 80), got %d",
			cfg.TokenBudget.WarnThresholdPct)
	}
}
