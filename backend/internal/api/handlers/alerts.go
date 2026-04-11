package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// alertWithDetails includes the associated package info.
type alertWithDetails struct {
	models.Alert
	PackageName     string `json:"packageName"`
	PackageRegistry string `json:"packageRegistry"`
}

// ListAlerts returns a paginated list of alerts scoped to the current org.
// Supports optional query params: severity, status, search.
func (h *AlertHandlers) ListAlerts(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	page, limit := parsePagination(r)
	sortOrder := parseSort(r, map[string]string{
		"severity":    "alerts.severity",
		"status":      "alerts.status",
		"createdAt":   "alerts.created_at",
		"message":     "alerts.message",
		"packageName": "packages.name",
	}, "alerts.created_at DESC")
	severity := r.URL.Query().Get("severity")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	query := h.DB.Model(&models.Alert{}).
		Where("alerts.org_id = ?", orgID).
		Joins("JOIN packages ON packages.id = alerts.package_id")

	validSeverities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	validFilterStatuses := map[string]bool{"new": true, "acknowledged": true, "resolved": true}

	if severity != "" {
		if !validSeverities[severity] {
			respondAppError(w, apperror.Validation("invalid severity filter"))
			return
		}
		query = query.Where("alerts.severity = ?", severity)
	}
	if status != "" {
		if !validFilterStatuses[status] {
			respondAppError(w, apperror.Validation("invalid status filter"))
			return
		}
		query = query.Where("alerts.status = ?", status)
	}
	if search != "" {
		search = escapeLike(search)
		query = query.Where("(packages.name ILIKE ? OR alerts.message ILIKE ?)",
			"%"+search+"%", "%"+search+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to count alerts"))
		return
	}

	var alerts []models.Alert
	if err := query.Preload("Package").
		Order(sortOrder).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&alerts).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to list alerts"))
		return
	}

	result := make([]alertWithDetails, len(alerts))
	for i, a := range alerts {
		result[i] = alertWithDetails{
			Alert:           a,
			PackageName:     a.Package.Name,
			PackageRegistry: string(a.Package.Registry),
		}
	}

	respondJSON(w, http.StatusOK, result, &Meta{Page: page, Limit: limit, Total: total})
}

// updateAlertRequest is the request body for updating an alert.
type updateAlertRequest struct {
	Status string `json:"status"`
}

// UpdateAlert updates the status of an alert scoped to the current org.
func (h *AlertHandlers) UpdateAlert(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid alert ID"))
		return
	}

	var req updateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, apperror.BadRequest("invalid request body"))
		return
	}

	validStatuses := map[string]bool{
		"new":          true,
		"acknowledged": true,
		"resolved":     true,
	}
	if !validStatuses[req.Status] {
		respondAppError(w, apperror.Validation("status must be 'new', 'acknowledged', or 'resolved'"))
		return
	}

	var alert models.Alert
	if err := h.DB.Where("id = ? AND org_id = ?", id, orgID).First(&alert).Error; err != nil {
		respondAppError(w, apperror.NotFound("alert"))
		return
	}

	if err := h.DB.Model(&alert).Update("status", req.Status).Error; err != nil {
		respondAppError(w, apperror.Internal("failed to update alert"))
		return
	}
	alert.Status = models.AlertStatus(req.Status)

	h.Audit.LogAction(r.Context(), "update", "alert", alert.ID, fmt.Sprintf("updated alert status to %q", req.Status))

	respondJSON(w, http.StatusOK, alert, nil)
}

// ListAlertNotes returns all notes for a specific alert.
func (h *AlertHandlers) ListAlertNotes(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	alertID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid alert ID"))
		return
	}

	// Verify alert belongs to org
	var alert models.Alert
	if err := h.DB.Where("id = ? AND org_id = ?", alertID, orgID).First(&alert).Error; err != nil {
		respondAppError(w, apperror.NotFound("alert"))
		return
	}

	notes, err := h.AlertNotes.FindByAlertID(r.Context(), uint(alertID))
	if err != nil {
		respondAppError(w, apperror.Internal("failed to list alert notes"))
		return
	}

	respondJSON(w, http.StatusOK, notes, nil)
}

// createAlertNoteRequest is the request body for creating a note on an alert.
type createAlertNoteRequest struct {
	Content string `json:"content"`
}

// CreateAlertNote adds a note/comment to an alert.
func (h *AlertHandlers) CreateAlertNote(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	userID := rbac.UserIDFromContext(r.Context())

	alertID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, apperror.BadRequest("invalid alert ID"))
		return
	}

	// Verify alert belongs to org
	var alert models.Alert
	if err := h.DB.Where("id = ? AND org_id = ?", alertID, orgID).First(&alert).Error; err != nil {
		respondAppError(w, apperror.NotFound("alert"))
		return
	}

	var req createAlertNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, apperror.BadRequest("invalid request body"))
		return
	}

	if req.Content == "" {
		respondAppError(w, apperror.Validation("content is required"))
		return
	}

	const maxNoteLength = 10000
	if len(req.Content) > maxNoteLength {
		respondAppError(w, apperror.Validation(fmt.Sprintf("content must be at most %d characters", maxNoteLength)))
		return
	}

	// Look up user email
	userEmail := h.getUserEmail(userID)

	note := &domain.AlertNote{
		AlertID:   uint(alertID),
		OrgID:     orgID,
		UserID:    userID,
		UserEmail: userEmail,
		Content:   req.Content,
	}

	if err := h.AlertNotes.Create(r.Context(), note); err != nil {
		respondAppError(w, apperror.Internal("failed to create alert note"))
		return
	}

	h.Audit.LogAction(r.Context(), "create", "alert_note", note.ID,
		fmt.Sprintf("added note on alert %d", alertID))

	respondJSON(w, http.StatusCreated, note, nil)
}

// getUserEmail retrieves a user's email from the database.
func (h *AlertHandlers) getUserEmail(userID uint) string {
	var email string
	h.DB.Table("users").Where("id = ?", userID).Pluck("email", &email)
	return email
}
