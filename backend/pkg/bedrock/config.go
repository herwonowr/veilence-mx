package bedrock

import (
	"fmt"
	"time"
)

// Config holds configuration for the AWS Bedrock LLM client.
type Config struct {
	// Region is the AWS region for the Bedrock endpoint (required).
	Region string
	// AccessKeyID is the AWS access key ID (required).
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key (required).
	SecretAccessKey string
	// SessionToken is the AWS session token (optional, for temporary credentials).
	SessionToken string
	// Model is the Bedrock model ID (required).
	Model string
	// MaxDiffSize limits diff size in bytes (required, must be > 0).
	MaxDiffSize int
	// RateInterval is the minimum time between requests (required, must be > 0).
	RateInterval time.Duration
}

// Validate ensures all required configuration is provided.
func (c Config) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("AWS_REGION is required for bedrock provider")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID is required for bedrock provider")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("AWS_SECRET_ACCESS_KEY is required for bedrock provider")
	}
	if c.Model == "" {
		return fmt.Errorf("LLM_MODEL is required for bedrock provider")
	}
	if c.MaxDiffSize <= 0 {
		return fmt.Errorf("MaxDiffSize must be > 0")
	}
	if c.RateInterval <= 0 {
		return fmt.Errorf("RateInterval must be > 0")
	}
	return nil
}
