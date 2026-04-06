package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/models"
)

// AllModels is the complete list of GORM models used for auto-migration in
// test databases. Keep this in sync with the models package.
var AllModels = []any{
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
	&models.PasswordResetToken{},
	&models.EmailVerificationToken{},
	&models.Session{},
}

// SetupTestDB creates an in-memory SQLite database with all models migrated.
// It registers a cleanup function that closes the database when the test
// completes. This is the canonical way to create test databases across the
// entire backend.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to open in-memory sqlite")

	err = db.AutoMigrate(AllModels...)
	require.NoError(t, err, "failed to auto-migrate models")

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}

// SetupTestDBWithModels creates an in-memory SQLite database migrating only the
// specified models. This is useful for tests that want to operate on a subset
// of the schema (e.g. auth-only tests).
func SetupTestDBWithModels(t *testing.T, models ...any) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "failed to open in-memory sqlite")

	err = db.AutoMigrate(models...)
	require.NoError(t, err, "failed to auto-migrate models")

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}

// TruncateAll deletes all rows from all tables. Useful between subtests
// when sharing a single test database.
func TruncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()

	tables := []string{
		"notifications", "notification_rules", "notification_channels",
		"audit_logs", "invitations", "org_members", "permissions", "roles",
		"alerts", "analyses", "diffs", "releases", "packages",
		"sessions", "email_verification_tokens", "password_reset_tokens",
		"api_keys", "refresh_tokens", "settings",
		"organizations", "users",
	}
	for _, table := range tables {
		db.Exec("DELETE FROM " + table)
	}
}
