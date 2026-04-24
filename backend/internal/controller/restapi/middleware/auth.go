package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
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

// Auth returns a Chi middleware that authenticates requests via
// Bearer JWT tokens or X-API-Key headers. On success, it sets user_id
// and email in the request context. For API key auth, it also sets the
// role and workspace ID so that downstream RBAC middleware can enforce
// permissions based on the API key's assigned role.
func Auth(svc *auth.Service) func(http.Handler) http.Handler {
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

					ctx := auth.WithUserID(r.Context(), claims.UserID)
					ctx = auth.WithEmail(ctx, claims.Email)
					ctx = auth.WithAuthMethod(ctx, auth.AuthMethodJWT)
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

				ctx := auth.WithUserID(r.Context(), userID)
				ctx = auth.WithEmail(ctx, email)
				ctx = auth.WithAuthMethod(ctx, auth.AuthMethodAPIKey)
				ctx = auth.WithAPIKeyRole(ctx, role)
				ctx = auth.WithAPIKeyWorkspaceID(ctx, workspaceID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			respondAuthError(w, http.StatusUnauthorized, "Authentication required")
		})
	}
}
