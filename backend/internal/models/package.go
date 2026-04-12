package models

import (
	"time"

	"gorm.io/gorm"
)

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
	ID            uint           `gorm:"primarykey" json:"id"`
	OrgID         uint           `gorm:"not null;index" json:"orgId"`
	Name          string         `gorm:"not null" json:"name"`
	Ecosystem     Ecosystem      `gorm:"column:ecosystem;not null;type:varchar(10)" json:"ecosystem"`
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
