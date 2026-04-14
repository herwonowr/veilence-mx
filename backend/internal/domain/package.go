package domain

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
)

// Package represents a monitored package from Python (PyPI) or npm.
type Package struct {
	ID            uint          `json:"id"`
	OrgID         uint          `json:"orgId"`
	Name          string        `json:"name"`
	Ecosystem     Ecosystem     `json:"ecosystem"`
	LatestVersion string        `json:"latestVersion"`
	Description   string        `json:"description"`
	Source        PackageSource `json:"source"`
	Status        PackageStatus `json:"status"`
	Rank          *uint         `json:"rank,omitempty"`
	BlockedAt     *time.Time    `json:"blockedAt,omitempty"`
	BlockedReason string        `json:"blockedReason,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	Releases      []Release     `json:"releases,omitempty"`
}
