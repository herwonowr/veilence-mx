// Package usecase defines the business logic interfaces for the application.
// All repository and service interfaces are defined here — the consumer defines the contract.
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
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
}

// RefreshTokenRepository defines persistence operations for RefreshToken entities.
type RefreshTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	Create(ctx context.Context, token *entity.RefreshToken) error
	Delete(ctx context.Context, id uint) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID uint) error
	DeleteByUserIDExceptTokenHash(ctx context.Context, userID uint, exceptTokenHash string) error
}

// DashboardRepository
// APIKeyRepository defines persistence operations for APIKey entities.
type APIKeyRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.APIKey, error)
	FindActiveByPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error)
	FindByUserID(ctx context.Context, userID uint) ([]entity.APIKey, error)
	FindByUserIDAndWorkspaceID(ctx context.Context, userID, workspaceID uint) ([]entity.APIKey, error)
	Create(ctx context.Context, key *entity.APIKey) error
	Update(ctx context.Context, key *entity.APIKey) error
	SoftDelete(ctx context.Context, userID, keyID uint) error
	SoftDeleteScoped(ctx context.Context, userID, workspaceID, keyID uint) error
}

// PackageRepository defines persistence operations for Package entities.
type PackageRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Package, error)
	FindByIDAndWorkspaceID(ctx context.Context, id, workspaceID uint) (*entity.Package, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	FindActiveByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.Package, error)
	FindByWorkspaceAndName(ctx context.Context, workspaceID uint, name string, ecosystem entity.Ecosystem) (*entity.Package, error)
	Create(ctx context.Context, pkg *entity.Package) error
	Update(ctx context.Context, pkg *entity.Package) error
	BlockPackage(ctx context.Context, workspaceID, pkgID uint, reason string) error
	UnblockPackage(ctx context.Context, workspaceID, pkgID uint) error
	RemovePackage(ctx context.Context, workspaceID, pkgID uint) error
	CountByWorkspace(ctx context.Context, workspaceID uint, ecosystem *entity.Ecosystem) (int64, error)
	ExistsByWorkspaceAndName(ctx context.Context, workspaceID uint, name string, ecosystem entity.Ecosystem) (bool, error)
	FindSuggestionsByWorkspaceID(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	ApprovePackage(ctx context.Context, workspaceID, pkgID uint) error
	RejectPackage(ctx context.Context, workspaceID, pkgID uint) error
	BulkApprovePackages(ctx context.Context, workspaceID uint, pkgIDs []uint) (int, error)
	BulkApproveAllSuggestions(ctx context.Context, workspaceID uint) (int, error)
	UpdateDownloadCounts(ctx context.Context, workspaceID uint, updates []entity.PackageDownloadUpdate) error
	FindStaleByWorkspaceID(ctx context.Context, workspaceID uint, staleBefore time.Time) ([]entity.Package, error)
	RemoveStaleByWorkspaceID(ctx context.Context, workspaceID uint, staleBefore time.Time) (int, error)
}

// ReleaseRepository defines persistence operations for Release entities.
type ReleaseRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Release, error)
	FindByIDWithPackage(ctx context.Context, id uint) (*entity.Release, *entity.Package, error)
	FindByIDWithPackageAndWorkspace(ctx context.Context, id, workspaceID uint) (*entity.Release, *entity.Package, error)
	FindByPackageID(ctx context.Context, packageID uint, page, limit int) ([]entity.Release, int64, error)
	FindByPackageIDAndWorkspace(ctx context.Context, packageID, workspaceID uint, page, limit int) ([]entity.Release, int64, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.Release, int64, error)
	FindByWorkspaceIDWithDetails(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error)
	FindByPackageIDAll(ctx context.Context, packageID uint) ([]entity.Release, error)
	UpdateStatus(ctx context.Context, id uint, status entity.ReleaseStatus) error
	Create(ctx context.Context, release *entity.Release) error
	Update(ctx context.Context, release *entity.Release) error
}

