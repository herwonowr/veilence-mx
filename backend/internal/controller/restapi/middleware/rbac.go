package middleware

import (
	"cmp"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// RequireWorkspace returns a Chi middleware that extracts the workspace ID from
// the URL parameter "workspaceId", verifies the authenticated user is a member,
// and sets both the workspace ID and member role in the request context.
//
// The user ID must already be set in the context (by the auth middleware).
// If no user ID is found, it returns 401. If the user is not a member
// of the workspace, it returns 403.
func RequireWorkspace(svc *rbac.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := rbac.UserIDFromContext(r.Context())
			if userID == "" {
				http.Error(w, `{"data":null,"error":"Authentication required"}`, http.StatusUnauthorized)
				return
			}

			// For API key auth, the workspace and role come from the key itself
			if auth.AuthMethodFromContext(r.Context()) == auth.AuthMethodAPIKey {
				apiKeyWsID := auth.APIKeyWorkspaceIDFromContext(r.Context())
				apiKeyRole := auth.APIKeyRoleFromContext(r.Context())

				if apiKeyWsID == "" {
					http.Error(w, `{"data":null,"error":"API key is not bound to a workspace"}`, http.StatusForbidden)
					return
				}

				// If a workspace ID was provided explicitly, it must match the API key's workspace
				wsIDStr := cmp.Or(
					chi.URLParam(r, "workspaceId"),
					r.Header.Get("X-Workspace-ID"),
					r.URL.Query().Get("workspace_id"),
				)
				if wsIDStr != "" {
					if wsIDStr != apiKeyWsID {
						http.Error(w, `{"data":null,"error":"API key is not authorized for this workspace"}`, http.StatusForbidden)
						return
					}
				}

				ctx := rbac.WithWorkspaceID(r.Context(), apiKeyWsID)
				ctx = rbac.WithMemberRole(ctx, string(apiKeyRole))
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// JWT auth: workspace ID from URL, header, or query param
			wsIDStr := cmp.Or(
				chi.URLParam(r, "workspaceId"),
				r.Header.Get("X-Workspace-ID"),
				r.URL.Query().Get("workspace_id"),
			)

			workspaceID := wsIDStr
			if workspaceID == "" {
				http.Error(w, `{"data":null,"error":"Valid workspace ID is required"}`, http.StatusBadRequest)
				return
			}
			if _, err := uuid.Parse(workspaceID); err != nil {
				http.Error(w, `{"data":null,"error":"invalid workspace ID format"}`, http.StatusBadRequest)
				return
			}

			member, err := svc.GetUserMembership(r.Context(), userID, workspaceID)
			if err != nil {
				slog.Debug("workspace membership check failed",
					"user_id", userID,
					"workspace_id", workspaceID,
					"error", err,
				)
				http.Error(w, `{"data":null,"error":"not a member of this workspace"}`, http.StatusForbidden)
				return
			}

			ctx := rbac.WithWorkspaceID(r.Context(), workspaceID)
			ctx = rbac.WithMemberRole(ctx, member.Role.Name)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission returns a Chi middleware that checks if the current user
// has the specified permission (resource:action) in the current workspace.
//
// For JWT auth: delegates to the RBAC service to check the user's workspace role permissions.
// For API key auth: checks if the API key's assigned role has the required permission
// by looking up the role's permissions in the database.
//
// This middleware must be used after RequireWorkspace, which sets the workspace ID and
// user context. Returns 403 if the user lacks the required permission.
func RequirePermission(svc *rbac.Service, resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := rbac.UserIDFromContext(r.Context())
			workspaceID := rbac.WorkspaceIDFromContext(r.Context())

			if userID == "" || workspaceID == "" {
				http.Error(w, `{"data":null,"error":"Authentication and workspace context required"}`, http.StatusUnauthorized)
				return
			}

			// For API key auth, check permission against the key's role rather
			// than the user's actual workspace membership role.
			if auth.AuthMethodFromContext(r.Context()) == auth.AuthMethodAPIKey {
				apiKeyRole := string(auth.APIKeyRoleFromContext(r.Context()))
				if err := svc.CheckRolePermission(r.Context(), workspaceID, apiKeyRole, resource, action); err != nil {
					slog.Debug("API key permission denied",
						"user_id", userID,
						"workspace_id", workspaceID,
						"api_key_role", apiKeyRole,
						"resource", resource,
						"action", action,
					)
					http.Error(w, `{"data":null,"error":"insufficient permissions"}`, http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if err := svc.CheckPermission(r.Context(), userID, workspaceID, resource, action); err != nil {
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
