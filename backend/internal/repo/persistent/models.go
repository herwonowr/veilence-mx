// Package persistent implements repository interfaces using PostgreSQL via GORM.
// GORM model structs and repository implementations live here together.
package persistent

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// --- Package Models ---

// Ecosystem represents a package ecosystem source (GORM model).
type Ecosystem string

const (
	EcosystemPython Ecosystem = "python"
	EcosystemNPM    Ecosystem = "npm"
)

// PackageSource describes how a package was added to monitoring (GORM model).
type PackageSource string

const (
	PackageSourceManual     PackageSource = "manual"
	PackageSourceDiscovered PackageSource = "discovered"
	PackageSourceImported   PackageSource = "imported"
)

// PackageStatus represents the monitoring status of a package (GORM model).
type PackageStatus string

const (
	PackageStatusActive    PackageStatus = "active"
	PackageStatusBlocked   PackageStatus = "blocked"
	PackageStatusRemoved   PackageStatus = "removed"
	PackageStatusSuggested PackageStatus = "suggested"
)

// Package is the GORM model for monitored packages.
type Package struct {
	ID                     string        `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID                  string        `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Name                   string        `gorm:"not null" json:"name"`
	Ecosystem              Ecosystem     `gorm:"column:ecosystem;not null;type:varchar(10)" json:"ecosystem"`
	LatestVersion          string        `gorm:"type:varchar(100)" json:"latestVersion"`
	Description            string        `gorm:"type:text" json:"description"`
	Source                 PackageSource `gorm:"not null;default:'manual';type:varchar(20)" json:"source"`
	Status                 PackageStatus `gorm:"not null;default:'active';type:varchar(20);index" json:"status"`
	Rank                   *int          `json:"rank,omitempty"`
	DownloadCount          int64         `gorm:"not null;default:0" json:"downloadCount"`
	DownloadCountUpdatedAt *time.Time    `json:"downloadCountUpdatedAt,omitempty"`
	BlockedAt              *time.Time    `json:"blockedAt,omitempty"`
	BlockedReason          string        `gorm:"type:text" json:"blockedReason,omitempty"`
	CreatedAt              time.Time     `json:"createdAt"`
	UpdatedAt              time.Time     `json:"updatedAt"`
	Releases               []Release     `gorm:"foreignKey:PackageID" json:"releases,omitempty"`
}

func (Package) TableName() string { return "packages" }

// --- Release Models ---

// ReleaseStatus represents the processing status of a release (GORM model).
type ReleaseStatus string

const (
	ReleaseStatusPending   ReleaseStatus = "pending"
	ReleaseStatusDiffing   ReleaseStatus = "diffing"
	ReleaseStatusAnalyzing ReleaseStatus = "analyzing"
	ReleaseStatusCompleted ReleaseStatus = "completed"
	ReleaseStatusError     ReleaseStatus = "error"
)

// Release is the GORM model for package releases.
type Release struct {
	ID           string        `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	PackageID    string        `gorm:"type:uuid;not null;index" json:"packageId"`
	Package      Package       `gorm:"foreignKey:PackageID" json:"-"`
	Version      string        `gorm:"not null;type:varchar(100)" json:"version"`
	PublishedAt  time.Time     `json:"publishedAt"`
	TarballURL   string        `gorm:"type:text" json:"tarballUrl"`
	SHA256       string        `gorm:"type:varchar(64)" json:"sha256"`
	Status       ReleaseStatus `gorm:"not null;type:varchar(20);default:'pending'" json:"status"`
	ErrorMessage string        `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt    time.Time     `json:"createdAt"`
	Diffs        []Diff        `gorm:"foreignKey:ReleaseID" json:"diffs,omitempty"`
}

func (Release) TableName() string { return "releases" }

// --- Diff Models ---

