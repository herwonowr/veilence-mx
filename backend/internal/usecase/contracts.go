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
}

// APIKeyRepository defines persistence operations for APIKey entities.
type APIKeyRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.APIKey, error)
	FindActiveByPrefix(ctx context.Context, prefix string) ([]entity.APIKey, error)
	FindByUserID(ctx context.Context, userID uint) ([]entity.APIKey, error)
	Create(ctx context.Context, key *entity.APIKey) error
	Update(ctx context.Context, key *entity.APIKey) error
	SoftDelete(ctx context.Context, userID, keyID uint) error
}

// PackageRepository defines persistence operations for Package entities.
type PackageRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Package, error)
	FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	FindActiveByOrgID(ctx context.Context, orgID uint) ([]entity.Package, error)
	FindByOrgAndName(ctx context.Context, orgID uint, name string, ecosystem entity.Ecosystem) (*entity.Package, error)
	Create(ctx context.Context, pkg *entity.Package) error
	Update(ctx context.Context, pkg *entity.Package) error
	BlockPackage(ctx context.Context, orgID, pkgID uint, reason string) error
	UnblockPackage(ctx context.Context, orgID, pkgID uint) error
	RemovePackage(ctx context.Context, orgID, pkgID uint) error
	CountByOrg(ctx context.Context, orgID uint, ecosystem *entity.Ecosystem) (int64, error)
	ExistsByOrgAndName(ctx context.Context, orgID uint, name string, ecosystem entity.Ecosystem) (bool, error)
}

// ReleaseRepository defines persistence operations for Release entities.
type ReleaseRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Release, error)
	FindByIDWithPackage(ctx context.Context, id uint) (*entity.Release, *entity.Package, error)
	FindByPackageID(ctx context.Context, packageID uint, page, limit int) ([]entity.Release, int64, error)
	FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.Release, int64, error)
	FindByOrgIDWithDetails(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error)
	FindByPackageIDAll(ctx context.Context, packageID uint) ([]entity.Release, error)
	UpdateStatus(ctx context.Context, id uint, status entity.ReleaseStatus) error
	Create(ctx context.Context, release *entity.Release) error
	Update(ctx context.Context, release *entity.Release) error
}

// DiffRepository defines persistence operations for Diff entities.
type DiffRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Diff, error)
	FindByReleaseID(ctx context.Context, releaseID uint) ([]entity.Diff, error)
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
	FindByIDWithPackage(ctx context.Context, id, orgID uint) (*entity.Alert, *entity.Package, error)
	FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.Alert, int64, error)
	FindByOrgIDWithPackage(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error)
	Create(ctx context.Context, alert *entity.Alert) error
	Update(ctx context.Context, alert *entity.Alert) error
	UpdateStatus(ctx context.Context, id uint, status entity.AlertStatus) error
	CountByOrgAndStatus(ctx context.Context, orgID uint) (map[entity.AlertStatus]int64, error)
}

// AlertNoteRepository defines persistence operations for AlertNote entities.
type AlertNoteRepository interface {
	FindByAlertID(ctx context.Context, alertID uint) ([]entity.AlertNote, error)
	FindByID(ctx context.Context, id uint) (*entity.AlertNote, error)
	Create(ctx context.Context, note *entity.AlertNote) error
	Update(ctx context.Context, note *entity.AlertNote) error
	Delete(ctx context.Context, id uint) error
}

// SettingRepository defines persistence operations for Setting entities.
type SettingRepository interface {
	FindByOrgID(ctx context.Context, orgID uint) ([]entity.Setting, error)
	FindByKey(ctx context.Context, orgID uint, key string) (*entity.Setting, error)
	Upsert(ctx context.Context, setting *entity.Setting) error
	UpsertByOrgAndKey(ctx context.Context, orgID uint, key, value string) error
	FindOrCreateByKey(ctx context.Context, key, defaultValue string) (*entity.Setting, error)
}

// OrganizationRepository defines persistence operations for Organization entities.
type OrganizationRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Organization, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Organization, error)
	CountBySlug(ctx context.Context, slug string, excludeID *uint) (int64, error)
	Create(ctx context.Context, org *entity.Organization) error
	Update(ctx context.Context, org *entity.Organization) error
	SoftDelete(ctx context.Context, id uint) error
	FindByUserID(ctx context.Context, userID uint) ([]entity.Organization, error)
}

// OrgMemberRepository defines persistence operations for OrgMember entities.
type OrgMemberRepository interface {
	FindByOrgID(ctx context.Context, orgID uint) ([]entity.OrgMember, error)
	FindByUserAndOrg(ctx context.Context, userID, orgID uint) (*entity.OrgMember, error)
	CountByUserAndOrg(ctx context.Context, userID, orgID uint) (int64, error)
	Create(ctx context.Context, member *entity.OrgMember) error
	Update(ctx context.Context, member *entity.OrgMember) error
	DeleteByUserAndOrg(ctx context.Context, userID, orgID uint) error
}

// RoleRepository defines persistence operations for Role entities.
type RoleRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.Role, error)
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*entity.Role, error)
	FindByOrgID(ctx context.Context, orgID uint) ([]entity.Role, error)
	Create(ctx context.Context, role *entity.Role) error
}

// PermissionRepository defines persistence operations for Permission entities.
type PermissionRepository interface {
	FindAll(ctx context.Context) ([]entity.Permission, error)
	FindOrCreate(ctx context.Context, perm *entity.Permission) error
	CheckUserPermission(ctx context.Context, userID, orgID uint, resource, action string) (bool, error)
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
	FindByOrgID(ctx context.Context, orgID uint, filters entity.AuditLogFilters, page, limit int) ([]entity.AuditLog, int64, error)
	FindByID(ctx context.Context, id uint) (*entity.AuditLog, error)
}

