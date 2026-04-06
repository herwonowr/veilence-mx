package testutil

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// ---------------------------------------------------------------------------
// User fixtures
// ---------------------------------------------------------------------------

// UserFixture holds the options for creating a test user.
type UserFixture struct {
	Email         string
	Password      string
	FirstName     string
	LastName      string
	IsActive      bool
	EmailVerified bool
}

// DefaultUserFixture returns a UserFixture with sensible defaults.
func DefaultUserFixture() UserFixture {
	return UserFixture{
		Email:         "test@example.com",
		Password:      "Password123",
		FirstName:     "Test",
		LastName:      "User",
		IsActive:      true,
		EmailVerified: false,
	}
}

// CreateUser creates a user in the database and returns the GORM model.
// The password is hashed before storage. Each call with the same email will
// fail, so callers should use unique emails (or use CreateUserN).
func CreateUser(t *testing.T, db *gorm.DB, opts ...func(*UserFixture)) *models.User {
	t.Helper()
	f := DefaultUserFixture()
	for _, o := range opts {
		o(&f)
	}
	user := &models.User{
		Email:         f.Email,
		FirstName:     f.FirstName,
		LastName:      f.LastName,
		IsActive:      f.IsActive,
		EmailVerified: f.EmailVerified,
	}
	require.NoError(t, user.HashPassword(f.Password))
	require.NoError(t, db.Create(user).Error)
	return user
}

// CreateUserN creates N users with emails user-0@example.com …
// user-(N-1)@example.com and returns them all.
func CreateUserN(t *testing.T, db *gorm.DB, n int) []*models.User {
	t.Helper()
	users := make([]*models.User, n)
	for i := range n {
		users[i] = CreateUser(t, db, func(f *UserFixture) {
			f.Email = fmt.Sprintf("user-%d@example.com", i)
		})
	}
	return users
}

// ---------------------------------------------------------------------------
// Organization fixtures
// ---------------------------------------------------------------------------

// OrgFixture holds the options for creating a test organization.
type OrgFixture struct {
	Name        string
	Slug        string
	Description string
	OwnerID     uint
}

// DefaultOrgFixture returns an OrgFixture with sensible defaults.
func DefaultOrgFixture() OrgFixture {
	return OrgFixture{
		Name:        "Test Org",
		Slug:        "test-org",
		Description: "A test organization",
		OwnerID:     1,
	}
}

// CreateOrg creates an organization in the database.
func CreateOrg(t *testing.T, db *gorm.DB, opts ...func(*OrgFixture)) *models.Organization {
	t.Helper()
	f := DefaultOrgFixture()
	for _, o := range opts {
		o(&f)
	}
	org := &models.Organization{
		Name:        f.Name,
		Slug:        f.Slug,
		Description: f.Description,
		OwnerID:     f.OwnerID,
		IsActive:    true,
	}
	require.NoError(t, db.Create(org).Error)
	return org
}

// ---------------------------------------------------------------------------
// Package fixtures
// ---------------------------------------------------------------------------

// PkgFixture holds the options for creating a test package.
type PkgFixture struct {
	OrgID         uint
	Name          string
	Registry      models.Registry
	LatestVersion string
	IsCustom      bool
}

// DefaultPkgFixture returns a PkgFixture with sensible defaults.
func DefaultPkgFixture() PkgFixture {
	return PkgFixture{
		OrgID:    1,
		Name:     "requests",
		Registry: models.RegistryPyPI,
		IsCustom: false,
	}
}

// CreatePackage creates a package in the database.
func CreatePackage(t *testing.T, db *gorm.DB, opts ...func(*PkgFixture)) *models.Package {
	t.Helper()
	f := DefaultPkgFixture()
	for _, o := range opts {
		o(&f)
	}
	pkg := &models.Package{
		OrgID:         f.OrgID,
		Name:          f.Name,
		Registry:      f.Registry,
		LatestVersion: f.LatestVersion,
		IsCustom:      f.IsCustom,
	}
	require.NoError(t, db.Create(pkg).Error)
	return pkg
}

