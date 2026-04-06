package domain

import "time"

// ReleaseStatus represents the processing status of a release.
type ReleaseStatus string

const (
	// ReleaseStatusPending indicates the release is awaiting processing.
	ReleaseStatusPending ReleaseStatus = "pending"
	// ReleaseStatusDiffing indicates the release is being diffed.
	ReleaseStatusDiffing ReleaseStatus = "diffing"
	// ReleaseStatusAnalyzing indicates the release diff is being analyzed.
	ReleaseStatusAnalyzing ReleaseStatus = "analyzing"
	// ReleaseStatusCompleted indicates the release has been fully processed.
	ReleaseStatusCompleted ReleaseStatus = "completed"
	// ReleaseStatusError indicates an error occurred during processing.
	ReleaseStatusError ReleaseStatus = "error"
)

// Release represents a specific version release of a package.
type Release struct {
	ID           uint          `json:"id"`
	PackageID    uint          `json:"packageId"`
	Version      string        `json:"version"`
	PublishedAt  time.Time     `json:"publishedAt"`
	TarballURL   string        `json:"tarballUrl"`
	SHA256       string        `json:"sha256"`
	Status       ReleaseStatus `json:"status"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
	CreatedAt    time.Time     `json:"createdAt"`
	Diffs        []Diff        `json:"diffs,omitempty"`
}
