package llm

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// codeBlockRe matches markdown code fences: ```json ... ``` or ``` ... ```
// It captures the content between the fences.
var codeBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)\\n?\\s*```")

// stripMarkdownFences removes markdown code fences from the response text.
// It handles ```json ... ```, ``` ... ```, and returns the inner content.
// If no fences are found, the original text is returned unchanged.
func stripMarkdownFences(raw string) string {
	matches := codeBlockRe.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return raw
}

// ParseResponse parses an LLM response that may be raw JSON or JSON wrapped
// in markdown code fences (```json ... ``` or ``` ... ```).
//
// The parsing strategy is:
//  1. Try direct JSON parse (handles raw JSON responses)
//  2. Strip markdown code fences and try again
//  3. Extract JSON by finding the outermost { } in the stripped text
//  4. Fallback: return suspicious classification with 50% confidence
func ParseResponse(rawResponse string) *Result {
	trimmed := strings.TrimSpace(rawResponse)

	// Step 1: Try direct JSON parse.
	var result Result
	if err := json.Unmarshal([]byte(trimmed), &result); err == nil {
		result.RawResponse = rawResponse
		ValidateResult(&result)
		return &result
	}

	// Step 2: Strip markdown code fences and try again.
	stripped := stripMarkdownFences(trimmed)
	if stripped != trimmed {
		if err := json.Unmarshal([]byte(stripped), &result); err == nil {
			result.RawResponse = rawResponse
			ValidateResult(&result)
			slog.Debug("parsed LLM response after stripping markdown fences")
			return &result
		}
	}

	// Step 3: Extract JSON object by finding outermost { } in the stripped text.
	if extracted, ok := extractJSONObject(stripped); ok {
		if err := json.Unmarshal([]byte(extracted), &result); err == nil {
			result.RawResponse = rawResponse
			ValidateResult(&result)
			slog.Debug("parsed LLM response via JSON extraction")
			return &result
		}
	}

	// Step 4: Fallback - mark as suspicious (fail-safe).
	slog.Warn("failed to parse LLM response, falling back to suspicious",
		"raw_length", len(rawResponse),
	)
	return &Result{
		Classification: "suspicious",
		Confidence:     0.5,
		Reasoning:      fmt.Sprintf("Failed to parse LLM response. Raw: %s", rawResponse),
		RawResponse:    rawResponse,
	}
}

// extractJSONObject finds the first top-level JSON object in text by matching
// braces. It handles nested braces and string literals correctly.
func extractJSONObject(text string) (string, bool) {
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return "", false
	}

	// Walk forward from start, counting brace depth, respecting strings.
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(text); i++ {
		ch := text[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1], true
			}
		}
	}

	return "", false
}
