package bedrock

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"github.com/veilence/veilence-mx/backend/pkg/llm"
)

// Client implements an AWS Bedrock Converse API client for LLM analysis.
type Client struct {
	httpClient  *http.Client
	region      string
	credentials aws.Credentials
	model       string
	maxDiffSize int
	rateLimiter *time.Ticker
}

// New creates a new Bedrock client.
// Config must be validated before calling this (use Config.Validate()).
func New(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		region:     cfg.Region,
		credentials: aws.Credentials{
			AccessKeyID:     cfg.AccessKeyID,
			SecretAccessKey: cfg.SecretAccessKey,
			SessionToken:    cfg.SessionToken,
		},
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
	return "bedrock"
}

// Converse API request/response types.
type converseRequest struct {
	Messages        []converseMessage `json:"messages"`
	System          []contentBlock    `json:"system"`
	InferenceConfig *inferenceConfig  `json:"inferenceConfig,omitempty"`
}

type converseMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Text string `json:"text"`
}

type inferenceConfig struct {
	MaxTokens int `json:"maxTokens"`
}

type converseResponse struct {
	Output struct {
		Message struct {
			Role    string         `json:"role"`
			Content []contentBlock `json:"content"`
		} `json:"message"`
	} `json:"output"`
	Usage struct {
		InputTokens  int `json:"inputTokens"`
		OutputTokens int `json:"outputTokens"`
	} `json:"usage"`
	StopReason string `json:"stopReason"`
}

// Analyze sends a diff to Bedrock for classification.
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

	reqBody := converseRequest{
		Messages: []converseMessage{
			{Role: "user", Content: []contentBlock{{Text: userPrompt}}},
		},
		System:          []contentBlock{{Text: llm.SecurityAnalysisPrompt}},
		InferenceConfig: &inferenceConfig{MaxTokens: 4096},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/converse", c.region, url.PathEscape(c.model))

	slog.Info("analyzing via bedrock", "package", packageName, "version", newVersion, "model", c.model)

	// Retry on 429
	maxRetries := 3
	var response converseResponse
	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		// SigV4 signing
		payloadHash := sha256Hash(bodyBytes)
		signer := v4.NewSigner()
		if err := signer.SignHTTP(ctx, c.credentials, req, payloadHash, "bedrock", c.region, time.Now()); err != nil {
			return nil, fmt.Errorf("signing request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("calling bedrock: %w", err)
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
			return nil, fmt.Errorf("bedrock rate limited after %d retries", maxRetries)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			return nil, fmt.Errorf("bedrock returned %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decoding response: %w", err)
		}
		resp.Body.Close()
		break
	}

	if len(response.Output.Message.Content) == 0 {
		return nil, fmt.Errorf("empty response from bedrock")
	}

	rawResponse := response.Output.Message.Content[0].Text
	if rawResponse == "" {
		return nil, fmt.Errorf("no text content in bedrock response")
	}

	result := llm.ParseResponse(rawResponse)
	return result, nil
}

// sha256Hash computes the hex-encoded SHA-256 hash of data.
func sha256Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
