// Package usecase defines the business logic interfaces for the application.
// All repository and service interfaces are defined here - the consumer defines the contract.
// Implementations live in repo/ (for persistence) or other outer-layer packages.
package usecase

import (
	"context"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// --- Repository Interfaces ---

// UserRepository defines persistence operations for User entities.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	CountAll(ctx context.Context) (int64, error)
	FindAll(ctx context.Context, page, limit int, sortClause string, filters entity.UserFilters) ([]entity.User, int64, error)
	CountSuperAdmins(ctx context.Context) (int64, error)
}

// RefreshTokenRepository defines persistence operations for RefreshToken entities.
type RefreshTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	Create(ctx context.Context, token *entity.RefreshToken) error
	Delete(ctx context.Context, id string) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteByUserIDExceptTokenHash(ctx context.Context, userID string, exceptTokenHash string) error
}

// APIKeyRepository defines persistence operations for APIKey entities.
type APIKeyRepository interface {
	FindActiveByPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error)
	FindByUserIDAndWorkspaceID(ctx context.Context, userID, workspaceID string) ([]entity.APIKey, error)
	Create(ctx context.Context, key *entity.APIKey) error
	SoftDeleteScoped(ctx context.Context, userID, workspaceID, keyID string) error
}

// PackageRepository defines persistence operations for Package entities.
type PackageRepository interface {
	FindByIDAndWorkspaceID(ctx context.Context, id, workspaceID string) (*entity.Package, error)
	FindByWorkspaceID(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	FindActiveByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.Package, error)
	Create(ctx context.Context, pkg *entity.Package) error
	Update(ctx context.Context, pkg *entity.Package) error
	BlockPackage(ctx context.Context, workspaceID, pkgID string, reason string) error
	UnblockPackage(ctx context.Context, workspaceID, pkgID string) error
	RemovePackage(ctx context.Context, workspaceID, pkgID string) error
	CountByWorkspace(ctx context.Context, workspaceID string, ecosystem *entity.Ecosystem) (int64, error)
	ExistsByWorkspaceAndName(ctx context.Context, workspaceID string, name string, ecosystem entity.Ecosystem) (bool, error)
	FindSuggestionsByWorkspaceID(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	ApprovePackage(ctx context.Context, workspaceID, pkgID string) error
	RejectPackage(ctx context.Context, workspaceID, pkgID string) error
	BulkApprovePackages(ctx context.Context, workspaceID string, pkgIDs []string) (int, error)
	BulkApproveAllSuggestions(ctx context.Context, workspaceID string) (int, error)
	FindStaleByWorkspaceID(ctx context.Context, workspaceID string, staleBefore time.Time, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
}

// ReleaseRepository defines persistence operations for Release entities.
type ReleaseRepository interface {
	FindByIDWithPackageAndWorkspace(ctx context.Context, id, workspaceID string) (*entity.Release, *entity.Package, error)
	FindByPackageIDAndWorkspace(ctx context.Context, packageID, workspaceID string, page, limit int) ([]entity.Release, int64, error)
	FindByWorkspaceIDWithDetails(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error)
	FindByPackageIDAll(ctx context.Context, packageID string) ([]entity.Release, error)
	UpdateStatus(ctx context.Context, id string, status entity.ReleaseStatus) error
	CountByWorkspaceAndStatus(ctx context.Context, workspaceID string) (*entity.PipelineStatus, error)
}

// DiffRepository defines persistence operations for Diff entities.
type DiffRepository interface {
	FindFirstByReleaseID(ctx context.Context, releaseID string) (*entity.Diff, error)
	FindByReleaseIDs(ctx context.Context, releaseIDs []string) ([]entity.Diff, error)
}

// AnalysisRepository defines persistence operations for Analysis entities.
type AnalysisRepository interface {
	FindByDiffID(ctx context.Context, diffID string) ([]entity.Analysis, error)
	FindByDiffIDs(ctx context.Context, diffIDs []string) ([]entity.Analysis, error)
}

// AlertRepository defines persistence operations for Alert entities.
type AlertRepository interface {
	FindByIDAndWorkspaceID(ctx context.Context, id, workspaceID string) (*entity.Alert, error)
	FindByIDWithPackage(ctx context.Context, id, workspaceID string) (*entity.Alert, *entity.Package, error)
	FindByWorkspaceIDWithPackage(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error)
	Create(ctx context.Context, alert *entity.Alert) error
	UpdateStatus(ctx context.Context, id, workspaceID string, status entity.AlertStatus) error
	CountByWorkspaceAndStatus(ctx context.Context, workspaceID string) (map[entity.AlertStatus]int64, error)
}

// AlertNoteRepository defines persistence operations for AlertNote entities.
type AlertNoteRepository interface {
	FindByAlertID(ctx context.Context, alertID, workspaceID string) ([]entity.AlertNote, error)
	FindByID(ctx context.Context, id, workspaceID string) (*entity.AlertNote, error)
	Create(ctx context.Context, note *entity.AlertNote) error
	Update(ctx context.Context, note *entity.AlertNote) error
	Delete(ctx context.Context, id, workspaceID string) error
}

// SettingRepository defines persistence operations for Setting entities.
type SettingRepository interface {
	FindByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.Setting, error)
	UpsertByWorkspaceAndKey(ctx context.Context, workspaceID string, key, value string) error
	// FindPlatformSettings returns settings where workspace_id IS NULL for the given keys.
	FindPlatformSettings(ctx context.Context, keys []string) ([]entity.Setting, error)
	// UpsertPlatformSetting upserts a setting where workspace_id IS NULL.
	UpsertPlatformSetting(ctx context.Context, key, value string) error
}

// AuditLogRepository defines persistence operations for AuditLog entities.
type AuditLogRepository interface {
	Create(ctx context.Context, entry *entity.AuditLog) error
	FindByWorkspaceID(ctx context.Context, workspaceID string, filters entity.AuditLogFilters, page, limit int) ([]entity.AuditLog, int64, error)
	FindAll(ctx context.Context, filters entity.AuditLogFilters, page, limit int, sortClause string) ([]entity.AuditLog, int64, error)
	FindByID(ctx context.Context, id string) (*entity.AuditLog, error)
}

// NotificationChannelRepository defines persistence operations for NotificationChannel entities.
type NotificationChannelRepository interface {
	FindByID(ctx context.Context, id string) (*entity.NotificationChannel, error)
	FindByIDAndWorkspace(ctx context.Context, id, workspaceID string) (*entity.NotificationChannel, error)
	FindByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.NotificationChannel, error)
	Create(ctx context.Context, channel *entity.NotificationChannel) error
	Update(ctx context.Context, channel *entity.NotificationChannel) error
	DeleteByIDAndWorkspace(ctx context.Context, id, workspaceID string) (int64, error)
}

