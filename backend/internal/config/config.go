// Package config loads and validates application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration. Loaded once at startup from environment variables.
type Config struct {
	// Required
	DatabaseURL string
	JWTSecret   string
	LLMApiURL   string
	LLMModel    string

	// Optional with defaults
	RedisURL    string
	Port        string
	FrontendURL string
	AppEnv      string

	// JWT rotation
	JWTSecretPrevious []string

	// LLM
	LLMProvider     string
	LLMApiKey       string
	LLMMaxDiffLen   int
	LLMRateInterval time.Duration

	// Pipeline
	MonitoringInterval time.Duration
	DiscoveryInterval  time.Duration
	PollerConcurrency  int
	DiffSizeLimit      int
	QueueMaxRetries    int
	QueueLockTimeout   time.Duration

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPUseTLS   bool

	// Settings seed defaults (from env)
	DiscoveryScanDepth           int
	DiscoveryAutoApprove         bool
	StaleAutoRemoveMonths        int
	PackageCountWarningThreshold int
	RequireEmailVerification     bool

	// Registration control
	RegistrationEnabled bool
	AllowedEmailDomains []string
}

// NewConfig loads configuration from environment variables with sensible defaults for optional fields.
func NewConfig() (*Config, error) {
	// Load .env file before reading env vars. Ignore error - file is optional
	// (env vars may be set directly in production).
	_ = godotenv.Load()

	cfg := &Config{
		// Required - no defaults
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		LLMApiURL:   os.Getenv("LLM_API_URL"),
		LLMModel:    os.Getenv("LLM_MODEL"),

		// Optional with defaults
		RedisURL:    envOrDefault("REDIS_URL", "redis://localhost:6379/0"),
		Port:        envOrDefault("SERVER_PORT", "8080"),
		FrontendURL: envOrDefault("FRONTEND_URL", "http://localhost:3000"),
		AppEnv:      envOrDefault("APP_ENV", "production"),

		// LLM
		LLMProvider:     os.Getenv("LLM_PROVIDER"),
		LLMApiKey:       os.Getenv("LLM_API_KEY"),
		LLMMaxDiffLen:   envIntOrDefault("LLM_MAX_DIFF_LEN", 20000),
		LLMRateInterval: envDurationOrDefault("LLM_RATE_INTERVAL", 6*time.Second),

		// Pipeline
		MonitoringInterval: envDurationOrDefault("MONITORING_INTERVAL", 1*time.Hour),
		DiscoveryInterval:  envDurationOrDefault("DISCOVERY_INTERVAL", 24*time.Hour),
		PollerConcurrency:  envIntOrDefault("POLLER_CONCURRENCY", 5),
		DiffSizeLimit:      envIntOrDefault("DIFF_SIZE_LIMIT", 102400),
		QueueMaxRetries:    envIntOrDefault("QUEUE_MAX_RETRIES", 5),
		QueueLockTimeout:   envDurationOrDefault("QUEUE_LOCK_TIMEOUT", 10*time.Minute),

		// SMTP
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     envOrDefault("SMTP_PORT", "587"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
		SMTPUseTLS:   envBoolOrDefault("SMTP_USE_TLS", true),

		// Settings seed defaults
		DiscoveryScanDepth:           envIntOrDefault("DISCOVERY_SCAN_DEPTH", 50),
		DiscoveryAutoApprove:         envBoolOrDefault("DISCOVERY_AUTO_APPROVE", false),
		StaleAutoRemoveMonths:        envIntOrDefault("STALE_AUTO_REMOVE_MONTHS", 0),
		PackageCountWarningThreshold: envIntOrDefault("PACKAGE_COUNT_WARNING_THRESHOLD", 0),
		RequireEmailVerification:     envBoolOrDefault("REQUIRE_EMAIL_VERIFICATION", false),

		// Registration control
		RegistrationEnabled: envBoolOrDefault("REGISTRATION_ENABLED", false),
	}

	// Parse comma-separated previous JWT secrets
	if prev := os.Getenv("JWT_SECRET_PREVIOUS"); prev != "" {
		for _, s := range strings.Split(prev, ",") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				cfg.JWTSecretPrevious = append(cfg.JWTSecretPrevious, trimmed)
			}
		}
	}

	// Parse comma-separated allowed email domains
	if domains := os.Getenv("ALLOWED_EMAIL_DOMAINS"); domains != "" {
		for _, d := range strings.Split(domains, ",") {
			if trimmed := strings.TrimSpace(d); trimmed != "" {
				cfg.AllowedEmailDomains = append(cfg.AllowedEmailDomains, trimmed)
			}
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate ensures all required configuration is present and valid. Fails fast with clear messages.
func (c *Config) Validate() error {
	var errs []string

	if c.DatabaseURL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	if c.JWTSecret == "" && c.AppEnv == "development" {
		c.JWTSecret = "veilence-mx-dev-jwt-secret-change-in-production"
	}
	if c.JWTSecret == "" {
		errs = append(errs, "JWT_SECRET is required")
	}
	if c.JWTSecret == "veilence-mx-dev-jwt-secret-change-in-production" && c.AppEnv != "development" {
		errs = append(errs, "JWT_SECRET must be set to a secure value in production")
	}
	if c.LLMProvider == "" {
		errs = append(errs, "LLM_PROVIDER is required (valid: copilot, openai, anthropic, ollama)")
	} else if c.LLMProvider != "copilot" && c.LLMProvider != "openai" && c.LLMProvider != "anthropic" && c.LLMProvider != "ollama" {
		errs = append(errs, fmt.Sprintf("unknown LLM_PROVIDER: %s (valid: copilot, openai, anthropic, ollama)", c.LLMProvider))
	} else {
		if c.LLMApiURL == "" {
			errs = append(errs, "LLM_API_URL is required")
		}
		if c.LLMModel == "" {
			errs = append(errs, "LLM_MODEL is required")
		}
		if (c.LLMProvider == "openai" || c.LLMProvider == "anthropic") && c.LLMApiKey == "" {
			errs = append(errs, fmt.Sprintf("LLM_API_KEY is required when using %s provider", c.LLMProvider))
		}
	}
	if c.LLMMaxDiffLen <= 0 {
		errs = append(errs, "LLM_MAX_DIFF_LEN must be > 0")
	}
	if c.LLMRateInterval <= 0 {
		errs = append(errs, "LLM_RATE_INTERVAL must be > 0")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// IsSMTPConfigured returns true if SMTP host and from address are set.
func (c *Config) IsSMTPConfigured() bool {
	return c.SMTPHost != "" && c.SMTPFrom != ""
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envBoolOrDefault(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