// Diff is the GORM model for release diffs.
type Diff struct {
	ID               string     `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	ReleaseID        string     `gorm:"type:uuid;not null;index" json:"releaseId"`
	Release          Release    `gorm:"foreignKey:ReleaseID" json:"-"`
	PrevReleaseID    string     `gorm:"type:uuid;not null" json:"prevReleaseId"`
	PrevRelease      Release    `gorm:"foreignKey:PrevReleaseID" json:"-"`
	DiffContent      string     `gorm:"type:text" json:"diffContent"`
	FileChangesCount int        `json:"fileChangesCount"`
	LinesAdded       int        `json:"linesAdded"`
	LinesRemoved     int        `json:"linesRemoved"`
	Truncated        bool       `gorm:"not null;default:false" json:"truncated"`
	OriginalSize     int        `gorm:"not null;default:0" json:"originalSize"`
	CreatedAt        time.Time  `json:"createdAt"`
	Analyses         []Analysis `gorm:"foreignKey:DiffID" json:"analyses,omitempty"`
}

func (Diff) TableName() string { return "diffs" }

// --- Analysis Models ---

// Classification represents the LLM's assessment of a diff (GORM model).
type Classification string

const (
	ClassificationBenign     Classification = "benign"
	ClassificationSuspicious Classification = "suspicious"
	ClassificationMalicious  Classification = "malicious"
)

// AnalyzerType represents which analysis backend was used (GORM model).
type AnalyzerType string

const (
	AnalyzerTypeCopilot   AnalyzerType = "copilot"
	AnalyzerTypeOpenAI    AnalyzerType = "openai"
	AnalyzerTypeAnthropic AnalyzerType = "anthropic"
	AnalyzerTypeOllama    AnalyzerType = "ollama"
)

// Analysis is the GORM model for LLM analysis results.
type Analysis struct {
	ID             string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	DiffID         string         `gorm:"type:uuid;not null;index" json:"diffId"`
	Diff           Diff           `gorm:"foreignKey:DiffID" json:"-"`
	Classification Classification `gorm:"not null;type:varchar(20)" json:"classification"`
	Confidence     float64        `gorm:"not null" json:"confidence"`
	Reasoning      string         `gorm:"type:text" json:"reasoning"`
	ModelUsed      string         `gorm:"type:varchar(100)" json:"modelUsed"`
	AnalyzerType   AnalyzerType   `gorm:"not null;type:varchar(20)" json:"analyzerType"`
	RawResponse    string         `gorm:"type:text" json:"-"`
	CreatedAt      time.Time      `json:"createdAt"`
}

func (Analysis) TableName() string { return "analyses" }

// --- Alert Models ---

// AlertSeverity represents the severity level of an alert (GORM model).
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the current status of an alert (GORM model).
type AlertStatus string

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// Alert is the GORM model for security alerts.
type Alert struct {
	ID         string        `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID      string        `gorm:"type:uuid;not null;index" json:"workspaceId"`
	AnalysisID string        `gorm:"type:uuid;not null;index" json:"analysisId"`
	Analysis   Analysis      `gorm:"foreignKey:AnalysisID" json:"-"`
	ReleaseID  *string       `gorm:"type:uuid;index" json:"releaseId"`
	Release    Release       `gorm:"foreignKey:ReleaseID" json:"-"`
	PackageID  string        `gorm:"type:uuid;not null;index" json:"packageId"`
	Package    Package       `gorm:"foreignKey:PackageID" json:"-"`
	Severity   AlertSeverity `gorm:"not null;type:varchar(20)" json:"severity"`
	Status     AlertStatus   `gorm:"not null;type:varchar(20);default:'new'" json:"status"`
	Message    string        `gorm:"type:text" json:"message"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

func (Alert) TableName() string { return "alerts" }

// AlertNote is the GORM model for alert notes/comments.
type AlertNote struct {
	ID        string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	AlertID   string    `gorm:"type:uuid;not null;index" json:"alertId"`
	Alert     Alert     `gorm:"foreignKey:AlertID" json:"-"`
	WorkspaceID string      `gorm:"type:uuid;not null;index" json:"workspaceId"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"userId"`
	UserEmail string    `gorm:"type:varchar(255)" json:"userEmail"`
	Content   string    `gorm:"not null;type:text" json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (AlertNote) TableName() string { return "alert_notes" }

// --- User Models ---

// User is the GORM model for system users.
type User struct {
	ID            string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Email         string         `gorm:"uniqueIndex;not null;type:varchar(255)" json:"email"`
	PasswordHash  string         `gorm:"type:varchar(255)" json:"-"`
	FirstName     string         `gorm:"type:varchar(100)" json:"firstName"`
	LastName      string         `gorm:"type:varchar(100)" json:"lastName"`
	IsActive      bool           `gorm:"not null;default:true" json:"isActive"`
	EmailVerified bool           `gorm:"not null;default:false" json:"emailVerified"`
	LastLoginAt   *time.Time     `json:"lastLoginAt,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

// HashPassword hashes the given plaintext password and stores it on the user.
func (u *User) HashPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword compares the given plaintext password against the stored hash.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// --- RefreshToken Models ---

// RefreshToken is the GORM model for refresh tokens.
type RefreshToken struct {
	ID        string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string    `gorm:"uniqueIndex;not null;column:token_hash;type:varchar(255)" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// --- APIKey Models ---

// APIKey is the GORM model for API keys.
type APIKey struct {
	ID          string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID      string         `gorm:"type:uuid;not null;index" json:"userId"`
	WorkspaceID string         `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Name        string         `gorm:"not null;type:varchar(100)" json:"name"`
	KeyHash     string         `gorm:"uniqueIndex;not null;type:varchar(255)" json:"-"`
	KeyPrefix   string         `gorm:"not null;type:varchar(10)" json:"keyPrefix"`
	Role        string         `gorm:"not null;type:varchar(20);default:'viewer'" json:"role"`
	LastUsedAt  *time.Time     `json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time     `json:"expiresAt,omitempty"`
	IsActive    bool           `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time      `json:"createdAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (APIKey) TableName() string { return "api_keys" }

// --- Workspace Models ---

// Workspace is the GORM model for tenant workspaces.
type Workspace struct {
	ID          string         `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"not null;type:varchar(100)" json:"name"`
	Slug        string         `gorm:"not null;type:varchar(100)" json:"slug"`
	Description string         `gorm:"type:text" json:"description"`
	OwnerID     string         `gorm:"type:uuid;not null" json:"ownerId"`
	IsActive    bool           `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Workspace) TableName() string { return "workspaces" }

// --- WorkspaceMember Models ---

// WorkspaceMember is the GORM model for workspace memberships.
type WorkspaceMember struct {
	ID        string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID string      `gorm:"type:uuid;not null;uniqueIndex:idx_workspace_user" json:"workspaceId"`
	UserID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_workspace_user" json:"userId"`
	RoleID    string    `gorm:"type:uuid;not null" json:"roleId"`
	Role      Role      `gorm:"foreignKey:RoleID" json:"role"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	JoinedAt  time.Time `gorm:"not null" json:"joinedAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (WorkspaceMember) TableName() string { return "workspace_members" }

// --- Role & Permission Models ---

// Role is the GORM model for roles.
type Role struct {
	ID          string       `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID       string       `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Name        string       `gorm:"not null;type:varchar(50)" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	IsSystem    bool         `gorm:"not null;default:false" json:"isSystem"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	Permissions []Permission `gorm:"many2many:role_permissions" json:"permissions"`
}

func (Role) TableName() string { return "roles" }

// Permission is the GORM model for permissions.
type Permission struct {
	ID       string `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Resource string `gorm:"not null;type:varchar(50)" json:"resource"`
	Action   string `gorm:"not null;type:varchar(50)" json:"action"`
}

func (Permission) TableName() string { return "permissions" }

// System role names.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// SystemPermissions defines all permissions available in the system.
var SystemPermissions = []Permission{
	{Resource: "packages", Action: "read"},
	{Resource: "packages", Action: "write"},
	{Resource: "packages", Action: "delete"},
	{Resource: "alerts", Action: "read"},
	{Resource: "alerts", Action: "write"},
	{Resource: "releases", Action: "read"},
	{Resource: "settings", Action: "read"},
	{Resource: "settings", Action: "write"},
	{Resource: "members", Action: "read"},
	{Resource: "members", Action: "invite"},
	{Resource: "members", Action: "remove"},
	{Resource: "roles", Action: "read"},
	{Resource: "roles", Action: "write"},
	{Resource: "workspace", Action: "read"},
	{Resource: "workspace", Action: "write"},
	{Resource: "workspace", Action: "delete"},
	{Resource: "api_keys", Action: "read"},
	{Resource: "api_keys", Action: "write"},
	{Resource: "audit", Action: "read"},
	{Resource: "notifications", Action: "read"},
	{Resource: "notifications", Action: "create"},
	{Resource: "notifications", Action: "update"},
	{Resource: "notifications", Action: "delete"},
}

// --- Invitation Models ---

// Invitation is the GORM model for workspace invitations.
type Invitation struct {
	ID         string     `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID      string     `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Email      string     `gorm:"not null;type:varchar(255)" json:"email"`
	RoleID     string     `gorm:"type:uuid;not null" json:"roleId"`
	TokenHash  string     `gorm:"uniqueIndex;not null;type:varchar(255);column:token_hash" json:"-"`
	InvitedBy  string     `gorm:"type:uuid;not null" json:"invitedBy"`
	ExpiresAt  time.Time  `gorm:"not null" json:"expiresAt"`
	AcceptedAt *time.Time `json:"acceptedAt,omitempty"`
	DeclinedAt *time.Time `json:"declinedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (Invitation) TableName() string { return "invitations" }

// --- AuditLog Models ---

// AuditLog is the GORM model for audit log entries.
type AuditLog struct {
	ID            string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID        *string   `gorm:"type:uuid;index" json:"userId"`
	WorkspaceID   *string   `gorm:"type:uuid;index" json:"workspaceId"`
	Action        string    `gorm:"not null;type:varchar(50)" json:"action"`
	Resource      string    `gorm:"not null;type:varchar(50)" json:"resource"`
	ResourceID    *string   `gorm:"type:uuid" json:"resourceId"`
	Details       string    `gorm:"type:text" json:"details"`
	IPAddress     string    `gorm:"type:varchar(45)" json:"ipAddress"`
	UserAgent     string    `gorm:"type:varchar(255)" json:"userAgent"`
	CorrelationID string    `gorm:"type:varchar(36);index" json:"correlationId"`
	CreatedAt     time.Time `gorm:"index" json:"createdAt"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// --- Setting Models ---

// Setting is the GORM model for system settings.
type Setting struct {
	ID        string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID string      `gorm:"type:uuid;index;not null;uniqueIndex:idx_settings_workspace_key" json:"workspaceId"`
	Key       string    `gorm:"not null;type:varchar(100);uniqueIndex:idx_settings_workspace_key" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Setting) TableName() string { return "settings" }

// --- Notification Models ---

// NotificationChannelType represents the type of a notification channel (GORM model).
type NotificationChannelType string

const (
	NotificationChannelEmail   NotificationChannelType = "email"
	NotificationChannelSlack   NotificationChannelType = "slack"
	NotificationChannelWebhook NotificationChannelType = "webhook"
)

// NotificationChannel is the GORM model for notification channels.
type NotificationChannel struct {
	ID        string                  `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID string                    `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Name      string                  `gorm:"not null;type:varchar(100)" json:"name"`
	Type      NotificationChannelType `gorm:"not null;type:varchar(20)" json:"type"`
	Config    string                  `gorm:"type:text" json:"config"`
	IsActive  bool                    `gorm:"not null;default:true" json:"isActive"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

func (NotificationChannel) TableName() string { return "notification_channels" }

// NotificationRule is the GORM model for notification rules.
type NotificationRule struct {
	ID        string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID string      `gorm:"type:uuid;not null;index" json:"workspaceId"`
	ChannelID string    `gorm:"type:uuid;not null;index" json:"channelId"`
	Severity  string    `gorm:"type:varchar(20)" json:"severity"`
	IsActive  bool      `gorm:"not null;default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (NotificationRule) TableName() string { return "notification_rules" }

// Notification is the GORM model for in-app notifications.
type Notification struct {
	ID            string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	WorkspaceID   string    `gorm:"type:uuid;not null;index" json:"workspaceId"`
	UserID        *string   `gorm:"type:uuid;index" json:"userId"`
	ChannelID     *string   `gorm:"type:uuid;index" json:"channelId"`
	Severity      string    `gorm:"not null;type:varchar(20);default:''" json:"severity"`
	EventType     string    `gorm:"not null;type:varchar(100);default:'';index" json:"eventType"`
	ReferenceID   *string   `gorm:"type:uuid" json:"referenceId"`
	ReferenceType string    `gorm:"not null;type:varchar(50);default:''" json:"referenceType"`
	Title         string    `gorm:"not null;type:varchar(255)" json:"title"`
	Message       string    `gorm:"type:text" json:"message"`
	IsRead        bool      `gorm:"not null;default:false" json:"isRead"`
	SentAt        time.Time `json:"sentAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (Notification) TableName() string { return "notifications" }

// --- Token Models ---

// PasswordResetToken is the GORM model for password reset tokens.
type PasswordResetToken struct {
	ID        string     `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID    string     `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string     `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

// EmailVerificationToken is the GORM model for email verification tokens.
type EmailVerificationToken struct {
	ID        string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string    `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func (EmailVerificationToken) TableName() string { return "email_verification_tokens" }

// --- Session Models ---

// Session is the GORM model for user sessions.
type Session struct {
	ID         string    `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID     string    `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash  string    `gorm:"not null;uniqueIndex;type:varchar(255)" json:"-"`
	IPAddress  string    `gorm:"type:varchar(45)" json:"ipAddress"`
	UserAgent  string    `gorm:"type:varchar(512)" json:"userAgent"`
	CreatedAt  time.Time `json:"createdAt"`
	LastActive time.Time `gorm:"not null" json:"lastActive"`
	ExpiresAt  time.Time `gorm:"not null" json:"expiresAt"`
}

func (Session) TableName() string { return "sessions" }

// AllModels is the complete list of GORM models for auto-migration and testing.
var AllModels = []any{
	&User{}, &RefreshToken{}, &APIKey{},
	&Package{}, &Release{}, &Diff{}, &Analysis{},
	&Alert{}, &AlertNote{},
	&Workspace{}, &WorkspaceMember{}, &Role{}, &Permission{},
	&Invitation{}, &AuditLog{},
	&Setting{},
	&NotificationChannel{}, &NotificationRule{}, &Notification{},
	&PasswordResetToken{}, &EmailVerificationToken{},
	&Session{},
}
