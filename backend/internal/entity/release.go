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
)

// Release represents a specific version release of a package.
type Release struct {
	ID             uint `json:"id"`
	PackageID      uint `json:"packageId"`
	Version        string `json:"version"`
	PublishedAt    time.Time `json:"publishedAt"`
	TarballURL     string `json:"tarballUrl"`
	SHA256         string `json:"sha256"`
	Status         ReleaseStatus `json:"status"`
	ErrorMessage   string `json:"errorMessage"`
	CreatedAt      time.Time `json:"createdAt"`
	Diffs          []Diff `json:"diffs,omitempty"`
}

// ReleaseFilters holds optional query filters for listing releases.
type ReleaseFilters struct {
	Ecosystem      *Ecosystem `json:"ecosystem,omitempty"`
	Status         *ReleaseStatus `json:"status,omitempty"`
	Search         *string `json:"search,omitempty"`
	Classification *string `json:"classification,omitempty"`
	LatestPerPackage bool `json:"latestPerPackage"`
}

// ReleaseDetail contains a release with its associated diff, analysis, and package info.
type ReleaseDetail struct {
	Release        Release `json:"release"`
	Package        *Package `json:"package,omitempty"`
	Diff           *Diff `json:"diff,omitempty"`
	Analysis       *Analysis `json:"analysis,omitempty"`
	IsBaseline     bool `json:"isBaseline"`
}

// ReleaseWithDetails contains a release with package info and classification for list views.
type ReleaseWithDetails struct {
	Release
	PackageName    string `json:"packageName"`
	PackageEcosystem string `json:"packageEcosystem"`
	Classification string `json:"classification"`
}

// AnalysisHistoryEntry holds a single entry in the analysis history for a package.
type AnalysisHistoryEntry struct {
	ReleaseID      uint `json:"releaseId"`
	Version        string `json:"version"`
	Classification string `json:"classification"`
	Confidence     float64 `json:"confidence"`
	Reasoning      string `json:"reasoning"`
	ModelUsed      string `json:"modelUsed"`
	AnalyzerType   string `json:"analyzerType"`
	AnalyzedAt     string `json:"analyzedAt"`
	PublishedAt    string `json:"publishedAt"`
}