// NotificationChannelRepository defines persistence operations for NotificationChannel entities.
type NotificationChannelRepository interface {
	FindByID(ctx context.Context, id uint) (*entity.NotificationChannel, error)
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*entity.NotificationChannel, error)
	FindByOrgID(ctx context.Context, orgID uint) ([]entity.NotificationChannel, error)
	Create(ctx context.Context, channel *entity.NotificationChannel) error
	Update(ctx context.Context, channel *entity.NotificationChannel) error
	DeleteByIDAndOrg(ctx context.Context, id, orgID uint) (int64, error)
}

// NotificationRuleRepository defines persistence operations for NotificationRule entities.
type NotificationRuleRepository interface {
	FindByOrgID(ctx context.Context, orgID uint) ([]entity.NotificationRule, error)
	FindActiveByOrgID(ctx context.Context, orgID uint) ([]entity.NotificationRule, error)
	Create(ctx context.Context, rule *entity.NotificationRule) error
	DeleteByIDAndOrg(ctx context.Context, id, orgID uint) (int64, error)
}

// NotificationRepository defines persistence operations for Notification entities.
type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.Notification) error
	FindByUserAndOrg(ctx context.Context, orgID, userID uint, onlyUnread bool) ([]entity.Notification, error)
	MarkRead(ctx context.Context, id, userID uint) (int64, error)
	MarkAllRead(ctx context.Context, orgID, userID uint) (int64, error)
	CountUnread(ctx context.Context, orgID, userID uint) (int64, error)
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
}

// DashboardRepository defines read-only aggregation queries for the dashboard.
// All queries are scoped to an organization via orgID.
type DashboardRepository interface {
	GetStats(ctx context.Context, orgID uint) (*entity.DashboardStats, error)
	GetReleaseActivity(ctx context.Context, orgID uint, from, to time.Time) ([]entity.ReleaseActivityPoint, error)
	GetClassificationDistribution(ctx context.Context, orgID uint, from, to time.Time) ([]entity.ClassificationCount, error)
	GetBaselineCount(ctx context.Context, orgID uint, from, to time.Time) (int64, error)
	GetEcosystemDistribution(ctx context.Context, orgID uint) ([]entity.EcosystemCount, error)
	GetAlertsBySeverity(ctx context.Context, orgID uint, from, to time.Time) ([]entity.AlertSeverityCount, error)
	GetReleaseStatusDistribution(ctx context.Context, orgID uint, from, to time.Time) ([]entity.ReleaseStatusCount, error)
	GetUnanalyzedDiffIDs(ctx context.Context, orgID uint) ([]uint, error)
}

// --- Service Interfaces ---

// PackageService defines the business logic operations for package lifecycle management.
type PackageService interface {
	ListPackages(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error)
	GetPackage(ctx context.Context, orgID, pkgID uint) (*entity.Package, error)
	CreatePackage(ctx context.Context, orgID uint, name string, ecosystem entity.Ecosystem) (*entity.Package, error)
	ImportPackages(ctx context.Context, orgID uint, entries []entity.ImportEntry) (*entity.ImportResult, error)
	BlockPackage(ctx context.Context, orgID, pkgID uint, reason string) (*entity.Package, error)
	UnblockPackage(ctx context.Context, orgID, pkgID uint) (*entity.Package, error)
	RemovePackage(ctx context.Context, orgID, pkgID uint) error
}

// AlertService defines the business logic operations for alerts.
type AlertService interface {
	ListAlerts(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error)
	GetAlert(ctx context.Context, orgID, alertID uint) (*entity.Alert, *entity.Package, error)
	UpdateAlertStatus(ctx context.Context, orgID, alertID uint, status entity.AlertStatus) (*entity.Alert, error)
}

// AlertNoteService defines the business logic operations for alert notes.
type AlertNoteService interface {
	ListByAlert(ctx context.Context, orgID, alertID uint) ([]entity.AlertNote, error)
	Create(ctx context.Context, orgID, alertID, userID uint, content string) (*entity.AlertNote, error)
	Update(ctx context.Context, orgID, alertID, noteID, userID uint, content string) (*entity.AlertNote, error)
	Delete(ctx context.Context, orgID, alertID, noteID, userID uint) error
}

// ReleaseService defines the business logic operations for releases.
type ReleaseService interface {
	ListByPackage(ctx context.Context, orgID, packageID uint, page, limit int) ([]entity.Release, int64, error)
	GetRelease(ctx context.Context, orgID, releaseID uint) (*entity.ReleaseDetail, error)
	ReanalyzeRelease(ctx context.Context, orgID, releaseID uint) (string, string, error)
	GetAnalysisHistory(ctx context.Context, orgID, packageID uint) ([]entity.AnalysisHistoryEntry, error)
}

// SettingService defines the business logic operations for settings.
type SettingService interface {
	GetSettings(ctx context.Context, orgID uint) (map[string]string, error)
	UpdateSettings(ctx context.Context, orgID uint, settings map[string]string) (map[string]string, error)
}

// DashboardService defines the business logic operations for the dashboard.
type DashboardService interface {
	GetStats(ctx context.Context, orgID uint) (*entity.DashboardStats, error)
	GetRecentReleases(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error)
	GetChartData(ctx context.Context, orgID uint, from, to time.Time) (*entity.ChartData, error)
	ReanalyzeAll(ctx context.Context, orgID uint) (int, error)
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
