package llm

// Result holds the output of an LLM analysis.
type Result struct {
	Classification string  `json:"classification"` // benign, suspicious, malicious
	Confidence     float64 `json:"confidence"`     // 0.0 - 1.0
	Reasoning      string  `json:"reasoning"`
	RawResponse    string  `json:"rawResponse"`
}
