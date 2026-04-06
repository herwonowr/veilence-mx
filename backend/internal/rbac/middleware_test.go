package rbac_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
	"github.com/veilence/veilence-mx/backend/internal/repository"
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
		&models.User{},
		&models.Organization{},
		&models.Role{},
		&models.Permission{},
		&models.OrgMember{},
		&models.Invitation{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.PasswordResetToken{},
		&models.EmailVerificationToken{},
		&models.Session{},
	)
	require.NoError(t, err)

	err = rbac.SeedPermissions(db)
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return &middlewareTestEnv{
		DB:      db,
		AuthSvc: auth.NewService(repository.NewUserRepo(db), repository.NewRefreshTokenRepo(db), repository.NewAPIKeyRepo(db), repository.NewPasswordResetTokenRepo(db), repository.NewEmailVerificationTokenRepo(db), repository.NewSessionRepo(db), testJWTSecret),
		RBACSvc: rbac.NewService(db),
	}
}

// registerAndLogin creates a user via the auth service and returns the access
// token and user model. This ensures user_id and email are properly set in
// context when the auth middleware processes the request.
func registerAndLogin(t *testing.T, env *middlewareTestEnv, email string) (*domain.User, string) {
	t.Helper()
	user, err := env.AuthSvc.Register(email, "Password123", "Test", "User")
	require.NoError(t, err)
	_, tokens, err := env.AuthSvc.Login(email, "Password123")
	require.NoError(t, err)
	return user, tokens.AccessToken
}

// createOrgWithOwner creates an org and returns it. The registering user
// automatically becomes the owner.
func createOrgWithOwner(t *testing.T, env *middlewareTestEnv, ownerID uint, slug string) *models.Organization {
	t.Helper()
	org, err := env.RBACSvc.CreateOrganization(ownerID, "Test Org "+slug, slug, "test")
	require.NoError(t, err)
	return org
}

// addMemberWithRole invites a user to the org with the given role name and
// accepts the invitation, returning the membership record.
func addMemberWithRole(t *testing.T, env *middlewareTestEnv, org *models.Organization, ownerID uint, member *domain.User, roleName string) *models.OrgMember {
	t.Helper()

	roles, err := env.RBACSvc.GetOrgRoles(org.ID)
	require.NoError(t, err)

	var targetRole *models.Role
	for i := range roles {
		if roles[i].Name == roleName {
			targetRole = &roles[i]
			break
		}
	}
	require.NotNil(t, targetRole, "role %q not found for org %d", roleName, org.ID)

	_, rawToken, err := env.RBACSvc.InviteMember(org.ID, member.Email, targetRole.ID, ownerID)
	require.NoError(t, err)

	membership, err := env.RBACSvc.AcceptInvitation(rawToken, member.ID, member.Email)
	require.NoError(t, err)
	return membership
}

// orgIDStr returns the string representation of an org ID for URL building.
func orgIDStr(id uint) string {
	return fmt.Sprintf("%d", id)
}

// successHandler returns a handler that writes 200 with context values so
// tests can verify the middleware set context correctly.
func successHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"orgId":      rbac.OrgIDFromContext(r.Context()),
			"memberRole": rbac.MemberRoleFromContext(r.Context()),
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

func TestRequireOrg_DenyWhenNoOrgIDInURL(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "no-orgid@example.com")

	// Route with NO {orgId} parameter — middleware can't extract it.
	r := chi.NewRouter()
	r.Route("/api/test", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "valid organization ID is required")
}

func TestRequireOrg_DenyWhenOrgDoesNotExist(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "nonexist-org@example.com")

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	// Org 99999 does not exist.
	req := httptest.NewRequest(http.MethodGet, "/api/orgs/99999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "not a member of this organization")
}

func TestRequireOrg_DenyWhenUserNotMember(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "org-owner@example.com")
	_, outsiderToken := registerAndLogin(t, env, "outsider@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "deny-nonmember")

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+outsiderToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "not a member of this organization")
}

