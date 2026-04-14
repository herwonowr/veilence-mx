package models

import (
	"time"
)

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
	// PackageSourceManual is a user-added package (replaces IsCustom=true).
	PackageSourceManual PackageSource = "manual"
	// PackageSourceDiscovered is an auto-discovered package from registry rankings (replaces IsCustom=false).
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
	// PackageStatusRemoved means the package was removed by a user (discovery may re-add it).
	PackageStatusRemoved PackageStatus = "removed"
)

// Package represents a monitored package from Python (PyPI) or npm.
type Package struct {
	ID            uint          `gorm:"primarykey" json:"id"`
	OrgID         uint          `gorm:"not null;index" json:"orgId"`
	Name          string        `gorm:"not null" json:"name"`
	Ecosystem     Ecosystem     `gorm:"column:ecosystem;not null;type:varchar(10)" json:"ecosystem"`
	LatestVersion string        `gorm:"type:varchar(100)" json:"latestVersion"`
	Description   string        `gorm:"type:text" json:"description"`
	Source        PackageSource `gorm:"not null;default:'manual';type:varchar(20)" json:"source"`
	Status        PackageStatus `gorm:"not null;default:'active';type:varchar(20);index" json:"status"`
	Rank          *uint         `json:"rank,omitempty"`
	BlockedAt     *time.Time    `json:"blockedAt,omitempty"`
	BlockedReason string        `gorm:"type:text" json:"blockedReason,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
	Releases      []Release     `gorm:"foreignKey:PackageID" json:"releases,omitempty"`
}

// TableName returns the table name for Package.
func (Package) TableName() string {
	return "packages"
}