// NotificationRuleRepository defines persistence operations for NotificationRule entities.
type NotificationRuleRepository interface {
	FindByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.NotificationRule, error)
	FindActiveByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.NotificationRule, error)
	Create(ctx context.Context, rule *entity.NotificationRule) error
	DeleteByIDAndWorkspace(ctx context.Context, id, workspaceID string) (int64, error)
}

// NotificationRepository defines persistence operations for Notification entities.
type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.Notification) error
	FindByUserAndWorkspace(ctx context.Context, workspaceID, userID string, onlyUnread bool) ([]entity.Notification, error)
	FindByUserAndWorkspaceIDs(ctx context.Context, workspaceIDs []string, userID string, onlyUnread bool) ([]entity.Notification, error)
	FindByID(ctx context.Context, id string) (*entity.Notification, error)
	MarkRead(ctx context.Context, id, workspaceID, userID string) (int64, error)
	MarkAllRead(ctx context.Context, workspaceID, userID string) (int64, error)
	MarkAllReadByWorkspaceIDs(ctx context.Context, workspaceIDs []string, userID string) (int64, error)
	CountUnread(ctx context.Context, workspaceID, userID string) (int64, error)
	CountUnreadByWorkspaceIDs(ctx context.Context, workspaceIDs []string, userID string) (int64, error)
	DeleteByID(ctx context.Context, id, workspaceID, userID string) (int64, error)
	DeleteAll(ctx context.Context, workspaceID, userID string) (int64, error)
	DeleteAllByWorkspaceIDs(ctx context.Context, workspaceIDs []string, userID string) (int64, error)
	DeleteBatch(ctx context.Context, ids []string, workspaceID, userID string) (int64, error)
	DeleteBatchByWorkspaceIDs(ctx context.Context, ids []string, workspaceIDs []string, userID string) (int64, error)
}

