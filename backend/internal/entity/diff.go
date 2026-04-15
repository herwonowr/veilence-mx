package entity

import "time"

// Diff represents the unified diff between two consecutive releases.
type Diff struct {
	ID             uint
	ReleaseID      uint
	PrevReleaseID  uint
	DiffContent    string
	FileChangesCount int
	LinesAdded     int
	LinesRemoved   int
	CreatedAt      time.Time
	Analyses       []Analysis
}