func TestRequireOrg_AllowWhenUserIsMember(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "member-test-owner@example.com")
	createOrgWithOwner(t, env, owner.ID, "allow-member")

	// The owner is automatically a member.
	var org models.Organization
	require.NoError(t, env.DB.First(&org, "slug = ?", "allow-member").Error)

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireOrg_SetsOrgContextCorrectly(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "ctx-owner@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "context-check")

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	resp := decodeJSON(t, w)
	// orgId is stored as uint, JSON decodes to float64.
	assert.Equal(t, float64(org.ID), resp["orgId"])
	assert.Equal(t, models.RoleOwner, resp["memberRole"])
}

func TestRequireOrg_DenyWhenNoAuthentication(t *testing.T) {
	env := setupMiddlewareEnv(t)

	// No auth middleware in front — simulates an unauthenticated request
	// reaching RequireOrg directly (userID will be 0).
	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "authentication required")
}

func TestRequireOrg_InvalidOrgIDFormat(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "badorgid@example.com")

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/not-a-number", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "valid organization ID is required")
}

func TestRequireOrg_ZeroOrgID(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "zero-org@example.com")

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRequireOrg_OrgIDFromHeader(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "header-org@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "header-org")

	// Route without {orgId} in the URL, middleware falls back to X-Org-ID header.
	r := chi.NewRouter()
	r.Route("/api/test", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	req.Header.Set("X-Org-ID", orgIDStr(org.ID))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, float64(org.ID), resp["orgId"])
}

func TestRequireOrg_OrgIDFromQueryParam(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, ownerToken := registerAndLogin(t, env, "query-org@example.com")
	org := createOrgWithOwner(t, env, owner.ID, "query-org")

	// Route without {orgId} in the URL, middleware falls back to org_id query param.
	r := chi.NewRouter()
	r.Route("/api/test", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/test?org_id="+orgIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, float64(org.ID), resp["orgId"])
}

func TestRequireOrg_InvitedMemberAllowed(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "inv-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "inv-member@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "invited-member")
	addMemberWithRole(t, env, org, owner.ID, member, models.RoleMember)

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, float64(org.ID), resp["orgId"])
	assert.Equal(t, models.RoleMember, resp["memberRole"])
}

// ========================================================================
// RequirePermission tests
// ========================================================================

func TestRequirePermission_DenyWhenNoOrgContext(t *testing.T) {
	env := setupMiddlewareEnv(t)
	_, token := registerAndLogin(t, env, "no-org-ctx@example.com")

	// Use RequirePermission WITHOUT RequireOrg — org context won't be set.
	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "read"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	resp := decodeJSON(t, w)
	assert.Contains(t, resp["error"], "authentication and organization context required")
}

