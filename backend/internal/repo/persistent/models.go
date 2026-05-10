// Package persistent implements repository interfaces using PostgreSQL via GORM.
// GORM model structs and repository implementations live here together.
package persistent

import (
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/pkg/id"
)

// --- Package Models ---

// Ecosystem represents a package ecosystem source (GORM model).
type Ecosystem string

const (
	EcosystemPython Ecosystem = "python"
	EcosystemNPM    Ecosystem = "npm"
	EcosystemGo     Ecosystem = "go"
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
	ID                     string        `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID            string        `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Name                   string        `gorm:"not null" json:"name"`
	Ecosystem              Ecosystem     `gorm:"column:ecosystem;not null;type:varchar(10)" json:"ecosystem"`
	LatestVersion          string        `gorm:"type:varchar(100)" json:"latestVersion"`
	Description            string        `gorm:"type:text" json:"description"`
	Source                 PackageSource `gorm:"not null;default:'manual';type:varchar(20)" json:"source"`
	Status                 PackageStatus `gorm:"not null;default:'active';type:varchar(20);index" json:"status"`
	DownloadCount          int64         `gorm:"not null;default:0" json:"downloadCount"`
	DownloadCountUpdatedAt *time.Time    `json:"downloadCountUpdatedAt,omitempty"`
	BlockedAt              *time.Time    `json:"blockedAt,omitempty"`
	BlockedReason          string        `gorm:"type:text" json:"blockedReason,omitempty"`
	LastReleaseAt          *time.Time    `gorm:"column:last_release_at;->" json:"-"`
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
	ID           string        `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID  string        `gorm:"type:uuid;not null;index" json:"workspaceId"`
	PackageID    string        `gorm:"type:uuid;not null;index" json:"packageId"`
	Package      Package       `gorm:"foreignKey:PackageID" json:"-"`
	Version      string        `gorm:"not null;type:varchar(100)" json:"version"`
	PublishedAt  time.Time     `json:"publishedAt"`
	TarballURL   string        `gorm:"type:text" json:"tarballUrl"`
	Status       ReleaseStatus `gorm:"not null;type:varchar(20);default:'pending'" json:"status"`
	ErrorMessage string        `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedAt    time.Time     `json:"createdAt"`
	Diffs        []Diff        `gorm:"foreignKey:ReleaseID" json:"diffs,omitempty"`
}

func (Release) TableName() string { return "releases" }

// --- Diff Models ---

// Diff is the GORM model for release diffs.
type Diff struct {
	ID               string     `gorm:"type:uuid;primarykey" json:"id"`
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
	ClassificationBaseline   Classification = "baseline"
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
	ID             string         `gorm:"type:uuid;primarykey" json:"id"`
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
	ID          string        `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID string        `gorm:"type:uuid;not null;index" json:"workspaceId"`
	AnalysisID  *string       `gorm:"type:uuid;index" json:"analysisId"`
	Analysis    Analysis      `gorm:"foreignKey:AnalysisID" json:"-"`
	ReleaseID   *string       `gorm:"type:uuid;index" json:"releaseId"`
	Release     Release       `gorm:"foreignKey:ReleaseID" json:"-"`
	PackageID   string        `gorm:"type:uuid;not null;index" json:"packageId"`
	Package     Package       `gorm:"foreignKey:PackageID" json:"-"`
	Severity    AlertSeverity `gorm:"not null;type:varchar(20)" json:"severity"`
	Status      AlertStatus   `gorm:"not null;type:varchar(20);default:'new'" json:"status"`
	Message     string        `gorm:"type:text" json:"message"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
}

func (Alert) TableName() string { return "alerts" }

