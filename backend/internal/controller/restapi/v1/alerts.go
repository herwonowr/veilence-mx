package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/response"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

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

	var filters entity.AlertFilters
	if sev := r.URL.Query().Get("severity"); sev != "" {
		validSeverities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if !validSeverities[sev] {
			respondAppError(w, Validation("invalid severity filter"))
			return
		}
		s := entity.AlertSeverity(sev)
		filters.Severity = &s
	}
	if st := r.URL.Query().Get("status"); st != "" {
		validStatuses := map[string]bool{"new": true, "acknowledged": true, "resolved": true}
		if !validStatuses[st] {
			respondAppError(w, Validation("invalid status filter"))
			return
		}
		s := entity.AlertStatus(st)
		filters.Status = &s
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}

	alerts, total, err := h.AlertSvc.ListAlerts(r.Context(), orgID, page, limit, sortOrder, filters)
	if err != nil {
		respondAppError(w, Internal("failed to list alerts"))
		return
	}

	respondJSON(w, http.StatusOK, response.AlertsWithPackageFromEntities(alerts), &Meta{Page: page, Limit: limit, Total: total})
}

// GetAlert returns a single alert by ID, scoped to the current org.
// Returns the alert with associated package info (packageName, packageEcosystem).
func (h *AlertHandlers) GetAlert(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, BadRequest("invalid alert ID"))
		return
	}

	alert, pkg, err := h.AlertSvc.GetAlert(r.Context(), orgID, uint(id))
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("alert"))
			return
		}
		respondAppError(w, Internal("failed to get alert"))
		return
	}

	respondJSON(w, http.StatusOK, response.AlertDetailFromEntity(alert, pkg), nil)
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
		respondAppError(w, BadRequest("invalid alert ID"))
		return
	}

	var req updateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	validStatuses := map[string]bool{
		"new":          true,
		"acknowledged": true,
		"resolved":     true,
	}
	if !validStatuses[req.Status] {
		respondAppError(w, Validation("status must be 'new', 'acknowledged', or 'resolved'"))
		return
	}

	alert, err := h.AlertSvc.UpdateAlertStatus(r.Context(), orgID, uint(id), entity.AlertStatus(req.Status))
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("alert"))
			return
		}
		respondAppError(w, Internal("failed to update alert"))
		return
	}

	respondJSON(w, http.StatusOK, response.AlertFromEntity(alert), nil)
}

// ListAlertNotes returns all notes for a specific alert.
func (h *AlertHandlers) ListAlertNotes(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())

	alertID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, BadRequest("invalid alert ID"))
		return
	}

	notes, err := h.Notes.ListByAlert(r.Context(), orgID, uint(alertID))
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("alert"))
			return
		}
		respondAppError(w, Internal("failed to list alert notes"))
		return
	}

	respondJSON(w, http.StatusOK, response.AlertNotesFromEntities(notes), nil)
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
		respondAppError(w, BadRequest("invalid alert ID"))
		return
	}

	var req createAlertNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	if req.Content == "" {
		respondAppError(w, Validation("content is required"))
		return
	}

	if len(req.Content) > entity.MaxNoteLength {
		respondAppError(w, Validation(fmt.Sprintf("content must be at most %d characters", entity.MaxNoteLength)))
		return
	}

	note, err := h.Notes.Create(r.Context(), orgID, uint(alertID), userID, req.Content)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("alert"))
			return
		}
		respondAppError(w, Internal("failed to create alert note"))
		return
	}

	h.Audit.LogAction(r.Context(), "create", "alert_note", note.ID,
		fmt.Sprintf("added note on alert %d", alertID))

	respondJSON(w, http.StatusCreated, response.AlertNoteFromEntity(note), nil)
}

// updateAlertNoteRequest is the request body for updating a note.
type updateAlertNoteRequest struct {
	Content string `json:"content"`
}

// UpdateAlertNote edits an existing note. Only the note's author may edit.
func (h *AlertHandlers) UpdateAlertNote(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	userID := rbac.UserIDFromContext(r.Context())

	alertID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, BadRequest("invalid alert ID"))
		return
	}

	noteID, err := strconv.ParseUint(chi.URLParam(r, "noteId"), 10, 64)
	if err != nil {
		respondAppError(w, BadRequest("invalid note ID"))
		return
	}

	var req updateAlertNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAppError(w, BadRequest("invalid request body"))
		return
	}

	if req.Content == "" {
		respondAppError(w, Validation("content is required"))
		return
	}

	if len(req.Content) > entity.MaxNoteLength {
		respondAppError(w, Validation(fmt.Sprintf("content must be at most %d characters", entity.MaxNoteLength)))
		return
	}

	note, err := h.Notes.Update(r.Context(), orgID, uint(alertID), uint(noteID), userID, req.Content)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("alert note"))
			return
		}
		if errors.Is(err, entity.ErrForbidden) {
			respondAppError(w, Forbidden("you can only edit your own notes"))
			return
		}
		respondAppError(w, Internal("failed to update alert note"))
		return
	}

	h.Audit.LogAction(r.Context(), "update", "alert_note", note.ID,
		fmt.Sprintf("edited note on alert %d", alertID))

	respondJSON(w, http.StatusOK, response.AlertNoteFromEntity(note), nil)
}

// DeleteAlertNote removes an existing note. Only the note's author may delete.
func (h *AlertHandlers) DeleteAlertNote(w http.ResponseWriter, r *http.Request) {
	orgID := rbac.OrgIDFromContext(r.Context())
	userID := rbac.UserIDFromContext(r.Context())

	alertID, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondAppError(w, BadRequest("invalid alert ID"))
		return
	}

	noteID, err := strconv.ParseUint(chi.URLParam(r, "noteId"), 10, 64)
	if err != nil {
		respondAppError(w, BadRequest("invalid note ID"))
		return
	}

	if err := h.Notes.Delete(r.Context(), orgID, uint(alertID), uint(noteID), userID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			respondAppError(w, NotFound("alert note"))
			return
		}
		if errors.Is(err, entity.ErrForbidden) {
			respondAppError(w, Forbidden("you can only delete your own notes"))
			return
		}
		respondAppError(w, Internal("failed to delete alert note"))
		return
	}

	h.Audit.LogAction(r.Context(), "delete", "alert_note", uint(noteID),
		fmt.Sprintf("deleted note on alert %d", alertID))

	w.WriteHeader(http.StatusNoContent)
}