// PasswordResetTokenRepository defines persistence operations for PasswordResetToken entities.
type PasswordResetTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordResetToken, error)
	Create(ctx context.Context, token *entity.PasswordResetToken) error
	MarkUsed(ctx context.Context, id string) error
	DeleteExpiredByUserID(ctx context.Context, userID string) error
}

// EmailVerificationTokenRepository defines persistence operations for EmailVerificationToken entities.
type EmailVerificationTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.EmailVerificationToken, error)
	Create(ctx context.Context, token *entity.EmailVerificationToken) error
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
}

// SessionRepository defines persistence operations for Session entities.
type SessionRepository interface {
	FindByID(ctx context.Context, id string) (*entity.Session, error)
	FindByUserID(ctx context.Context, userID string) ([]entity.Session, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.Session, error)
	Create(ctx context.Context, session *entity.Session) error
	UpdateLastActive(ctx context.Context, id string, lastActive time.Time) error
	UpdateTokenHash(ctx context.Context, id string, tokenHash string) error
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) (int64, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	DeleteOldestByUserID(ctx context.Context, userID string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteByUserIDExceptTokenHash(ctx context.Context, userID string, exceptTokenHash string) error
}

// DashboardRepository defines read-only aggregation queries for the dashboard.
// All queries are scoped to a workspace via workspaceID.
type DashboardRepository interface {
	GetStats(ctx context.Context, workspaceID string) (*entity.DashboardStats, error)
	GetReleaseActivity(ctx context.Context, workspaceID string, from, to time.Time) ([]entity.ReleaseActivityPoint, error)
	GetClassificationDistribution(ctx context.Context, workspaceID string, from, to time.Time) ([]entity.ClassificationCount, error)
	GetBaselineCount(ctx context.Context, workspaceID string, from, to time.Time) (int64, error)
	GetEcosystemDistribution(ctx context.Context, workspaceID string) ([]entity.EcosystemCount, error)
	GetAlertsBySeverity(ctx context.Context, workspaceID string, from, to time.Time) ([]entity.AlertSeverityCount, error)
	GetReleaseStatusDistribution(ctx context.Context, workspaceID string, from, to time.Time) ([]entity.ReleaseStatusCount, error)
	GetUnanalyzedDiffIDs(ctx context.Context, workspaceID string) ([]string, error)
}

// ReleaseHashRepository defines persistence operations for ReleaseHash entities.
type ReleaseHashRepository interface {
	CreateBatch(ctx context.Context, hashes []entity.ReleaseHash) error
	FindByReleaseID(ctx context.Context, releaseID string) ([]entity.ReleaseHash, error)
	CountByReleaseID(ctx context.Context, releaseID string) (int64, error)
	CountByReleaseIDs(ctx context.Context, releaseIDs []string) (map[string]int64, error)
}

// --- Service Interfaces ---

// PackageService defines the business logic operations for package lifecycle management.
type PackageService interface {
	ListPackages(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	GetPackage(ctx context.Context, workspaceID, pkgID string) (*entity.Package, error)
	CreatePackage(ctx context.Context, workspaceID string, name string, ecosystem entity.Ecosystem) (*entity.Package, error)
	ImportPackages(ctx context.Context, workspaceID string, entries []entity.ImportEntry) (*entity.ImportResult, error)
	BlockPackage(ctx context.Context, workspaceID, pkgID string, reason string) (*entity.Package, error)
	UnblockPackage(ctx context.Context, workspaceID, pkgID string) (*entity.Package, error)
	RemovePackage(ctx context.Context, workspaceID, pkgID string) error
	ApprovePackage(ctx context.Context, workspaceID, pkgID string) (*entity.Package, error)
	RejectPackage(ctx context.Context, workspaceID, pkgID string) error
	BulkApprovePackages(ctx context.Context, workspaceID string, pkgIDs []string) (int, error)
	BulkApproveAllSuggestions(ctx context.Context, workspaceID string) (int, error)
	ListSuggestions(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	ListStalePackages(ctx context.Context, workspaceID string, staleBefore time.Time, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	CountPackages(ctx context.Context, workspaceID string) (int64, error)
}

// AlertService defines the business logic operations for alerts.
type AlertService interface {
	ListAlerts(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error)
	GetAlert(ctx context.Context, workspaceID, alertID string) (*entity.Alert, *entity.Package, error)
	UpdateAlertStatus(ctx context.Context, workspaceID, alertID string, status entity.AlertStatus) (*entity.Alert, error)
}

// AlertNoteService defines the business logic operations for alert notes.
type AlertNoteService interface {
	ListByAlert(ctx context.Context, workspaceID, alertID string) ([]entity.AlertNote, error)
	Create(ctx context.Context, workspaceID, alertID, userID string, content string) (*entity.AlertNote, error)
	Update(ctx context.Context, workspaceID, alertID, noteID, userID string, content string) (*entity.AlertNote, error)
	Delete(ctx context.Context, workspaceID, alertID, noteID, userID string) error
}

// ReleaseService defines the business logic operations for releases.
type ReleaseService interface {
	ListByPackage(ctx context.Context, workspaceID, packageID string, page, limit int) ([]entity.Release, int64, error)
	GetRelease(ctx context.Context, workspaceID, releaseID string) (*entity.ReleaseDetail, error)
	ReanalyzeRelease(ctx context.Context, workspaceID, releaseID string) (string, string, error)
	GetAnalysisHistory(ctx context.Context, workspaceID, packageID string) ([]entity.AnalysisHistoryEntry, error)
	GetPipelineStatus(ctx context.Context, workspaceID string) (*entity.PipelineStatus, error)
}

// SettingService defines the business logic operations for settings.
type SettingService interface {
	GetSettings(ctx context.Context, workspaceID string) (map[string]string, error)
	UpdateSettings(ctx context.Context, workspaceID string, settings map[string]string) (map[string]string, error)
}

// DashboardService defines the business logic operations for the dashboard.
type DashboardService interface {
	GetStats(ctx context.Context, workspaceID string) (*entity.DashboardStats, error)
	GetRecentReleases(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error)
	GetChartData(ctx context.Context, workspaceID string, from, to time.Time) (*entity.ChartData, error)
	ReanalyzeAll(ctx context.Context, workspaceID string) (int, error)
}

// HealthService defines the business logic operations for health checks.
type HealthService interface {
	HealthCheck(ctx context.Context) *entity.HealthResponse
	ReadinessCheck(ctx context.Context) *entity.ReadinessResponse
}

// QueueEnqueuer defines the interface for enqueueing analysis jobs.
// Implementations live in repo/queue or pkg/queue.
type QueueEnqueuer interface {
	Enqueue(ctx context.Context, jobType string, workspaceID, referenceID string, metadata map[string]string) (string, error)
}

// AuditLogger defines the interface for audit logging used by usecase layer.
// The implementation lives in the outer layer (repo or controller).
type AuditLogger interface {
	LogAction(ctx context.Context, action, resource string, resourceID string, details string)
	LogActionWithUser(ctx context.Context, userID, userEmail, action, resource, resourceID, details string)
}

// Registry defines the interface for interacting with a package registry.
type Registry interface {
	GetPackage(ctx context.Context, name string) (*entity.RegistryPackageInfo, error)
	GetTopPackages(ctx context.Context, limit int) ([]entity.PackageRanking, error)
	DownloadTarball(ctx context.Context, url string) (string, error)
	Name() string
}

// AuthEmailSender defines the interface for sending authentication-related emails.
// Implementations live in the outer layer (pkg/mailer).
type AuthEmailSender interface {
	SendPasswordResetEmail(ctx context.Context, email, token string) error
	SendVerificationEmail(ctx context.Context, email, token string) error
}

// InvitationEmailSender defines the interface for sending workspace invitation emails.
// Implementations live in the outer layer (pkg/mailer).
type InvitationEmailSender interface {
	SendInvitationEmail(ctx context.Context, email, token, workspaceName, inviterEmail string) error
}

// UserEmailResolver resolves users by ID or email.
// Used by the RBAC service to display inviter identity in invitation emails
// and to look up invited users for targeted notifications.
type UserEmailResolver interface {
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
}

// UserWorkspaceLister returns the workspace IDs a user belongs to.
// Used by the notification service to scope cross-workspace queries.
type UserWorkspaceLister interface {
	FindWorkspaceIDsByUserID(ctx context.Context, userID string) ([]string, error)
}

// NotificationDispatcher defines the interface for dispatching system notifications.
// Implementations route notifications to in-app storage and external channels
// based on the workspace's configured notification rules.
type NotificationDispatcher interface {
	Dispatch(ctx context.Context, workspaceID string, severity, title, message string)
	DispatchEvent(ctx context.Context, workspaceID string, evt entity.NotificationEvent)
}

// RBACRepository defines the persistence operations needed by the RBAC service.
// Defined here (in the shared usecase package) so that repo implementations can
// reference it without creating import cycles.
type RBACRepository interface {
	// Workspace operations
	CountWorkspacesBySlug(ctx context.Context, slug string, excludeID *string) (int64, error)
	CreateWorkspace(ctx context.Context, ws *entity.Workspace) error
	FindWorkspaceByID(ctx context.Context, id string) (*entity.Workspace, error)
	UpdateWorkspace(ctx context.Context, ws *entity.Workspace) error
	SoftDeleteWorkspace(ctx context.Context, id string) error
	FindWorkspacesByUserID(ctx context.Context, userID string, params entity.WorkspaceListParams) (*entity.WorkspaceListResult, error)

	// Role & Permission operations
	FindAllPermissions(ctx context.Context) ([]entity.Permission, error)
	CreateRole(ctx context.Context, role *entity.Role) error
	FindRoleByIDAndWorkspace(ctx context.Context, roleID, workspaceID string) (*entity.Role, error)
	FindRolesByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.Role, error)

	// Member operations
	CreateMember(ctx context.Context, member *entity.WorkspaceMember) error
	FindMembersByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.WorkspaceMember, error)
	FindMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (*entity.WorkspaceMember, error)
	CountMembersByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (int64, error)
	UpdateMember(ctx context.Context, member *entity.WorkspaceMember) error
	DeleteMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID string) error

	// Invitation operations
	CreateInvitation(ctx context.Context, invitation *entity.Invitation) error
	FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*entity.Invitation, error)
	UpdateInvitation(ctx context.Context, invitation *entity.Invitation) error
	FindPendingInvitations(ctx context.Context, workspaceID string) ([]entity.Invitation, error)
	FindInvitationByID(ctx context.Context, workspaceID, invitationID string) (*entity.Invitation, error)
	FindInvitationByIDGlobal(ctx context.Context, invitationID string) (*entity.Invitation, error)
	FindPendingInvitationsByEmail(ctx context.Context, email string) ([]entity.Invitation, error)
	DeletePendingInvitation(ctx context.Context, workspaceID, invitationID string) error

	// Permission check
	CheckUserPermission(ctx context.Context, userID, workspaceID string, resource, action string) (bool, error)
	CheckRolePermission(ctx context.Context, workspaceID string, roleName, resource, action string) (bool, error)

	// SeedPermission ensures a permission exists (idempotent).
	SeedPermission(ctx context.Context, perm entity.Permission) error

	// Transaction support: runs fn within a transaction.
	WithTransaction(ctx context.Context, fn func(tx RBACRepository) error) error
}

