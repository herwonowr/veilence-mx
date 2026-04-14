package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// SessionHandlers handles session management endpoints.
type SessionHandlers struct {
	Auth *auth.Service
}

// SessionResponse is the response DTO for a session, including a flag
// indicating whether it is the caller's current session.
type SessionResponse struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"userId"`
	IPAddress  string    `json:"ipAddress"`
	UserAgent  string    `json:"userAgent"`
	CreatedAt  time.Time `json:"createdAt"`
	LastActive time.Time `json:"lastActive"`
	ExpiresAt  time.Time `json:"expiresAt"`
	IsCurrent  bool      `json:"isCurrent"`
}

// ListSessions returns all active sessions for the authenticated user.
// If the X-Refresh-Token header is provided, the response marks the
// matching session as the current session (isCurrent: true).
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

	// Hash the caller's refresh token to identify the current session.
	var currentTokenHash string
	if rt := r.Header.Get("X-Refresh-Token"); rt != "" {
		h := sha256.Sum256([]byte(rt))
		currentTokenHash = hex.EncodeToString(h[:])
	}

	resp := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = SessionResponse{
			ID:         s.ID,
			UserID:     s.UserID,
			IPAddress:  s.IPAddress,
			UserAgent:  s.UserAgent,
			CreatedAt:  s.CreatedAt,
			LastActive: s.LastActive,
			ExpiresAt:  s.ExpiresAt,
			IsCurrent:  currentTokenHash != "" && s.TokenHash == currentTokenHash,
		}
	}

	respondJSON(w, http.StatusOK, resp, nil)
}

// RevokeSession deletes a specific session belonging to the authenticated user.
// It refuses to delete the caller's current session (identified via X-Refresh-Token header).
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

	// Prevent deleting the current session — identify it via X-Refresh-Token header
	if rt := r.Header.Get("X-Refresh-Token"); rt != "" {
		rtHash := sha256.Sum256([]byte(rt))
		currentTokenHash := hex.EncodeToString(rtHash[:])
		if err := h.Auth.GuardCurrentSession(userID, uint(id), currentTokenHash); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	if err := h.Auth.RevokeSession(userID, uint(id)); err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			respondAppError(w, NotFound("session"))
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke session")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "session revoked"}, nil)
}
