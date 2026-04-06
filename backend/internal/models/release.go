package models

import (
	"time"
)

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
	ID           uint          `gorm:"primarykey" json:"id"`
	PackageID    uint          `gorm:"not null;index" json:"packageId"`
	Package      Package       `gorm:"foreignKey:PackageID" json:"-"`
	Version      string        `gorm:"not null;type:varchar(100)" json:"version"`
	PublishedAt  time.Time     `json:"publishedAt"`
	TarballURL   string        `gorm:"type:text" json:"tarballUrl"`
	SHA256       string        `gorm:"type:varchar(64)" json:"sha256"`
	Status       ReleaseStatus `gorm:"not null;type:varchar(20);default:'pending'" json:"status"`
	ErrorMessage string        `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt    time.Time     `json:"createdAt"`
	Diffs        []Diff        `gorm:"foreignKey:ReleaseID" json:"diffs,omitempty"`
}

// TableName returns the table name for Release.
func (Release) TableName() string {
	return "releases"
}
