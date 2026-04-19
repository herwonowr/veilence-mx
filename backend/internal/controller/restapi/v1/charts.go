package v1

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// GetChartData returns aggregated data for dashboard charts, scoped to the current org.
func (h *DashboardHandlers) GetChartData(w http.ResponseWriter, r *http.Request) {
	workspaceID := rbac.WorkspaceIDFromContext(r.Context())

	// Parse time range from query params
	now := time.Now()
	var from, to time.Time

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed.Add(24*time.Hour - time.Second) // end of day
		}
	}

	// Default to last 30 days if no range specified
	if from.IsZero() {
		from = now.AddDate(0, 0, -30)
	}
	if to.IsZero() {
		to = now
	}

	data, err := h.DashboardSvc.GetChartData(r.Context(), workspaceID, from, to)
	if err != nil {
		slog.Error("failed to get chart data", "workspace_id", workspaceID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to get chart data")
		return
	}

	respondJSON(w, http.StatusOK, response.ChartDataFromEntity(data), nil)
}
