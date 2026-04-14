package domain

import (
	"context"
	"time"
)

// UserRepository defines persistence operations for User entities.
type UserRepository interface {
	// FindByID returns a user by their ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*User, error)
	// FindByEmail returns a user by their email address. Returns an error if not found.
	FindByEmail(ctx context.Context, email string) (*User, error)
	// Create persists a new user.
	Create(ctx context.Context, user *User) error
	// Update saves changes to an existing user.
	Update(ctx context.Context, user *User) error
}

// RefreshTokenRepository defines persistence operations for RefreshToken entities.
type RefreshTokenRepository interface {
	// FindByTokenHash returns a refresh token by its SHA-256 hash. Returns an error if not found.
	FindByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	// Create persists a new refresh token.
	Create(ctx context.Context, token *RefreshToken) error
	// Delete removes a refresh token by ID.
	Delete(ctx context.Context, id uint) error
	// DeleteByTokenHash removes a refresh token by its hash.
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}

// APIKeyRepository defines persistence operations for APIKey entities.
type APIKeyRepository interface {
	// FindByID returns an API key by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*APIKey, error)
	// FindActiveByPrefix returns active API keys matching the given prefix.
	FindActiveByPrefix(ctx context.Context, prefix string) ([]APIKey, error)
	// FindByUserID returns all API keys for a user (including soft-deleted check at repo level).
	FindByUserID(ctx context.Context, userID uint) ([]APIKey, error)
	// Create persists a new API key.
	Create(ctx context.Context, key *APIKey) error
	// Update saves changes to an existing API key.
	Update(ctx context.Context, key *APIKey) error
	// SoftDelete marks an API key as deleted (by user ownership).
	SoftDelete(ctx context.Context, userID, keyID uint) error
}

// PackageRepository defines persistence operations for Package entities.
type PackageRepository interface {
	// FindByID returns a package by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Package, error)
	// FindByOrgID returns all packages belonging to an organization.
	// By default excludes removed packages unless includeRemoved is true.
	FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string) ([]Package, int64, error)
	// FindActiveByOrgID returns all active packages for an org (for monitoring loop).
	FindActiveByOrgID(ctx context.Context, orgID uint) ([]Package, error)
	// FindByOrgAndName returns a package by org, name, and ecosystem. Returns an error if not found.
	FindByOrgAndName(ctx context.Context, orgID uint, name string, ecosystem Ecosystem) (*Package, error)
	// Create persists a new package.
	Create(ctx context.Context, pkg *Package) error
	// Update saves changes to an existing package.
	Update(ctx context.Context, pkg *Package) error
	// BlockPackage sets a package's status to 'blocked' with a reason and timestamp.
	BlockPackage(ctx context.Context, orgID, pkgID uint, reason string) error
	// UnblockPackage sets a package's status back to 'active', clearing block fields.
	UnblockPackage(ctx context.Context, orgID, pkgID uint) error
	// RemovePackage sets a package's status to 'removed' (replaces SoftDelete).
	RemovePackage(ctx context.Context, orgID, pkgID uint) error
	// CountByOrg returns the count of packages in an org, optionally filtered by ecosystem.
	CountByOrg(ctx context.Context, orgID uint, ecosystem *Ecosystem) (int64, error)
}

// ReleaseRepository defines persistence operations for Release entities.
type ReleaseRepository interface {
	// FindByID returns a release by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Release, error)
	// FindByPackageID returns all releases for a package.
	FindByPackageID(ctx context.Context, packageID uint, page, limit int) ([]Release, int64, error)
	// Create persists a new release.
	Create(ctx context.Context, release *Release) error
	// Update saves changes to an existing release.
	Update(ctx context.Context, release *Release) error
}

// DiffRepository defines persistence operations for Diff entities.
type DiffRepository interface {
	// FindByID returns a diff by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Diff, error)
	// FindByReleaseID returns diffs for a release.
	FindByReleaseID(ctx context.Context, releaseID uint) ([]Diff, error)
	// Create persists a new diff.
	Create(ctx context.Context, diff *Diff) error
}

// AnalysisRepository defines persistence operations for Analysis entities.
type AnalysisRepository interface {
	// FindByID returns an analysis by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Analysis, error)
	// FindByDiffID returns all analyses for a diff.
	FindByDiffID(ctx context.Context, diffID uint) ([]Analysis, error)
	// Create persists a new analysis.
	Create(ctx context.Context, analysis *Analysis) error
	// CountByDiffID returns the number of analyses for a diff.
	CountByDiffID(ctx context.Context, diffID uint) (int64, error)
}

// AlertRepository defines persistence operations for Alert entities.
type AlertRepository interface {
	// FindByID returns an alert by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Alert, error)
	// FindByOrgID returns all alerts for an organization with pagination.
	FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string, filters AlertFilters) ([]Alert, int64, error)
	// Create persists a new alert.
	Create(ctx context.Context, alert *Alert) error
	// Update saves changes to an existing alert.
	Update(ctx context.Context, alert *Alert) error
	// CountByOrgAndStatus returns counts of alerts by status for an org.
	CountByOrgAndStatus(ctx context.Context, orgID uint) (map[AlertStatus]int64, error)
}

