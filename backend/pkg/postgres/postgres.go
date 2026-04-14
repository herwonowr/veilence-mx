package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	pgMigrate "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
	"github.com/veilence/veilence-mx/backend/migrations"
)

// Connect establishes a connection to the PostgreSQL database and runs versioned migrations.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("getting underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	slog.Info("connected to PostgreSQL database")

	if err := RunMigrations(sqlDB); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	// Seed system permissions after schema is ready
	if err := rbac.SeedPermissions(db); err != nil {
		return nil, fmt.Errorf("seeding permissions: %w", err)
	}

	return db, nil
}

// RunMigrations applies all pending up migrations using golang-migrate.
func RunMigrations(db *sql.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}

	slog.Info("running database migrations")

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("applying migrations: %w", err)
	}

	version, dirty, _ := m.Version()
	slog.Info("database migrations completed", "version", version, "dirty", dirty)
	return nil
}

// RollbackMigrations rolls back one migration step.
func RollbackMigrations(db *sql.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("rolling back migration: %w", err)
	}

	version, _, _ := m.Version()
	slog.Info("migration rolled back", "version", version)
	return nil
}

// MigrateDown rolls back all migrations.
func MigrateDown(db *sql.DB) error {
	m, err := newMigrator(db)
	if err != nil {
		return err
	}

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("rolling back all migrations: %w", err)
	}

	slog.Info("all migrations rolled back")
	return nil
}

// MigrationVersion returns the current migration version and dirty state.
func MigrationVersion(db *sql.DB) (uint, bool, error) {
	m, err := newMigrator(db)
	if err != nil {
		return 0, false, err
	}
	return m.Version()
}

func newMigrator(db *sql.DB) (*migrate.Migrate, error) {
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("creating migration source: %w", err)
	}

	dbDriver, err := pgMigrate.WithInstance(db, &pgMigrate.Config{})
	if err != nil {
		return nil, fmt.Errorf("creating migration db driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return nil, fmt.Errorf("creating migrator: %w", err)
	}

	return m, nil
}
