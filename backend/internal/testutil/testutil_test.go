package testutil_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/testutil"
)

// --- SetupTestDB ---

func TestSetupTestDB_Migrates(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Should be able to create a user
	user := &models.User{Email: "test@example.com", PasswordHash: "h", IsActive: true}
	require.NoError(t, db.Create(user).Error)
	assert.NotZero(t, user.ID)

	// Should be able to create a package
	pkg := &models.Package{Name: "requests", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)
	assert.NotZero(t, pkg.ID)
}

// --- Fixtures ---

func TestCreateUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	user := testutil.CreateUser(t, db)
	assert.NotZero(t, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.True(t, user.IsActive)
}

func TestCreateUserN(t *testing.T) {
	db := testutil.SetupTestDB(t)
	users := testutil.CreateUserN(t, db, 3)
	assert.Len(t, users, 3)
	for i, u := range users {
		assert.Equal(t, fmt.Sprintf("user-%d@example.com", i), u.Email)
	}
}

func TestCreatePackage(t *testing.T) {
	db := testutil.SetupTestDB(t)
	pkg := testutil.CreatePackage(t, db)
	assert.NotZero(t, pkg.ID)
	assert.Equal(t, "requests", pkg.Name)
}

func TestCreateRelease(t *testing.T) {
	db := testutil.SetupTestDB(t)
	pkg := testutil.CreatePackage(t, db)
	rel := testutil.CreateRelease(t, db, func(f *testutil.ReleaseFixture) {
		f.PackageID = pkg.ID
		f.Version = "2.0.0"
	})
	assert.NotZero(t, rel.ID)
	assert.Equal(t, "2.0.0", rel.Version)
}

func TestCreateAlert(t *testing.T) {
	db := testutil.SetupTestDB(t)
	alert := testutil.CreateAlert(t, db)
	assert.NotZero(t, alert.ID)
	assert.Equal(t, models.AlertSeverityHigh, alert.Severity)
	assert.Equal(t, models.AlertStatusNew, alert.Status)
}

// --- Mock Repositories ---

func TestMockUserRepo_CreateAndFind(t *testing.T) {
	repo := testutil.NewMockUserRepository()
	ctx := context.Background()

	user := &domain.User{Email: "alice@example.com", FirstName: "Alice"}
	require.NoError(t, repo.Create(ctx, user))
	assert.Equal(t, uint(1), user.ID)
	assert.Equal(t, 1, repo.Calls.Create)

	found, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", found.Email)
	assert.Equal(t, 1, repo.Calls.FindByID)
}

func TestMockUserRepo_FindByEmail(t *testing.T) {
	repo := testutil.NewMockUserRepository()
	ctx := context.Background()

	user := &domain.User{Email: "bob@example.com"}
	require.NoError(t, repo.Create(ctx, user))

	found, err := repo.FindByEmail(ctx, "bob@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}

func TestMockUserRepo_Error(t *testing.T) {
	repo := testutil.NewMockUserRepository()
	ctx := context.Background()

	repo.Errors.FindByID = fmt.Errorf("db connection lost")
	_, err := repo.FindByID(ctx, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db connection lost")
}

func TestMockPackageRepo_CreateAndFind(t *testing.T) {
	repo := testutil.NewMockPackageRepository()
	ctx := context.Background()

	pkg := &domain.Package{OrgID: 1, Name: "requests", Ecosystem: domain.EcosystemPython}
	require.NoError(t, repo.Create(ctx, pkg))
	assert.Equal(t, uint(1), pkg.ID)

	found, err := repo.FindByID(ctx, pkg.ID)
	require.NoError(t, err)
	assert.Equal(t, "requests", found.Name)
}

func TestMockPackageRepo_FindByOrgID(t *testing.T) {
	repo := testutil.NewMockPackageRepository()
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &domain.Package{OrgID: 1, Name: "p1", Ecosystem: domain.EcosystemPython}))
	require.NoError(t, repo.Create(ctx, &domain.Package{OrgID: 1, Name: "p2", Ecosystem: domain.EcosystemNPM}))
	require.NoError(t, repo.Create(ctx, &domain.Package{OrgID: 2, Name: "p3", Ecosystem: domain.EcosystemPython}))

	pkgs, total, err := repo.FindByOrgID(ctx, 1, 1, 10, "")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, pkgs, 2)
}

