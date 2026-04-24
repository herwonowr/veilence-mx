package llm

import "log/slog"

// validClassifications is the set of allowed classification values from the LLM.
var validClassifications = map[string]bool{
	"benign":     true,
	"suspicious": true,
	"malicious":  true,
}

// ValidateResult ensures the LLM result has valid classification and confidence values.
// Invalid classifications default to "suspicious" (fail-safe).
// Confidence is clamped to [0.0, 1.0].
func ValidateResult(result *Result) {
	if !validClassifications[result.Classification] {
		slog.Warn("invalid LLM classification, defaulting to suspicious",
			"classification", result.Classification)
		result.Classification = "suspicious"
		if result.Confidence > 0.5 {
			result.Confidence = 0.5
		}
	}
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 1 {
		result.Confidence = 1
	}
}