// RateLimiter defines a simple cooldown-based rate limiter.
// Implementations live in the outer layer (repo/cache).
type RateLimiter interface {
	// Allow checks if the given key is allowed. If allowed, it sets a cooldown
	// for the specified duration and returns true. If still in cooldown, returns false.
	Allow(ctx context.Context, key string, cooldown time.Duration) (bool, error)
}

// TokenClaims represents the essential claims extracted from a validated token.
type TokenClaims struct {
	UserID    string
	Email     string
	TokenType string
}

// TokenProvider handles JWT token generation and validation.
// Implementations live in the outer layer (pkg/jwt or similar).
type TokenProvider interface {
	// GenerateAccessToken creates a signed JWT access token.
	GenerateAccessToken(userID, email string, duration time.Duration) (string, error)
	// GenerateRefreshToken creates a cryptographically random refresh token string.
	GenerateRefreshToken() (string, error)
	// ValidateAccessToken parses and validates a JWT access token, returning its claims.
	ValidateAccessToken(tokenString string) (*TokenClaims, error)
}

// PasswordHasher handles password hashing and comparison.
// Implementations live in the outer layer (pkg/hasher or similar).
type PasswordHasher interface {
	// Hash produces a secure hash of the given password.
	Hash(password string) (string, error)
	// Compare checks whether the given password matches the stored hash.
	// Returns nil on match, error otherwise.
	Compare(hash, password string) error
}

