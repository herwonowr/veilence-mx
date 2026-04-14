package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	validation "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1/request"
	
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/entity"
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
	Scope     string `json:"scope,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// createAPIKeyResponse is the response body for API key creation.
// It flattens the domain.APIKey fields and includes the raw key string
// (only available at creation time, before the key is hashed).
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
		respondAppError(w, Validation(err.Error()))
		return
	}

	if err := validation.ValidatePassword(req.Password); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}

	_, err := h.Auth.Register(req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		if errors.Is(err, auth.ErrEmailAlreadyRegistered) {
			respondAppError(w, Conflict("email already registered"))
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	// Automatically log in the newly registered user to return tokens
	user, tokens, err := h.Auth.Login(req.Email, req.Password, r.RemoteAddr, r.UserAgent())
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
		respondAppError(w, Validation("email and password are required"))
		return
	}

	user, tokens, err := h.Auth.Login(req.Email, req.Password, r.RemoteAddr, r.UserAgent())
	if err != nil {
		// Log failed login attempt
		h.Audit.LogAuthEvent(r.Context(), "login_failed", 0, fmt.Sprintf("failed login attempt for email %s", req.Email))
		respondError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// Log successful login
	h.Audit.LogAuthEvent(r.Context(), "login", user.ID, fmt.Sprintf("user %s logged in", user.Email))

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
		respondAppError(w, Validation("refreshToken is required"))
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
		respondAppError(w, Validation("refreshToken is required"))
		return
	}

	if err := h.Auth.Logout(req.RefreshToken); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	// Log logout event
	userID := auth.UserIDFromContext(r.Context())
	h.Audit.LogAuthEvent(r.Context(), "logout", userID, "user logged out")

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
		respondAppError(w, Validation("name is required"))
		return
	}
	if len(req.Name) > 100 {
		respondAppError(w, Validation("name must be at most 100 characters"))
		return
	}

	// Validate scope (default to "read" if not specified)
	scope := entity.APIKeyScope(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = entity.APIKeyScopeRead
	}
	if !entity.IsValidAPIKeyScope(string(scope)) {
		respondAppError(w, Validation("scope must be one of: read, write, admin"))
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			respondAppError(w, Validation("expiresAt must be in RFC3339 format"))
			return
		}
		if t.Before(time.Now()) {
			respondAppError(w, Validation("expiresAt must be in the future"))
			return
		}
		expiresAt = &t
	}

	apiKey, rawKey, err := h.Auth.CreateAPIKey(userID, req.Name, scope, expiresAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create API key")
		return
	}

	h.Audit.LogAction(r.Context(), "create", "api_key", apiKey.ID, fmt.Sprintf("created API key %q with scope %q", req.Name, scope))

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
			respondAppError(w, NotFound("API key"))
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to revoke API key")
		return
	}

	h.Audit.LogAction(r.Context(), "revoke", "api_key", uint(id), fmt.Sprintf("revoked API key %d", id))

	respondJSON(w, http.StatusOK, map[string]string{"message": "API key revoked"}, nil)
}

// forgotPasswordRequest is the request body for initiating a password reset.
type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles POST /api/auth/forgot-password — initiates a password reset.
// Always returns 200 to prevent user enumeration.
func (h *AuthHandlers) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		respondAppError(w, Validation("email is required"))
		return
	}

	// TODO: token should be emailed by the service layer via SMTP.
	// The raw token is intentionally not included in the HTTP response.
	_, err := h.Auth.ForgotPassword(req.Email)
	if err != nil {
		// Log the error but don't reveal it to the client
		respondJSON(w, http.StatusOK, map[string]string{
			"message": "if an account with that email exists, a password reset link has been sent",
		}, nil)
		return
	}

	// The token is sent via email, never exposed in the HTTP response.
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "if an account with that email exists, a password reset link has been sent",
	}, nil)
}

// resetPasswordRequest is the request body for resetting a password.
type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

// ResetPassword handles POST /api/auth/reset-password — resets user password with a valid token.
func (h *AuthHandlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		respondAppError(w, Validation("token is required"))
		return
	}

	if err := validation.ValidatePassword(req.NewPassword); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}

	if err := h.Auth.ResetPassword(req.Token, req.NewPassword); err != nil {
		if errors.Is(err, auth.ErrResetTokenInvalid) || errors.Is(err, auth.ErrResetTokenUsed) {
			respondAppError(w, BadRequest("invalid or expired reset token"))
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	h.Audit.LogAction(r.Context(), "reset", "password", 0, "password reset via token")

	respondJSON(w, http.StatusOK, map[string]string{"message": "password reset successfully"}, nil)
}

// verifyEmailRequest is the request body for verifying an email address.
type verifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmail handles POST /api/auth/verify-email — verifies user email with a valid token.
func (h *AuthHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		respondAppError(w, Validation("token is required"))
		return
	}

	if err := h.Auth.VerifyEmail(req.Token); err != nil {
		if errors.Is(err, auth.ErrVerificationInvalid) {
			respondAppError(w, BadRequest("invalid or expired verification token"))
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to verify email")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "email verified successfully"}, nil)
}

// SendVerificationEmail handles POST /api/auth/send-verification — sends a new verification email.
func (h *AuthHandlers) SendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// TODO: token should be emailed by the service layer via SMTP.
	// The raw token is intentionally not included in the HTTP response.
	_, err := h.Auth.GenerateEmailVerificationToken(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate verification token")
		return
	}

	// The token is sent via email, never exposed in the HTTP response.
	respondJSON(w, http.StatusOK, map[string]string{
		"message": "verification email sent",
	}, nil)
}

// updateProfileRequest is the request body for updating user profile.
type updateProfileRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// UpdateProfile handles PUT /api/auth/me — updates the current user's profile.
func (h *AuthHandlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req updateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)

	if req.FirstName == "" {
		respondAppError(w, Validation("firstName is required"))
		return
	}
	if req.LastName == "" {
		respondAppError(w, Validation("lastName is required"))
		return
	}
	if len(req.FirstName) > 100 {
		respondAppError(w, Validation("firstName must be at most 100 characters"))
		return
	}
	if len(req.LastName) > 100 {
		respondAppError(w, Validation("lastName must be at most 100 characters"))
		return
	}

	user, err := h.Auth.UpdateProfile(userID, req.FirstName, req.LastName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "user", userID, "updated user profile")

	respondJSON(w, http.StatusOK, user, nil)
}

// changePasswordRequest is the request body for changing password.
type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePassword handles POST /api/auth/change-password — changes the user's password.
func (h *AuthHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CurrentPassword == "" {
		respondAppError(w, Validation("currentPassword is required"))
		return
	}

	if err := validation.ValidatePassword(req.NewPassword); err != nil {
		respondAppError(w, Validation(err.Error()))
		return
	}

	if err := h.Auth.ChangePassword(userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, auth.ErrInvalidPassword) {
			respondAppError(w, BadRequest("current password is incorrect"))
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to change password")
		return
	}

	h.Audit.LogAction(r.Context(), "update", "password", userID, "changed password")

	respondJSON(w, http.StatusOK, map[string]string{"message": "password changed successfully"}, nil)
}
