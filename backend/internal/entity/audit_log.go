package entity

import "time"

// AuditLog records a user action for compliance and debugging purposes.
type AuditLog struct {
	ID             uint `json:"id"`
	UserID         uint `json:"userId"`
	OrgID          uint `json:"orgId"`
	Action         string `json:"action"`
	Resource       string `json:"resource"`
	ResourceID     uint `json:"resourceId"`
	Details        string `json:"details"`
	IPAddress      string `json:"iPAddress"`
	UserAgent      string `json:"userAgent"`
	CorrelationID  string `json:"correlationId"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AuditLogFilters holds the query parameters for filtering audit logs.
type AuditLogFilters struct {
	Action         string `json:"action"`
	Resource       string `json:"resource"`
	UserID         uint `json:"userId"`
	FromDate       *time.Time `json:"fromDate,omitempty"`
	ToDate         *time.Time `json:"toDate,omitempty"`
}
