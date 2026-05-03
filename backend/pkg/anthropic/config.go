package anthropic

import (
	"fmt"
	"time"
)

// Config holds configuration for the Anthropic LLM client.
type Config struct {
	// APIKey is the Anthropic API key (required).
	APIKey string
	// Model is the model to use (defaults to "claude-sonnet-4-20250514").
	Model string
	// BaseURL is the API base URL (defaults to "https://api.anthropic.com/v1").
	BaseURL string
	// MaxDiffLen limits diff size in characters (required, must be > 0).
	MaxDiffLen int
	// RateInterval is the minimum time between requests (required, must be > 0).
	RateInterval time.Duration
}

// Validate ensures all required configuration is provided.
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("anthropic API key is required (set LLM_API_KEY env var)")
	}
	if c.Model == "" {
		return fmt.Errorf("anthropic model is required (set LLM_MODEL env var)")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("anthropic base URL is required (set LLM_API_URL env var)")
	}
	if c.MaxDiffLen <= 0 {
		return fmt.Errorf("anthropic max diff length must be > 0")
	}
	if c.RateInterval <= 0 {
		return fmt.Errorf("anthropic rate interval must be > 0")
	}
	return nil
}
