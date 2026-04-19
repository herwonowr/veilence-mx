// Package app is the composition root. It wires dependencies and runs the CLI application.
package app

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"

	"github.com/veilence/veilence-mx/backend/internal/config"
)

var version = "dev"

// Run creates the CLI application and executes the appropriate subcommand.
func Run(cfg *config.Config) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	_ = godotenv.Load()

	app := &cli.App{
		Name:    "veilence-mx",
		Usage:   "Supply Chain Monitoring Platform",
		Version: version,
		Commands: []*cli.Command{
			serveCommand(cfg),
			migrateCommand(cfg),
			seedCommand(cfg),
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}