// AlertNote is the GORM model for alert notes/comments.
type AlertNote struct {
	ID          string    `gorm:"type:uuid;primarykey" json:"id"`
	AlertID     string    `gorm:"type:uuid;not null;index" json:"alertId"`
	Alert       Alert     `gorm:"foreignKey:AlertID" json:"-"`
	WorkspaceID string    `gorm:"type:uuid;not null;index" json:"workspaceId"`
	UserID      string    `gorm:"type:uuid;not null;index" json:"userId"`
	UserEmail   string    `gorm:"type:varchar(255)" json:"userEmail"`
	Content     string    `gorm:"not null;type:text" json:"content"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (AlertNote) TableName() string { return "alert_notes" }

// --- User Models ---

// User is the GORM model for system users.
type User struct {
	ID                 string         `gorm:"type:uuid;primarykey" json:"id"`
	Email              string         `gorm:"uniqueIndex;not null;type:varchar(255)" json:"email"`
	PasswordHash       string         `gorm:"type:varchar(255)" json:"-"`
	FirstName          string         `gorm:"type:varchar(100)" json:"firstName"`
	LastName           string         `gorm:"type:varchar(100)" json:"lastName"`
	IsActive           bool           `gorm:"not null;default:true" json:"isActive"`
	IsSuperAdmin       bool           `gorm:"column:is_superadmin;not null;default:false" json:"isSuperAdmin"`
	DeactivatedAt      *time.Time     `json:"deactivatedAt,omitempty"`
	EmailVerified      bool           `gorm:"not null;default:false" json:"emailVerified"`
	MustChangePassword bool           `gorm:"not null;default:false" json:"mustChangePassword"`
	AuthProvider       string         `gorm:"not null;type:varchar(20);default:'local'" json:"authProvider"`
	LastLoginAt        *time.Time     `json:"lastLoginAt,omitempty"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

// --- RefreshToken Models ---

// RefreshToken is the GORM model for refresh tokens.
type RefreshToken struct {
	ID        string    `gorm:"type:uuid;primarykey" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string    `gorm:"uniqueIndex;not null;column:token_hash;type:varchar(255)" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// --- APIKey Models ---

// APIKey is the GORM model for API keys.
type APIKey struct {
	ID          string         `gorm:"type:uuid;primarykey" json:"id"`
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
	ID          string         `gorm:"type:uuid;primarykey" json:"id"`
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
	ID          string    `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID string    `gorm:"type:uuid;not null;uniqueIndex:idx_workspace_user" json:"workspaceId"`
	UserID      string    `gorm:"type:uuid;not null;uniqueIndex:idx_workspace_user" json:"userId"`
	RoleID      string    `gorm:"type:uuid;not null" json:"roleId"`
	Role        Role      `gorm:"foreignKey:RoleID" json:"role"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	JoinedAt    time.Time `gorm:"not null" json:"joinedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (WorkspaceMember) TableName() string { return "workspace_members" }

// --- Role & Permission Models ---

// Role is the GORM model for roles.
type Role struct {
	ID          string       `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID string       `gorm:"type:uuid;not null;index" json:"workspaceId"`
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
	ID       string `gorm:"type:uuid;primarykey" json:"id"`
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

// --- Invitation Models ---

// Invitation is the GORM model for workspace invitations.
type Invitation struct {
	ID          string     `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID string     `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Email       string     `gorm:"not null;type:varchar(255)" json:"email"`
	RoleID      string     `gorm:"type:uuid;not null" json:"roleId"`
	TokenHash   string     `gorm:"uniqueIndex;not null;type:varchar(255);column:token_hash" json:"-"`
	InvitedBy   string     `gorm:"type:uuid;not null" json:"invitedBy"`
	ExpiresAt   time.Time  `gorm:"not null" json:"expiresAt"`
	AcceptedAt  *time.Time `json:"acceptedAt,omitempty"`
	DeclinedAt  *time.Time `json:"declinedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (Invitation) TableName() string { return "invitations" }

// --- AuditLog Models ---

// AuditLog is the GORM model for audit log entries.
type AuditLog struct {
	ID            string    `gorm:"type:uuid;primarykey" json:"id"`
	UserID        *string   `gorm:"type:uuid;index" json:"userId"`
	UserEmail     string    `gorm:"type:varchar(255)" json:"userEmail"`
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
	ID          string    `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID *string   `gorm:"type:uuid;index;uniqueIndex:idx_settings_workspace_key" json:"workspaceId"`
	Key         string    `gorm:"not null;type:varchar(100);uniqueIndex:idx_settings_workspace_key" json:"key"`
	Value       string    `gorm:"type:text" json:"value"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	ID          string                  `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID string                  `gorm:"type:uuid;not null;index" json:"workspaceId"`
	Name        string                  `gorm:"not null;type:varchar(100)" json:"name"`
	Type        NotificationChannelType `gorm:"not null;type:varchar(20)" json:"type"`
	Config      string                  `gorm:"type:text" json:"config"`
	IsActive    bool                    `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time               `json:"createdAt"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

func (NotificationChannel) TableName() string { return "notification_channels" }

// NotificationRule is the GORM model for notification rules.
type NotificationRule struct {
	ID          string    `gorm:"type:uuid;primarykey" json:"id"`
	WorkspaceID string    `gorm:"type:uuid;not null;index" json:"workspaceId"`
	ChannelID   string    `gorm:"type:uuid;not null;index" json:"channelId"`
	Severity    string    `gorm:"type:varchar(20)" json:"severity"`
	IsActive    bool      `gorm:"not null;default:true" json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (NotificationRule) TableName() string { return "notification_rules" }

// Notification is the GORM model for in-app notifications.
type Notification struct {
	ID            string    `gorm:"type:uuid;primarykey" json:"id"`
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
	ID        string     `gorm:"type:uuid;primarykey" json:"id"`
	UserID    string     `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string     `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

// EmailVerificationToken is the GORM model for email verification tokens.
type EmailVerificationToken struct {
	ID        string    `gorm:"type:uuid;primarykey" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash string    `gorm:"not null;uniqueIndex" json:"-"`
	ExpiresAt time.Time `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

func (EmailVerificationToken) TableName() string { return "email_verification_tokens" }

// --- Session Models ---

// Session is the GORM model for user sessions.
type Session struct {
	ID           string    `gorm:"type:uuid;primarykey" json:"id"`
	UserID       string    `gorm:"type:uuid;not null;index" json:"userId"`
	TokenHash    string    `gorm:"not null;uniqueIndex;type:varchar(255)" json:"-"`
	IPAddress    string    `gorm:"type:varchar(45)" json:"ipAddress"`
	UserAgent    string    `gorm:"type:varchar(512)" json:"userAgent"`
	AuthProvider string    `gorm:"type:varchar(20);not null;default:'local'" json:"authProvider"`
	CreatedAt    time.Time `json:"createdAt"`
	LastActive   time.Time `gorm:"not null" json:"lastActive"`
	ExpiresAt    time.Time `gorm:"not null" json:"expiresAt"`
}

func (Session) TableName() string { return "sessions" }

// --- SSO Models ---

// SSOConfig is the GORM model for platform-level SSO configurations.
type SSOConfig struct {
	ID                   string    `gorm:"type:uuid;primarykey" json:"id"`
	Provider             string    `gorm:"not null;type:varchar(20)" json:"provider"`
	DisplayName          string    `gorm:"type:varchar(100);not null;default:''" json:"displayName"`
	IsEnabled            bool      `gorm:"not null;default:false" json:"isEnabled"`
	AllowedDomains       string    `gorm:"type:text" json:"allowedDomains"`
	AutoCreateUser       bool      `gorm:"not null;default:false" json:"autoCreateUser"`
	SAMLEntityID         string    `gorm:"type:text" json:"samlEntityId"`
	SAMLSsoURL           string    `gorm:"type:text" json:"samlSsoUrl"`
	SAMLCertificate      string    `gorm:"type:text" json:"samlCertificate"`
	SAMLAttrEmail        string    `gorm:"type:varchar(100)" json:"samlAttrEmail"`
	SAMLAttrFirstName    string    `gorm:"type:varchar(100)" json:"samlAttrFirstName"`
	SAMLAttrLastName     string    `gorm:"type:varchar(100)" json:"samlAttrLastName"`
	OAuthClientID        string    `gorm:"type:varchar(255);column:oauth_client_id" json:"oauthClientId"`
	OAuthClientSecretEnc string    `gorm:"type:text;column:oauth_client_secret_enc" json:"-"`
	GoogleHostedDomain   string    `gorm:"type:varchar(255);column:google_hosted_domain" json:"googleHostedDomain"`
	GitHubOrgs           string    `gorm:"type:text;column:github_orgs" json:"gitHubOrgs"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

func (SSOConfig) TableName() string { return "sso_configs" }

// UserIdentity is the GORM model for linked external user identities.
type UserIdentity struct {
	ID             string    `gorm:"type:uuid;primarykey" json:"id"`
	UserID         string    `gorm:"type:uuid;not null;index" json:"userId"`
	Provider       string    `gorm:"not null;type:varchar(20)" json:"provider"`
	ProviderUserID string    `gorm:"not null;type:varchar(255)" json:"providerUserId"`
	Email          string    `gorm:"not null;type:varchar(255)" json:"email"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (UserIdentity) TableName() string { return "user_identities" }

// SSOState is the GORM model for pending SSO authentication state (CSRF).
type SSOState struct {
	ID            string    `gorm:"type:uuid;primarykey"`
	ConfigID      string    `gorm:"type:uuid;not null"`
	State         string    `gorm:"not null;uniqueIndex;type:varchar(255)"`
	UserID        *string   `gorm:"type:uuid"`
	Provider      string    `gorm:"not null;type:varchar(20)"`
	CallbackURL   string    `gorm:"type:text"`
	Mode          string    `gorm:"not null;type:varchar(20)"`
	CodeVerifier  string    `gorm:"type:text" json:"-"`
	SAMLRequestID string    `gorm:"type:text;column:saml_request_id"`
	ExpiresAt     time.Time `gorm:"not null"`
	CreatedAt     time.Time
}

func (SSOState) TableName() string { return "sso_states" }

// --- BeforeCreate hooks (UUIDv7 via pkg/id) ---

func (m *Package) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Release) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Diff) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Analysis) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Alert) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *AlertNote) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *User) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *APIKey) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Workspace) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *WorkspaceMember) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Role) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Permission) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Invitation) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Setting) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *NotificationChannel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *NotificationRule) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Notification) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *PasswordResetToken) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *EmailVerificationToken) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *Session) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *SSOConfig) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *UserIdentity) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

func (m *SSOState) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = id.New()
	}
	return nil
}

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
	&SSOConfig{}, &UserIdentity{}, &SSOState{},
}
