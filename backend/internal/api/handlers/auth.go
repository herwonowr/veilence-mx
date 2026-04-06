package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/veilence/veilence-mx/backend/internal/api/validation"
	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/auth"
)

// registerRequest is the request body for user registration.
type registerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// loginRequest is the request body for user login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// refreshRequest is the request body for token refresh.
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// logoutRequest is the request body for logout.
type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// createAPIKeyRequest is the request body for creating an API key.
type createAPIKeyRequest struct {
	Name      string `json:"name"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// Register handles user registration.
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate fields using the validation package
	req.Email = strings.TrimSpace(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if err := validation.ValidateEmail(req.Email); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}

	if err := validation.ValidatePassword(req.Password); err != nil {
		respondAppError(w, apperror.Validation(err.Error()))
		return
	}

	_, err := h.Auth.Register(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		if errors.Is(err, auth.ErrEmailAlreadyRegistered) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	// Automatically log in the newly registered user to return tokens
	user, tokens, err := h.Auth.Login(req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "registration succeeded but failed to generate tokens")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "user", user.ID, fmt.Sprintf("registered user %s", req.Email))

	respondJSON(w, http.StatusCreated, map[string]any{
		"user":         user,
		"accessToken":  tokens.AccessToken,
		"refreshToken": tokens.RefreshToken,
	}, nil)
}

// Login handles user authentication and returns a token pair.
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || req.Password == "" {
		respondAppError(w, apperror.Validation("email and password are required"))
		return
	}

	user, tokens, err := h.Auth.Login(req.Email, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"user":         user,
		"accessToken":  tokens.AccessToken,
		"refreshToken": tokens.RefreshToken,
	}, nil)
}

// RefreshToken handles token refresh using a refresh token.
func (h *AuthHandlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		respondAppError(w, apperror.Validation("refreshToken is required"))
		return
	}

	tokens, err := h.Auth.RefreshTokens(req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	respondJSON(w, http.StatusOK, tokens, nil)
}

// Logout handles user logout by invalidating the refresh token.
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		respondAppError(w, apperror.Validation("refreshToken is required"))
		return
	}

	if err := h.Auth.Logout(req.RefreshToken); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"}, nil)
}

// GetMe returns the currently authenticated user.
func (h *AuthHandlers) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := h.Auth.GetUserByID(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user, nil)
}

// CreateAPIKey creates a new API key for the authenticated user.
func (h *AuthHandlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req createAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		respondAppError(w, apperror.Validation("name is required"))
		return
	}
	if len(req.Name) > 100 {
		respondAppError(w, apperror.Validation("name must be at most 100 characters"))
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			respondAppError(w, apperror.Validation("expiresAt must be in RFC3339 format"))
			return
		}
		if t.Before(time.Now()) {
			respondAppError(w, apperror.Validation("expiresAt must be in the future"))
			return
		}
		expiresAt = &t
	}

	apiKey, rawKey, err := h.Auth.CreateAPIKey(userID, req.Name, expiresAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "api_key", apiKey.ID, fmt.Sprintf("created API key %q", req.Name))

	respondJSON(w, http.StatusCreated, map[string]any{
		"apiKey": apiKey,
		"key":    rawKey,
	}, nil)
}

// ListAPIKeys returns all API keys for the authenticated user.
func (h *AuthHandlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	keys, err := h.Auth.ListAPIKeys(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list API keys")
		return
	}

	respondJSON(w, http.StatusOK, keys, nil)
}

// RevokeAPIKey deletes an API key belonging to the authenticated user.
func (h *AuthHandlers) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid API key ID")
		return
	}

	if err := h.Auth.RevokeAPIKey(userID, uint(id)); err != nil {
		if errors.Is(err, auth.ErrAPIKeyNotFound) {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke API key")
		return
	}

	h.Audit.LogAction(r.Context(), "revoke", "api_key", uint(id), fmt.Sprintf("revoked API key %d", id))

	respondJSON(w, http.StatusOK, map[string]string{"message": "API key revoked"}, nil)
}