// AlertFilters holds optional query filters for listing alerts.
type AlertFilters struct {
	Severity *AlertSeverity
	Status   *AlertStatus
	Search   *string
}

// AlertNoteRepository defines persistence operations for AlertNote entities.
type AlertNoteRepository interface {
	// FindByAlertID returns all notes for an alert, ordered by created_at DESC.
	FindByAlertID(ctx context.Context, alertID uint) ([]AlertNote, error)
	// FindByID returns a single alert note by ID. Returns ErrNotFound if not found.
	FindByID(ctx context.Context, id uint) (*AlertNote, error)
	// Create persists a new alert note.
	Create(ctx context.Context, note *AlertNote) error
	// Update saves changes to an existing alert note (content and updated_at).
	Update(ctx context.Context, note *AlertNote) error
	// Delete removes an alert note by ID.
	Delete(ctx context.Context, id uint) error
}

// SettingRepository defines persistence operations for Setting entities.
type SettingRepository interface {
	// FindByOrgID returns all settings for an organization (including global defaults).
	FindByOrgID(ctx context.Context, orgID uint) ([]Setting, error)
	// FindByKey returns a setting by org and key. Returns an error if not found.
	FindByKey(ctx context.Context, orgID uint, key string) (*Setting, error)
	// Upsert creates or updates a setting by org and key.
	Upsert(ctx context.Context, setting *Setting) error
	// FindOrCreateByKey finds a setting by key or creates it with the given default value.
	FindOrCreateByKey(ctx context.Context, key, defaultValue string) (*Setting, error)
}

// OrganizationRepository defines persistence operations for Organization entities.
type OrganizationRepository interface {
	// FindByID returns an organization by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Organization, error)
	// FindBySlug returns an organization by its slug. Returns an error if not found.
	FindBySlug(ctx context.Context, slug string) (*Organization, error)
	// CountBySlug returns the number of organizations with the given slug, optionally excluding an ID.
	CountBySlug(ctx context.Context, slug string, excludeID *uint) (int64, error)
	// Create persists a new organization.
	Create(ctx context.Context, org *Organization) error
	// Update saves changes to an existing organization.
	Update(ctx context.Context, org *Organization) error
	// SoftDelete marks an organization as deleted.
	SoftDelete(ctx context.Context, id uint) error
	// FindByUserID returns all organizations the user is a member of.
	FindByUserID(ctx context.Context, userID uint) ([]Organization, error)
}

// OrgMemberRepository defines persistence operations for OrgMember entities.
type OrgMemberRepository interface {
	// FindByOrgID returns all members of an organization with their roles.
	FindByOrgID(ctx context.Context, orgID uint) ([]OrgMember, error)
	// FindByUserAndOrg returns a membership by user and org. Returns an error if not found.
	FindByUserAndOrg(ctx context.Context, userID, orgID uint) (*OrgMember, error)
	// CountByUserAndOrg returns 1 if member exists, 0 if not.
	CountByUserAndOrg(ctx context.Context, userID, orgID uint) (int64, error)
	// Create persists a new membership.
	Create(ctx context.Context, member *OrgMember) error
	// Update saves changes to an existing membership.
	Update(ctx context.Context, member *OrgMember) error
	// DeleteByUserAndOrg removes a membership.
	DeleteByUserAndOrg(ctx context.Context, userID, orgID uint) error
}

// RoleRepository defines persistence operations for Role entities.
type RoleRepository interface {
	// FindByID returns a role by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Role, error)
	// FindByIDAndOrg returns a role by ID and org. Returns an error if not found.
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*Role, error)
	// FindByOrgID returns all roles for an organization with permissions.
	FindByOrgID(ctx context.Context, orgID uint) ([]Role, error)
	// Create persists a new role.
	Create(ctx context.Context, role *Role) error
}

// PermissionRepository defines persistence operations for Permission entities.
type PermissionRepository interface {
	// FindAll returns all system permissions.
	FindAll(ctx context.Context) ([]Permission, error)
	// FindOrCreate finds a permission by resource and action, or creates it.
	FindOrCreate(ctx context.Context, perm *Permission) error
	// CheckUserPermission verifies whether a user has a specific permission in an org.
	// Returns true if the user has the permission.
	CheckUserPermission(ctx context.Context, userID, orgID uint, resource, action string) (bool, error)
}

// InvitationRepository defines persistence operations for Invitation entities.
type InvitationRepository interface {
	// FindByTokenHash returns an invitation by its hashed token. Returns an error if not found.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Invitation, error)
	// Create persists a new invitation.
	Create(ctx context.Context, invitation *Invitation) error
	// Update saves changes to an existing invitation.
	Update(ctx context.Context, invitation *Invitation) error
}

