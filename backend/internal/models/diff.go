package models

import (
	"time"
)

// Diff represents the unified diff between two consecutive releases.
type Diff struct {
	ID               uint       `gorm:"primarykey" json:"id"`
	ReleaseID        uint       `gorm:"not null;index" json:"releaseId"`
	Release          Release    `gorm:"foreignKey:ReleaseID" json:"-"`
	PrevReleaseID    uint       `gorm:"not null" json:"prevReleaseId"`
	PrevRelease      Release    `gorm:"foreignKey:PrevReleaseID" json:"-"`
	DiffContent      string     `gorm:"type:text" json:"diffContent"`
	FileChangesCount int        `json:"fileChangesCount"`
	LinesAdded       int        `json:"linesAdded"`
	LinesRemoved     int        `json:"linesRemoved"`
	CreatedAt        time.Time  `json:"createdAt"`
	Analyses         []Analysis `gorm:"foreignKey:DiffID" json:"analyses,omitempty"`
}

// TableName returns the table name for Diff.
func (Diff) TableName() string {
	return "diffs"
}
