package entity

import "time"

// Diff represents the unified diff between two consecutive releases.
type Diff struct {
	ID             string
	ReleaseID      string
	PrevReleaseID  string
	DiffContent    string
	FileChangesCount int
	LinesAdded     int
	LinesRemoved   int
	Truncated      bool
	OriginalSize   int
	CreatedAt      time.Time
	Analyses       []Analysis
}