// DiffRepository defines persistence operations for Diff entities.
type DiffRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Diff, error)
	FindByIDAndWorkspace(ctx context.Context, id, workspaceID uint) (*entity.Diff, error)
	FindByReleaseID(ctx context.Context, releaseID uint) ([]entity.Diff, error)
	FindByReleaseIDAndWorkspace(ctx context.Context, releaseID, workspaceID uint) ([]entity.Diff, error)
	FindFirstByReleaseID(ctx context.Context, releaseID uint) (*entity.Diff, error)
	FindByReleaseIDs(ctx context.Context, releaseIDs []uint) ([]entity.Diff, error)
	Create(ctx context.Context, diff *entity.Diff) error
}

// AnalysisRepository defines persistence operations for Analysis entities.
type AnalysisRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Analysis, error)
	FindByDiffID(ctx context.Context, diffID uint) ([]entity.Analysis, error)
	FindByDiffIDs(ctx context.Context, diffIDs []uint) ([]entity.Analysis, error)
	Create(ctx context.Context, analysis *entity.Analysis) error
	CountByDiffID(ctx context.Context, diffID uint) (int64, error)
}

// AlertRepository defines persistence operations for Alert entities.
type AlertRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Alert, error)
	FindByIDAndWorkspaceID(ctx context.Context, id, workspaceID uint) (*entity.Alert, error)
	FindByIDWithPackage(ctx context.Context, id, workspaceID uint) (*entity.Alert, *entity.Package, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.Alert, int64, error)
	FindByWorkspaceIDWithPackage(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error)
	Create(ctx context.Context, alert *entity.Alert) error
	Update(ctx context.Context, alert *entity.Alert) error
	UpdateStatus(ctx context.Context, id, workspaceID uint, status entity.AlertStatus) error
	CountByWorkspaceAndStatus(ctx context.Context, workspaceID uint) (map[entity.AlertStatus]int64, error)
}

// AlertNoteRepository defines persistence operations for AlertNote entities.
type AlertNoteRepository interface {
	FindByAlertID(ctx context.Context, alertID, workspaceID uint) ([]entity.AlertNote, error)
	FindByID(ctx context.Context, id, workspaceID uint) (*entity.AlertNote, error)
	Create(ctx context.Context, note *entity.AlertNote) error
	Update(ctx context.Context, note *entity.AlertNote) error
	Delete(ctx context.Context, id, workspaceID uint) error
}

// SettingRepository defines persistence operations for Setting entities.
type SettingRepository interface {
	FindByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.Setting, error)
	FindByKey(ctx context.Context, workspaceID uint, key string) (*entity.Setting, error)
	Upsert(ctx context.Context, setting *entity.Setting) error
	UpsertByWorkspaceAndKey(ctx context.Context, workspaceID uint, key, value string) error
	FindOrCreateByKey(ctx context.Context, key, defaultValue string) (*entity.Setting, error)
}

// WorkspaceRepository defines persistence operations for Workspace entities.
type WorkspaceRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Workspace, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Workspace, error)
	CountBySlug(ctx context.Context, slug string, excludeID *uint) (int64, error)
	Create(ctx context.Context, org *entity.Workspace) error
	Update(ctx context.Context, org *entity.Workspace) error
	SoftDelete(ctx context.Context, id uint) error
	FindByUserID(ctx context.Context, userID uint) ([]entity.Workspace, error)
}

// WorkspaceMemberRepository defines persistence operations for WorkspaceMember entities.
type WorkspaceMemberRepository interface {
	FindByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.WorkspaceMember, error)
	FindByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) (*entity.WorkspaceMember, error)
	CountByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) (int64, error)
	Create(ctx context.Context, member *entity.WorkspaceMember) error
	Update(ctx context.Context, member *entity.WorkspaceMember) error
	DeleteByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) error
}

// RoleRepository defines persistence operations for Role entities.
type RoleRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Role, error)
	FindByIDAndWorkspace(ctx context.Context, id, workspaceID uint) (*entity.Role, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.Role, error)
	Create(ctx context.Context, role *entity.Role) error
}

// PermissionRepository defines persistence operations for Permission entities.
type PermissionRepository interface {
	FindAll(ctx context.Context) ([]entity.Permission, error)
	FindOrCreate(ctx context.Context, perm *entity.Permission) error
	CheckUserPermission(ctx context.Context, userID, workspaceID uint, resource, action string) (bool, error)
}

// InvitationRepository defines persistence operations for Invitation entities.
type InvitationRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.Invitation, error)
	Create(ctx context.Context, invitation *entity.Invitation) error
	Update(ctx context.Context, invitation *entity.Invitation) error
}