// CreatePackageN creates N packages named pkg-0 … pkg-(N-1).
func CreatePackageN(t *testing.T, db *gorm.DB, orgID uint, n int) []*models.Package {
	t.Helper()
	pkgs := make([]*models.Package, n)
	for i := range n {
		pkgs[i] = CreatePackage(t, db, func(f *PkgFixture) {
			f.OrgID = orgID
			f.Name = fmt.Sprintf("pkg-%d", i)
		})
	}
	return pkgs
}

// ---------------------------------------------------------------------------
// Release fixtures
// ---------------------------------------------------------------------------

// ReleaseFixture holds the options for creating a test release.
type ReleaseFixture struct {
	PackageID uint
	Version   string
	Status    models.ReleaseStatus
}

// DefaultReleaseFixture returns a ReleaseFixture with sensible defaults.
func DefaultReleaseFixture() ReleaseFixture {
	return ReleaseFixture{
		PackageID: 1,
		Version:   "1.0.0",
		Status:    models.ReleaseStatusCompleted,
	}
}

// CreateRelease creates a release in the database.
func CreateRelease(t *testing.T, db *gorm.DB, opts ...func(*ReleaseFixture)) *models.Release {
	t.Helper()
	f := DefaultReleaseFixture()
	for _, o := range opts {
		o(&f)
	}
	rel := &models.Release{
		PackageID: f.PackageID,
		Version:   f.Version,
		Status:    f.Status,
	}
	require.NoError(t, db.Create(rel).Error)
	return rel
}

// ---------------------------------------------------------------------------
// Diff fixtures
// ---------------------------------------------------------------------------

// DiffFixture holds the options for creating a test diff.
type DiffFixture struct {
	ReleaseID        uint
	PrevReleaseID    uint
	DiffContent      string
	FileChangesCount int
	LinesAdded       int
	LinesRemoved     int
}

// DefaultDiffFixture returns a DiffFixture with sensible defaults.
func DefaultDiffFixture() DiffFixture {
	return DiffFixture{
		ReleaseID:        2,
		PrevReleaseID:    1,
		DiffContent:      "--- a/file.py\n+++ b/file.py\n@@ -1,3 +1,4 @@\n+import os\n def main():\n     pass\n",
		FileChangesCount: 1,
		LinesAdded:       1,
		LinesRemoved:     0,
	}
}

// CreateDiff creates a diff in the database.
func CreateDiff(t *testing.T, db *gorm.DB, opts ...func(*DiffFixture)) *models.Diff {
	t.Helper()
	f := DefaultDiffFixture()
	for _, o := range opts {
		o(&f)
	}
	diff := &models.Diff{
		ReleaseID:        f.ReleaseID,
		PrevReleaseID:    f.PrevReleaseID,
		DiffContent:      f.DiffContent,
		FileChangesCount: f.FileChangesCount,
		LinesAdded:       f.LinesAdded,
		LinesRemoved:     f.LinesRemoved,
	}
	require.NoError(t, db.Create(diff).Error)
	return diff
}

// ---------------------------------------------------------------------------
// Analysis fixtures
// ---------------------------------------------------------------------------

// AnalysisFixture holds the options for creating a test analysis.
type AnalysisFixture struct {
	DiffID         uint
	Classification models.Classification
	Confidence     float64
	Reasoning      string
	ModelUsed      string
	AnalyzerType   models.AnalyzerType
}

// DefaultAnalysisFixture returns an AnalysisFixture with sensible defaults.
func DefaultAnalysisFixture() AnalysisFixture {
	return AnalysisFixture{
		DiffID:         1,
		Classification: models.ClassificationBenign,
		Confidence:     0.95,
		Reasoning:      "Standard library usage, no suspicious patterns.",
		ModelUsed:      "claude-sonnet-4-20250514",
		AnalyzerType:   models.AnalyzerTypeAPI,
	}
}

