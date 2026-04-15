package entity

import "time"

// AuditLog records a user action for compliance and debugging purposes.
type AuditLog struct {
	ID             uint
	UserID         uint
	OrgID          uint
	Action         string
	Resource       string
	ResourceID     uint
	Details        string
	IPAddress      string
	UserAgent      string
	CorrelationID  string
	CreatedAt      time.Time
}

// AuditLogFilters holds the query parameters for filtering audit logs.
type AuditLogFilters struct {
	Action         string
	Resource       string
	UserID         uint
	FromDate       *time.Time
	ToDate         *time.Time
}
