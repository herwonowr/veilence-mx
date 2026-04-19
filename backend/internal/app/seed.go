package app

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/veilence/veilence-mx/backend/internal/config"
	"github.com/veilence/veilence-mx/backend/pkg/postgres"
)

func seedCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "seed",
		Usage: "Seed default data into the database",
		Action: func(c *cli.Context) error {
			db, err := postgres.Connect(cfg.DatabaseURL)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			seedSettingsDefaults(cfg, db)
			fmt.Println("seed data applied")
			return nil
		},
	}
}
