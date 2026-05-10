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
	MetricsPort string
	FrontendURL string
	BackendURL  string
	AppEnv      string

	// JWT rotation
	JWTSecretPrevious []string

	// LLM
	LLMProvider     string
	LLMApiKey       string
	LLMMaxDiffSize  int
	LLMRateInterval time.Duration

	// Pipeline
	MonitoringInterval         time.Duration
	DiscoveryInterval          time.Duration
	PollerConcurrency          int
	PollerWorkspaceConcurrency int
	DiffSizeLimit              int
	QueueMaxRetries            int
	QueueLockTimeout           time.Duration
	DiffWorkerConcurrency      int
	AnalyzeWorkerConcurrency   int
	DiffJobTimeoutSeconds      int
	AnalyzeJobTimeoutSeconds   int

	// Archive extraction limits
	MaxArchiveFileCount int
	MaxArchiveSize      int
	MaxFileExtractSize  int
	MaxFileReadSize     int

	// Registry download limits
	MaxRegistryDownloadSize int

	// Bulk operation limits
	MaxBulkImport  int
	MaxBulkApprove int

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

	// Ecosystems
	EcosystemsEnabled []string

	// Registration control
	RegistrationEnabled bool
	AllowedEmailDomains []string

	// SSO
	SSOEnabled       bool
	SSOEncryptionKey string
	SSOSAMLClockSkew time.Duration
	SSOStateTTL      time.Duration

	// OAuth provider URLs (overridable for Keycloak/local dev)
	OAuthGoogleAuthURL     string
	OAuthGoogleTokenURL    string
	OAuthGoogleUserInfoURL string
	OAuthGitHubAuthURL     string
	OAuthGitHubTokenURL    string
	OAuthGitHubUserInfoURL string
	OAuthGitHubEmailsURL   string
	OAuthGitHubOrgsURL     string
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
		MetricsPort: envOrDefault("METRICS_PORT", "9090"),
		FrontendURL: envOrDefault("FRONTEND_URL", "http://localhost:3000"),
		BackendURL:  envOrDefault("BACKEND_URL", "http://localhost:8080"),
		AppEnv:      envOrDefault("APP_ENV", "production"),

		// LLM
		LLMProvider:     os.Getenv("LLM_PROVIDER"),
		LLMApiKey:       os.Getenv("LLM_API_KEY"),
		LLMMaxDiffSize:  envIntOrDefault("LLM_MAX_DIFF_SIZE", 150*1024),
		LLMRateInterval: envDurationOrDefault("LLM_RATE_INTERVAL", 6*time.Second),

		// Pipeline
		MonitoringInterval:         envDurationOrDefault("MONITORING_INTERVAL", 1*time.Hour),
		DiscoveryInterval:          envDurationOrDefault("DISCOVERY_INTERVAL", 24*time.Hour),
		PollerConcurrency:          envIntOrDefault("POLLER_CONCURRENCY", 5),
		PollerWorkspaceConcurrency: envIntOrDefault("POLLER_WORKSPACE_CONCURRENCY", 5),
		DiffSizeLimit:              envIntOrDefault("DIFF_SIZE_LIMIT", 150*1024),
		QueueMaxRetries:            envIntOrDefault("QUEUE_MAX_RETRIES", 5),
		QueueLockTimeout:           envDurationOrDefault("QUEUE_LOCK_TIMEOUT", 10*time.Minute),
		DiffWorkerConcurrency:      envIntOrDefault("DIFF_WORKER_CONCURRENCY", 5),
		AnalyzeWorkerConcurrency:   envIntOrDefault("ANALYZE_WORKER_CONCURRENCY", 3),
		DiffJobTimeoutSeconds:      envIntOrDefault("DIFF_JOB_TIMEOUT_SECONDS", 600),
		AnalyzeJobTimeoutSeconds:   envIntOrDefault("ANALYZE_JOB_TIMEOUT_SECONDS", 300),

		// Archive extraction limits
		MaxArchiveFileCount: envIntOrDefault("MAX_ARCHIVE_FILE_COUNT", 50000),
		MaxArchiveSize:      envIntOrDefault("MAX_ARCHIVE_SIZE", 500*1024*1024),
		MaxFileExtractSize:  envIntOrDefault("MAX_FILE_EXTRACT_SIZE", 50*1024*1024),
		MaxFileReadSize:     envIntOrDefault("MAX_FILE_READ_SIZE", 1*1024*1024),

		// Registry download limits
		MaxRegistryDownloadSize: envIntOrDefault("MAX_REGISTRY_DOWNLOAD_SIZE", 200*1024*1024),

		// Bulk operation limits
		MaxBulkImport:  envIntOrDefault("MAX_BULK_IMPORT", 500),
		MaxBulkApprove: envIntOrDefault("MAX_BULK_APPROVE", 1000),

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

		// SSO
		SSOEnabled:       envBoolOrDefault("SSO_ENABLED", false),
		SSOEncryptionKey: os.Getenv("SSO_ENCRYPTION_KEY"),
		SSOSAMLClockSkew: envDurationOrDefault("SSO_SAML_CLOCK_SKEW", 30*time.Second),
		SSOStateTTL:      envDurationOrDefault("SSO_STATE_TTL", 5*time.Minute),

		// OAuth provider URLs
		OAuthGoogleAuthURL:     envOrDefault("OAUTH_GOOGLE_AUTH_URL", "https://accounts.google.com/o/oauth2/v2/auth"),
		OAuthGoogleTokenURL:    envOrDefault("OAUTH_GOOGLE_TOKEN_URL", "https://oauth2.googleapis.com/token"),
		OAuthGoogleUserInfoURL: envOrDefault("OAUTH_GOOGLE_USERINFO_URL", "https://www.googleapis.com/oauth2/v3/userinfo"),
		OAuthGitHubAuthURL:     envOrDefault("OAUTH_GITHUB_AUTH_URL", "https://github.com/login/oauth/authorize"),
		OAuthGitHubTokenURL:    envOrDefault("OAUTH_GITHUB_TOKEN_URL", "https://github.com/login/oauth/access_token"),
		OAuthGitHubUserInfoURL: envOrDefault("OAUTH_GITHUB_USERINFO_URL", "https://api.github.com/user"),
		OAuthGitHubEmailsURL:   envOrDefault("OAUTH_GITHUB_EMAILS_URL", "https://api.github.com/user/emails"),
		OAuthGitHubOrgsURL:     envOrDefault("OAUTH_GITHUB_ORGS_URL", "https://api.github.com/user/orgs"),
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

	// Parse comma-separated enabled ecosystems
	if ecosystems := os.Getenv("ECOSYSTEMS_ENABLED"); ecosystems != "" {
		seen := make(map[string]bool)
		for _, e := range strings.Split(ecosystems, ",") {
			if trimmed := strings.ToLower(strings.TrimSpace(e)); trimmed != "" {
				if !seen[trimmed] {
					seen[trimmed] = true
					switch trimmed {
					case "npm", "pypi", "go":
						cfg.EcosystemsEnabled = append(cfg.EcosystemsEnabled, trimmed)
					default:
						return nil, fmt.Errorf("unknown ecosystem in ECOSYSTEMS_ENABLED: %q (valid: npm, pypi, go)", trimmed)
					}
				}
			}
		}
	} else {
		cfg.EcosystemsEnabled = []string{"npm", "pypi"}
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
	if c.LLMMaxDiffSize <= 0 {
		errs = append(errs, "LLM_MAX_DIFF_SIZE must be > 0")
	}
	if c.LLMRateInterval <= 0 {
		errs = append(errs, "LLM_RATE_INTERVAL must be > 0")
	}

	// SSO validation
	if c.SSOEnabled && c.SSOEncryptionKey == "" {
		errs = append(errs, "SSO_ENCRYPTION_KEY is required when SSO_ENABLED=true")
	}
	if c.SSOEnabled && c.SSOEncryptionKey != "" && len(c.SSOEncryptionKey) != 64 {
		errs = append(errs, "SSO_ENCRYPTION_KEY must be a 64-character hex string (32 bytes)")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
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
