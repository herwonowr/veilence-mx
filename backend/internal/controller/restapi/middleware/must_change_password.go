package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// UserByIDFinder retrieves a user by ID. Middleware depends on this interface
// instead of a concrete service type (dependency inversion).
type UserByIDFinder interface {
	GetUserByID(ctx context.Context, id string) (*entity.User, error)
}

// mustChangePasswordResponse is the structured response when the user must change their password.
type mustChangePasswordResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// RequireMustChangePassword blocks requests from users whose MustChangePassword
// flag is true, except for the password-change endpoint itself (and logout).
// This forces users created by an admin to set their own password before
// accessing any other API.
func RequireMustChangePassword(userFinder UserByIDFinder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow password-change and logout endpoints through unconditionally.
			path := r.URL.Path
			if strings.HasPrefix(path, "/api/") &&
				(strings.HasSuffix(path, "/auth/change-password") ||
					strings.HasSuffix(path, "/auth/logout") ||
					strings.HasSuffix(path, "/auth/me") ||
					strings.HasSuffix(path, "/auth/refresh")) {
				next.ServeHTTP(w, r)
				return
			}

			userID := auth.UserIDFromContext(r.Context())
			if userID == "" {
				// No authenticated user - let the auth middleware handle it.
				next.ServeHTTP(w, r)
				return
			}

			user, err := userFinder.GetUserByID(r.Context(), userID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if user.MustChangePassword {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				resp := mustChangePasswordResponse{
					Error:   "must_change_password",
					Message: "you must change your password before accessing this resource",
				}
				if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
					slog.Error("failed to write must-change-password response", "error", encErr)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
