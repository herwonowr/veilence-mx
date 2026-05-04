package v1

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// normalizeFilterValue normalizes a filter string by replacing spaces with
// underscores and lowercasing, so display-formatted values like "Auth Settings"
// match stored values like "auth_settings".
func normalizeFilterValue(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(strings.ReplaceAll(s, " ", "_"))
}

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
		Action:   normalizeFilterValue(r.URL.Query().Get("action")),
		Resource: normalizeFilterValue(r.URL.Query().Get("resource")),
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

	logs, total, err := h.Audit.ListAuditLogs(r.Context(), workspaceID, filters, page, limit)
	if err != nil {
		respondAppError(w, Internal("failed to list audit logs"))
		return
	}

	result := make([]response.AuditLogResponse, len(logs))
	for i, l := range logs {
		result[i] = response.AuditLogResponse{
			ID: l.ID, UserID: l.UserID, UserEmail: l.UserEmail,
			WorkspaceID: l.WorkspaceID, WorkspaceName: l.WorkspaceName,
			Action: l.Action, Resource: l.Resource, ResourceID: l.ResourceID,
			Details: l.Details, IPAddress: l.IPAddress, UserAgent: l.UserAgent,
			CorrelationID: l.CorrelationID, CreatedAt: l.CreatedAt,
		}
	}
	respondJSON(w, http.StatusOK, result, &Meta{Page: page, Limit: limit, Total: total})
}

// ListAllAuditLogs handles GET /api/admin/audit-logs - returns a paginated list of
// audit logs across all workspaces. Only accessible by super admins.
func (h *AuditHandlers) ListAllAuditLogs(w http.ResponseWriter, r *http.Request) {
	page, limit := parsePagination(r)
	sortClause := parseSort(r, map[string]string{
		"createdAt": "created_at",
		"action":    "action",
		"resource":  "resource",
	}, "created_at DESC")

	filters := audit.AuditLogFilters{
		Action:        normalizeFilterValue(r.URL.Query().Get("action")),
		Resource:      normalizeFilterValue(r.URL.Query().Get("resource")),
		UserEmail:     r.URL.Query().Get("user_email"),
		WorkspaceName: r.URL.Query().Get("workspace_name"),
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

	logs, total, err := h.Audit.ListAllAuditLogs(r.Context(), filters, page, limit, sortClause)
	if err != nil {
		respondAppError(w, Internal("failed to list audit logs"))
		return
	}

	result := make([]response.AuditLogResponse, len(logs))
	for i, l := range logs {
		result[i] = response.AuditLogResponse{
			ID: l.ID, UserID: l.UserID, UserEmail: l.UserEmail,
			WorkspaceID: l.WorkspaceID, WorkspaceName: l.WorkspaceName,
			Action: l.Action, Resource: l.Resource, ResourceID: l.ResourceID,
			Details: l.Details, IPAddress: l.IPAddress, UserAgent: l.UserAgent,
			CorrelationID: l.CorrelationID, CreatedAt: l.CreatedAt,
		}
	}
	respondJSON(w, http.StatusOK, result, &Meta{Page: page, Limit: limit, Total: total})
}