// CreateAnalysis creates an analysis in the database.
func CreateAnalysis(t *testing.T, db *gorm.DB, opts ...func(*AnalysisFixture)) *models.Analysis {
	t.Helper()
	f := DefaultAnalysisFixture()
	for _, o := range opts {
		o(&f)
	}
	analysis := &models.Analysis{
		DiffID:         f.DiffID,
		Classification: f.Classification,
		Confidence:     f.Confidence,
		Reasoning:      f.Reasoning,
		ModelUsed:      f.ModelUsed,
		AnalyzerType:   f.AnalyzerType,
	}
	require.NoError(t, db.Create(analysis).Error)
	return analysis
}

// ---------------------------------------------------------------------------
// Alert fixtures
// ---------------------------------------------------------------------------

// AlertFixture holds the options for creating a test alert.
type AlertFixture struct {
	OrgID      uint
	AnalysisID uint
	PackageID  uint
	Severity   models.AlertSeverity
	Status     models.AlertStatus
	Message    string
}

// DefaultAlertFixture returns an AlertFixture with sensible defaults.
func DefaultAlertFixture() AlertFixture {
	return AlertFixture{
		OrgID:      1,
		AnalysisID: 1,
		PackageID:  1,
		Severity:   models.AlertSeverityHigh,
		Status:     models.AlertStatusNew,
		Message:    "Suspicious code injection detected.",
	}
}

// CreateAlert creates an alert in the database.
func CreateAlert(t *testing.T, db *gorm.DB, opts ...func(*AlertFixture)) *models.Alert {
	t.Helper()
	f := DefaultAlertFixture()
	for _, o := range opts {
		o(&f)
	}
	alert := &models.Alert{
		OrgID:      f.OrgID,
		AnalysisID: f.AnalysisID,
		PackageID:  f.PackageID,
		Severity:   f.Severity,
		Status:     f.Status,
		Message:    f.Message,
	}
	require.NoError(t, db.Create(alert).Error)
	return alert
}

// ---------------------------------------------------------------------------
// Setting fixtures
// ---------------------------------------------------------------------------

// CreateSetting creates a setting in the database.
func CreateSetting(t *testing.T, db *gorm.DB, orgID uint, key, value string) *models.Setting {
	t.Helper()
	s := &models.Setting{OrgID: orgID, Key: key, Value: value}
	require.NoError(t, db.Create(s).Error)
	return s
}

// ---------------------------------------------------------------------------
// Notification fixtures
// ---------------------------------------------------------------------------

// CreateNotificationChannel creates a notification channel in the database.
func CreateNotificationChannel(t *testing.T, db *gorm.DB, orgID uint, name string, channelType models.NotificationChannelType, config string) *models.NotificationChannel {
	t.Helper()
	ch := &models.NotificationChannel{
		OrgID:    orgID,
		Name:     name,
		Type:     channelType,
		Config:   config,
		IsActive: true,
	}
	require.NoError(t, db.Create(ch).Error)
	return ch
}

// CreateNotificationRule creates a notification rule in the database.
func CreateNotificationRule(t *testing.T, db *gorm.DB, orgID, channelID uint, severity string) *models.NotificationRule {
	t.Helper()
	rule := &models.NotificationRule{
		OrgID:     orgID,
		ChannelID: channelID,
		Severity:  severity,
		IsActive:  true,
	}
	require.NoError(t, db.Create(rule).Error)
	return rule
}

// CreateNotification creates an in-app notification in the database.
func CreateNotification(t *testing.T, db *gorm.DB, orgID, userID, channelID uint, title, message string, isRead bool) *models.Notification {
	t.Helper()
	n := &models.Notification{
		OrgID:     orgID,
		UserID:    userID,
		ChannelID: channelID,
		Title:     title,
		Message:   message,
		IsRead:    isRead,
		SentAt:    time.Now(),
	}
	require.NoError(t, db.Create(n).Error)
	return n
}

