package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/controller/restapi"
	v1 "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/repo/registry"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	alertnoteuc "github.com/veilence/veilence-mx/backend/internal/usecase/alertnote"
	"github.com/veilence/veilence-mx/backend/internal/usecase/alertuc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/analyzer"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/dashboarduc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/differ"
	"github.com/veilence/veilence-mx/backend/internal/usecase/digest"
	"github.com/veilence/veilence-mx/backend/internal/usecase/healthuc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
	"github.com/veilence/veilence-mx/backend/internal/usecase/pkguc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/poller"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
	"github.com/veilence/veilence-mx/backend/internal/usecase/releaseuc"
	"github.com/veilence/veilence-mx/backend/internal/usecase/settinguc"
	"github.com/veilence/veilence-mx/backend/pkg/mailer"
	"github.com/veilence/veilence-mx/backend/pkg/postgres"
	"github.com/veilence/veilence-mx/backend/pkg/queue"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	_ = godotenv.Load()

	app := &cli.App{
		Name:    "veilence-mx",
		Usage:   "Supply Chain Monitoring Platform",
		Version: version,
		Commands: []*cli.Command{
			serveCommand(),
			migrateCommand(),
			seedCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

// --- Shared flags ---

var sharedFlags = []cli.Flag{
	&cli.StringFlag{Name: "database-url", EnvVars: []string{"DATABASE_URL"}, Required: true, Usage: "PostgreSQL connection URL"},
	&cli.StringFlag{Name: "redis-url", EnvVars: []string{"REDIS_URL"}, Value: "redis://localhost:6379/0", Usage: "Redis connection URL"},
}

// --- Serve command ---

func serveCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the HTTP server and background workers",
		Flags: append(sharedFlags,
			&cli.StringFlag{Name: "port", EnvVars: []string{"SERVER_PORT"}, Value: "8080", Usage: "HTTP server port"},
			&cli.StringFlag{Name: "frontend-url", EnvVars: []string{"FRONTEND_URL"}, Value: "http://localhost:3000", Usage: "Frontend URL for CORS"},
			&cli.StringFlag{Name: "jwt-secret", EnvVars: []string{"JWT_SECRET"}, Required: true, Usage: "JWT signing secret"},
			&cli.StringFlag{Name: "jwt-secret-previous", EnvVars: []string{"JWT_SECRET_PREVIOUS"}, Usage: "Previous JWT secrets for rotation (comma-separated)"},
			&cli.StringFlag{Name: "app-env", EnvVars: []string{"APP_ENV"}, Value: "production", Usage: "Application environment (development|production)"},

			// LLM configuration — all mandatory
			&cli.StringFlag{Name: "llm-api-url", EnvVars: []string{"LLM_API_URL"}, Required: true, Usage: "LLM API endpoint URL"},
			&cli.StringFlag{Name: "llm-model", EnvVars: []string{"LLM_MODEL"}, Required: true, Usage: "LLM model name"},
			&cli.IntFlag{Name: "llm-max-diff-len", EnvVars: []string{"LLM_MAX_DIFF_LEN"}, Value: 20000, Usage: "Max diff length (chars) sent to LLM"},
			&cli.DurationFlag{Name: "llm-rate-interval", EnvVars: []string{"LLM_RATE_INTERVAL"}, Value: 6 * time.Second, Usage: "Min interval between LLM requests"},

			// Pipeline
			&cli.DurationFlag{Name: "monitoring-interval", EnvVars: []string{"MONITORING_INTERVAL"}, Value: 1 * time.Hour, Usage: "Package polling interval"},
			&cli.DurationFlag{Name: "discovery-interval", EnvVars: []string{"DISCOVERY_INTERVAL"}, Value: 24 * time.Hour, Usage: "Top-N discovery interval"},
			&cli.IntFlag{Name: "poller-concurrency", EnvVars: []string{"POLLER_CONCURRENCY"}, Value: 5, Usage: "Poller concurrency"},
			&cli.IntFlag{Name: "diff-size-limit", EnvVars: []string{"DIFF_SIZE_LIMIT"}, Value: 102400, Usage: "Max diff size in bytes"},
			&cli.IntFlag{Name: "queue-max-retries", EnvVars: []string{"QUEUE_MAX_RETRIES"}, Value: 5, Usage: "Max job retries"},
			&cli.DurationFlag{Name: "queue-lock-timeout", EnvVars: []string{"QUEUE_LOCK_TIMEOUT"}, Value: 10 * time.Minute, Usage: "Queue job lock timeout"},

			// SMTP
			&cli.StringFlag{Name: "smtp-host", EnvVars: []string{"SMTP_HOST"}, Usage: "SMTP host"},
			&cli.StringFlag{Name: "smtp-port", EnvVars: []string{"SMTP_PORT"}, Value: "587", Usage: "SMTP port"},
			&cli.StringFlag{Name: "smtp-username", EnvVars: []string{"SMTP_USERNAME"}, Usage: "SMTP username"},
			&cli.StringFlag{Name: "smtp-password", EnvVars: []string{"SMTP_PASSWORD"}, Usage: "SMTP password"},
			&cli.StringFlag{Name: "smtp-from", EnvVars: []string{"SMTP_FROM"}, Usage: "SMTP from address"},
			&cli.BoolFlag{Name: "smtp-use-tls", EnvVars: []string{"SMTP_USE_TLS"}, Value: true, Usage: "Use TLS for SMTP"},
		),
		Action: actionServe,
	}
}

