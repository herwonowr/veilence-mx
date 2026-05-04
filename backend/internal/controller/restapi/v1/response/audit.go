package response

import "time"

// AuditLogResponse is the JSON representation of an audit log entry.
type AuditLogResponse struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	UserEmail     string    `json:"userEmail"`
	WorkspaceID   string    `json:"workspaceId"`
	WorkspaceName string    `json:"workspaceName"`
	Action        string    `json:"action"`
	Resource      string    `json:"resource"`
	ResourceID    string    `json:"resourceId"`
	Details       string    `json:"details"`
	IPAddress     string    `json:"ipAddress"`
	UserAgent     string    `json:"userAgent"`
	CorrelationID string    `json:"correlationId"`
	CreatedAt     time.Time `json:"createdAt"`
}