// AuditLogRepository defines persistence operations for AuditLog entities.
type AuditLogRepository interface {
	Create(ctx context.Context, entry *entity.AuditLog) error
	FindByWorkspaceID(ctx context.Context, workspaceID uint, filters entity.AuditLogFilters, page, limit int) ([]entity.AuditLog, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.AuditLog, error)
}

// NotificationChannelRepository defines persistence operations for NotificationChannel entities.
type NotificationChannelRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.NotificationChannel, error)
	FindByIDAndWorkspace(ctx context.Context, id, workspaceID uint) (*entity.NotificationChannel, error)
	FindByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.NotificationChannel, error)
	Create(ctx context.Context, channel *entity.NotificationChannel) error
	Update(ctx context.Context, channel *entity.NotificationChannel) error
	DeleteByIDAndWorkspace(ctx context.Context, id, workspaceID uint) (int64, error)
}

// NotificationRuleRepository defines persistence operations for NotificationRule entities.
type NotificationRuleRepository interface {
	FindByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.NotificationRule, error)
	FindActiveByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.NotificationRule, error)
	Create(ctx context.Context, rule *entity.NotificationRule) error
	DeleteByIDAndWorkspace(ctx context.Context, id, workspaceID uint) (int64, error)
}

// NotificationRepository defines persistence operations for Notification entities.
type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.Notification) error
	FindByUserAndWorkspace(ctx context.Context, workspaceID, userID uint, onlyUnread bool) ([]entity.Notification, error)
	MarkRead(ctx context.Context, id, userID uint) (int64, error)
	MarkAllRead(ctx context.Context, workspaceID, userID uint) (int64, error)
	CountUnread(ctx context.Context, workspaceID, userID uint) (int64, error)
	DeleteByID(ctx context.Context, id, workspaceID, userID uint) (int64, error)
	DeleteAll(ctx context.Context, workspaceID, userID uint) (int64, error)
	DeleteBatch(ctx context.Context, ids []uint, workspaceID, userID uint) (int64, error)
}

// PasswordResetTokenRepository defines persistence operations for PasswordResetToken entities.
type PasswordResetTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordResetToken, error)
	Create(ctx context.Context, token *entity.PasswordResetToken) error
	MarkUsed(ctx context.Context, id uint) error
	DeleteExpiredByUserID(ctx context.Context, userID uint) error
}

// EmailVerificationTokenRepository defines persistence operations for EmailVerificationToken entities.
type EmailVerificationTokenRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.EmailVerificationToken, error)
	Create(ctx context.Context, token *entity.EmailVerificationToken) error
	Delete(ctx context.Context, id uint) error
	DeleteByUserID(ctx context.Context, userID uint) error
}

// SessionRepository defines persistence operations for Session entities.
type SessionRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Session, error)
	FindByUserID(ctx context.Context, userID uint) ([]entity.Session, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (*entity.Session, error)
	Create(ctx context.Context, session *entity.Session) error
	UpdateLastActive(ctx context.Context, id uint, lastActive time.Time) error
	UpdateTokenHash(ctx context.Context, id uint, tokenHash string) error
	Delete(ctx context.Context, id uint) error
	DeleteExpired(ctx context.Context) (int64, error)
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	DeleteOldestByUserID(ctx context.Context, userID uint) error
	DeleteByUserID(ctx context.Context, userID uint) error
	DeleteByUserIDExceptTokenHash(ctx context.Context, userID uint, exceptTokenHash string) error
}

// DashboardRepository defines read-only aggregation queries for the dashboard.
// All queries are scoped to a workspace via workspaceID.
type DashboardRepository interface {
	GetStats(ctx context.Context, workspaceID uint) (*entity.DashboardStats, error)
	GetReleaseActivity(ctx context.Context, workspaceID uint, from, to time.Time) ([]entity.ReleaseActivityPoint, error)
	GetClassificationDistribution(ctx context.Context, workspaceID uint, from, to time.Time) ([]entity.ClassificationCount, error)
	GetBaselineCount(ctx context.Context, workspaceID uint, from, to time.Time) (int64, error)
	GetEcosystemDistribution(ctx context.Context, workspaceID uint) ([]entity.EcosystemCount, error)
	GetAlertsBySeverity(ctx context.Context, workspaceID uint, from, to time.Time) ([]entity.AlertSeverityCount, error)
	GetReleaseStatusDistribution(ctx context.Context, workspaceID uint, from, to time.Time) ([]entity.ReleaseStatusCount, error)
	GetUnanalyzedDiffIDs(ctx context.Context, workspaceID uint) ([]uint, error)
}

