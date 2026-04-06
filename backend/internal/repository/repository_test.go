package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.APIKey{},
		&models.Package{},
		&models.Release{},
		&models.Diff{},
		&models.Analysis{},
		&models.Alert{},
		&models.Setting{},
		&models.Organization{},
		&models.Role{},
		&models.Permission{},
		&models.OrgMember{},
		&models.Invitation{},
		&models.AuditLog{},
		&models.NotificationChannel{},
		&models.NotificationRule{},
		&models.Notification{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

var ctx = context.Background()

// =========================================================================
// UserRepo
// =========================================================================

func TestUserRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepo(db)

	user := &models.User{
		Email:        "test@example.com",
		PasswordHash: "hash123",
		FirstName:    "Test",
		LastName:     "User",
		IsActive:     true,
	}
	require.NoError(t, db.Create(user).Error)

	found, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", found.Email)
	assert.Equal(t, "Test", found.FirstName)
}

func TestUserRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserRepo_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepo(db)

	user := &models.User{Email: "find@example.com", PasswordHash: "hash", IsActive: true}
	require.NoError(t, db.Create(user).Error)

	found, err := repo.FindByEmail(ctx, "find@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepo(db)

	_, err := repo.FindByEmail(ctx, "nope@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestUserRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepo(db)

	user := &models.User{Email: "new@example.com", PasswordHash: "hash", IsActive: true}
	// Use domain create
	domainUser := userModelToDomainForTest(user)
	err := repo.Create(ctx, domainUser)
	require.NoError(t, err)
	assert.NotZero(t, domainUser.ID)
}

func TestUserRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepo(db)

	user := &models.User{Email: "update@example.com", PasswordHash: "hash", IsActive: true}
	require.NoError(t, db.Create(user).Error)

	found, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	found.FirstName = "Updated"
	err = repo.Update(ctx, found)
	require.NoError(t, err)

	reloaded, err := repo.FindByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", reloaded.FirstName)
}

// =========================================================================
// RefreshTokenRepo
// =========================================================================

func TestRefreshTokenRepo_CreateAndFindByHash(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRefreshTokenRepo(db)

	token := &models.RefreshToken{
		UserID:    1,
		TokenHash: "abc123hash",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, db.Create(token).Error)

	found, err := repo.FindByTokenHash(ctx, "abc123hash")
	require.NoError(t, err)
	assert.Equal(t, uint(1), found.UserID)
}

func TestRefreshTokenRepo_DeleteByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRefreshTokenRepo(db)

	token := &models.RefreshToken{UserID: 1, TokenHash: "to_delete", ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, db.Create(token).Error)

	err := repo.DeleteByTokenHash(ctx, "to_delete")
	require.NoError(t, err)

	_, err = repo.FindByTokenHash(ctx, "to_delete")
	require.Error(t, err)
}

// =========================================================================
// APIKeyRepo
// =========================================================================

func TestAPIKeyRepo_CreateAndFindByPrefix(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	key := &models.APIKey{
		UserID:    1,
		Name:      "test-key",
		KeyHash:   "hash123",
		KeyPrefix: "vmx_ab",
		IsActive:  true,
	}
	require.NoError(t, db.Create(key).Error)

	found, err := repo.FindActiveByPrefix(ctx, "vmx_ab")
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "test-key", found[0].Name)
}

func TestAPIKeyRepo_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	key1 := &models.APIKey{UserID: 1, Name: "k1", KeyHash: "h1", KeyPrefix: "p1", IsActive: true}
	key2 := &models.APIKey{UserID: 1, Name: "k2", KeyHash: "h2", KeyPrefix: "p2", IsActive: true}
	key3 := &models.APIKey{UserID: 2, Name: "k3", KeyHash: "h3", KeyPrefix: "p3", IsActive: true}
	require.NoError(t, db.Create(key1).Error)
	require.NoError(t, db.Create(key2).Error)
	require.NoError(t, db.Create(key3).Error)

	keys, err := repo.FindByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestAPIKeyRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	key := &models.APIKey{UserID: 1, Name: "to-delete", KeyHash: "h", KeyPrefix: "p", IsActive: true}
	require.NoError(t, db.Create(key).Error)

	err := repo.SoftDelete(ctx, 1, key.ID)
	require.NoError(t, err)

	// Should not be found by user anymore
	keys, err := repo.FindByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, keys, 0)
}

