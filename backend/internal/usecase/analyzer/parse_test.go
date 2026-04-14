package analyzer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripMarkdownFences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no fences",
			input:    `{"classification":"benign","confidence":0.95,"reasoning":"OK"}`,
			expected: `{"classification":"benign","confidence":0.95,"reasoning":"OK"}`,
		},
		{
			name:     "json fences",
			input:    "```json\n{\"classification\":\"benign\",\"confidence\":0.95,\"reasoning\":\"OK\"}\n```",
			expected: `{"classification":"benign","confidence":0.95,"reasoning":"OK"}`,
		},
		{
			name:     "plain fences",
			input:    "```\n{\"classification\":\"benign\",\"confidence\":0.95,\"reasoning\":\"OK\"}\n```",
			expected: `{"classification":"benign","confidence":0.95,"reasoning":"OK"}`,
		},
		{
			name:     "fences with extra whitespace",
			input:    "```json  \n  {\"classification\":\"benign\",\"confidence\":0.95,\"reasoning\":\"OK\"}  \n  ```",
			expected: `{"classification":"benign","confidence":0.95,"reasoning":"OK"}`,
		},
		{
			name:     "multiline JSON in fences",
			input:    "```json\n{\n  \"classification\": \"benign\",\n  \"confidence\": 0.95,\n  \"reasoning\": \"OK\"\n}\n```",
			expected: "{\n  \"classification\": \"benign\",\n  \"confidence\": 0.95,\n  \"reasoning\": \"OK\"\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownFences(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractJSONObject(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		ok       bool
	}{
		{
			name:     "simple object",
			input:    `{"key":"value"}`,
			expected: `{"key":"value"}`,
			ok:       true,
		},
		{
			name:     "object with surrounding text",
			input:    `Here is: {"key":"value"} done.`,
			expected: `{"key":"value"}`,
			ok:       true,
		},
		{
			name:     "nested braces in value",
			input:    `{"reasoning":"saw func() { return x; } pattern"}`,
			expected: `{"reasoning":"saw func() { return x; } pattern"}`,
			ok:       true,
		},
		{
			name:     "nested JSON objects",
			input:    `{"outer":{"inner":"value"}}`,
			expected: `{"outer":{"inner":"value"}}`,
			ok:       true,
		},
		{
			name:     "braces after object",
			input:    `{"key":"value"} and then func() {}`,
			expected: `{"key":"value"}`,
			ok:       true,
		},
		{
			name:     "no braces",
			input:    "plain text",
			expected: "",
			ok:       false,
		},
		{
			name:     "unmatched opening brace",
			input:    `{incomplete`,
			expected: "",
			ok:       false,
		},
		{
			name:     "escaped quotes in string",
			input:    `{"key":"value with \"escaped\" quotes"}`,
			expected: `{"key":"value with \"escaped\" quotes"}`,
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractJSONObject(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestParseLLMResponse_RawJSON(t *testing.T) {
	raw := `{"classification":"benign","confidence":0.95,"reasoning":"Normal update"}`
	result := ParseLLMResponse(raw)

	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 0.95, result.Confidence)
	assert.Equal(t, "Normal update", result.Reasoning)
	assert.Equal(t, raw, result.RawResponse)
}

func TestParseLLMResponse_MarkdownJSONFence(t *testing.T) {
	raw := "```json\n{\"classification\":\"benign\",\"confidence\":0.95,\"reasoning\":\"Normal update\"}\n```"
	result := ParseLLMResponse(raw)

	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 0.95, result.Confidence)
	assert.Equal(t, "Normal update", result.Reasoning)
	assert.Equal(t, raw, result.RawResponse)
}

func TestParseLLMResponse_MarkdownPlainFence(t *testing.T) {
	raw := "```\n{\"classification\":\"suspicious\",\"confidence\":0.7,\"reasoning\":\"Unusual pattern\"}\n```"
	result := ParseLLMResponse(raw)

	assert.Equal(t, "suspicious", result.Classification)
	assert.Equal(t, 0.7, result.Confidence)
	assert.Equal(t, "Unusual pattern", result.Reasoning)
}

func TestParseLLMResponse_MarkdownFenceWithWhitespace(t *testing.T) {
	raw := "  ```json  \n  {\"classification\":\"benign\",\"confidence\":0.9,\"reasoning\":\"OK\"}  \n  ```  "
	result := ParseLLMResponse(raw)

	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 0.9, result.Confidence)
	assert.Equal(t, "OK", result.Reasoning)
}

