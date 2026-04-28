package middleware

import (
	"net/http"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

// RequireMustChangePassword blocks requests from users whose MustChangePassword
// flag is true, except for the password-change endpoint itself (and logout).
// This forces users created by an admin to set their own password before
// accessing any other API.
func RequireMustChangePassword(authService *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow password-change and logout endpoints through unconditionally.
			path := r.URL.Path
			if strings.HasSuffix(path, "/auth/change-password") ||
				strings.HasSuffix(path, "/auth/logout") ||
				strings.HasSuffix(path, "/auth/me") ||
				strings.HasSuffix(path, "/auth/refresh") {
				next.ServeHTTP(w, r)
				return
			}

			userID := auth.UserIDFromContext(r.Context())
			if userID == "" {
				// No authenticated user - let the auth middleware handle it.
				next.ServeHTTP(w, r)
				return
			}

			user, err := authService.GetUserByID(userID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if user.MustChangePassword {
				respondAuthError(w, http.StatusForbidden, "Password change required before accessing the application")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
