package models

import (
	"time"
)

// AuditLog records a user action for compliance and debugging purposes.
type AuditLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	UserID        uint      `gorm:"index" json:"userId"`
	OrgID         uint      `gorm:"index" json:"orgId"`
	Action        string    `gorm:"not null;type:varchar(50)" json:"action"`
	Resource      string    `gorm:"not null;type:varchar(50)" json:"resource"`
	ResourceID    uint      `json:"resourceId"`
	Details       string    `gorm:"type:text" json:"details"`
	IPAddress     string    `gorm:"type:varchar(45)" json:"ipAddress"`
	UserAgent     string    `gorm:"type:varchar(255)" json:"userAgent"`
	CorrelationID string    `gorm:"type:varchar(36);index" json:"correlationId"`
	CreatedAt     time.Time `gorm:"index" json:"createdAt"`
}

// TableName returns the table name for AuditLog.
func (AuditLog) TableName() string {
	return "audit_logs"
}
