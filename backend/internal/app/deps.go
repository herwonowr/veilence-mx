package app

import (
	"context"
	"fmt"
	"log/slog"

	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/redis/go-redis/v9"

	"github.com/veilence/veilence-mx/backend/internal/config"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi"
	v1 "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/cache"
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

// Dependencies holds all initialized services, repos, and infrastructure.
type Dependencies struct {
	DB              *gorm.DB
	Queue           *queue.Queue
	Router          http.Handler
	PollerService   *poller.Poller
	DiffWorker      *queue.Worker
	AnalyzeWorker   *queue.Worker
	DigestScheduler *digest.Scheduler
}

// BuildDependencies wires all application dependencies from config.
func BuildDependencies(ctx context.Context, cfg *config.Config) (*Dependencies, error) {
	// Connect to database
	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	slog.Info("database connected")

	// Seed settings defaults
	seedSettingsDefaults(cfg, db)

	// Connect to Redis queue
	jobQueue, err := queue.New(queue.Config{
		RedisURL:    cfg.RedisURL,
		MaxAttempts: cfg.QueueMaxRetries,
		LockTimeout: cfg.QueueLockTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	slog.Info("redis queue connected")

	// Recover stuck jobs
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

	// SMTP
	smtpConfig := notifications.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		UseTLS:   cfg.SMTPUseTLS,
	}
	if smtpConfig.IsConfigured() {
		slog.Info("SMTP configured for email notifications", "host", smtpConfig.Host, "port", smtpConfig.Port, "from", smtpConfig.From)
	} else {
		slog.Warn("SMTP not configured - email notifications will be skipped")
	}

	// Notification service
	notificationChannelRepo := persistent.NewNotificationChannelRepo(db)
	notificationRuleRepo := persistent.NewNotificationRuleRepo(db)
	notificationRepo := persistent.NewNotificationRepo(db)
	workspaceMemberRepo := persistent.NewWorkspaceMemberRepo(db)
	notificationService := notifications.NewService(notificationChannelRepo, notificationRuleRepo, notificationRepo, workspaceMemberRepo, smtpConfig)

	// Pipeline
	pollerRepo := persistent.NewPollerRepo(db)
	pollerService := poller.New(pollerRepo, pythonClient, npmClient, poller.Config{
		MonitoringInterval: cfg.MonitoringInterval,
		DiscoveryInterval:  cfg.DiscoveryInterval,
		Concurrency:        cfg.PollerConcurrency,
	}, jobQueue, notificationService)

	differRepo := persistent.NewDifferRepo(db)
	differService := differ.New(differRepo, pythonClient, npmClient, differ.Config{
		DiffSizeLimit: cfg.DiffSizeLimit,
	}, jobQueue, notificationService)

	// LLM analyzer
	llmConfig := analyzer.CLIClientConfig{
		BaseURL:      cfg.LLMApiURL,
		Model:        cfg.LLMModel,
		MaxDiffLen:   cfg.LLMMaxDiffLen,
		RateInterval: cfg.LLMRateInterval,
	}
	if err := llmConfig.Validate(); err != nil {
		return nil, fmt.Errorf("LLM configuration error: %w", err)
	}

	llmClient := analyzer.NewCLIClient(llmConfig)
	slog.Info("LLM analyzer enabled", "url", llmConfig.BaseURL, "model", llmConfig.Model)
	pipelineRepo := persistent.NewPipelineRepo(db)
	pipeline := analyzer.NewPipeline(pipelineRepo, notificationService, llmClient)

	// Queue workers
	diffWorker := queue.NewWorker(jobQueue, queue.JobTypeDiff, differService.ProcessJob, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  2,
	})
	analyzeWorker := queue.NewWorker(jobQueue, queue.JobTypeAnalyze, pipeline.ProcessJob, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  1,
	})

	// Auth
	userRepo := persistent.NewUserRepo(db)
	refreshTokenRepo := persistent.NewRefreshTokenRepo(db)
	apiKeyRepo := persistent.NewAPIKeyRepo(db)
	passwordResetTokenRepo := persistent.NewPasswordResetTokenRepo(db)
	emailVerificationTokenRepo := persistent.NewEmailVerificationTokenRepo(db)
	sessionRepo := persistent.NewSessionRepo(db)
	settingRepo := persistent.NewSettingRepo(db)

	// Redis client for rate limiting (reuses the same Redis instance as the queue)
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL for rate limiter: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)
	rateLimiter := cache.NewRateLimiter(redisClient)

	var authEmailSender usecase.AuthEmailSender
	if smtpConfig.IsConfigured() {
		authEmailSender = mailer.New(mailer.SMTPConfig{
			Host:     smtpConfig.Host,
			Port:     smtpConfig.Port,
			Username: smtpConfig.Username,
			Password: smtpConfig.Password,
			From:     smtpConfig.From,
		}, cfg.FrontendURL)
	}

	var previousSecrets []string
	if len(cfg.JWTSecretPrevious) > 0 {
		previousSecrets = cfg.JWTSecretPrevious
		slog.Info("JWT secret rotation enabled", "previous_secrets_count", len(previousSecrets))
	}

	authService := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, authEmailSender, settingRepo, rateLimiter, cfg.JWTSecret, previousSecrets...)
	auditLogRepo := persistent.NewAuditLogRepo(db)
	rbacRepo := persistent.NewRBACRepo(db)
	if err := rbac.SeedPermissions(rbacRepo); err != nil {
		return nil, fmt.Errorf("seeding permissions: %w", err)
	}
	rbacService := rbac.NewService(rbacRepo)
	auditService := audit.NewService(auditLogRepo)
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
	digestRepo := persistent.NewDigestRepo(db)
	digestScheduler := digest.New(digestRepo, smtpConfig, digest.Config{})

	// HTTP handlers + router
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
	router := restapi.NewRouter(h, cfg.FrontendURL, authService, rbacService)

	return &Dependencies{
		DB:              db,
		Queue:           jobQueue,
		Router:          router,
		PollerService:   pollerService,
		DiffWorker:      diffWorker,
		AnalyzeWorker:   analyzeWorker,
		DigestScheduler: digestScheduler,
	}, nil
}

// Close cleans up resources held by dependencies.
func (d *Dependencies) Close() {
	if d.Queue != nil {
		d.Queue.Close()
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

// recoverStuckReleases re-enqueues releases stuck in diffing/analyzing state.
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

// seedSettingsDefaults seeds default settings values into the database.
func seedSettingsDefaults(cfg *config.Config, db *gorm.DB) {
	seedDefaults := map[string]string{
		entity.SettingMonitoringInterval:           cfg.MonitoringInterval.String(),
		entity.SettingDiscoveryScanDepth:           cfg.DiscoveryScanDepth,
		entity.SettingDiscoveryInterval:            cfg.DiscoveryInterval.String(),
		entity.SettingDiffSizeLimit:                fmt.Sprintf("%d", cfg.DiffSizeLimit),
		entity.SettingDiscoveryAutoApprove:         cfg.DiscoveryAutoApprove,
		entity.SettingStaleAutoRemoveMonths:        cfg.StaleAutoRemoveMonths,
		entity.SettingPackageCountWarningThreshold: cfg.PackageCountWarningThreshold,
		entity.SettingRequireEmailVerification:     cfg.RequireEmailVerification,
	}
	for key, value := range seedDefaults {
		db.Where("key = ?", key).FirstOrCreate(&persistent.Setting{Key: key, Value: value})
	}
}
