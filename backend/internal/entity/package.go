package entity

import "time"

// Ecosystem represents a package ecosystem source.
type Ecosystem string

const (
	// EcosystemPython is the Python Package Index.
	EcosystemPython Ecosystem = "python"
	// EcosystemNPM is the npm ecosystem.
	EcosystemNPM Ecosystem = "npm"
)

// PackageSource describes how a package was added to monitoring.
type PackageSource string

const (
	// PackageSourceManual is a user-added package.
	PackageSourceManual PackageSource = "manual"
	// PackageSourceDiscovered is an auto-discovered package from registry rankings.
	PackageSourceDiscovered PackageSource = "discovered"
	// PackageSourceImported is a bulk-imported package.
	PackageSourceImported PackageSource = "imported"
)

// PackageStatus represents the monitoring status of a package.
type PackageStatus string

const (
	// PackageStatusActive means the package is being monitored.
	PackageStatusActive PackageStatus = "active"
	// PackageStatusBlocked means the package is permanently excluded from discovery and analysis.
	PackageStatusBlocked PackageStatus = "blocked"
	// PackageStatusRemoved means the package was removed by a user.
	PackageStatusRemoved PackageStatus = "removed"
	// PackageStatusSuggested means the package was discovered but not yet approved for monitoring.
	PackageStatusSuggested PackageStatus = "suggested"
)

// Package represents a monitored package from Python (PyPI) or npm.
type Package struct {
	ID                     uint `json:"id"`
	OrgID                  uint `json:"orgId"`
	Name                   string `json:"name"`
	Ecosystem              Ecosystem `json:"ecosystem"`
	LatestVersion          string `json:"latestVersion"`
	Description            string `json:"description"`
	Source                 PackageSource `json:"source"`
	Status                 PackageStatus `json:"status"`
	Rank                   *uint `json:"rank,omitempty"`
	DownloadCount          int64 `json:"downloadCount"`
	PopularityScore        float64 `json:"popularityScore"`
	DownloadCountUpdatedAt *time.Time `json:"downloadCountUpdatedAt,omitempty"`
	BlockedAt              *time.Time `json:"blockedAt,omitempty"`
	BlockedReason          string `json:"blockedReason"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
	Releases               []Release `json:"releases,omitempty"`
}

// PackageFilters holds optional query filters for listing packages.
type PackageFilters struct {
	Ecosystem      *Ecosystem `json:"ecosystem,omitempty"`
	Source         *PackageSource `json:"source,omitempty"`
	Status         *PackageStatus `json:"status,omitempty"`
	Search         *string `json:"search,omitempty"`
}

// ImportEntry represents a single package in a bulk import request.
type ImportEntry struct {
	Name           string `json:"name"`
	Ecosystem      Ecosystem `json:"ecosystem"`
}

// ImportErrorEntry represents a single error in the import result.
type ImportErrorEntry struct {
	Name           string `json:"name"`
	Error          string `json:"error"`
}

// ImportResult summarizes the outcome of a bulk import.
type ImportResult struct {
	Imported       int `json:"imported"`
	Skipped        int `json:"skipped"`
	Errors         []ImportErrorEntry `json:"errors,omitempty"`
}

// PackageDownloadUpdate holds download metrics for a single package.
type PackageDownloadUpdate struct {
	PackageID       uint
	DownloadCount   int64
	PopularityScore float64
}

// PackageRanking holds a package's ranking data from a registry.
type PackageRanking struct {
	Name            string
	DownloadCount   int64
	PopularityScore float64
	Rank            uint
}
