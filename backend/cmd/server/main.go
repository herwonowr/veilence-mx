package main

import (
	"log"

	"github.com/veilence/veilence-mx/backend/internal/app"
	"github.com/veilence/veilence-mx/backend/internal/config"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}
	app.Run(cfg)
}
