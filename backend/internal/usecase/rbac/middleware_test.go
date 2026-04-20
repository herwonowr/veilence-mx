package rbac_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// ---------- Test infrastructure ----------

const testJWTSecret = "test-middleware-jwt-secret-1234567890"

// middlewareTestEnv bundles everything needed to test RBAC middleware.
type middlewareTestEnv struct {
	DB      *gorm.DB
	AuthSvc *auth.Service
	RBACSvc *rbac.Service
}

// setupMiddlewareEnv creates an in-memory SQLite database with all models
// migrated and system permissions seeded.
func setupMiddlewareEnv(t *testing.T) *middlewareTestEnv {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&persistent.User{},
		&persistent.Workspace{},
		&persistent.Role{},
		&persistent.Permission{},
		&persistent.WorkspaceMember{},
		&persistent.Invitation{},
		&persistent.RefreshToken{},
		&persistent.APIKey{},
		&persistent.PasswordResetToken{},
		&persistent.EmailVerificationToken{},
		&persistent.Session{},
	)
	require.NoError(t, err)

	err = rbac.SeedPermissions(persistent.NewRBACRepo(db))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return &middlewareTestEnv{
		DB:      db,
		AuthSvc: auth.NewService(persistent.NewUserRepo(db), persistent.NewRefreshTokenRepo(db), persistent.NewAPIKeyRepo(db), persistent.NewPasswordResetTokenRepo(db), persistent.NewEmailVerificationTokenRepo(db), persistent.NewSessionRepo(db), nil, false, nil, testJWTSecret),
		RBACSvc: rbac.NewService(persistent.NewRBACRepo(db)),
	}
}

// registerAndLogin creates a user via the auth service and returns the access
// token and user model. This ensures user_id and email are properly set in
// context when the auth middleware processes the request.
func registerAndLogin(t *testing.T, env *middlewareTestEnv, email string) (*entity.User, string) {
	t.Helper()
	user, err := env.AuthSvc.Register(email, "Password123", "Test", "User")
	require.NoError(t, err)
	_, tokens, err := env.AuthSvc.Login(email, "Password123", "127.0.0.1", "TestBrowser/1.0")
	require.NoError(t, err)
	return user, tokens.AccessToken
}

// createOrgWithOwner creates an org and returns it. The registering user
// automatically becomes the owner.
func createOrgWithOwner(t *testing.T, env *middlewareTestEnv, ownerID string, slug string) *entity.Workspace {
	t.Helper()
	org, err := env.RBACSvc.CreateWorkspace(ownerID, "Test Workspace "+slug, slug, "test")
	require.NoError(t, err)
	return org
}

// addMemberWithRole invites a user to the org with the given role name and
// accepts the invitation, returning the membership record.
func addMemberWithRole(t *testing.T, env *middlewareTestEnv, org *entity.Workspace, ownerID string, member *entity.User, roleName string) *entity.WorkspaceMember {
	t.Helper()

	roles, err := env.RBACSvc.GetWorkspaceRoles(org.ID)
	require.NoError(t, err)

	var targetRole *entity.Role
	for i := range roles {
		if roles[i].Name == roleName {
			targetRole = &roles[i]
			break
		}
	}
	require.NotNil(t, targetRole, "role %q not found for org %s", roleName, org.ID)

	_, rawToken, err := env.RBACSvc.InviteMember(org.ID, member.Email, targetRole.ID, ownerID)
	require.NoError(t, err)

	membership, err := env.RBACSvc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)
	return membership
}

// workspaceIDStr returns the string representation of an org ID for URL building.
func workspaceIDStr(id string) string {
	return id
}

// successHandler returns a handler that writes 200 with context values so
// tests can verify the middleware set context correctly.
func successHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"workspaceId": rbac.WorkspaceIDFromContext(r.Context()),
			"memberRole":  rbac.MemberRoleFromContext(r.Context()),
		})
	}
}

// decodeJSON is a test helper that decodes the response body.
func decodeJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}

// ========================================================================
// RequireOrg tests
// ========================================================================

func TestRequireWorkspace_DenyWhenNoWorkspaceIDInURL(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "no-wsid@example.com")

	// Route with NO {workspaceId} parameter - middleware can't extract it.
	r := chi.NewRouter()
	r.Route("/api/test", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "valid workspace ID is required")
}

func TestRequireWorkspace_DenyWhenOrgDoesNotExist(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "nonexist-org@example.com")

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	// Workspace 99999 does not exist.
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "not a member of this workspace")
}

func TestRequireWorkspace_DenyWhenUserNotMember(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "workspace-owner@example.com")
	_, outsiderToken := registerAndLogin(t, env, "outsider@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "deny-nonmember")

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+outsiderToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "not a member of this workspace")
}

