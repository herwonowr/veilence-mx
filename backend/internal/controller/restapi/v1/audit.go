package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// ListAuditLogs handles GET /api/orgs/{orgId}/audit-logs — returns a paginated
// list of audit logs for an organization with optional filters.
func (h *AuditHandlers) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	if orgID == 0 {
		respondError(w, http.StatusBadRequest, "organization context required")
		return
	}

	page, limit := parsePagination(r)

	filters := audit.AuditLogFilters{
		Action:   r.URL.Query().Get("action"),
		Resource: r.URL.Query().Get("resource"),
	}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		uid, err := strconv.ParseUint(userIDStr, 10, 64)
		if err == nil {
			filters.UserID = uint(uid)
		}
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

	logs, total, err := h.Audit.ListAuditLogs(orgID, filters, page, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	respondJSON(w, http.StatusOK, logs, &Meta{Page: page, Limit: limit, Total: total})
}