// --- Service Interfaces ---

// PackageService defines the business logic operations for package lifecycle management.
type PackageService interface {
	ListPackages(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	GetPackage(ctx context.Context, workspaceID, pkgID uint) (*entity.Package, error)
	CreatePackage(ctx context.Context, workspaceID uint, name string, ecosystem entity.Ecosystem) (*entity.Package, error)
	ImportPackages(ctx context.Context, workspaceID uint, entries []entity.ImportEntry) (*entity.ImportResult, error)
	BlockPackage(ctx context.Context, workspaceID, pkgID uint, reason string) (*entity.Package, error)
	UnblockPackage(ctx context.Context, workspaceID, pkgID uint) (*entity.Package, error)
	RemovePackage(ctx context.Context, workspaceID, pkgID uint) error
	ApprovePackage(ctx context.Context, workspaceID, pkgID uint) (*entity.Package, error)
	RejectPackage(ctx context.Context, workspaceID, pkgID uint) error
	BulkApprovePackages(ctx context.Context, workspaceID uint, pkgIDs []uint) (int, error)
	BulkApproveAllSuggestions(ctx context.Context, workspaceID uint) (int, error)
	ListSuggestions(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	ListStalePackages(ctx context.Context, workspaceID uint, staleBefore time.Time) ([]entity.Package, error)
	RemoveStalePackages(ctx context.Context, workspaceID uint, months int) (int, error)
}

// AlertService defines the business logic operations for alerts.
type AlertService interface {
	ListAlerts(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error)
	GetAlert(ctx context.Context, workspaceID, alertID uint) (*entity.Alert, *entity.Package, error)
	UpdateAlertStatus(ctx context.Context, workspaceID, alertID uint, status entity.AlertStatus) (*entity.Alert, error)
}

// AlertNoteService defines the business logic operations for alert notes.
type AlertNoteService interface {
	ListByAlert(ctx context.Context, workspaceID, alertID uint) ([]entity.AlertNote, error)
	Create(ctx context.Context, workspaceID, alertID, userID uint, content string) (*entity.AlertNote, error)
	Update(ctx context.Context, workspaceID, alertID, noteID, userID uint, content string) (*entity.AlertNote, error)
	Delete(ctx context.Context, workspaceID, alertID, noteID, userID uint) error
}

// ReleaseService defines the business logic operations for releases.
type ReleaseService interface {
	ListByPackage(ctx context.Context, workspaceID, packageID uint, page, limit int) ([]entity.Release, int64, error)
	GetRelease(ctx context.Context, workspaceID, releaseID uint) (*entity.ReleaseDetail, error)
	ReanalyzeRelease(ctx context.Context, workspaceID, releaseID uint) (string, string, error)
	GetAnalysisHistory(ctx context.Context, workspaceID, packageID uint) ([]entity.AnalysisHistoryEntry, error)
}

// SettingService defines the business logic operations for settings.
type SettingService interface {
	GetSettings(ctx context.Context, workspaceID uint) (map[string]string, error)
	UpdateSettings(ctx context.Context, workspaceID uint, settings map[string]string) (map[string]string, error)
}

// DashboardService defines the business logic operations for the dashboard.
type DashboardService interface {
	GetStats(ctx context.Context, workspaceID uint) (*entity.DashboardStats, error)
	GetRecentReleases(ctx context.Context, workspaceID uint, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error)
	GetChartData(ctx context.Context, workspaceID uint, from, to time.Time) (*entity.ChartData, error)
	ReanalyzeAll(ctx context.Context, workspaceID uint) (int, error)
}

// HealthService defines the business logic operations for health checks.
type HealthService interface {
	HealthCheck(ctx context.Context) *entity.HealthResponse
	ReadinessCheck(ctx context.Context) *entity.ReadinessResponse
}

// QueueEnqueuer defines the interface for enqueueing analysis jobs.
// Implementations live in repo/queue or pkg/queue.
type QueueEnqueuer interface {
	Enqueue(ctx context.Context, jobType string, referenceID uint) (string, error)
}

// AuditLogger defines the interface for audit logging used by usecase layer.
// The implementation lives in the outer layer (repo or controller).
type AuditLogger interface {
	LogAction(ctx context.Context, action, resource string, resourceID uint, details string)
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

// SettingGetter defines a minimal read-only interface for retrieving a single
// setting value. Used by the auth service to check settings (e.g. email
// verification toggle) without depending on the full SettingRepository.
type SettingGetter interface {
	GetSettingValue(ctx context.Context, workspaceID uint, key string) (string, error)
	IsSettingEnabledForAnyWorkspace(ctx context.Context, userID uint, key string) (bool, error)
}

// NotificationDispatcher defines the interface for dispatching system notifications.
// Implementations route notifications to in-app storage and external channels
// based on the workspace's configured notification rules.
type NotificationDispatcher interface {
	Dispatch(ctx context.Context, workspaceID uint, severity, title, message string)
	DispatchEvent(ctx context.Context, workspaceID uint, evt entity.NotificationEvent)
}

// RBACRepository defines the persistence operations needed by the RBAC service.
// Defined here (in the shared usecase package) so that repo implementations can
// reference it without creating import cycles.
type RBACRepository interface {
	// Workspace operations
	CountWorkspacesBySlug(ctx context.Context, slug string, excludeID *uint) (int64, error)
	CreateWorkspace(ctx context.Context, ws *entity.Workspace) error
	FindWorkspaceByID(ctx context.Context, id uint) (*entity.Workspace, error)
	UpdateWorkspace(ctx context.Context, ws *entity.Workspace) error
	SoftDeleteWorkspace(ctx context.Context, id uint) error
	FindWorkspacesByUserID(ctx context.Context, userID uint) ([]entity.Workspace, error)

	// Role & Permission operations
	FindAllPermissions(ctx context.Context) ([]entity.Permission, error)
	CreateRole(ctx context.Context, role *entity.Role) error
	FindRoleByIDAndWorkspace(ctx context.Context, roleID, workspaceID uint) (*entity.Role, error)
	FindRolesByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.Role, error)

	// Member operations
	CreateMember(ctx context.Context, member *entity.WorkspaceMember) error
	FindMembersByWorkspaceID(ctx context.Context, workspaceID uint) ([]entity.WorkspaceMember, error)
	FindMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) (*entity.WorkspaceMember, error)
	CountMembersByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) (int64, error)
	UpdateMember(ctx context.Context, member *entity.WorkspaceMember) error
	DeleteMemberByUserAndWorkspace(ctx context.Context, userID, workspaceID uint) error

	// Invitation operations
	CreateInvitation(ctx context.Context, invitation *entity.Invitation) error
	FindInvitationByTokenHash(ctx context.Context, tokenHash string) (*entity.Invitation, error)
	UpdateInvitation(ctx context.Context, invitation *entity.Invitation) error
	FindPendingInvitations(ctx context.Context, workspaceID uint) ([]entity.Invitation, error)
	DeletePendingInvitation(ctx context.Context, workspaceID, invitationID uint) error

	// Permission check
	CheckUserPermission(ctx context.Context, userID, workspaceID uint, resource, action string) (bool, error)
	CheckRolePermission(ctx context.Context, workspaceID uint, roleName, resource, action string) (bool, error)

	// SeedPermission ensures a permission exists (idempotent).
	SeedPermission(ctx context.Context, perm entity.Permission) error

	// Transaction support: runs fn within a transaction.
	WithTransaction(ctx context.Context, fn func(tx RBACRepository) error) error
}

// DigestRepository defines the persistence operations needed by the digest scheduler.
type DigestRepository interface {
	FindEnabledDigestConfigs(ctx context.Context) ([]entity.DigestOrgConfig, error)
	CountAlertsSince(ctx context.Context, workspaceID uint, since time.Time) (int64, error)
	CountPackagesAnalyzedSince(ctx context.Context, workspaceID uint, since time.Time) (int64, error)
	GetClassificationBreakdownSince(ctx context.Context, workspaceID uint, since time.Time) (map[string]int64, error)
	GetTopAlertsSince(ctx context.Context, workspaceID uint, since time.Time, limit int) ([]entity.DigestTopAlert, error)
}
