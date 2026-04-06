package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/auth"
)

// SessionHandlers handles session management endpoints.
type SessionHandlers struct {
	Auth *auth.Service
}

// ListSessions returns all active sessions for the authenticated user.
// GET /api/v1/sessions
func (h *SessionHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	sessions, err := h.Auth.ListSessions(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list sessions")
		return
	}

	respondJSON(w, http.StatusOK, sessions, nil)
}

// RevokeSession deletes a specific session belonging to the authenticated user.
// DELETE /api/v1/sessions/:id
func (h *SessionHandlers) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid session ID")
		return
	}

	if err := h.Auth.RevokeSession(userID, uint(id)); err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			respondAppError(w, apperror.NotFound("session"))
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "session revoked"}, nil)
}
