package llm

import (
	"testing"
)

func TestValidateResult_ValidClassifications(t *testing.T) {
	for _, class := range []string{"benign", "suspicious", "malicious"} {
		r := &Result{Classification: class, Confidence: 0.8}
		ValidateResult(r)
		if r.Classification != class {
			t.Errorf("expected %s to remain unchanged, got %s", class, r.Classification)
		}
		if r.Confidence != 0.8 {
			t.Errorf("expected confidence 0.8 unchanged, got %f", r.Confidence)
		}
	}
}

func TestValidateResult_InvalidClassification(t *testing.T) {
	r := &Result{Classification: "unknown", Confidence: 0.9}
	ValidateResult(r)
	if r.Classification != "suspicious" {
		t.Errorf("expected suspicious, got %s", r.Classification)
	}
	if r.Confidence > 0.5 {
		t.Errorf("expected confidence capped at 0.5, got %f", r.Confidence)
	}
}

func TestValidateResult_ConfidenceClamp(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"negative", -0.5, 0},
		{"above one", 1.5, 1},
		{"normal", 0.7, 0.7},
		{"zero", 0, 0},
		{"one", 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{Classification: "benign", Confidence: tt.input}
			ValidateResult(r)
			if r.Confidence != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, r.Confidence)
			}
		})
	}
}
