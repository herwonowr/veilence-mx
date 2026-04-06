package models

import (
	"time"

	"gorm.io/gorm"
)

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
	ID            uint           `gorm:"primarykey" json:"id"`
	OrgID         uint           `gorm:"not null;index" json:"orgId"`
	Name          string         `gorm:"not null" json:"name"`
	Registry      Registry       `gorm:"not null;type:varchar(10)" json:"registry"`
	LatestVersion string         `gorm:"type:varchar(100)" json:"latestVersion"`
	Description   string         `gorm:"type:text" json:"description"`
	IsCustom      bool           `gorm:"not null;default:false" json:"isCustom"`
	Rank          *uint          `json:"rank"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Releases      []Release      `gorm:"foreignKey:PackageID" json:"releases,omitempty"`
}

// TableName returns the table name for Package.
func (Package) TableName() string {
	return "packages"
}
