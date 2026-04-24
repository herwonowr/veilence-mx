package anthropic

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

// Client implements an Anthropic Messages API client for LLM analysis.
type Client struct {
	httpClient  *http.Client
	apiKey      string
	baseURL     string
	model       string
	maxDiffLen  int
	rateLimiter *time.Ticker
}

// New creates a new Anthropic client.
// Config must be validated before calling this (use Config.Validate()).
func New(cfg Config) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		apiKey:      cfg.APIKey,
		baseURL:     cfg.BaseURL,
		model:       cfg.Model,
		maxDiffLen:  cfg.MaxDiffLen,
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
	return "anthropic"
}

// Anthropic Messages API request/response types.
type messagesRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []messageContent `json:"messages"`
}

type messageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// Analyze sends a diff to Anthropic for classification.
func (c *Client) Analyze(ctx context.Context, diff string, packageName string, ecosystem string, oldVersion string, newVersion string, truncated bool) (*llm.Result, error) {
	// Rate limit
	select {
	case <-c.rateLimiter.C:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Truncate diff if needed
	if len(diff) > c.maxDiffLen {
		diff = diff[:c.maxDiffLen]
		truncated = true
	}

	userPrompt := llm.BuildUserPrompt(packageName, ecosystem, oldVersion, newVersion, diff, truncated)

	reqBody := messagesRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    llm.SecurityAnalysisPrompt,
		Messages: []messageContent{
			{Role: "user", Content: userPrompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	slog.Info("analyzing via anthropic", "package", packageName, "version", newVersion, "model", c.model)

	// Retry on 429
	maxRetries := 3
	var msgResp messagesResponse
	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("calling anthropic: %w", err)
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
			return nil, fmt.Errorf("anthropic rate limited after %d retries", maxRetries)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("anthropic returned %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()
		break
	}

	if len(msgResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from anthropic")
	}

	// Find the first text content block
	var rawResponse string
	for _, block := range msgResp.Content {
		if block.Type == "text" {
			rawResponse = block.Text
			break
		}
	}
	if rawResponse == "" {
		return nil, fmt.Errorf("no text content in anthropic response")
	}

	result := llm.ParseResponse(rawResponse)
	return result, nil
}
