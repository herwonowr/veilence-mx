package response

import "time"

// AuditLogResponse is the JSON representation of an audit log entry.
// Mapped from persistent.AuditLog at the handler level because the audit
// service currently returns persistent types (pre-existing arch compromise).
type AuditLogResponse struct {
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