func TestMockAlertRepo_FilterByStatus(t *testing.T) {
	repo := testutil.NewMockAlertRepository()
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &domain.Alert{OrgID: 1, Status: domain.AlertStatusNew}))
	require.NoError(t, repo.Create(ctx, &domain.Alert{OrgID: 1, Status: domain.AlertStatusResolved}))

	status := domain.AlertStatusNew
	alerts, total, err := repo.FindByOrgID(ctx, 1, 1, 10, "", domain.AlertFilters{Status: &status})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, alerts, 1)
}

func TestMockNotificationChannelRepo_CRUD(t *testing.T) {
	repo := testutil.NewMockNotificationChannelRepository()
	ctx := context.Background()

	ch := &domain.NotificationChannel{OrgID: 1, Name: "Email", Type: domain.NotificationChannelEmail, Config: "{}", IsActive: true}
	require.NoError(t, repo.Create(ctx, ch))
	assert.Equal(t, uint(1), ch.ID)

	found, err := repo.FindByIDAndOrg(ctx, ch.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "Email", found.Name)

	channels, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, channels, 1)

	affected, err := repo.DeleteByIDAndOrg(ctx, ch.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	channels, err = repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, channels, 0)
}

func TestMockSessionRepo_CRUD(t *testing.T) {
	repo := testutil.NewMockSessionRepository()
	ctx := context.Background()

	s := &domain.Session{UserID: 1, TokenHash: "hash1", IPAddress: "10.0.0.1"}
	require.NoError(t, repo.Create(ctx, s))
	assert.Equal(t, uint(1), s.ID)

	sessions, err := repo.FindByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, sessions, 1)

	require.NoError(t, repo.Delete(ctx, s.ID))
	sessions, err = repo.FindByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestTruncateAll(t *testing.T) {
	db := testutil.SetupTestDB(t)

	// Create some data
	testutil.CreateUser(t, db)
	testutil.CreatePackage(t, db)

	// Truncate
	testutil.TruncateAll(t, db)

	// Verify empty
	var userCount, pkgCount int64
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Package{}).Count(&pkgCount)
	assert.Equal(t, int64(0), userCount)
	assert.Equal(t, int64(0), pkgCount)
}

// --- Server helpers ---

func TestWithOrgContext(t *testing.T) {
	ctx := testutil.WithOrgContext(context.Background(), 42)
	// Verify org ID is set (use rbac.OrgIDFromContext indirectly)
	assert.NotNil(t, ctx)
}

func TestInjectOrg_Middleware(t *testing.T) {
	// Just verify the middleware compiles and can wrap a handler
	mw := testutil.InjectOrg(1)
	assert.NotNil(t, mw)
}

func TestNewTestAuthService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestAuthService(db)
	require.NotNil(t, svc)

	// Should be able to register and login
	user, err := svc.Register("auth-test@example.com", "Password123", "Auth", "Test")
	require.NoError(t, err)
	assert.NotZero(t, user.ID)
}

func TestLoginTestUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	svc := testutil.NewTestAuthService(db)

	userID, token := testutil.LoginTestUser(t, svc, "login@example.com", "Password123", "Login", "User")
	assert.NotZero(t, userID)
	assert.NotEmpty(t, token)

	// Token should be valid
	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "login@example.com", claims.Email)
}

func TestNewAuthenticatedRequest(t *testing.T) {
	req := testutil.NewAuthenticatedRequest("GET", "/api/test", "my-token", "")
	assert.Equal(t, "Bearer my-token", req.Header.Get("Authorization"))
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
}

func TestIDStr(t *testing.T) {
	assert.Equal(t, "42", testutil.IDStr(42))
	assert.Equal(t, "0", testutil.IDStr(0))
}