// EmailNotificationSender sends notification emails via SMTP.
// Implementations live in the outer layer (pkg/mailer or similar).
type EmailNotificationSender interface {
	// SendNotificationEmail sends using the globally configured SMTP (for system/digest emails).
	SendNotificationEmail(from string, recipients []string, subject, body string) error
	// SendChannelEmail sends using per-channel SMTP config (for notification channels).
	SendChannelEmail(host, port, username, password, from string, recipients []string, subject, body string) error
}

// WebhookSender sends HTTP webhook notifications.
// Implementations live in the outer layer (pkg/ or repo/).
type WebhookSender interface {
	// SendWebhook posts a JSON payload to the given URL.
	// If signingSecret is non-empty, it adds an HMAC-SHA256 signature header.
	SendWebhook(url string, payload []byte, signingSecret string) error
}

// SlackSender sends Slack webhook notifications.
// Implementations live in the outer layer (pkg/ or repo/).
type SlackSender interface {
	// SendSlack posts a text message to a Slack incoming webhook URL.
	SendSlack(webhookURL string, text string) error
}

// UserAccountCreator creates user accounts. Used by the RBAC service
// for the "Add User to Workspace" flow without importing the auth package.
type UserAccountCreator interface {
	CreateUserWithoutPassword(ctx context.Context, email, firstName, lastName string) (*entity.User, error)
	CreateUserWithPassword(ctx context.Context, email, firstName, lastName, password string) (*entity.User, error)
	CreateSSOUser(ctx context.Context, email, firstName, lastName string, provider entity.SSOProvider) (*entity.User, error)
}

