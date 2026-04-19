package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

type contextKey string

const (
	// contextKeyUserID is the context key for the authenticated user's ID.
	contextKeyUserID contextKey = "user_id"
	// contextKeyEmail is the context key for the authenticated user's email.
	contextKeyEmail contextKey = "email"
	// contextKeyAuthMethod is the context key for the authentication method used.
	contextKeyAuthMethod contextKey = "auth_method"
	// contextKeyAPIKeyRole is the context key for the API key role (only set for API key auth).
	contextKeyAPIKeyRole contextKey = "api_key_role"
	// contextKeyAPIKeyWorkspaceID is the context key for the API key's workspace ID (only set for API key auth).
	contextKeyAPIKeyWorkspaceID contextKey = "api_key_workspace_id"

	// AuthMethodJWT indicates authentication via JWT Bearer token.
	AuthMethodJWT = "jwt"
	// AuthMethodAPIKey indicates authentication via X-API-Key header.
	AuthMethodAPIKey = "api_key"
)

// authErrorResponse matches the existing APIResponse envelope for error responses.
type authErrorResponse struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
	Meta  any     `json:"meta,omitempty"`
}

// respondAuthError writes a JSON error response matching the APIResponse format.
func respondAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(authErrorResponse{Error: &msg}); err != nil {
		slog.Error("failed to write auth error response", "error", err)
	}
}

// Middleware returns a Chi middleware that authenticates requests via
// Bearer JWT tokens or X-API-Key headers. On success, it sets user_id
// and email in the request context. For API key auth, it also sets the
// role and workspace ID so that downstream RBAC middleware can enforce
// permissions based on the API key's assigned role.
func Middleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try Bearer token first
			if authHeader := r.Header.Get("Authorization"); authHeader != "" {
				if tokenString, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
					claims, err := svc.ValidateAccessToken(tokenString)
					if err != nil {
						slog.Debug("invalid JWT token", "error", err)
						respondAuthError(w, http.StatusUnauthorized, "invalid or expired token")
						return
					}

					ctx := context.WithValue(r.Context(), contextKeyUserID, claims.UserID)
					ctx = context.WithValue(ctx, contextKeyEmail, claims.Email)
					ctx = context.WithValue(ctx, contextKeyAuthMethod, AuthMethodJWT)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Try X-API-Key header
			if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
				userID, email, role, workspaceID, err := svc.ValidateAPIKey(apiKey)
				if err != nil {
					slog.Debug("invalid API key", "error", err)
					respondAuthError(w, http.StatusUnauthorized, "invalid or expired API key")
					return
				}

				ctx := context.WithValue(r.Context(), contextKeyUserID, userID)
				ctx = context.WithValue(ctx, contextKeyEmail, email)
				ctx = context.WithValue(ctx, contextKeyAuthMethod, AuthMethodAPIKey)
				ctx = context.WithValue(ctx, contextKeyAPIKeyRole, role)
				ctx = context.WithValue(ctx, contextKeyAPIKeyWorkspaceID, workspaceID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			respondAuthError(w, http.StatusUnauthorized, "authentication required")
		})
	}
}

// UserIDFromContext extracts the authenticated user's ID from the request context.
// Returns 0 if no user is authenticated.
func UserIDFromContext(ctx context.Context) uint {
	if v, ok := ctx.Value(contextKeyUserID).(uint); ok {
		return v
	}
	return 0
}

// EmailFromContext extracts the authenticated user's email from the request context.
// Returns an empty string if no user is authenticated.
func EmailFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyEmail).(string); ok {
		return v
	}
	return ""
}

// AuthMethodFromContext extracts the authentication method from the request context.
// Returns an empty string if no auth method is set (unauthenticated request).
func AuthMethodFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyAuthMethod).(string); ok {
		return v
	}
	return ""
}

// APIKeyRoleFromContext extracts the API key role from the request context.
// Returns empty string if the request was not authenticated via API key.
func APIKeyRoleFromContext(ctx context.Context) entity.APIKeyRole {
	if v, ok := ctx.Value(contextKeyAPIKeyRole).(entity.APIKeyRole); ok {
		return v
	}
	return ""
}

// APIKeyWorkspaceIDFromContext extracts the API key's workspace ID from the request context.
// Returns 0 if the request was not authenticated via API key.
func APIKeyWorkspaceIDFromContext(ctx context.Context) uint {
	if v, ok := ctx.Value(contextKeyAPIKeyWorkspaceID).(uint); ok {
		return v
	}
	return 0
}
