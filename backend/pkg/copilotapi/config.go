package copilotapi

import (
	"fmt"
	"time"
)

// Config holds configuration for the copilot-api LLM client.
// All fields are mandatory - the application MUST fail to start if any are missing.
type Config struct {
	// BaseURL is the LLM API proxy URL (required).
	BaseURL string
	// Model is the model to use (required).
	Model string
	// MaxDiffLen limits diff size in characters (required, must be > 0).
	MaxDiffLen int
	// RateInterval is the minimum time between requests (required, must be > 0).
	RateInterval time.Duration
}

// Validate ensures all required configuration is provided. Returns an error if any field is missing.
func (c Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("LLM API base URL is required (set LLM_API_URL env var)")
	}
	if c.Model == "" {
		return fmt.Errorf("LLM model is required (set LLM_MODEL env var)")
	}
	if c.MaxDiffLen <= 0 {
		return fmt.Errorf("LLM max diff length must be > 0 (set LLM_MAX_DIFF_LEN env var)")
	}
	if c.RateInterval <= 0 {
		return fmt.Errorf("LLM rate interval must be > 0 (set LLM_RATE_INTERVAL env var)")
	}
	return nil
}