// PasswordResetInitiator triggers a password-set email for newly created users.
type PasswordResetInitiator interface {
	InitiatePasswordReset(ctx context.Context, email string) error
}

// SSOConfigRepository defines persistence operations for platform-level SSO configurations.
type SSOConfigRepository interface {
	FindAll(ctx context.Context) ([]entity.SSOConfig, error)
	FindByID(ctx context.Context, id string) (*entity.SSOConfig, error)
	FindEnabled(ctx context.Context) ([]entity.SSOConfig, error)
	FindBySAMLEntityID(ctx context.Context, entityID string) (*entity.SSOConfig, error)
	FindByProvider(ctx context.Context, provider entity.SSOProvider) (*entity.SSOConfig, error)
	Create(ctx context.Context, config *entity.SSOConfig) error
	Update(ctx context.Context, config *entity.SSOConfig) error
	Delete(ctx context.Context, id string) error
}

// UserIdentityRepository defines persistence operations for external user identities.
type UserIdentityRepository interface {
	FindByUserID(ctx context.Context, userID string) ([]entity.UserIdentity, error)
	FindByProviderAndProviderUserID(ctx context.Context, provider entity.AuthProvider, providerUserID string) (*entity.UserIdentity, error)
	Create(ctx context.Context, identity *entity.UserIdentity) error
	Update(ctx context.Context, identity *entity.UserIdentity) error
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteByProvider(ctx context.Context, provider entity.AuthProvider) error
}