func TestParseLLMResponse_MultilineJSONInFences(t *testing.T) {
	raw := "```json\n{\n  \"classification\": \"malicious\",\n  \"confidence\": 0.99,\n  \"reasoning\": \"Contains obfuscated eval\"\n}\n```"
	result := ParseLLMResponse(raw)

	assert.Equal(t, "malicious", result.Classification)
	assert.Equal(t, 0.99, result.Confidence)
	assert.Equal(t, "Contains obfuscated eval", result.Reasoning)
}

func TestParseLLMResponse_JSONWithSurroundingText(t *testing.T) {
	raw := `Here is my analysis: {"classification":"suspicious","confidence":0.7,"reasoning":"Unusual pattern"} based on the diff.`
	result := ParseLLMResponse(raw)

	assert.Equal(t, "suspicious", result.Classification)
	assert.Equal(t, 0.7, result.Confidence)
}

func TestParseLLMResponse_FenceWithTrailingBraces(t *testing.T) {
	// This is the key bug case: text after the fence contains } characters
	raw := "```json\n{\"classification\":\"benign\",\"confidence\":0.95,\"reasoning\":\"OK\"}\n```\n\nThe function changed from func(x int) {} to func(x, y int) {}"
	result := ParseLLMResponse(raw)

	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 0.95, result.Confidence)
	assert.Equal(t, "OK", result.Reasoning)
}

func TestParseLLMResponse_RawJSONWithWhitespace(t *testing.T) {
	raw := "  \n  {\"classification\":\"benign\",\"confidence\":0.95,\"reasoning\":\"OK\"}  \n  "
	result := ParseLLMResponse(raw)

	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestParseLLMResponse_InvalidJSON_Fallback(t *testing.T) {
	raw := "This is not JSON at all, just plain text analysis."
	result := ParseLLMResponse(raw)

	assert.Equal(t, "suspicious", result.Classification)
	assert.Equal(t, 0.5, result.Confidence)
	assert.Contains(t, result.Reasoning, "Failed to parse")
	assert.Equal(t, raw, result.RawResponse)
}

func TestParseLLMResponse_InvalidClassification_Validated(t *testing.T) {
	raw := `{"classification":"unknown","confidence":0.8,"reasoning":"test"}`
	result := ParseLLMResponse(raw)

	// ValidateResult should correct invalid classification to "suspicious"
	assert.Equal(t, "suspicious", result.Classification)
	assert.LessOrEqual(t, result.Confidence, 0.5) // clamped down
}

func TestParseLLMResponse_ConfidenceClamped(t *testing.T) {
	raw := `{"classification":"benign","confidence":1.5,"reasoning":"test"}`
	result := ParseLLMResponse(raw)

	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 1.0, result.Confidence) // clamped to max
}

func TestParseLLMResponse_ReasoningWithBraces(t *testing.T) {
	raw := "```json\n{\"classification\":\"suspicious\",\"confidence\":0.7,\"reasoning\":\"Found eval({data}) and function() { exec(); }\"}\n```"
	result := ParseLLMResponse(raw)

	assert.Equal(t, "suspicious", result.Classification)
	assert.Equal(t, 0.7, result.Confidence)
	assert.Contains(t, result.Reasoning, "eval({data})")
}

func TestParseLLMResponse_WindowsLineEndings(t *testing.T) {
	raw := "```json\r\n{\"classification\":\"benign\",\"confidence\":0.95,\"reasoning\":\"OK\"}\r\n```"
	result := ParseLLMResponse(raw)

	assert.Equal(t, "benign", result.Classification)
	assert.Equal(t, 0.95, result.Confidence)
}

func TestParseLLMResponse_EmptyFences(t *testing.T) {
	raw := "```json\n\n```"
	result := ParseLLMResponse(raw)

	// Should fallback to suspicious
	assert.Equal(t, "suspicious", result.Classification)
	assert.Equal(t, 0.5, result.Confidence)
	assert.Contains(t, result.Reasoning, "Failed to parse")
}
