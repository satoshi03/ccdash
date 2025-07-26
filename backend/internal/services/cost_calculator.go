package services

import (
	"strings"
)

// PricingCalculator provides cost calculation for Claude models
type PricingCalculator struct {
	pricing map[string]map[string]float64
}

// NewPricingCalculator creates a new pricing calculator with fallback pricing
func NewPricingCalculator() *PricingCalculator {
	// Fallback pricing per million tokens (USD)
	fallbackPricing := map[string]map[string]float64{
		"opus": {
			"input":          15.0,
			"output":         75.0,
			"cache_creation": 18.75,
			"cache_read":     1.5,
		},
		"sonnet": {
			"input":          3.0,
			"output":         15.0,
			"cache_creation": 3.75,
			"cache_read":     0.3,
		},
		"haiku": {
			"input":          0.25,
			"output":         1.25,
			"cache_creation": 0.3,
			"cache_read":     0.03,
		},
	}

	// Map specific model names to pricing
	pricing := map[string]map[string]float64{
		"claude-3-opus":               fallbackPricing["opus"],
		"claude-3-sonnet":             fallbackPricing["sonnet"],
		"claude-3-haiku":              fallbackPricing["haiku"],
		"claude-3-5-sonnet":           fallbackPricing["sonnet"],
		"claude-3-5-haiku":            fallbackPricing["haiku"],
		"claude-sonnet-4-20250514":    fallbackPricing["sonnet"],
		"claude-opus-4-20250514":      fallbackPricing["opus"],
	}

	return &PricingCalculator{
		pricing: pricing,
	}
}

// CalculateCost calculates the cost for given token usage and model
func (pc *PricingCalculator) CalculateCost(
	model string,
	inputTokens int,
	outputTokens int,
	cacheCreationTokens int,
	cacheReadTokens int,
) float64 {
	// Handle synthetic model
	if model == "<synthetic>" {
		return 0.0
	}

	// Get pricing for model
	pricing := pc.getPricingForModel(model)

	// Calculate costs (pricing is per million tokens)
	cost := (float64(inputTokens)/1_000_000)*pricing["input"] +
		(float64(outputTokens)/1_000_000)*pricing["output"] +
		(float64(cacheCreationTokens)/1_000_000)*pricing["cache_creation"] +
		(float64(cacheReadTokens)/1_000_000)*pricing["cache_read"]

	// Round to 6 decimal places
	return roundToDecimals(cost, 6)
}

// getPricingForModel gets pricing for a model with fallback logic
func (pc *PricingCalculator) getPricingForModel(model string) map[string]float64 {
	// Normalize model name
	normalized := normalizeModelName(model)

	// Check configured pricing
	if pricing, exists := pc.pricing[normalized]; exists {
		return pricing
	}

	// Check original model name
	if pricing, exists := pc.pricing[model]; exists {
		return pricing
	}

	// Fallback to hardcoded pricing based on model type
	modelLower := strings.ToLower(model)
	if strings.Contains(modelLower, "opus") {
		return pc.pricing["claude-3-opus"]
	}
	if strings.Contains(modelLower, "haiku") {
		return pc.pricing["claude-3-haiku"]
	}
	// Default to Sonnet pricing
	return pc.pricing["claude-3-sonnet"]
}

// normalizeModelName normalizes model names for consistent lookup
func normalizeModelName(model string) string {
	// Remove common prefixes and normalize
	model = strings.TrimSpace(model)
	model = strings.ToLower(model)
	
	// Map various model name formats to standard names
	if strings.Contains(model, "opus") {
		if strings.Contains(model, "4") || strings.Contains(model, "2025") {
			return "claude-opus-4-20250514"
		}
		return "claude-3-opus"
	}
	
	if strings.Contains(model, "haiku") {
		if strings.Contains(model, "3.5") || strings.Contains(model, "3-5") {
			return "claude-3-5-haiku"
		}
		return "claude-3-haiku"
	}
	
	if strings.Contains(model, "sonnet") {
		if strings.Contains(model, "4") || strings.Contains(model, "2025") {
			return "claude-sonnet-4-20250514"
		}
		if strings.Contains(model, "3.5") || strings.Contains(model, "3-5") {
			return "claude-3-5-sonnet"
		}
		return "claude-3-sonnet"
	}
	
	return model
}

// roundToDecimals rounds a float64 to specified decimal places
func roundToDecimals(num float64, decimals int) float64 {
	multiplier := 1.0
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	return float64(int(num*multiplier+0.5)) / multiplier
}

// CalculateMessageCost calculates cost for a single message
func (pc *PricingCalculator) CalculateMessageCost(
	model *string,
	inputTokens int,
	outputTokens int,
	cacheCreationTokens int,
	cacheReadTokens int,
) float64 {
	if model == nil {
		return 0.0
	}
	
	return pc.CalculateCost(
		*model,
		inputTokens,
		outputTokens,
		cacheCreationTokens,
		cacheReadTokens,
	)
}