// SSOStateRepository defines persistence operations for SSO state parameters (CSRF).
type SSOStateRepository interface {
	Create(ctx context.Context, state *entity.SSOState) error
	FindByState(ctx context.Context, state string) (*entity.SSOState, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// AuthSessionCreator issues JWT sessions. Used by SSO service to delegate JWT issuance.
type AuthSessionCreator interface {
	CreateSessionForUser(ctx context.Context, userID, provider, ipAddress, userAgent string) (*TokenPair, error)
}

// TokenPair holds an access/refresh token pair returned after authentication.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// SAMLProvider handles SAML protocol operations.
type SAMLProvider interface {
	GenerateAuthnRequest(config *entity.SSOConfig) (redirectURL string, requestID string, err error)
	ValidateResponse(config *entity.SSOConfig, samlResponse string, requestID string) (*SAMLAssertion, error)
	GenerateMetadata(config *entity.SSOConfig) ([]byte, error)
	ParseLogoutRequest(samlRequest string) (nameID string, issuer string, err error)
	VerifyLogoutSignature(samlRequest, signature, sigAlg, pemCertificate string) error
}

// SAMLAssertion represents extracted SAML assertion data.
// Placed in contracts.go as a return type of SAMLProvider (same pattern as TokenClaims).
type SAMLAssertion struct {
	NameID       string
	Email        string
	FirstName    string
	LastName     string
	Groups       []string
	SessionIndex string
}

// SAMLMetadataFetcher fetches and parses SAML metadata from a URL.
type SAMLMetadataFetcher interface {
	FetchAndParse(ctx context.Context, metadataURL string) (*SAMLMetadataInfo, error)
}

// SAMLMetadataInfo holds the extracted fields from a SAML IdP metadata document.
type SAMLMetadataInfo struct {
	EntityID    string
	SSOURL      string
	SloURL      string
	Certificate string
}

// SSOTestResult holds the outcome of an SSO config connectivity test.
type SSOTestResult struct {
	IdpEntityID string
	IdpSSOURL   string
}

// OAuthTokenExchanger exchanges authorization codes for user info.
type OAuthTokenExchanger interface {
	ExchangeGoogle(ctx context.Context, config *entity.SSOConfig, code, codeVerifier string) (*OAuthUserInfo, error)
	ExchangeGitHub(ctx context.Context, config *entity.SSOConfig, code, codeVerifier string) (*OAuthUserInfo, error)
}

// OAuthUserInfo represents normalized user info from an OAuth provider.
type OAuthUserInfo struct {
	ProviderUserID string
	Email          string
	FirstName      string
	LastName       string
	AvatarURL      string
	Organizations  []string // GitHub orgs
	HostedDomain   string   // Google Workspace domain
}

// MonitoringTrigger allows triggering an immediate monitoring cycle for a workspace.
type MonitoringTrigger interface {
	TriggerMonitoring(workspaceID string)
}

// DigestRepository defines the persistence operations needed by the digest scheduler.
type DigestRepository interface {
	FindEnabledDigestConfigs(ctx context.Context) ([]entity.DigestOrgConfig, error)
	CountAlertsSince(ctx context.Context, workspaceID string, since time.Time) (int64, error)
	CountPackagesAnalyzedSince(ctx context.Context, workspaceID string, since time.Time) (int64, error)
	GetClassificationBreakdownSince(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error)
	GetTopAlertsSince(ctx context.Context, workspaceID string, since time.Time, limit int) ([]entity.DigestTopAlert, error)
}
