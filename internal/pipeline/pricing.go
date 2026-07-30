// Pricing module for token cost estimation.
//
// Embeds default per-million-token prices for well-known models (public pricing).
// Users can override via config. Cost is computed at write time and stored in
// flywheel entries for historically accurate cost analysis.
package pipeline

// ModelPrice holds per-million-token pricing for a model.
// Prices are in USD. These are public list prices — no secrets.
type ModelPrice struct {
	PromptPer1M     float64 `yaml:"prompt"`
	CompletionPer1M float64 `yaml:"completion"`
}

// DefaultPricing returns embedded prices for well-known models.
// Users can override via config's model_pricing section.
// Source: public API pricing pages as of July 2026.
func DefaultPricing() map[string]ModelPrice {
	return map[string]ModelPrice{
		"gpt-4o":                   {PromptPer1M: 2.50, CompletionPer1M: 10.00},
		"gpt-4o-mini":              {PromptPer1M: 0.15, CompletionPer1M: 0.60},
		"gpt-4.1":                  {PromptPer1M: 2.00, CompletionPer1M: 8.00},
		"gpt-4.1-mini":             {PromptPer1M: 0.40, CompletionPer1M: 1.60},
		"claude-sonnet-4-20250514": {PromptPer1M: 3.00, CompletionPer1M: 15.00},
		"claude-haiku-3.5":         {PromptPer1M: 0.80, CompletionPer1M: 4.00},
		"kimi-k3":                  {PromptPer1M: 0.70, CompletionPer1M: 2.80},
	}
}

// EstimateCost computes USD cost given token counts and a pricing table.
// Returns 0 if the model is not in the table (not an error — unknown models
// are expected for non-reporting backends or new models not yet priced).
func EstimateCost(model string, promptTokens, completionTokens int, pricing map[string]ModelPrice) float64 {
	price, ok := pricing[model]
	if !ok {
		return 0
	}
	promptCost := float64(promptTokens) / 1_000_000 * price.PromptPer1M
	completionCost := float64(completionTokens) / 1_000_000 * price.CompletionPer1M
	return promptCost + completionCost
}
