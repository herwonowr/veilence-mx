package domain

import "time"

// Registry represents a package registry source.
type Registry string

const (
	// RegistryPyPI is the Python Package Index.
	RegistryPyPI Registry = "pypi"
	// RegistryNPM is the npm registry.
	RegistryNPM Registry = "npm"
)

// Package represents a monitored package from PyPI or npm.
type Package struct {
	ID            uint      `json:"id"`
	OrgID         uint      `json:"orgId"`
	Name          string    `json:"name"`
	Registry      Registry  `json:"registry"`
	LatestVersion string    `json:"latestVersion"`
	Description   string    `json:"description"`
	IsCustom      bool      `json:"isCustom"`
	Rank          *uint     `json:"rank,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Releases      []Release `json:"releases,omitempty"`
}