func TestAPIKeyRepo_SoftDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	err := repo.SoftDelete(ctx, 1, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api key not found")
}

// =========================================================================
// PackageRepo
// =========================================================================

func TestPackageRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "requests", Registry: "pypi"}
	require.NoError(t, db.Create(pkg).Error)

	found, err := repo.FindByID(ctx, pkg.ID)
	require.NoError(t, err)
	assert.Equal(t, "requests", found.Name)
}

func TestPackageRepo_FindByOrgID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "pkg1", Registry: "pypi"}).Error)
	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "pkg2", Registry: "npm"}).Error)
	require.NoError(t, db.Create(&models.Package{OrgID: 2, Name: "pkg3", Registry: "pypi"}).Error)

	pkgs, total, err := repo.FindByOrgID(ctx, 1, 1, 10, "name asc")
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, pkgs, 2)
}

func TestPackageRepo_CountByOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "p1", Registry: "pypi"}).Error)
	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "p2", Registry: "npm"}).Error)

	count, err := repo.CountByOrg(ctx, 1, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestPackageRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "to-delete", Registry: "pypi"}
	require.NoError(t, db.Create(pkg).Error)

	err := repo.SoftDelete(ctx, 1, pkg.ID)
	require.NoError(t, err)

	// Should no longer appear in org listing
	pkgs, total, err := repo.FindByOrgID(ctx, 1, 1, 10, "name asc")
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, pkgs, 0)
}

func TestPackageRepo_SoftDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	err := repo.SoftDelete(ctx, 1, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package not found")
}

func TestPackageRepo_SoftDelete_CrossTenantBlocked(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	// Package belongs to org 1
	pkg := &models.Package{OrgID: 1, Name: "secret-pkg", Registry: "pypi"}
	require.NoError(t, db.Create(pkg).Error)

	// Org 2 attempts to delete org 1's package — must fail
	err := repo.SoftDelete(ctx, 2, pkg.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "package not found")

	// Package must still exist for org 1
	found, err := repo.FindByID(ctx, pkg.ID)
	require.NoError(t, err)
	assert.Equal(t, "secret-pkg", found.Name)
}

// =========================================================================
// NotificationChannelRepo
// =========================================================================

func TestNotifChannelRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationChannelRepo(db)

	ch := &models.NotificationChannel{OrgID: 1, Name: "Email", Type: "email", Config: `{}`, IsActive: true}
	require.NoError(t, db.Create(ch).Error)

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

// =========================================================================
// NotificationRuleRepo
// =========================================================================

func TestNotifRuleRepo_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRuleRepo(db)

	rule := &models.NotificationRule{OrgID: 1, ChannelID: 1, Severity: "high", IsActive: true}
	require.NoError(t, db.Create(rule).Error)

	rules, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, rules, 1)

	activeRules, err := repo.FindActiveByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, activeRules, 1)

	affected, err := repo.DeleteByIDAndOrg(ctx, rule.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)
}

// =========================================================================
// NotificationRepo
// =========================================================================

func TestNotifRepo_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRepo(db)

	notif := &models.Notification{
		OrgID: 1, UserID: 0, ChannelID: 1,
		Title: "Test", Message: "msg", IsRead: false, SentAt: time.Now(),
	}
	require.NoError(t, db.Create(notif).Error)

	notifs, err := repo.FindByUserAndOrg(ctx, 1, 42, false)
	require.NoError(t, err)
	assert.Len(t, notifs, 1, "org-wide notification should be visible to any user")
}