func TestRequireWorkspace_AllowWhenUserIsMember(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "member-test-owner@example.com")
	createOrgWithOwner(t, env, owner.ID, "allow-member")

	// The owner is automatically a member.
	var org persistent.Workspace
	require.NoError(t, env.DB.First(&org, "slug = ?", "allow-member").Error)

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireWorkspace_SetsOrgContextCorrectly(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "ctx-owner@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "context-check")

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	resp := decodeJSON(t, w)
	assert.Equal(t, org.ID, resp["workspaceId"])
	assert.Equal(t, entity.RoleOwner, resp["memberRole"])
}

func TestRequireWorkspace_DenyWhenNoAuthentication(t *testing.T) {
	env := setupMiddlewareEnv(t)

	// No auth middleware in front - simulates an unauthenticated request
	// reaching RequireOrg directly (userID will be 0).
	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "authentication required")
}

func TestRequireWorkspace_InvalidWorkspaceIDFormat(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "bad-wsid@example.com")

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	// With string UUIDs, any non-empty string is accepted as workspace ID format;
	// it just won't find a membership, resulting in 403.
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "not a member of this workspace")
}

func TestRequireWorkspace_ZeroWorkspaceID(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "zero-org@example.com")

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	// "0" is a non-empty string, so it passes format check but won't find a membership.
	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireWorkspace_WorkspaceIDFromHeader(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "header-org@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "header-org")

	// Route without {workspaceId} in the URL, middleware falls back to X-Workspace-ID header.
	r := chi.NewRouter()
	r.Route("/api/test", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	req.Header.Set("X-Workspace-ID", workspaceIDStr(org.ID))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, org.ID, resp["workspaceId"])
}

func TestRequireWorkspace_WorkspaceIDFromQueryParam(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "query-org@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "query-org")

	// Route without {workspaceId} in the URL, middleware falls back to workspace_id query param.
	r := chi.NewRouter()
	r.Route("/api/test", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test?workspace_id="+workspaceIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, org.ID, resp["workspaceId"])
}

func TestRequireWorkspace_InvitedMemberAllowed(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "inv-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "inv-member@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "invited-member")
	addMemberWithRole(t, env, org, owner.ID, member, entity.RoleMember)

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, org.ID, resp["workspaceId"])
	assert.Equal(t, entity.RoleMember, resp["memberRole"])
}

// ========================================================================
// RequirePermission tests
// ========================================================================

func TestRequirePermission_DenyWhenNoOrgContext(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "no-org-ctx@example.com")

	// Use RequirePermission WITHOUT RequireOrg - org context won't be set.
	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "read"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "authentication and workspace context required")
}

func TestRequirePermission_DenyWhenUserLacksPermission(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "perm-deny-owner@example.com")
	viewer, viewerToken := registerAndLogin(t, env, "perm-deny-viewer@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "perm-deny")
	addMemberWithRole(t, env, org, owner.ID, viewer, entity.RoleViewer)

	// Viewer cannot write packages.
	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "write"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "insufficient permissions")
}

func TestRequirePermission_AllowWhenUserHasPermission(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "perm-allow-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "perm-allow-member@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "perm-allow")
	addMemberWithRole(t, env, org, owner.ID, member, entity.RoleMember)

	// Member CAN write packages.
	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "write"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_DenyNoAuthentication(t *testing.T) {
	env := setupMiddlewareEnv(t)

	// No auth middleware, no user context - both userID and workspaceID are 0.
	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "read"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========================================================================
// Edge cases: Role-based permission coverage
// ========================================================================

func TestOwnerRole_HasAllPermissions(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "all-perms-owner@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "owner-all-perms")

	// Build a route for each system permission and verify the owner can access it.
	for _, perm := range persistent.SystemPermissions {
		t.Run(perm.Resource+":"+perm.Action, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireWorkspace(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.Resource, perm.Action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
			req.Header.Set("Authorization", "Bearer "+ownerToken)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code,
				"owner should have permission %s:%s", perm.Resource, perm.Action)
		})
	}
}

func TestViewerRole_HasOnlyReadPermissions(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "viewer-perms-owner@example.com")
	viewer, viewerToken := registerAndLogin(t, env, "viewer-perms@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "viewer-perms")
	addMemberWithRole(t, env, org, owner.ID, viewer, entity.RoleViewer)

	// Viewer allowed permissions.
	viewerAllowed := map[string]bool{
		"packages:read":      true,
		"alerts:read":        true,
		"releases:read":      true,
		"settings:read":      true,
		"notifications:read": true,
	}

	for _, perm := range persistent.SystemPermissions {
		key := perm.Resource + ":" + perm.Action
		t.Run(key, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireWorkspace(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.Resource, perm.Action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
			req.Header.Set("Authorization", "Bearer "+viewerToken)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if viewerAllowed[key] {
				assert.Equal(t, http.StatusOK, w.Code,
					"viewer SHOULD have permission %s", key)
			} else {
				assert.Equal(t, http.StatusForbidden, w.Code,
					"viewer should NOT have permission %s", key)
			}
		})
	}
}

