package rbac

import (
	"cmp"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// RequireWorkspace returns a Chi middleware that extracts the workspace ID from
// the URL parameter "workspaceId", verifies the authenticated user is a member,
// and sets both the workspace ID and member role in the request context.
//
// The user ID must already be set in the context (by the auth middleware).
// If no user ID is found, it returns 401. If the user is not a member
// of the workspace, it returns 403.
func RequireWorkspace(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID == 0 {
				http.Error(w, `{"data":null,"error":"authentication required"}`, http.StatusUnauthorized)
				return
			}

			wsIDStr := cmp.Or(
				chi.URLParam(r, "workspaceId"),
				r.Header.Get("X-Workspace-ID"),
				r.URL.Query().Get("workspace_id"),
			)

			workspaceID, err := strconv.ParseUint(wsIDStr, 10, 64)
			if err != nil || workspaceID == 0 {
				http.Error(w, `{"data":null,"error":"valid workspace ID is required"}`, http.StatusBadRequest)
				return
			}

			member, err := svc.GetUserMembership(userID, uint(workspaceID))
			if err != nil {
				slog.Debug("workspace membership check failed",
					"user_id", userID,
					"workspace_id", workspaceID,
					"error", err,
				)
				http.Error(w, `{"data":null,"error":"not a member of this workspace"}`, http.StatusForbidden)
				return
			}

			ctx := WithWorkspaceID(r.Context(), uint(workspaceID))
			ctx = WithMemberRole(ctx, member.Role.Name)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns a Chi middleware that checks if the current user
// has the specified permission (resource:action) in the current workspace.
//
// This middleware must be used after RequireWorkspace, which sets the workspace ID and
// user context. Returns 403 if the user lacks the required permission.
func RequirePermission(svc *Service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			workspaceID := WorkspaceIDFromContext(r.Context())

			if userID == 0 || workspaceID == 0 {
				http.Error(w, `{"data":null,"error":"authentication and workspace context required"}`, http.StatusUnauthorized)
				return
			}

			if err := svc.CheckPermission(userID, workspaceID, resource, action); err != nil {
				slog.Debug("permission denied",
					"user_id", userID,
					"workspace_id", workspaceID,
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