func TestNotifRepo_MarkRead(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRepo(db)

	notif := &models.Notification{
		OrgID: 1, UserID: 42, ChannelID: 1,
		Title: "Unread", Message: "msg", IsRead: false, SentAt: time.Now(),
	}
	require.NoError(t, db.Create(notif).Error)

	affected, err := repo.MarkRead(ctx, notif.ID, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	count, err := repo.CountUnread(ctx, 1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestNotifRepo_CountUnread(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRepo(db)

	require.NoError(t, db.Create(&models.Notification{
		OrgID: 1, UserID: 0, ChannelID: 1, Title: "N1", Message: "m", IsRead: false, SentAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&models.Notification{
		OrgID: 1, UserID: 0, ChannelID: 1, Title: "N2", Message: "m", IsRead: true, SentAt: time.Now(),
	}).Error)

	count, err := repo.CountUnread(ctx, 1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// =========================================================================
// AuditLogRepo
// =========================================================================

func TestAuditLogRepo_CreateAndFindByOrgID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAuditLogRepo(db)

	entry := &models.AuditLog{
		UserID: 1, OrgID: 1, Action: "create", Resource: "package", ResourceID: 1,
	}
	require.NoError(t, db.Create(entry).Error)

	logs, total, err := repo.FindByOrgID(ctx, 1, domain.AuditLogFilters{}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
	assert.Equal(t, "create", logs[0].Action)
}

// =========================================================================
// SettingRepo
// =========================================================================

func TestSettingRepo_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSettingRepo(db)

	setting := &domain.Setting{OrgID: 1, Key: "test_key", Value: "value1"}
	err := repo.Upsert(ctx, setting)
	require.NoError(t, err)
	assert.NotZero(t, setting.ID)

	// Upsert again with new value
	setting2 := &domain.Setting{OrgID: 1, Key: "test_key", Value: "value2"}
	err = repo.Upsert(ctx, setting2)
	require.NoError(t, err)

	// Should have been updated, not duplicated
	settings, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, settings, 1)
}

func TestSettingRepo_FindByKey(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSettingRepo(db)

	require.NoError(t, db.Create(&models.Setting{OrgID: 1, Key: "found_key", Value: "v"}).Error)

	found, err := repo.FindByKey(ctx, 1, "found_key")
	require.NoError(t, err)
	assert.Equal(t, "v", found.Value)
}

// =========================================================================
// OrganizationRepo
// =========================================================================

func TestOrgRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	org := &domain.Organization{Name: "Acme", Slug: "acme", OwnerID: 1, IsActive: true}
	err := repo.Create(ctx, org)
	require.NoError(t, err)
	assert.NotZero(t, org.ID)

	found, err := repo.FindByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "Acme", found.Name)
}

func TestOrgRepo_CountBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	org := &domain.Organization{Name: "Acme", Slug: "acme", OwnerID: 1, IsActive: true}
	require.NoError(t, repo.Create(ctx, org))

	count, err := repo.CountBySlug(ctx, "acme", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = repo.CountBySlug(ctx, "acme", &org.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "excluding own ID should return 0")
}

// =========================================================================
// InvitationRepo
// =========================================================================

func TestInvitationRepo_CreateAndFindByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewInvitationRepo(db)

	inv := &domain.Invitation{
		OrgID: 1, Email: "new@example.com", RoleID: 1,
		TokenHash: "hashed-secret-token", InvitedBy: 1, ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	err := repo.Create(ctx, inv)
	require.NoError(t, err)
	assert.NotZero(t, inv.ID)

	found, err := repo.FindByTokenHash(ctx, "hashed-secret-token")
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", found.Email)
}

// =========================================================================
// Helpers
// =========================================================================

// userModelToDomainForTest converts a models.User to a domain.User for testing.
func userModelToDomainForTest(m *models.User) *domain.User {
	return &domain.User{
		ID:            m.ID,
		Email:         m.Email,
		PasswordHash:  m.PasswordHash,
		FirstName:     m.FirstName,
		LastName:      m.LastName,
		IsActive:      m.IsActive,
		EmailVerified: m.EmailVerified,
		LastLoginAt:   m.LastLoginAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
