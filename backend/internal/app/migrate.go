package app

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
	"github.com/veilence/veilence-mx/backend/internal/config"
	"github.com/veilence/veilence-mx/backend/pkg/postgres"
)

func migrateCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Run database migrations",
		Subcommands: []*cli.Command{
			{
				Name:  "up",
				Usage: "Apply all pending migrations",
				Action: func(c *cli.Context) error {
					db, err := openRawDB(cfg.DatabaseURL)
					if err != nil {
						return err
					}
					defer db.Close()
					if err := postgres.RunMigrations(db); err != nil {
						return fmt.Errorf("migration failed: %w", err)
					}
					fmt.Println("migrations applied successfully")
					return nil
				},
			},
			{
				Name:  "down",
				Usage: "Roll back all migrations",
				Action: func(c *cli.Context) error {
					db, err := openRawDB(cfg.DatabaseURL)
					if err != nil {
						return err
					}
					defer db.Close()
					if err := postgres.MigrateDown(db); err != nil {
						return fmt.Errorf("rollback failed: %w", err)
					}
					fmt.Println("all migrations rolled back")
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "Roll back one migration step",
				Action: func(c *cli.Context) error {
					db, err := openRawDB(cfg.DatabaseURL)
					if err != nil {
						return err
					}
					defer db.Close()
					if err := postgres.RollbackMigrations(db); err != nil {
						return fmt.Errorf("rollback failed: %w", err)
					}
					fmt.Println("one migration step rolled back")
					return nil
				},
			},
			{
				Name:  "version",
				Usage: "Show current migration version",
				Action: func(c *cli.Context) error {
					db, err := openRawDB(cfg.DatabaseURL)
					if err != nil {
						return err
					}
					defer db.Close()
					ver, dirty, err := postgres.MigrationVersion(db)
					if err != nil {
						return fmt.Errorf("failed to get version: %w", err)
					}
					fmt.Printf("version: %d, dirty: %t\n", ver, dirty)
					return nil
				},
			},
		},
	}
}

func openRawDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return db, nil
}
