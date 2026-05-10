package ollama

import (
	"fmt"
	"time"
)

// Config holds configuration for the Ollama local LLM client.
type Config struct {
	// Model is the model to use (defaults to "llama3.1").
	Model string
	// BaseURL is the Ollama API base URL (defaults to "http://localhost:11434/v1").
	BaseURL string
	// MaxDiffSize limits diff size in bytes (required, must be > 0).
	MaxDiffSize int
	// RateInterval is the minimum time between requests (required, must be > 0).
	RateInterval time.Duration
}

// Validate ensures all required configuration is provided.
func (c Config) Validate() error {
	if c.Model == "" {
		return fmt.Errorf("ollama model is required (set LLM_MODEL env var)")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("ollama base URL is required (set LLM_API_URL env var)")
	}
	if c.MaxDiffSize <= 0 {
		return fmt.Errorf("ollama max diff size must be > 0")
	}
	if c.RateInterval <= 0 {
		return fmt.Errorf("ollama rate interval must be > 0")
	}
	return nil
}
