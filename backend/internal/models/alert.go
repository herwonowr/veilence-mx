package models

import (
	"time"
)

// AlertSeverity represents the severity level of an alert.
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the current status of an alert.
type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// Alert represents a security alert generated from an analysis.
type Alert struct {
	ID         uint          `gorm:"primarykey" json:"id"`
	OrgID      uint          `gorm:"not null;index" json:"orgId"`
	AnalysisID uint          `gorm:"not null;index" json:"analysisId"`
	Analysis   Analysis      `gorm:"foreignKey:AnalysisID" json:"-"`
	PackageID  uint          `gorm:"not null;index" json:"packageId"`
	Package    Package       `gorm:"foreignKey:PackageID" json:"-"`
	Severity   AlertSeverity `gorm:"not null;type:varchar(20)" json:"severity"`
	Status     AlertStatus   `gorm:"not null;type:varchar(20);default:'new'" json:"status"`
	Message    string        `gorm:"type:text" json:"message"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

// TableName returns the table name for Alert.
func (Alert) TableName() string {
	return "alerts"
}
