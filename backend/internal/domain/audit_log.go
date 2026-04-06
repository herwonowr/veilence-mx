package domain

import "time"

// AuditLog records a user action for compliance and debugging purposes.
type AuditLog struct {
	ID            uint      `json:"id"`
	UserID        uint      `json:"userId"`
	OrgID         uint      `json:"orgId"`
	Action        string    `json:"action"`
	Resource      string    `json:"resource"`
	ResourceID    uint      `json:"resourceId"`
	Details       string    `json:"details"`
	IPAddress     string    `json:"ipAddress"`
	UserAgent     string    `json:"userAgent"`
	CorrelationID string    `json:"correlationId"`
	CreatedAt     time.Time `json:"createdAt"`
}

// AuditLogFilters holds the query parameters for filtering audit logs.
type AuditLogFilters struct {
	Action   string
	Resource string
	UserID   uint
	FromDate *time.Time
	ToDate   *time.Time
}
