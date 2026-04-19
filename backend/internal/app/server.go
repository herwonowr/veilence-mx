package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/veilence/veilence-mx/backend/internal/config"
)

func serveCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the HTTP server and background workers",
		Action: func(c *cli.Context) error {
			return actionServe(cfg)
		},
	}
}

func actionServe(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps, err := BuildDependencies(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to build dependencies: %w", err)
	}
	defer deps.Close()

	// Start background workers
	go deps.PollerService.Start(ctx)
	go deps.DiffWorker.Start(ctx)
	go deps.AnalyzeWorker.Start(ctx)
	go deps.DigestScheduler.Start(ctx)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           deps.Router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("shutdown signal received")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("server starting", "port", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server error: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
