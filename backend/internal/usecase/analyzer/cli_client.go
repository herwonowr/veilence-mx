package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// CLIClient implements the Analyzer interface using an OpenAI-compatible LLM API.
// It sends requests to a configured LLM API endpoint for code analysis.
type CLIClient struct {
	httpClient  *http.Client
	baseURL     string
	model       string
	maxDiffLen  int
	rateLimiter *time.Ticker
}

// CLIClientConfig holds configuration for the LLM API client.
// All fields are mandatory — the application MUST fail to start if any are missing.
type CLIClientConfig struct {
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
func (c CLIClientConfig) Validate() error {
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

// NewCLIClient creates a new LLM analyzer client.
// Config must be validated before calling this (use CLIClientConfig.Validate()).
func NewCLIClient(config CLIClientConfig) *CLIClient {
	return &CLIClient{
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		baseURL:     config.BaseURL,
		model:       config.Model,
		maxDiffLen:  config.MaxDiffLen,
		rateLimiter: time.NewTicker(config.RateInterval),
	}
}

// Close stops the rate limiter ticker, releasing its goroutine.
func (c *CLIClient) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}

// Type returns the analyzer type identifier.
func (c *CLIClient) Type() string {
	return "copilot"
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Max      int           `json:"max_tokens,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Analyze sends a diff to copilot-api for classification via GitHub Copilot.
func (c *CLIClient) Analyze(ctx context.Context, diff string, packageName string, ecosystem string, oldVersion string, newVersion string) (*Result, error) {
	// Rate limit
	select {
	case <-c.rateLimiter.C:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Truncate diff
	truncated := false
	if len(diff) > c.maxDiffLen {
		diff = diff[:c.maxDiffLen]
		truncated = true
	}

	userPrompt := fmt.Sprintf(
		"Analyze the following diff for package \"%s\" (%s ecosystem) between versions %s and %s:\n\n```diff\n%s\n```",
		packageName, ecosystem, oldVersion, newVersion, diff,
	)
	if truncated {
		userPrompt += "\n\n(Note: diff was truncated due to size limits. Analyze what is visible.)"
	}

	reqBody := chatRequest{
		Model: c.model,
		Max:   4096,
		Messages: []chatMessage{
			{Role: "system", Content: SystemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	slog.Info("analyzing via copilot-api", "package", packageName, "version", newVersion, "model", c.model)

	// Retry on 429
	maxRetries := 3
	var chatResp chatResponse
	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer dummy")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("calling copilot-api: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt < maxRetries-1 {
				backoff := time.Duration(10*(attempt+1)) * time.Second
				slog.Warn("rate limited, retrying", "package", packageName, "backoff", backoff)
				select {
				case <-time.After(backoff):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, fmt.Errorf("copilot-api rate limited after %d retries", maxRetries)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("copilot-api returned %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()
		break
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from copilot-api")
	}

	rawResponse := chatResp.Choices[0].Message.Content

	result := ParseLLMResponse(rawResponse)
	return result, nil
}