func actionServe(c *cli.Context) error {
	dbURL := c.String("database-url")
	redisURL := c.String("redis-url")
	port := c.String("port")
	frontendURL := c.String("frontend-url")
	jwtSecret := c.String("jwt-secret")
	appEnv := c.String("app-env")

	// Validate JWT secret in production
	if jwtSecret == "veilence-mx-dev-jwt-secret-change-in-production" && appEnv != "development" {
		return fmt.Errorf("JWT_SECRET must be set to a secure value in production")
	}

	// Validate LLM config
	llmConfig := analyzer.CLIClientConfig{
		BaseURL:      c.String("llm-api-url"),
		Model:        c.String("llm-model"),
		MaxDiffLen:   c.Int("llm-max-diff-len"),
		RateInterval: c.Duration("llm-rate-interval"),
	}
	if err := llmConfig.Validate(); err != nil {
		return fmt.Errorf("LLM configuration error: %w", err)
	}

	// Connect to database
	db, err := postgres.Connect(dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	slog.Info("database connected")

	// Seed settings defaults
	seedSettingsDefaults(c, db)

	// Connect to Redis queue
	jobQueue, err := queue.New(queue.Config{
		RedisURL:    redisURL,
		MaxAttempts: c.Int("queue-max-retries"),
		LockTimeout: c.Duration("queue-lock-timeout"),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	defer jobQueue.Close()
	slog.Info("redis queue connected")

	// Recover stuck jobs
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

	// Registry clients
	pythonClient := registry.NewPyPIClient()
	npmClient := registry.NewNPMClient()

	// SMTP configuration
	smtpConfig := notifications.SMTPConfig{
		Host:     c.String("smtp-host"),
		Port:     c.String("smtp-port"),
		Username: c.String("smtp-username"),
		Password: c.String("smtp-password"),
		From:     c.String("smtp-from"),
		UseTLS:   c.Bool("smtp-use-tls"),
	}
	if smtpConfig.IsConfigured() {
		slog.Info("SMTP configured for email notifications", "host", smtpConfig.Host, "port", smtpConfig.Port, "from", smtpConfig.From)
	} else {
		slog.Warn("SMTP not configured — email notifications will be skipped")
	}

	// Notification service
	notificationChannelRepo := persistent.NewNotificationChannelRepo(db)
	notificationRuleRepo := persistent.NewNotificationRuleRepo(db)
	notificationRepo := persistent.NewNotificationRepo(db)
	notificationService := notifications.NewService(notificationChannelRepo, notificationRuleRepo, notificationRepo, smtpConfig)

	// Pipeline services
	pollerService := poller.New(db, pythonClient, npmClient, poller.Config{
		MonitoringInterval: c.Duration("monitoring-interval"),
		DiscoveryInterval:  c.Duration("discovery-interval"),
		Concurrency:        c.Int("poller-concurrency"),
	}, jobQueue, notificationService)

	differService := differ.New(db, pythonClient, npmClient, differ.Config{
		DiffSizeLimit: c.Int("diff-size-limit"),
	}, jobQueue, notificationService)

	// LLM analyzer
	var analyzers []analyzer.Analyzer
	llmClient := analyzer.NewCLIClient(llmConfig)
	analyzers = append(analyzers, llmClient)
	slog.Info("LLM analyzer enabled", "url", llmConfig.BaseURL, "model", llmConfig.Model)

	pipeline := analyzer.NewPipeline(db, notificationService, analyzers...)

	// Queue workers
	diffWorker := queue.NewWorker(jobQueue, queue.JobTypeDiff, differService.ProcessJob, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  2,
	})
	analyzeWorker := queue.NewWorker(jobQueue, queue.JobTypeAnalyze, pipeline.ProcessJob, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  1,
	})

	go pollerService.Start(ctx)
	go diffWorker.Start(ctx)
	go analyzeWorker.Start(ctx)

	// Auth and services
	userRepo := persistent.NewUserRepo(db)
	refreshTokenRepo := persistent.NewRefreshTokenRepo(db)
	apiKeyRepo := persistent.NewAPIKeyRepo(db)
	passwordResetTokenRepo := persistent.NewPasswordResetTokenRepo(db)
	emailVerificationTokenRepo := persistent.NewEmailVerificationTokenRepo(db)
	sessionRepo := persistent.NewSessionRepo(db)
	settingRepo := persistent.NewSettingRepo(db)

	var authEmailSender usecase.AuthEmailSender
	if smtpConfig.IsConfigured() {
		authEmailSender = mailer.New(mailer.SMTPConfig{
			Host:     smtpConfig.Host,
			Port:     smtpConfig.Port,
			Username: smtpConfig.Username,
			Password: smtpConfig.Password,
			From:     smtpConfig.From,
		}, frontendURL)
	}

	var previousSecrets []string
	if prev := c.String("jwt-secret-previous"); prev != "" {
		for _, s := range strings.Split(prev, ",") {
			if trimmed := strings.TrimSpace(s); trimmed != "" {
				previousSecrets = append(previousSecrets, trimmed)
			}
		}
		slog.Info("JWT secret rotation enabled", "previous_secrets_count", len(previousSecrets))
	}

	authService := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, authEmailSender, settingRepo, jwtSecret, previousSecrets...)
	rbacService := rbac.NewService(db)
	auditService := audit.NewService(db)
	dashboardRepo := persistent.NewDashboardRepo(db)
	alertNoteRepo := persistent.NewAlertNoteRepo(db)
	alertRepo := persistent.NewAlertRepo(db)
	alertNoteService := alertnoteuc.New(alertNoteRepo, alertRepo, userRepo)
	packageRepo := persistent.NewPackageRepo(db)
	packageService := pkguc.New(packageRepo, auditService)
	releaseRepo := persistent.NewReleaseRepo(db)
	diffRepo := persistent.NewDiffRepo(db)
	analysisRepo := persistent.NewAnalysisRepo(db)

	alertService := alertuc.New(alertRepo, auditService)
	releaseService := releaseuc.New(packageRepo, releaseRepo, diffRepo, analysisRepo, jobQueue)
	settingService := settinguc.New(settingRepo)
	dashboardService := dashboarduc.New(dashboardRepo, releaseRepo, diffRepo, analysisRepo, jobQueue)

	healthService := healthuc.New(dbPinger{db: db}, jobQueue)

	// Digest scheduler
	digestScheduler := digest.New(db, smtpConfig, digest.Config{})
	go digestScheduler.Start(ctx)

	h := v1.NewHandlers(
		authService,
		rbacService,
		auditService,
		notificationService,
		pollerService,
		pythonClient,
		npmClient,
		jobQueue,
		alertNoteService,
		packageService,
		alertService,
		releaseService,
		settingService,
		dashboardService,
		healthService,
	)

	router := restapi.NewRouter(h, frontendURL, authService, rbacService)

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
		return fmt.Errorf("server error: %w", err)
	}

	slog.Info("server stopped")
	return nil
}

