package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/veilence/veilence-mx/backend/pkg/llm"
)

// Client implements an OpenAI direct API client for LLM analysis.
type Client struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	model       string
	maxDiffSize int
	rateLimiter *time.Ticker
}

// New creates a new OpenAI client.
// Config must be validated before calling this (use Config.Validate()).
func New(cfg Config) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		apiKey:      cfg.APIKey,
		baseURL:     cfg.BaseURL,
		model:       cfg.Model,
		maxDiffSize: cfg.MaxDiffSize,
		rateLimiter: time.NewTicker(cfg.RateInterval),
	}
}

// Close stops the rate limiter ticker, releasing its goroutine.
func (c *Client) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
}

// Type returns the provider type identifier.
func (c *Client) Type() string {
	return "openai"
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

// Analyze sends a diff to OpenAI for classification.
func (c *Client) Analyze(ctx context.Context, diff string, packageName string, ecosystem string, oldVersion string, newVersion string, truncated bool) (*llm.Result, error) {
	// Rate limit
	select {
	case <-c.rateLimiter.C:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Truncate diff if needed
	if len(diff) > c.maxDiffSize {
		diff = diff[:c.maxDiffSize]
		truncated = true
	}

	userPrompt := llm.BuildUserPrompt(packageName, ecosystem, oldVersion, newVersion, diff, truncated)

	reqBody := chatRequest{
		Model: c.model,
		Max:   4096,
		Messages: []chatMessage{
			{Role: "system", Content: llm.SecurityAnalysisPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	slog.Info("analyzing via openai", "package", packageName, "version", newVersion, "model", c.model)

	// Retry on 429
	maxRetries := 3
	var chatResp chatResponse
	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("calling openai: %w", err)
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
			return nil, fmt.Errorf("openai rate limited after %d retries", maxRetries)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("openai returned %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()
		break
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from openai")
	}

	rawResponse := chatResp.Choices[0].Message.Content
	result := llm.ParseResponse(rawResponse)
	return result, nil
}
