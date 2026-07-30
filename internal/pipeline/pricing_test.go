package pipeline

import (
	"math"
	"testing"
)

func TestEstimateCost_KnownModel(t *testing.T) {
	pricing := DefaultPricing()

	tests := []struct {
		name             string
		model            string
		promptTokens     int
		completionTokens int
		wantCost         float64
	}{
		{
			name:             "gpt-4o typical usage",
			model:            "gpt-4o",
			promptTokens:     50000,
			completionTokens: 10000,
			// prompt: 50000/1M * 2.50 = 0.125
			// completion: 10000/1M * 10.00 = 0.10
			// total: 0.225
			wantCost: 0.225,
		},
		{
			name:             "gpt-4o-mini cheap model",
			model:            "gpt-4o-mini",
			promptTokens:     100000,
			completionTokens: 20000,
			// prompt: 100000/1M * 0.15 = 0.015
			// completion: 20000/1M * 0.60 = 0.012
			// total: 0.027
			wantCost: 0.027,
		},
		{
			name:             "claude-sonnet-4",
			model:            "claude-sonnet-4-20250514",
			promptTokens:     80000,
			completionTokens: 15000,
			// prompt: 80000/1M * 3.00 = 0.24
			// completion: 15000/1M * 15.00 = 0.225
			// total: 0.465
			wantCost: 0.465,
		},
		{
			name:             "kimi-k3",
			model:            "kimi-k3",
			promptTokens:     200000,
			completionTokens: 50000,
			// prompt: 200000/1M * 0.70 = 0.14
			// completion: 50000/1M * 2.80 = 0.14
			// total: 0.28
			wantCost: 0.28,
		},
		{
			name:             "zero tokens",
			model:            "gpt-4o",
			promptTokens:     0,
			completionTokens: 0,
			wantCost:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCost(tt.model, tt.promptTokens, tt.completionTokens, pricing)
			if math.Abs(got-tt.wantCost) > 0.0001 {
				t.Errorf("EstimateCost(%s, %d, %d) = %f, want %f",
					tt.model, tt.promptTokens, tt.completionTokens, got, tt.wantCost)
			}
		})
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	pricing := DefaultPricing()

	got := EstimateCost("unknown-model-xyz", 100000, 50000, pricing)
	if got != 0 {
		t.Errorf("EstimateCost for unknown model should be 0, got %f", got)
	}
}

func TestEstimateCost_EmptyPricing(t *testing.T) {
	got := EstimateCost("gpt-4o", 100000, 50000, map[string]ModelPrice{})
	if got != 0 {
		t.Errorf("EstimateCost with empty pricing should be 0, got %f", got)
	}
}

func TestEstimateCost_NilPricing(t *testing.T) {
	got := EstimateCost("gpt-4o", 100000, 50000, nil)
	if got != 0 {
		t.Errorf("EstimateCost with nil pricing should be 0, got %f", got)
	}
}

func TestDefaultPricing_ContainsExpectedModels(t *testing.T) {
	pricing := DefaultPricing()

	expectedModels := []string{
		"gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini",
		"claude-sonnet-4-20250514", "claude-haiku-3.5", "kimi-k3",
	}

	for _, model := range expectedModels {
		if _, ok := pricing[model]; !ok {
			t.Errorf("DefaultPricing() missing expected model: %s", model)
		}
	}
}

func TestDefaultPricing_PositivePrices(t *testing.T) {
	pricing := DefaultPricing()

	for model, price := range pricing {
		if price.PromptPer1M <= 0 {
			t.Errorf("model %s has non-positive prompt price: %f", model, price.PromptPer1M)
		}
		if price.CompletionPer1M <= 0 {
			t.Errorf("model %s has non-positive completion price: %f", model, price.CompletionPer1M)
		}
	}
}
