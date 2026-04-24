package openai

import (
	"fmt"
	"time"
)

// Config holds configuration for the OpenAI LLM client.
type Config struct {
	// APIKey is the OpenAI API key (required).
	APIKey string
	// Model is the model to use (defaults to "gpt-4o").
	Model string
	// BaseURL is the API base URL (defaults to "https://api.openai.com/v1").
	BaseURL string
	// MaxDiffLen limits diff size in characters (required, must be > 0).
	MaxDiffLen int
	// RateInterval is the minimum time between requests (required, must be > 0).
	RateInterval time.Duration
}

// Validate ensures all required configuration is provided.
func (c Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required (set OPENAI_API_KEY env var)")
	}
	if c.Model == "" {
		return fmt.Errorf("OpenAI model is required (set OPENAI_MODEL env var)")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("OpenAI base URL is required (set OPENAI_BASE_URL env var)")
	}
	if c.MaxDiffLen <= 0 {
		return fmt.Errorf("OpenAI max diff length must be > 0")
	}
	if c.RateInterval <= 0 {
		return fmt.Errorf("OpenAI rate interval must be > 0")
	}
	return nil
}
