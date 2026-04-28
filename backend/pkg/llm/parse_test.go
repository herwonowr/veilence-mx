package llm

import (
	"testing"
)

func TestParseResponse_RawJSON(t *testing.T) {
	raw := `{"classification":"benign","confidence":0.95,"reasoning":"Normal update"}`
	result := ParseResponse(raw)
	if result.Classification != "benign" {
		t.Errorf("expected benign, got %s", result.Classification)
	}
	if result.Confidence != 0.95 {
		t.Errorf("expected 0.95, got %f", result.Confidence)
	}
	if result.Reasoning != "Normal update" {
		t.Errorf("expected 'Normal update', got %s", result.Reasoning)
	}
	if result.RawResponse != raw {
		t.Errorf("expected RawResponse to be preserved")
	}
}

func TestParseResponse_MarkdownFences(t *testing.T) {
	raw := "```json\n{\"classification\":\"malicious\",\"confidence\":0.9,\"reasoning\":\"Backdoor detected\"}\n```"
	result := ParseResponse(raw)
	if result.Classification != "malicious" {
		t.Errorf("expected malicious, got %s", result.Classification)
	}
	if result.Confidence != 0.9 {
		t.Errorf("expected 0.9, got %f", result.Confidence)
	}
}

func TestParseResponse_ExtraTextAroundJSON(t *testing.T) {
	raw := "Here is my analysis:\n{\"classification\":\"suspicious\",\"confidence\":0.7,\"reasoning\":\"Unusual pattern\"}\nEnd."
	result := ParseResponse(raw)
	if result.Classification != "suspicious" {
		t.Errorf("expected suspicious, got %s", result.Classification)
	}
}

func TestParseResponse_InvalidJSON_Fallback(t *testing.T) {
	raw := "I cannot parse this as JSON"
	result := ParseResponse(raw)
	if result.Classification != "suspicious" {
		t.Errorf("expected suspicious fallback, got %s", result.Classification)
	}
	if result.Confidence != 0.5 {
		t.Errorf("expected 0.5 fallback confidence, got %f", result.Confidence)
	}
}

func TestParseResponse_InvalidClassification_DefaultsSuspicious(t *testing.T) {
	raw := `{"classification":"unknown","confidence":0.8,"reasoning":"test"}`
	result := ParseResponse(raw)
	if result.Classification != "suspicious" {
		t.Errorf("expected suspicious (invalid classification corrected), got %s", result.Classification)
	}
	if result.Confidence > 0.5 {
		t.Errorf("expected confidence capped at 0.5 for invalid classification, got %f", result.Confidence)
	}
}

func TestExtractJSONObject(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{"simple", `{"a":1}`, `{"a":1}`, true},
		{"nested", `{"a":{"b":2}}`, `{"a":{"b":2}}`, true},
		{"with prefix", `text {"a":1} more`, `{"a":1}`, true},
		{"string with braces", `{"a":"{}"}`, `{"a":"{}"}`, true},
		{"no json", "no json here", "", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractJSONObject(tt.input)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
