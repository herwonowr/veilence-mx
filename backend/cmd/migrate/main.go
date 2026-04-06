// Command migrate provides a CLI tool for managing database migrations.
//
// Usage:
//
//	migrate up       - Apply all pending migrations
//	migrate down     - Roll back all migrations
//	migrate rollback - Roll back one migration step
//	migrate version  - Show current migration version
package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"github.com/veilence/veilence-mx/backend/internal/database"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	_ = godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down|rollback|version>")
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://veilence:veilence_dev@localhost:5432/veilence_mx?sslmode=disable"
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		if err := database.RunMigrations(db); err != nil {
			slog.Error("migration failed", "error", err)
			os.Exit(1)
		}
		fmt.Println("migrations applied successfully")

	case "down":
		if err := database.MigrateDown(db); err != nil {
			slog.Error("rollback failed", "error", err)
			os.Exit(1)
		}
		fmt.Println("all migrations rolled back")

	case "rollback":
		if err := database.RollbackMigrations(db); err != nil {
			slog.Error("rollback failed", "error", err)
			os.Exit(1)
		}
		fmt.Println("one migration step rolled back")

	case "version":
		version, dirty, err := database.MigrationVersion(db)
		if err != nil {
			slog.Error("failed to get version", "error", err)
			os.Exit(1)
		}
		fmt.Printf("version: %d, dirty: %t\n", version, dirty)

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nusage: migrate <up|down|rollback|version>\n", os.Args[1])
		os.Exit(1)
	}
}
