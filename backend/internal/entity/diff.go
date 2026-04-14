package entity

import "time"

// Diff represents the unified diff between two consecutive releases.
type Diff struct {
	ID             uint `json:"id"`
	ReleaseID      uint `json:"releaseId"`
	PrevReleaseID  uint `json:"prevReleaseId"`
	DiffContent    string `json:"diffContent"`
	FileChangesCount int `json:"fileChangesCount"`
	LinesAdded     int `json:"linesAdded"`
	LinesRemoved   int `json:"linesRemoved"`
	CreatedAt      time.Time `json:"createdAt"`
	Analyses       []Analysis `json:"analyses,omitempty"`
}
