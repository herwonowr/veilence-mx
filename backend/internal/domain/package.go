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

// Package represents a monitored package from Python (PyPI) or npm.
type Package struct {
	ID            uint      `json:"id"`
	OrgID         uint      `json:"orgId"`
	Name          string    `json:"name"`
	Ecosystem     Ecosystem `json:"ecosystem"`
	LatestVersion string    `json:"latestVersion"`
	Description   string    `json:"description"`
	IsCustom      bool      `json:"isCustom"`
	Rank          *uint     `json:"rank,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Releases      []Release `json:"releases,omitempty"`
}