func seedSettingsDefaults(c *cli.Context, db *gorm.DB) {
	seedDefaults := map[string]string{
		entity.SettingMonitoringInterval:           c.Duration("monitoring-interval").String(),
		entity.SettingDiscoveryScanDepth:           getEnvOrDefault("DISCOVERY_SCAN_DEPTH", "50"),
		entity.SettingDiscoveryInterval:            c.Duration("discovery-interval").String(),
		entity.SettingDiffSizeLimit:                fmt.Sprintf("%d", c.Int("diff-size-limit")),
		entity.SettingDiscoveryAutoApprove:         getEnvOrDefault("DISCOVERY_AUTO_APPROVE", "false"),
		entity.SettingStaleAutoRemoveMonths:        getEnvOrDefault("STALE_AUTO_REMOVE_MONTHS", "0"),
		entity.SettingPackageCountWarningThreshold: getEnvOrDefault("PACKAGE_COUNT_WARNING_THRESHOLD", "0"),
		entity.SettingRequireEmailVerification:     getEnvOrDefault("REQUIRE_EMAIL_VERIFICATION", "false"),
	}
	for key, value := range seedDefaults {
		db.Where("key = ?", key).FirstOrCreate(&persistent.Setting{Key: key, Value: value})
	}
}

// --- Migrate command ---

func migrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "Run database migrations",
		Subcommands: []*cli.Command{
			{
				Name:  "up",
				Usage: "Apply all pending migrations",
				Flags: sharedFlags,
				Action: func(c *cli.Context) error {
					db, err := openRawDB(c.String("database-url"))
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
				Flags: sharedFlags,
				Action: func(c *cli.Context) error {
					db, err := openRawDB(c.String("database-url"))
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
				Flags: sharedFlags,
				Action: func(c *cli.Context) error {
					db, err := openRawDB(c.String("database-url"))
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
				Flags: sharedFlags,
				Action: func(c *cli.Context) error {
					db, err := openRawDB(c.String("database-url"))
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

// --- Seed command ---

func seedCommand() *cli.Command {
	return &cli.Command{
		Name:  "seed",
		Usage: "Seed default data into the database",
		Flags: sharedFlags,
		Action: func(c *cli.Context) error {
			db, err := postgres.Connect(c.String("database-url"))
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			seedSettingsDefaults(c, db)
			fmt.Println("seed data applied")
			return nil
		},
	}
}

// --- Helpers ---

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

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func recoverStuckReleases(ctx context.Context, jobQueue *queue.Queue, db *gorm.DB) {
	var stuckDiffing []persistent.Release
	db.Where("status IN ?", []string{string(persistent.ReleaseStatusDiffing), string(persistent.ReleaseStatusAnalyzing)}).Find(&stuckDiffing)

	for _, rel := range stuckDiffing {
		if rel.Status == persistent.ReleaseStatusDiffing {
			db.Model(&rel).Update("status", persistent.ReleaseStatusPending)
			if _, err := jobQueue.Enqueue(ctx, queue.JobTypeDiff, rel.ID); err != nil {
				slog.Error("failed to re-enqueue stuck diffing release", "release_id", rel.ID, "error", err)
			} else {
				slog.Info("re-enqueued stuck diffing release", "release_id", rel.ID)
			}
		} else if rel.Status == persistent.ReleaseStatusAnalyzing {
			var diff persistent.Diff
			result := db.Where("release_id = ?", rel.ID).Limit(1).Find(&diff)
			if result.RowsAffected > 0 {
				var analysisCount int64
				db.Model(&persistent.Analysis{}).Where("diff_id = ?", diff.ID).Count(&analysisCount)
				if analysisCount == 0 {
					if _, err := jobQueue.Enqueue(ctx, queue.JobTypeAnalyze, diff.ID); err != nil {
						slog.Error("failed to re-enqueue stuck analyzing release", "release_id", rel.ID, "error", err)
					} else {
						slog.Info("re-enqueued stuck analyzing release", "release_id", rel.ID, "diff_id", diff.ID)
					}
				} else {
					db.Model(&rel).Update("status", persistent.ReleaseStatusCompleted)
					slog.Info("fixed stuck analyzing release (analysis exists)", "release_id", rel.ID)
				}
			} else {
				db.Model(&rel).Update("status", persistent.ReleaseStatusPending)
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

// dbPinger adapts *gorm.DB to satisfy healthuc.DBPinger.
type dbPinger struct {
	db *gorm.DB
}

func (p dbPinger) PingDB(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
