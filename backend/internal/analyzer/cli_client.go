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

// CLIClient implements the Analyzer interface using copilot-api proxy.
// It sends requests to a local copilot-api server that proxies to GitHub Copilot.
type CLIClient struct {
	httpClient  *http.Client
	baseURL     string
	model       string
	maxDiffLen  int
	rateLimiter *time.Ticker
}

// CLIClientConfig holds configuration for the copilot-api client.
type CLIClientConfig struct {
	// BaseURL is the copilot-api proxy URL. Defaults to "http://localhost:4141".
	BaseURL string
	// Model is the model to use. Defaults to "claude-sonnet-4.6".
	Model string
	// MaxDiffLen limits diff size in characters. Defaults to 20000.
	MaxDiffLen int
	// RateInterval is the minimum time between requests. Defaults to 6s.
	RateInterval time.Duration
}

// NewCLIClient creates a new copilot-api analyzer client.
func NewCLIClient(config CLIClientConfig) *CLIClient {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:4141"
	}
	if config.Model == "" {
		config.Model = "claude-opus-4.6"
	}
	if config.MaxDiffLen == 0 {
		config.MaxDiffLen = 20000
	}
	if config.RateInterval == 0 {
		config.RateInterval = 6 * time.Second
	}
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
func (c *CLIClient) Analyze(ctx context.Context, diff string, packageName string, registry string, oldVersion string, newVersion string) (*Result, error) {
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
		"Analyze the following diff for package \"%s\" (%s registry) between versions %s and %s:\n\n```diff\n%s\n```",
		packageName, registry, oldVersion, newVersion, diff,
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

	var result Result
	if err := json.Unmarshal([]byte(rawResponse), &result); err != nil {
		// Try to extract JSON from the response text
		start := bytes.IndexByte([]byte(rawResponse), '{')
		end := bytes.LastIndexByte([]byte(rawResponse), '}')
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(rawResponse[start:end+1]), &result); err2 == nil {
				result.RawResponse = rawResponse
				ValidateResult(&result)
				return &result, nil
			}
		}
		return &Result{
			Classification: "suspicious",
			Confidence:     0.5,
			Reasoning:      "Failed to parse LLM response. Raw: " + rawResponse,
			RawResponse:    rawResponse,
		}, nil
	}

	result.RawResponse = rawResponse
	ValidateResult(&result)
	return &result, nil
}

