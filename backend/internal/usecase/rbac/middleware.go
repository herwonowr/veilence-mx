package rbac

import (
	"cmp"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// RequireOrg returns a Chi middleware that extracts the organization ID from
// the URL parameter "orgId", verifies the authenticated user is a member,
// and sets both the org ID and member role in the request context.
//
// The user ID must already be set in the context (by the auth middleware).
// If no user ID is found, it returns 401. If the user is not a member
// of the organization, it returns 403.
func RequireOrg(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID == 0 {
				http.Error(w, `{"data":null,"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			orgIDStr := cmp.Or(
				chi.URLParam(r, "orgId"),
				r.Header.Get("X-Org-ID"),
				r.URL.Query().Get("org_id"),
			)

			orgID, err := strconv.ParseUint(orgIDStr, 10, 64)
			if err != nil || orgID == 0 {
				http.Error(w, `{"data":null,"error":"valid organization ID is required"}`, http.StatusBadRequest)
				return
			}

			member, err := svc.GetUserMembership(userID, uint(orgID))
			if err != nil {
				slog.Debug("org membership check failed",
					"user_id", userID,
					"org_id", orgID,
					"error", err,
				)
				http.Error(w, `{"data":null,"error":"not a member of this organization"}`, http.StatusForbidden)
				return
			}

			ctx := WithOrgID(r.Context(), uint(orgID))
			ctx = WithMemberRole(ctx, member.Role.Name)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns a Chi middleware that checks if the current user
// has the specified permission (resource:action) in the current organization.
//
// This middleware must be used after RequireOrg, which sets the org ID and
// user context. Returns 403 if the user lacks the required permission.
func RequirePermission(svc *Service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			orgID := OrgIDFromContext(r.Context())

			if userID == 0 || orgID == 0 {
				http.Error(w, `{"data":null,"error":"authentication and organization context required"}`, http.StatusUnauthorized)
				return
			}

			if err := svc.CheckPermission(userID, orgID, resource, action); err != nil {
				slog.Debug("permission denied",
					"user_id", userID,
					"org_id", orgID,
					"resource", resource,
					"action", action,
				)
				http.Error(w, `{"data":null,"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
