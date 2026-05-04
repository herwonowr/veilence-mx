package v1

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// ListAuditLogs handles GET /api/workspaces/{workspaceId}/audit-logs - returns a paginated
// list of audit logs for a workspace with optional filters.
func (h *AuditHandlers) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())
	if workspaceID == "" {
		respondAppError(w, BadRequest("workspace context required"))
		return
	}

	page, limit := parsePagination(r)

	filters := audit.AuditLogFilters{
		Action:   r.URL.Query().Get("action"),
		Resource: r.URL.Query().Get("resource"),
	}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if _, err := uuid.Parse(userIDStr); err != nil {
			respondAppError(w, BadRequest("invalid user_id format"))
			return
		}
		filters.UserID = userIDStr
	}

	if fromStr := r.URL.Query().Get("from_date"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filters.FromDate = &t
		}
	}

	if toStr := r.URL.Query().Get("to_date"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filters.ToDate = &t
		}
	}

	logs, total, err := h.Audit.ListAuditLogs(workspaceID, filters, page, limit)
	if err != nil {
		respondAppError(w, Internal("failed to list audit logs"))
		return
	}

	result := make([]response.AuditLogResponse, len(logs))
	for i, l := range logs {
		result[i] = response.AuditLogResponse{
			ID: l.ID, UserID: l.UserID, UserEmail: l.UserEmail, WorkspaceID: l.WorkspaceID,
			Action: l.Action, Resource: l.Resource, ResourceID: l.ResourceID,
			Details: l.Details, IPAddress: l.IPAddress, UserAgent: l.UserAgent,
			CorrelationID: l.CorrelationID, CreatedAt: l.CreatedAt,
		}
	}
	respondJSON(w, http.StatusOK, result, &Meta{Page: page, Limit: limit, Total: total})
}
