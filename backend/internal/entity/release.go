package entity

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
	// ReleaseStatusInProgress is a virtual status that matches pending, diffing, and analyzing.
	ReleaseStatusInProgress ReleaseStatus = "in_progress"
)

// InProgressStatuses returns the real statuses that "in_progress" expands to.
var InProgressStatuses = []ReleaseStatus{ReleaseStatusPending, ReleaseStatusDiffing, ReleaseStatusAnalyzing}

// Release represents a specific version release of a package.
type Release struct {
	ID           string
	WorkspaceID  string
	PackageID    string
	Version      string
	PublishedAt  time.Time
	TarballURL   string
	SHA256       string
	Status       ReleaseStatus
	ErrorMessage string
	CreatedAt    time.Time
	Diffs        []Diff
}

// ReleaseFilters holds optional query filters for listing releases.
type ReleaseFilters struct {
	Ecosystem        *Ecosystem
	Status           *ReleaseStatus
	Search           *string
	Classification   *string
	LatestPerPackage bool
}

// ReleaseDetail contains a release with its associated diff, analysis, and package info.
type ReleaseDetail struct {
	Release    Release
	Package    *Package
	Diff       *Diff
	Analysis   *Analysis
	IsBaseline bool
}

// ReleaseWithDetails contains a release with package info and classification for list views.
type ReleaseWithDetails struct {
	Release
	PackageName      string
	PackageEcosystem string
	Classification   string
}

// AnalysisHistoryEntry holds a single entry in the analysis history for a package.
type AnalysisHistoryEntry struct {
	ReleaseID      string
	Version        string
	Classification string
	Confidence     float64
	Reasoning      string
	ModelUsed      string
	AnalyzerType   string
	AnalyzedAt     string
	PublishedAt    string
}