// AuditLogRepository defines persistence operations for AuditLog entities.
type AuditLogRepository interface {
	// Create persists a new audit log entry.
	Create(ctx context.Context, entry *AuditLog) error
	// FindByOrgID returns paginated audit logs for an org with optional filters.
	FindByOrgID(ctx context.Context, orgID uint, filters AuditLogFilters, page, limit int) ([]AuditLog, int64, error)
	// FindByID returns a single audit log entry. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*AuditLog, error)
}

// NotificationChannelRepository defines persistence operations for NotificationChannel entities.
type NotificationChannelRepository interface {
	// FindByID returns a notification channel by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*NotificationChannel, error)
	// FindByIDAndOrg returns a channel by ID and org. Returns an error if not found.
	FindByIDAndOrg(ctx context.Context, id, orgID uint) (*NotificationChannel, error)
	// FindByOrgID returns all channels for an organization.
	FindByOrgID(ctx context.Context, orgID uint) ([]NotificationChannel, error)
	// Create persists a new notification channel.
	Create(ctx context.Context, channel *NotificationChannel) error
	// Update saves changes to an existing channel.
	Update(ctx context.Context, channel *NotificationChannel) error
	// DeleteByIDAndOrg deletes a channel by ID and org. Returns rows affected.
	DeleteByIDAndOrg(ctx context.Context, id, orgID uint) (int64, error)
}

// NotificationRuleRepository defines persistence operations for NotificationRule entities.
type NotificationRuleRepository interface {
	// FindByOrgID returns all rules for an organization.
	FindByOrgID(ctx context.Context, orgID uint) ([]NotificationRule, error)
	// FindActiveByOrgID returns all active rules for an organization.
	FindActiveByOrgID(ctx context.Context, orgID uint) ([]NotificationRule, error)
	// Create persists a new notification rule.
	Create(ctx context.Context, rule *NotificationRule) error
	// DeleteByIDAndOrg deletes a rule by ID and org. Returns rows affected.
	DeleteByIDAndOrg(ctx context.Context, id, orgID uint) (int64, error)
}

// NotificationRepository defines persistence operations for Notification entities.
type NotificationRepository interface {
	// Create persists a new notification.
	Create(ctx context.Context, notification *Notification) error
	// FindByUserAndOrg returns notifications for a user in an org.
	// If orgID is 0, returns across all orgs. If onlyUnread is true, filters to unread only.
	FindByUserAndOrg(ctx context.Context, orgID, userID uint, onlyUnread bool) ([]Notification, error)
	// MarkRead marks a notification as read (checking user ownership).
	MarkRead(ctx context.Context, id, userID uint) (int64, error)
	// MarkAllRead marks all notifications as read for a user, optionally scoped to an org.
	MarkAllRead(ctx context.Context, orgID, userID uint) (int64, error)
	// CountUnread returns the number of unread notifications for a user.
	CountUnread(ctx context.Context, orgID, userID uint) (int64, error)
}

// PasswordResetTokenRepository defines persistence operations for PasswordResetToken entities.
type PasswordResetTokenRepository interface {
	// FindByTokenHash returns a password reset token by its hash. Returns an error if not found.
	FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error)
	// Create persists a new password reset token.
	Create(ctx context.Context, token *PasswordResetToken) error
	// MarkUsed marks a token as used by setting its used_at timestamp.
	MarkUsed(ctx context.Context, id uint) error
	// DeleteExpiredByUserID removes all expired or used tokens for a user.
	DeleteExpiredByUserID(ctx context.Context, userID uint) error
}

// EmailVerificationTokenRepository defines persistence operations for EmailVerificationToken entities.
type EmailVerificationTokenRepository interface {
	// FindByTokenHash returns an email verification token by its hash. Returns an error if not found.
	FindByTokenHash(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)
	// Create persists a new email verification token.
	Create(ctx context.Context, token *EmailVerificationToken) error
	// Delete removes a token by ID.
	Delete(ctx context.Context, id uint) error
	// DeleteByUserID removes all tokens for a user.
	DeleteByUserID(ctx context.Context, userID uint) error
}

// SessionRepository defines persistence operations for Session entities.
type SessionRepository interface {
	// FindByID returns a session by its ID. Returns an error if not found.
	FindByID(ctx context.Context, id uint) (*Session, error)
	// FindByUserID returns all active sessions for a user.
	FindByUserID(ctx context.Context, userID uint) ([]Session, error)
	// FindByTokenHash returns a session by its token hash. Returns an error if not found.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)
	// Create persists a new session.
	Create(ctx context.Context, session *Session) error
	// UpdateLastActive updates the last_active timestamp for a session.
	UpdateLastActive(ctx context.Context, id uint, lastActive time.Time) error
	// UpdateTokenHash updates the token_hash for a session (called after refresh token rotation).
	UpdateTokenHash(ctx context.Context, id uint, tokenHash string) error
	// Delete removes a session by ID.
	Delete(ctx context.Context, id uint) error
	// DeleteExpired removes all sessions that have passed their expiration time.
	DeleteExpired(ctx context.Context) (int64, error)
	// CountByUserID returns the number of active sessions for a user.
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	// DeleteOldestByUserID removes the oldest session for a user.
	DeleteOldestByUserID(ctx context.Context, userID uint) error
}
