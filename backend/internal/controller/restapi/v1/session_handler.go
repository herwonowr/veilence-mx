package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// SessionHandlers handles session management endpoints.
type SessionHandlers struct {
	Auth  *auth.Service
	Audit *audit.Service
}

// SessionResponse is the response DTO for a session, including a flag
// indicating whether it is the caller's current session.
type SessionResponse struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	IPAddress    string    `json:"ipAddress"`
	UserAgent    string    `json:"userAgent"`
	AuthProvider string    `json:"authProvider"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActive   time.Time `json:"lastActive"`
	ExpiresAt    time.Time `json:"expiresAt"`
	IsCurrent    bool      `json:"isCurrent"`
}

// ListSessions returns all active sessions for the authenticated user.
// If the X-Refresh-Token header is provided, the response marks the
// matching session as the current session (isCurrent: true).
// GET /api/v1/sessions
func (h *SessionHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	sessions, err := h.Auth.ListSessions(userID)
	if err != nil {
		respondAppError(w, Internal("failed to list sessions"))
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
			ID:           s.ID,
			UserID:       s.UserID,
			IPAddress:    s.IPAddress,
			UserAgent:    s.UserAgent,
			AuthProvider: s.AuthProvider,
			CreatedAt:    s.CreatedAt,
			LastActive:   s.LastActive,
			ExpiresAt:    s.ExpiresAt,
			IsCurrent:    currentTokenHash != "" && s.TokenHash == currentTokenHash,
		}
	}

	respondJSON(w, http.StatusOK, resp, nil)
}

// RevokeSession deletes a specific session belonging to the authenticated user.
// It refuses to delete the caller's current session (identified via X-Refresh-Token header).
// DELETE /api/v1/sessions/:id
func (h *SessionHandlers) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		respondAppError(w, Unauthorized("authentication required"))
		return
	}

	id, ok := parseUUID(r, "id")
	if !ok {
		respondAppError(w, BadRequest("invalid session ID"))
		return
	}

	// Prevent deleting the current session - identify it via X-Refresh-Token header
	if rt := r.Header.Get("X-Refresh-Token"); rt != "" {
		rtHash := sha256.Sum256([]byte(rt))
		currentTokenHash := hex.EncodeToString(rtHash[:])
		if err := h.Auth.GuardCurrentSession(userID, id, currentTokenHash); err != nil {
			respondAppError(w, BadRequest("cannot revoke the current session"))
			return
		}
	}

	if err := h.Auth.RevokeSession(userID, id); err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			respondAppError(w, NotFound("session"))
			return
		}
		respondAppError(w, Internal("failed to revoke session"))
		return
	}

	h.Audit.LogAction(r.Context(), "revoke", "session", id, fmt.Sprintf("revoked session %s", id))

	respondJSON(w, http.StatusOK, map[string]string{"message": "session revoked"}, nil)
}