// ---------------------------------------------------------------------------
// AuditLog fixtures
// ---------------------------------------------------------------------------

// CreateAuditLog creates an audit log entry in the database.
func CreateAuditLog(t *testing.T, db *gorm.DB, userID, orgID uint, action, resource string, resourceID uint) *models.AuditLog {
	t.Helper()
	entry := &models.AuditLog{
		UserID:     userID,
		OrgID:      orgID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
	}
	require.NoError(t, db.Create(entry).Error)
	return entry
}

// ---------------------------------------------------------------------------
// Domain entity helpers (for mock-based tests)
// ---------------------------------------------------------------------------

// NewDomainUser creates a domain.User with sensible defaults. Fields can be
// overridden after creation.
func NewDomainUser(id uint, email string) *domain.User {
	return &domain.User{
		ID:        id,
		Email:     email,
		FirstName: "Test",
		LastName:  "User",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// NewDomainOrg creates a domain.Organization with sensible defaults.
func NewDomainOrg(id, ownerID uint, name, slug string) *domain.Organization {
	return &domain.Organization{
		ID:       id,
		Name:     name,
		Slug:     slug,
		OwnerID:  ownerID,
		IsActive: true,
	}
}

// NewDomainPackage creates a domain.Package with sensible defaults.
func NewDomainPackage(id, orgID uint, name string, registry domain.Registry) *domain.Package {
	return &domain.Package{
		ID:       id,
		OrgID:    orgID,
		Name:     name,
		Registry: registry,
	}
}

// NewDomainRelease creates a domain.Release with sensible defaults.
func NewDomainRelease(id, packageID uint, version string) *domain.Release {
	return &domain.Release{
		ID:        id,
		PackageID: packageID,
		Version:   version,
		Status:    domain.ReleaseStatusCompleted,
	}
}

// NewDomainDiff creates a domain.Diff with sensible defaults.
func NewDomainDiff(id, releaseID, prevReleaseID uint) *domain.Diff {
	return &domain.Diff{
		ID:               id,
		ReleaseID:        releaseID,
		PrevReleaseID:    prevReleaseID,
		DiffContent:      "test diff content",
		FileChangesCount: 1,
		LinesAdded:       5,
		LinesRemoved:     2,
	}
}

// NewDomainAnalysis creates a domain.Analysis with sensible defaults.
func NewDomainAnalysis(id, diffID uint, classification domain.Classification) *domain.Analysis {
	return &domain.Analysis{
		ID:             id,
		DiffID:         diffID,
		Classification: classification,
		Confidence:     0.92,
		Reasoning:      "Test analysis reasoning",
		ModelUsed:      "test-model",
		AnalyzerType:   domain.AnalyzerTypeAPI,
	}
}

// NewDomainAlert creates a domain.Alert with sensible defaults.
func NewDomainAlert(id, orgID, analysisID, packageID uint, severity domain.AlertSeverity) *domain.Alert {
	return &domain.Alert{
		ID:         id,
		OrgID:      orgID,
		AnalysisID: analysisID,
		PackageID:  packageID,
		Severity:   severity,
		Status:     domain.AlertStatusNew,
		Message:    "Test alert message",
	}
}

// NewDomainNotificationChannel creates a domain.NotificationChannel with defaults.
func NewDomainNotificationChannel(id, orgID uint, name string, channelType domain.NotificationChannelType) *domain.NotificationChannel {
	return &domain.NotificationChannel{
		ID:       id,
		OrgID:    orgID,
		Name:     name,
		Type:     channelType,
		Config:   `{"recipients":"test@example.com"}`,
		IsActive: true,
	}
}

// NewDomainNotificationRule creates a domain.NotificationRule with defaults.
func NewDomainNotificationRule(id, orgID, channelID uint, severity string) *domain.NotificationRule {
	return &domain.NotificationRule{
		ID:        id,
		OrgID:     orgID,
		ChannelID: channelID,
		Severity:  severity,
		IsActive:  true,
	}
}
