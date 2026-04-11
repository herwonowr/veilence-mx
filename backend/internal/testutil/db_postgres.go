//go:build integration

package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// defaultTestDSN is the default PostgreSQL DSN for integration tests.
// Override with TEST_DATABASE_URL environment variable.
const defaultTestDSN = "postgres://veilence:veilence_dev@localhost:5432/veilence_mx_test?sslmode=disable"

// SetupPostgresDB creates a PostgreSQL-backed test database using the
// docker-compose postgres service. It uses a dedicated test database and
// auto-migrates all models. Each test gets an isolated schema via TruncateAll.
//
// Prerequisites:
//   - PostgreSQL running: `docker-compose up -d postgres`
//   - Test database created: `CREATE DATABASE veilence_mx_test;`
//     (or set TEST_DATABASE_URL to an existing database)
//
// Usage:
//
//	go test -tags=integration ./... -v
func SetupPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		t.Skipf("PostgreSQL not available (set TEST_DATABASE_URL or run docker-compose up -d postgres): %v", err)
	}

	// Verify connectivity
	sqlDB, err := db.DB()
	require.NoError(t, err, "failed to get underlying sql.DB")

	if err := sqlDB.Ping(); err != nil {
		t.Skipf("PostgreSQL not reachable: %v", err)
	}

	// Auto-migrate all models (creates tables if needed)
	err = db.AutoMigrate(AllModels...)
	require.NoError(t, err, "failed to auto-migrate models on PostgreSQL")

	// Truncate all tables to start clean
	TruncateAllPostgres(t, db)

	t.Cleanup(func() {
		TruncateAllPostgres(t, db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return db
}

// TruncateAllPostgres truncates all tables using PostgreSQL TRUNCATE CASCADE.
func TruncateAllPostgres(t *testing.T, db *gorm.DB) {
	t.Helper()

	tables := []string{
		"alert_notes", "notifications", "notification_rules", "notification_channels",
		"audit_logs", "invitations", "org_members", "permissions", "roles",
		"alerts", "analyses", "diffs", "releases", "packages",
		"sessions", "email_verification_tokens", "password_reset_tokens",
		"api_keys", "refresh_tokens", "settings",
		"organizations", "users",
	}
	for _, table := range tables {
		db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
	}
}
