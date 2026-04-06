package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/analyzer"
	"github.com/veilence/veilence-mx/backend/internal/api"
	"github.com/veilence/veilence-mx/backend/internal/api/handlers"
	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/database"
	"github.com/veilence/veilence-mx/backend/internal/differ"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/notifications"
	"github.com/veilence/veilence-mx/backend/internal/poller"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
	"github.com/veilence/veilence-mx/backend/internal/registry"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	_ = godotenv.Load()

	dbURL := getEnv("DATABASE_URL", "postgres://veilence:veilence_dev@localhost:5432/veilence_mx?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")
	port := getEnv("SERVER_PORT", "8080")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")
	jwtSecret := getEnv("JWT_SECRET", "")
	appEnv := getEnv("APP_ENV", "production")
	if jwtSecret == "" || jwtSecret == "veilence-mx-dev-jwt-secret-change-in-production" {
		if appEnv == "development" {
			jwtSecret = "veilence-mx-dev-jwt-secret-change-in-production"
			slog.Warn("using default JWT secret — NOT SAFE FOR PRODUCTION. Set JWT_SECRET env var.")
		} else {
			slog.Error("FATAL: JWT_SECRET environment variable must be set to a secure value. Set APP_ENV=development to use a default secret for local development.")
			os.Exit(1)
		}
	}
	copilotAPIURL := getEnv("COPILOT_API_URL", "http://localhost:4141")
	copilotModel := getEnv("COPILOT_MODEL", "claude-opus-4.6")
	pypiInterval := parseDuration(getEnv("PYPI_POLL_INTERVAL", "5m"), 5*time.Minute)
	npmInterval := parseDuration(getEnv("NPM_POLL_INTERVAL", "5m"), 5*time.Minute)
	concurrency := parseInt(getEnv("POLLER_CONCURRENCY", "5"), 5)
	diffSizeLimit := parseInt(getEnv("DIFF_SIZE_LIMIT", "102400"), 102400)
	queueMaxRetries := parseInt(getEnv("QUEUE_MAX_RETRIES", "5"), 5)
	queueLockTimeout := parseDuration(getEnv("QUEUE_LOCK_TIMEOUT", "10m"), 10*time.Minute)

	db, err := database.Connect(dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// Seed settings from .env defaults (only inserts if key doesn't exist)
	seedDefaults := map[string]string{
		models.SettingPyPIPollInterval:    getEnv("PYPI_POLL_INTERVAL", "5m"),
		models.SettingNPMPollInterval:     getEnv("NPM_POLL_INTERVAL", "5m"),
		models.SettingPyPITopN:            getEnv("PYPI_TOP_N", "100"),
		models.SettingNPMTopN:             getEnv("NPM_TOP_N", "100"),
		models.SettingTopNRefreshInterval: getEnv("TOP_N_REFRESH_INTERVAL", "24h"),
		models.SettingDiffSizeLimit:       getEnv("DIFF_SIZE_LIMIT", "102400"),
		models.SettingVersionDepthMode:    getEnv("VERSION_DEPTH_MODE", "latest"),
		models.SettingVersionDepthCount:   getEnv("VERSION_DEPTH_COUNT", "3"),
	}
	for key, value := range seedDefaults {
		db.Where("key = ?", key).FirstOrCreate(&models.Setting{Key: key, Value: value})
	}

	// Connect to Redis queue
	jobQueue, err := queue.New(queue.Config{
		RedisURL:    redisURL,
		MaxAttempts: queueMaxRetries,
		LockTimeout: queueLockTimeout,
	})
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer jobQueue.Close()
	slog.Info("redis queue connected")

	// Recover stuck jobs from previous crash
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, jobType := range []string{queue.JobTypeDiff, queue.JobTypeAnalyze} {
		recovered, err := jobQueue.RecoverStuckJobs(ctx, jobType)
		if err != nil {
			slog.Error("failed to recover stuck queue jobs", "type", jobType, "error", err)
		} else if recovered > 0 {
			slog.Info("recovered stuck queue jobs", "type", jobType, "count", recovered)
		}
	}
	recoverStuckReleases(ctx, jobQueue, db)

	pypiClient := registry.NewPyPIClient()
	npmClient := registry.NewNPMClient()

	pollerService := poller.New(db, pypiClient, npmClient, poller.Config{
		PyPIInterval:        pypiInterval,
		NPMInterval:         npmInterval,
		Concurrency:         concurrency,
		TopNRefreshInterval: parseDuration(getEnv("TOP_N_REFRESH_INTERVAL", "24h"), 24*time.Hour),
	}, jobQueue)

	differService := differ.New(db, pypiClient, npmClient, differ.Config{
		DiffSizeLimit: diffSizeLimit,
	}, jobQueue)

	var analyzers []analyzer.Analyzer
	analyzers = append(analyzers, analyzer.NewCLIClient(analyzer.CLIClientConfig{
		BaseURL: copilotAPIURL,
		Model:   copilotModel,
	}))
	slog.Info("Copilot analyzer enabled", "url", copilotAPIURL, "model", copilotModel)

	pipeline := analyzer.NewPipeline(db, analyzers...)

	// Start queue workers
	diffWorker := queue.NewWorker(jobQueue, queue.JobTypeDiff, differService.ProcessJob, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  2,
	})
	analyzeWorker := queue.NewWorker(jobQueue, queue.JobTypeAnalyze, pipeline.ProcessJob, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  1, // LLM calls are rate-limited, keep at 1
	})

	go pollerService.Start(ctx)
	go diffWorker.Start(ctx)
	go analyzeWorker.Start(ctx)

	userRepo := repository.NewUserRepo(db)
	refreshTokenRepo := repository.NewRefreshTokenRepo(db)
	apiKeyRepo := repository.NewAPIKeyRepo(db)
	passwordResetTokenRepo := repository.NewPasswordResetTokenRepo(db)
	emailVerificationTokenRepo := repository.NewEmailVerificationTokenRepo(db)
	authService := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, jwtSecret)
	rbacService := rbac.NewService(db)
	auditService := audit.NewService(db)
	notificationChannelRepo := repository.NewNotificationChannelRepo(db)
	notificationRuleRepo := repository.NewNotificationRuleRepo(db)
	notificationRepo := repository.NewNotificationRepo(db)
	smtpConfig := notifications.SMTPConfig{
		Host:     getEnv("SMTP_HOST", ""),
		Port:     getEnv("SMTP_PORT", "587"),
		Username: getEnv("SMTP_USERNAME", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", ""),
		UseTLS:   getEnv("SMTP_USE_TLS", "true") == "true",
	}
	if smtpConfig.IsConfigured() {
		slog.Info("SMTP configured for email notifications", "host", smtpConfig.Host, "port", smtpConfig.Port, "from", smtpConfig.From)
	} else {
		slog.Warn("SMTP not configured — email notifications will be skipped. Set SMTP_HOST, SMTP_PORT, SMTP_FROM env vars.")
	}
	notificationService := notifications.NewService(notificationChannelRepo, notificationRuleRepo, notificationRepo, smtpConfig)
	dashboardRepo := repository.NewDashboardRepo(db)

	h := handlers.NewHandlers(
		db,
		authService,
		rbacService,
		auditService,
		notificationService,
		pollerService,
		pypiClient,
		npmClient,
		jobQueue,
		dashboardRepo,
	)

	router := api.NewRouter(h, frontendURL, authService, rbacService)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

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

	slog.Info("server starting", "port", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

func recoverStuckReleases(ctx context.Context, jobQueue *queue.Queue, db *gorm.DB) {
	var stuckDiffing []models.Release
	db.Where("status IN ?", []string{string(models.ReleaseStatusDiffing), string(models.ReleaseStatusAnalyzing)}).Find(&stuckDiffing)

	for _, rel := range stuckDiffing {
		if rel.Status == models.ReleaseStatusDiffing {
			// Reset to pending and re-enqueue a diff job
			db.Model(&rel).Update("status", models.ReleaseStatusPending)
			if _, err := jobQueue.Enqueue(ctx, queue.JobTypeDiff, rel.ID); err != nil {
				slog.Error("failed to re-enqueue stuck diffing release", "release_id", rel.ID, "error", err)
			} else {
				slog.Info("re-enqueued stuck diffing release", "release_id", rel.ID)
			}
		} else if rel.Status == models.ReleaseStatusAnalyzing {
			// Check if a diff exists — if so, re-enqueue analysis
			var diff models.Diff
			result := db.Where("release_id = ?", rel.ID).Limit(1).Find(&diff)
			if result.RowsAffected > 0 {
				// Check if analysis already exists
				var analysisCount int64
				db.Model(&models.Analysis{}).Where("diff_id = ?", diff.ID).Count(&analysisCount)
				if analysisCount == 0 {
					if _, err := jobQueue.Enqueue(ctx, queue.JobTypeAnalyze, diff.ID); err != nil {
						slog.Error("failed to re-enqueue stuck analyzing release", "release_id", rel.ID, "error", err)
					} else {
						slog.Info("re-enqueued stuck analyzing release", "release_id", rel.ID, "diff_id", diff.ID)
					}
				} else {
					// Analysis exists, just mark completed
					db.Model(&rel).Update("status", models.ReleaseStatusCompleted)
					slog.Info("fixed stuck analyzing release (analysis exists)", "release_id", rel.ID)
				}
			} else {
				// No diff, reset to pending for re-diffing
				db.Model(&rel).Update("status", models.ReleaseStatusPending)
				if _, err := jobQueue.Enqueue(ctx, queue.JobTypeDiff, rel.ID); err != nil {
					slog.Error("failed to re-enqueue stuck release for diffing", "release_id", rel.ID, "error", err)
				} else {
					slog.Info("re-enqueued stuck analyzing release for diffing (no diff)", "release_id", rel.ID)
				}
			}
		}
	}

	if len(stuckDiffing) > 0 {
		slog.Info("recovered stuck releases from database", "count", len(stuckDiffing))
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func parseInt(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