func TestMemberRole_PermissionBoundaries(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "member-perms-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "member-perms@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "member-perms")
	addMemberWithRole(t, env, org, owner.ID, member, entity.RoleMember)

	memberAllowed := map[string]bool{
		"packages:read":  true,
		"packages:write": true,
		"alerts:read":    true,
		"alerts:write":   true,
		"releases:read":  true,
		"settings:read":  true,
		"members:read":   true,
		"roles:read":     true,
		"workspace:read": true,
		"api_keys:read":  true,
		"api_keys:write": true,
	}

	// Member should NOT have these.
	memberDenied := []struct {
		resource string
		action   string
	}{
		{"packages", "delete"},
		{"settings", "write"},
		{"members", "invite"},
		{"members", "remove"},
		{"roles", "write"},
		{"workspace", "write"},
		{"workspace", "delete"},
		{"audit", "read"},
	}

	// Test allowed permissions.
	for key := range memberAllowed {
		t.Run("allowed/"+key, func(t *testing.T) {
			// Parse resource:action.
			resource, action, _ := strings.Cut(key, ":")

			r := chi.NewRouter()
			r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireWorkspace(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, resource, action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
			req.Header.Set("Authorization", "Bearer "+memberToken)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code,
				"member SHOULD have permission %s", key)
		})
	}

	// Test denied permissions.
	for _, perm := range memberDenied {
		key := perm.resource + ":" + perm.action
		t.Run("denied/"+key, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireWorkspace(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.resource, perm.action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
			req.Header.Set("Authorization", "Bearer "+memberToken)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code,
				"member should NOT have permission %s", key)
		})
	}
}

func TestAdminRole_HasAllExceptOrgDelete(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "admin-perms-owner@example.com")
	admin, adminToken := registerAndLogin(t, env, "admin-perms@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "admin-perms")
	addMemberWithRole(t, env, org, owner.ID, admin, entity.RoleAdmin)

	for _, perm := range persistent.SystemPermissions {
		key := perm.Resource + ":" + perm.Action
		t.Run(key, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireWorkspace(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.Resource, perm.Action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID), nil)
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if key == "workspace:delete" {
				assert.Equal(t, http.StatusForbidden, w.Code,
					"admin should NOT have org:delete permission")
			} else {
				assert.Equal(t, http.StatusOK, w.Code,
					"admin should have permission %s", key)
			}
		})
	}
}

// ========================================================================
// Middleware chaining integration tests
// ========================================================================

func TestMiddlewareChain_RequireOrgThenPermission(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "chain-owner@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "chain-test")

	// Full middleware chain: Auth → RequireOrg → RequirePermission.
	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}/packages", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "read"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(org.ID)+"/packages", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, org.ID, resp["workspaceId"])
	assert.Equal(t, entity.RoleOwner, resp["memberRole"])
}

func TestMiddlewareChain_MultiplePermissionChecks(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "multi-perm-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "multi-perm-member@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "multi-perm")
	addMemberWithRole(t, env, org, owner.ID, member, entity.RoleMember)

	// Route requires packages:write - member should have it.
	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))

		// Nested routes with different permissions.
		r.Route("/packages", func(r chi.Router) {
			r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "write"))
			r.Post("/", successHandler())
		})
		r.Route("/settings", func(r chi.Router) {
			r.Use(rbac.RequirePermission(env.RBACSvc, "settings", "write"))
			r.Put("/", successHandler())
		})
	})

	// Member can write packages.
	req := httptest.NewRequest(http.MethodPost, "/api/workspaces/"+workspaceIDStr(org.ID)+"/packages", nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "member should write packages")

	// Member cannot write settings.
	req2 := httptest.NewRequest(http.MethodPut, "/api/workspaces/"+workspaceIDStr(org.ID)+"/settings", nil)
	req2.Header.Set("Authorization", "Bearer "+memberToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code, "member should not write settings")
}

func TestRequireWorkspace_DifferentOrgsAreSeparate(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "cross-org-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "cross-org-member@example.com")

	orgA := createOrgWithOwner(t, env, owner.ID, "cross-org-a")
	orgB := createOrgWithOwner(t, env, owner.ID, "cross-org-b")

	// Member belongs to org A only.
	addMemberWithRole(t, env, orgA, owner.ID, member, entity.RoleMember)

	r := chi.NewRouter()
	r.Route("/api/workspaces/{workspaceId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireWorkspace(env.RBACSvc))
		r.Get("/", successHandler())
	})

	// Access org A - should succeed.
	reqA := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(orgA.ID), nil)
	reqA.Header.Set("Authorization", "Bearer "+memberToken)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	assert.Equal(t, http.StatusOK, wA.Code, "member should access org A")

	// Access org B - should fail.
	reqB := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+workspaceIDStr(orgB.ID), nil)
	reqB.Header.Set("Authorization", "Bearer "+memberToken)
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	assert.Equal(t, http.StatusForbidden, wB.Code, "member should not access org B")
}
