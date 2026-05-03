package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// RequireSuperAdmin blocks requests from non-super-admin users.
// Must be placed after Auth middleware (requires user ID in context).
func RequireSuperAdmin(userFinder UserByIDFinder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := auth.UserIDFromContext(r.Context())
			if userID == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
				return
			}

			user, err := userFinder.GetUserByID(r.Context(), userID)
			if err != nil {
				slog.Error("RequireSuperAdmin: fetching user", "userID", userID, "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
				return
			}

			if !user.IsSuperAdmin {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "super admin access required"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
