package app

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/config"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi"
	v1 "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/cache"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/repo/registry"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/internal/usecase/alert"
	alertnoteuc "github.com/veilence/veilence-mx/backend/internal/usecase/alertnote"
	"github.com/veilence/veilence-mx/backend/internal/usecase/analyzer"
	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/dashboard"
	"github.com/veilence/veilence-mx/backend/internal/usecase/differ"
	"github.com/veilence/veilence-mx/backend/internal/usecase/digest"
	"github.com/veilence/veilence-mx/backend/internal/usecase/health"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
	pkguc "github.com/veilence/veilence-mx/backend/internal/usecase/pkg"
	"github.com/veilence/veilence-mx/backend/internal/usecase/poller"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
	"github.com/veilence/veilence-mx/backend/internal/usecase/release"
	"github.com/veilence/veilence-mx/backend/internal/usecase/setting"
	"github.com/veilence/veilence-mx/backend/internal/usecase/setup"
	"github.com/veilence/veilence-mx/backend/internal/usecase/sso"
	"github.com/veilence/veilence-mx/backend/pkg/anthropic"
	"github.com/veilence/veilence-mx/backend/pkg/copilotapi"
	"github.com/veilence/veilence-mx/backend/pkg/crypto"
	"github.com/veilence/veilence-mx/backend/pkg/hasher"
	"github.com/veilence/veilence-mx/backend/pkg/llm"
	"github.com/veilence/veilence-mx/backend/pkg/mailer"
	"github.com/veilence/veilence-mx/backend/pkg/oauth"
	"github.com/veilence/veilence-mx/backend/pkg/ollama"
	"github.com/veilence/veilence-mx/backend/pkg/openai"
	"github.com/veilence/veilence-mx/backend/pkg/postgres"
	"github.com/veilence/veilence-mx/backend/pkg/queue"
	pkgsaml "github.com/veilence/veilence-mx/backend/pkg/saml"
	"github.com/veilence/veilence-mx/backend/pkg/sender"
	"github.com/veilence/veilence-mx/backend/pkg/token"
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
	SSOService      *sso.Service
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

	// Registry clients - conditionally instantiated based on ECOSYSTEMS_ENABLED
	var pythonClient usecase.Registry
	if slices.Contains(cfg.EcosystemsEnabled, "pypi") {
		pythonClient = registry.NewPyPIClient(registry.WithPyPIMaxDownloadSize(cfg.MaxRegistryDownloadSize))
	}
	var npmClient usecase.Registry
	if slices.Contains(cfg.EcosystemsEnabled, "npm") {
		npmClient = registry.NewNPMClient(registry.WithNPMMaxDownloadSize(cfg.MaxRegistryDownloadSize))
	}
	var goClient usecase.Registry
	if slices.Contains(cfg.EcosystemsEnabled, "go") {
		goClient = registry.NewGoModulesClient(registry.WithGoModulesMaxDownloadSize(cfg.MaxRegistryDownloadSize))
	}

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

	// Notification senders
	var emailNotifSender usecase.EmailNotificationSender
	if smtpConfig.IsConfigured() {
		emailNotifSender = sender.NewSMTPSender(sender.SMTPConfig{
			Host:     smtpConfig.Host,
			Port:     smtpConfig.Port,
			Username: smtpConfig.Username,
			Password: smtpConfig.Password,
			From:     smtpConfig.From,
			UseTLS:   smtpConfig.UseTLS,
		})
	}
	webhookSender := sender.NewHTTPWebhookSender()
	slackSender := sender.NewHTTPSlackSender()

	// RBAC repo (single instance - also satisfies workspace lister interface for notifications)
	rbacRepo := persistent.NewRBACRepo(db)

	// Notification service
	notificationChannelRepo := persistent.NewNotificationChannelRepo(db)
	notificationRuleRepo := persistent.NewNotificationRuleRepo(db)
	notificationRepo := persistent.NewNotificationRepo(db)
	notificationService := notifications.NewService(notificationChannelRepo, notificationRuleRepo, notificationRepo, rbacRepo, smtpConfig, emailNotifSender, webhookSender, slackSender)

	// Pipeline
	pollerRepo := persistent.NewPollerRepo(db)
	pollerService := poller.New(pollerRepo, pythonClient, npmClient, goClient, poller.Config{
		MonitoringInterval:   cfg.MonitoringInterval,
		DiscoveryInterval:    cfg.DiscoveryInterval,
		Concurrency:          cfg.PollerConcurrency,
		WorkspaceConcurrency: cfg.PollerWorkspaceConcurrency,
	}, jobQueue, notificationService)

	differRepo := persistent.NewDifferRepo(db)
	differService := differ.New(differRepo, pythonClient, npmClient, goClient, differ.Config{
		DiffSizeLimit:       cfg.DiffSizeLimit,
		MaxArchiveFileCount: cfg.MaxArchiveFileCount,
		MaxArchiveSize:      cfg.MaxArchiveSize,
		MaxFileExtractSize:  cfg.MaxFileExtractSize,
		MaxFileReadSize:     cfg.MaxFileReadSize,
	}, jobQueue, notificationService)

	// LLM analyzer - provider selection
	var llmAdapter analyzer.LLMProvider
	switch cfg.LLMProvider {
	case "copilot":
		llmConfig := copilotapi.Config{
			BaseURL:      cfg.LLMApiURL,
			Model:        cfg.LLMModel,
			MaxDiffSize:  cfg.LLMMaxDiffSize,
			RateInterval: cfg.LLMRateInterval,
		}
		if err := llmConfig.Validate(); err != nil {
			return nil, fmt.Errorf("copilot LLM config error: %w", err)
		}
		client := copilotapi.New(llmConfig)
		llmAdapter = &copilotAdapter{client: client}
		slog.Info("LLM provider: copilot", "url", llmConfig.BaseURL, "model", llmConfig.Model)

	case "openai":
		llmConfig := openai.Config{
			APIKey:       cfg.LLMApiKey,
			Model:        cfg.LLMModel,
			BaseURL:      cfg.LLMApiURL,
			MaxDiffSize:  cfg.LLMMaxDiffSize,
			RateInterval: cfg.LLMRateInterval,
		}
		if err := llmConfig.Validate(); err != nil {
			return nil, fmt.Errorf("openai LLM config error: %w", err)
		}
		client := openai.New(llmConfig)
		llmAdapter = &openaiAdapter{client: client}
		slog.Info("LLM provider: openai", "model", llmConfig.Model)

	case "anthropic":
		llmConfig := anthropic.Config{
			APIKey:       cfg.LLMApiKey,
			Model:        cfg.LLMModel,
			BaseURL:      cfg.LLMApiURL,
			MaxDiffSize:  cfg.LLMMaxDiffSize,
			RateInterval: cfg.LLMRateInterval,
		}
		if err := llmConfig.Validate(); err != nil {
			return nil, fmt.Errorf("anthropic LLM config error: %w", err)
		}
		client := anthropic.New(llmConfig)
		llmAdapter = &anthropicAdapter{client: client}
		slog.Info("LLM provider: anthropic", "model", llmConfig.Model)

	case "ollama":
		llmConfig := ollama.Config{
			Model:        cfg.LLMModel,
			BaseURL:      cfg.LLMApiURL,
			MaxDiffSize:  cfg.LLMMaxDiffSize,
			RateInterval: cfg.LLMRateInterval,
		}
		if err := llmConfig.Validate(); err != nil {
			return nil, fmt.Errorf("ollama LLM config error: %w", err)
		}
		client := ollama.New(llmConfig)
		llmAdapter = &ollamaAdapter{client: client}
		slog.Info("LLM provider: ollama", "url", llmConfig.BaseURL, "model", llmConfig.Model)

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLMProvider)
	}

	pipelineRepo := persistent.NewPipelineRepo(db)
	pipeline := analyzer.NewPipeline(pipelineRepo, notificationService, cfg.DiffSizeLimit, llmAdapter)

	// Queue workers
	diffJobTimeout := time.Duration(cfg.DiffJobTimeoutSeconds) * time.Second
	diffWorker := queue.NewWorker(jobQueue, queue.JobTypeDiff, func(ctx context.Context, job *queue.Job) error {
		jobCtx, cancel := context.WithTimeout(ctx, diffJobTimeout)
		defer cancel()
		return differService.ProcessRelease(jobCtx, job.ReferenceID)
	}, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  cfg.DiffWorkerConcurrency,
	})
	analyzeJobTimeout := time.Duration(cfg.AnalyzeJobTimeoutSeconds) * time.Second
	analyzeWorker := queue.NewWorker(jobQueue, queue.JobTypeAnalyze, func(ctx context.Context, job *queue.Job) error {
		jobCtx, cancel := context.WithTimeout(ctx, analyzeJobTimeout)
		defer cancel()
		return pipeline.ProcessDiff(jobCtx, job.ReferenceID)
	}, queue.WorkerConfig{
		PollInterval: 2 * time.Second,
		Concurrency:  cfg.AnalyzeWorkerConcurrency,
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
	var invitationEmailSender usecase.InvitationEmailSender
	if smtpConfig.IsConfigured() {
		m := mailer.New(mailer.SMTPConfig{
			Host:     smtpConfig.Host,
			Port:     smtpConfig.Port,
			Username: smtpConfig.Username,
			Password: smtpConfig.Password,
			From:     smtpConfig.From,
			SkipTLS:  !smtpConfig.UseTLS,
		}, cfg.FrontendURL)
		authEmailSender = m
		invitationEmailSender = m
	}

	var previousSecrets []string
	if len(cfg.JWTSecretPrevious) > 0 {
		previousSecrets = cfg.JWTSecretPrevious
		slog.Info("JWT secret rotation enabled", "previous_secrets_count", len(previousSecrets))
	}

	tokenProvider := &tokenProviderAdapter{provider: token.New(cfg.JWTSecret, previousSecrets...)}
	passwordHasher := hasher.New()

	authService := auth.NewService(userRepo, refreshTokenRepo, apiKeyRepo, passwordResetTokenRepo, emailVerificationTokenRepo, sessionRepo, authEmailSender, cfg.RequireEmailVerification, rateLimiter, tokenProvider, passwordHasher, cfg.RegistrationEnabled, cfg.AllowedEmailDomains)
	slog.Info("auth service initialized", "require_email_verification", cfg.RequireEmailVerification, "email_sender_configured", authEmailSender != nil, "registration_enabled", cfg.RegistrationEnabled)
	auditLogRepo := persistent.NewAuditLogRepo(db)
	if err := rbac.SeedPermissions(ctx, rbacRepo); err != nil {
		return nil, fmt.Errorf("seeding permissions: %w", err)
	}
	rbacService := rbac.NewService(rbacRepo, invitationEmailSender,
		rbac.WithUserEmailResolver(userRepo),
		rbac.WithRegistrationEnabled(cfg.RegistrationEnabled),
		rbac.WithAllowedEmailDomains(cfg.AllowedEmailDomains),
		rbac.WithUserAccountCreator(authService),
		rbac.WithPasswordResetInitiator(authService),
	)
	auditService := audit.NewService(auditLogRepo)
	dashboardRepo := persistent.NewDashboardRepo(db)
	alertNoteRepo := persistent.NewAlertNoteRepo(db)
	alertRepo := persistent.NewAlertRepo(db)
	alertNoteService := alertnoteuc.New(alertNoteRepo, alertRepo, userRepo)
	packageRepo := persistent.NewPackageRepo(db)
	registries := make(map[entity.Ecosystem]usecase.Registry)
	if pythonClient != nil {
		registries[entity.EcosystemPython] = pythonClient
	}
	if npmClient != nil {
		registries[entity.EcosystemNPM] = npmClient
	}
	if goClient != nil {
		registries[entity.EcosystemGo] = goClient
	}
	packageService := pkguc.New(packageRepo, auditService, registries, pollerService)
	releaseRepo := persistent.NewReleaseRepo(db)
	diffRepo := persistent.NewDiffRepo(db)
	analysisRepo := persistent.NewAnalysisRepo(db)

	alertService := alert.New(alertRepo, auditService)
	releaseService := release.New(packageRepo, releaseRepo, diffRepo, analysisRepo, jobQueue)
	settingService := setting.New(settingRepo)
	dashboardService := dashboard.New(dashboardRepo, releaseRepo, jobQueue)
	healthService := health.New(dbPinger{db: db}, jobQueue, version)

	// Setup service (initial platform setup)
	setupService := setup.NewService(userRepo, rbacRepo, passwordHasher, tokenProvider)

	// Digest scheduler
	digestRepo := persistent.NewDigestRepo(db)
	digestScheduler := digest.New(digestRepo, emailNotifSender, smtpConfig.From, digest.Config{})

	// SSO (conditionally wired when enabled)
	var ssoService *sso.Service
	var userIdentityRepo *persistent.UserIdentityRepo
	if cfg.SSOEnabled {
		encryptor, err := crypto.NewAESEncryptor(cfg.SSOEncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSO encryptor: %w", err)
		}

		ssoConfigRepo := persistent.NewSSOConfigRepo(db, encryptor)
		userIdentityRepo = persistent.NewUserIdentityRepo(db)
		ssoStateRepo := persistent.NewSSOStateRepo(db)

		samlProvider := pkgsaml.NewProvider(cfg.BackendURL, cfg.SSOSAMLClockSkew)
		oauthExchanger := newOAuthExchangerAdapter(oauth.NewExchanger(oauth.ExchangerConfig{
			CallbackBaseURL:   cfg.BackendURL,
			GoogleTokenURL:    cfg.OAuthGoogleTokenURL,
			GoogleUserInfoURL: cfg.OAuthGoogleUserInfoURL,
			GitHubTokenURL:    cfg.OAuthGitHubTokenURL,
			GitHubUserInfoURL: cfg.OAuthGitHubUserInfoURL,
			GitHubEmailsURL:   cfg.OAuthGitHubEmailsURL,
			GitHubOrgsURL:     cfg.OAuthGitHubOrgsURL,
		}))

		ssoService = sso.NewService(
			ssoConfigRepo,
			userIdentityRepo,
			ssoStateRepo,
			newSAMLProviderAdapter(samlProvider),
			oauthExchanger,
			authService,
			userRepo,
			authService,
			sessionRepo,
			refreshTokenRepo,
			settingRepo,
			cfg.SSOStateTTL,
			cfg.BackendURL,
			sso.WithAuditLogger(auditService),
			sso.WithMetadataFetcher(pkgsaml.NewMetadataFetcher()),
			sso.WithRegistrationEnabled(cfg.RegistrationEnabled),
			sso.WithOAuthAuthURLs(cfg.OAuthGoogleAuthURL, cfg.OAuthGitHubAuthURL),
		)

		// Load or generate SP signing key and configure the SAML provider.
		spKey, spCertPEM, spKeyErr := ssoService.EnsureSPSigningKey(context.Background())
		if spKeyErr != nil {
			slog.Error("Failed to ensure SP signing key", "error", spKeyErr)
		} else if spCertPEM != "" {
			block, _ := pem.Decode([]byte(spCertPEM))
			if block != nil {
				spCert, parseErr := x509.ParseCertificate(block.Bytes)
				if parseErr == nil {
					samlProvider.SetSPKeyPair(spKey, spCert)
					slog.Info("SAML SP signing key configured")
				}
			}
		}

		slog.Info("SSO enabled")
	} else {
		slog.Info("SSO disabled")
	}

	// HTTP handlers + router
	var identityRepoIface usecase.UserIdentityRepository
	if userIdentityRepo != nil {
		identityRepoIface = userIdentityRepo
	}
	h := v1.NewHandlers(
		authService,
		rbacService,
		auditService,
		notificationService,
		pollerService,
		pythonClient,
		npmClient,
		goClient,
		jobQueue,
		alertNoteService,
		packageService,
		alertService,
		releaseService,
		settingService,
		dashboardService,
		healthService,
		setupService,
		ssoService,
		cfg.FrontendURL,
		rbacRepo,
		identityRepoIface,
		cfg.EcosystemsEnabled,
		cfg.MaxBulkImport,
		cfg.MaxBulkApprove,
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
		SSOService:      ssoService,
	}, nil
}

// Close cleans up resources held by dependencies.
func (d *Dependencies) Close() {
	if d.Queue != nil {
		d.Queue.Close()
	}
}

// dbPinger adapts *gorm.DB to satisfy health.DBPinger.
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
	db.Preload("Package").Where("status IN ?", []string{string(persistent.ReleaseStatusDiffing), string(persistent.ReleaseStatusAnalyzing)}).Find(&stuckDiffing)

	for _, rel := range stuckDiffing {
		wsID := rel.Package.WorkspaceID
		meta := map[string]string{
			"package":   rel.Package.Name,
			"version":   rel.Version,
			"ecosystem": string(rel.Package.Ecosystem),
		}
		switch rel.Status {
		case persistent.ReleaseStatusDiffing:
			db.Model(&rel).Update("status", persistent.ReleaseStatusPending)
			if _, err := jobQueue.Enqueue(ctx, queue.JobTypeDiff, wsID, rel.ID, meta); err != nil {
				slog.Error("failed to re-enqueue stuck diffing release", "release_id", rel.ID, "error", err)
			} else {
				slog.Info("re-enqueued stuck diffing release", "release_id", rel.ID)
			}
		case persistent.ReleaseStatusAnalyzing:
			var diff persistent.Diff
			result := db.Where("release_id = ?", rel.ID).Limit(1).Find(&diff)
			if result.RowsAffected > 0 {
				var analysisCount int64
				db.Model(&persistent.Analysis{}).Where("diff_id = ?", diff.ID).Count(&analysisCount)
				if analysisCount == 0 {
					if _, err := jobQueue.Enqueue(ctx, queue.JobTypeAnalyze, wsID, diff.ID, meta); err != nil {
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
				if _, err := jobQueue.Enqueue(ctx, queue.JobTypeDiff, wsID, rel.ID, meta); err != nil {
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

// llmClientAdapter is a generic interface for all LLM provider clients.
type llmClientAdapter interface {
	Analyze(ctx context.Context, diff string, packageName string, ecosystem string, oldVersion string, newVersion string, truncated bool) (*llm.Result, error)
	Type() string
}

// genericLLMAdapter adapts any llmClientAdapter to satisfy analyzer.LLMProvider.
type genericLLMAdapter struct {
	client llmClientAdapter
}

func (a *genericLLMAdapter) Analyze(ctx context.Context, diff string, packageName string, ecosystem string, oldVersion string, newVersion string, truncated bool) (*analyzer.LLMResult, error) {
	result, err := a.client.Analyze(ctx, diff, packageName, ecosystem, oldVersion, newVersion, truncated)
	if err != nil {
		return nil, err
	}
	return &analyzer.LLMResult{
		Classification: result.Classification,
		Confidence:     result.Confidence,
		Reasoning:      result.Reasoning,
		RawResponse:    result.RawResponse,
	}, nil
}

func (a *genericLLMAdapter) Type() string {
	return a.client.Type()
}

// Type aliases for clarity in the provider switch.
type copilotAdapter = genericLLMAdapter
type openaiAdapter = genericLLMAdapter
type anthropicAdapter = genericLLMAdapter
type ollamaAdapter = genericLLMAdapter

// tokenProviderAdapter adapts pkg/token.JWTProvider to satisfy usecase.TokenProvider.
type tokenProviderAdapter struct {
	provider *token.JWTProvider
}

func (a *tokenProviderAdapter) GenerateAccessToken(userID, email string, duration time.Duration) (string, error) {
	return a.provider.GenerateAccessToken(userID, email, duration)
}

func (a *tokenProviderAdapter) GenerateRefreshToken() (string, error) {
	return a.provider.GenerateRefreshToken()
}

func (a *tokenProviderAdapter) ValidateAccessToken(tokenString string) (*usecase.TokenClaims, error) {
	c, err := a.provider.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &usecase.TokenClaims{
		UserID:    c.UserID,
		Email:     c.Email,
		TokenType: c.TokenType,
	}, nil
}

// seedSettingsDefaults seeds global/system-level default settings into the database.
// These are platform-wide defaults (not workspace-scoped), so WorkspaceID is left
// empty. Workspace-specific overrides are created separately via the settings API.
func seedSettingsDefaults(cfg *config.Config, db *gorm.DB) {
	seedDefaults := map[string]string{
		entity.SettingMonitoringInterval:           cfg.MonitoringInterval.String(),
		entity.SettingDiscoveryScanDepth:           fmt.Sprintf("%d", cfg.DiscoveryScanDepth),
		entity.SettingDiscoveryInterval:            cfg.DiscoveryInterval.String(),
		entity.SettingDiscoveryAutoApprove:         fmt.Sprintf("%t", cfg.DiscoveryAutoApprove),
		entity.SettingStaleAutoRemoveMonths:        fmt.Sprintf("%d", cfg.StaleAutoRemoveMonths),
		entity.SettingPackageCountWarningThreshold: fmt.Sprintf("%d", cfg.PackageCountWarningThreshold),
		entity.SettingEmailDigestEnabled:           "false",
		entity.SettingEmailDigestFrequency:         "daily",
		entity.SettingEmailDigestRecipients:        "",
	}
	for key, value := range seedDefaults {
		db.Where("key = ?", key).FirstOrCreate(&persistent.Setting{Key: key, Value: value})
	}
}