func TestRequirePermission_DenyWhenUserLacksPermission(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "perm-deny-owner@example.com")
	viewer, viewerToken := registerAndLogin(t, env, "perm-deny-viewer@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "perm-deny")
	addMemberWithRole(t, env, org, owner.ID, viewer, models.RoleViewer)

	// Viewer cannot write packages.
	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "write"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
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
	addMemberWithRole(t, env, org, owner.ID, member, models.RoleMember)

	// Member CAN write packages.
	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "write"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePermission_DenyNoAuthentication(t *testing.T) {
	env := setupMiddlewareEnv(t)

	// No auth middleware, no user context — both userID and orgID are 0.
	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "read"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/1", nil)
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
	for _, perm := range models.SystemPermissions {
		t.Run(perm.Resource+":"+perm.Action, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/api/orgs/{orgId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireOrg(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.Resource, perm.Action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
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
	addMemberWithRole(t, env, org, owner.ID, viewer, models.RoleViewer)

	// Viewer allowed permissions.
	viewerAllowed := map[string]bool{
		"packages:read":      true,
		"alerts:read":        true,
		"releases:read":      true,
		"settings:read":      true,
		"notifications:read": true,
	}

	for _, perm := range models.SystemPermissions {
		key := perm.Resource + ":" + perm.Action
		t.Run(key, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/api/orgs/{orgId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireOrg(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.Resource, perm.Action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
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
	addMemberWithRole(t, env, org, owner.ID, member, models.RoleMember)

	memberAllowed := map[string]bool{
		"packages:read":  true,
		"packages:write": true,
		"alerts:read":    true,
		"alerts:write":   true,
		"releases:read":  true,
		"settings:read":  true,
		"members:read":   true,
		"roles:read":     true,
		"org:read":       true,
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
		{"org", "write"},
		{"org", "delete"},
		{"audit", "read"},
	}

	// Test allowed permissions.
	for key := range memberAllowed {
		t.Run("allowed/"+key, func(t *testing.T) {
			// Parse resource:action.
			resource, action, _ := strings.Cut(key, ":")

			r := chi.NewRouter()
			r.Route("/api/orgs/{orgId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireOrg(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, resource, action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
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
			r.Route("/api/orgs/{orgId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireOrg(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.resource, perm.action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
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
	addMemberWithRole(t, env, org, owner.ID, admin, models.RoleAdmin)

	for _, perm := range models.SystemPermissions {
		key := perm.Resource + ":" + perm.Action
		t.Run(key, func(t *testing.T) {
			r := chi.NewRouter()
			r.Route("/api/orgs/{orgId}", func(r chi.Router) {
				r.Use(auth.Middleware(env.AuthSvc))
				r.Use(rbac.RequireOrg(env.RBACSvc))
				r.Use(rbac.RequirePermission(env.RBACSvc, perm.Resource, perm.Action))
				r.Get("/", successHandler())
			})

			req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID), nil)
			req.Header.Set("Authorization", "Bearer "+adminToken)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if key == "org:delete" {
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
	r.Route("/api/orgs/{orgId}/packages", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Use(rbac.RequirePermission(env.RBACSvc, "packages", "read"))
		r.Get("/", successHandler())
	})

	req := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(org.ID)+"/packages", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := decodeJSON(t, w)
	assert.Equal(t, float64(org.ID), resp["orgId"])
	assert.Equal(t, models.RoleOwner, resp["memberRole"])
}

func TestMiddlewareChain_MultiplePermissionChecks(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "multi-perm-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "multi-perm-member@example.com")

	org := createOrgWithOwner(t, env, owner.ID, "multi-perm")
	addMemberWithRole(t, env, org, owner.ID, member, models.RoleMember)

	// Route requires packages:write — member should have it.
	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))

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
	req := httptest.NewRequest(http.MethodPost, "/api/orgs/"+orgIDStr(org.ID)+"/packages", nil)
	req.Header.Set("Authorization", "Bearer "+memberToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "member should write packages")

	// Member cannot write settings.
	req2 := httptest.NewRequest(http.MethodPut, "/api/orgs/"+orgIDStr(org.ID)+"/settings", nil)
	req2.Header.Set("Authorization", "Bearer "+memberToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusForbidden, w2.Code, "member should not write settings")
}

func TestRequireOrg_DifferentOrgsAreSeparate(t *testing.T) {
	env := setupMiddlewareEnv(t)

	owner, _ := registerAndLogin(t, env, "cross-org-owner@example.com")
	member, memberToken := registerAndLogin(t, env, "cross-org-member@example.com")

	orgA := createOrgWithOwner(t, env, owner.ID, "cross-org-a")
	orgB := createOrgWithOwner(t, env, owner.ID, "cross-org-b")

	// Member belongs to org A only.
	addMemberWithRole(t, env, orgA, owner.ID, member, models.RoleMember)

	r := chi.NewRouter()
	r.Route("/api/orgs/{orgId}", func(r chi.Router) {
		r.Use(auth.Middleware(env.AuthSvc))
		r.Use(rbac.RequireOrg(env.RBACSvc))
		r.Get("/", successHandler())
	})

	// Access org A — should succeed.
	reqA := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(orgA.ID), nil)
	reqA.Header.Set("Authorization", "Bearer "+memberToken)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	assert.Equal(t, http.StatusOK, wA.Code, "member should access org A")

	// Access org B — should fail.
	reqB := httptest.NewRequest(http.MethodGet, "/api/orgs/"+orgIDStr(orgB.ID), nil)
	reqB.Header.Set("Authorization", "Bearer "+memberToken)
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	assert.Equal(t, http.StatusForbidden, wB.Code, "member should not access org B")
}
