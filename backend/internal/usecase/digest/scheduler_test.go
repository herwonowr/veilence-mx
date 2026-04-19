package digest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&persistent.Package{},
		&persistent.Release{},
		&persistent.Diff{},
		&persistent.Analysis{},
		&persistent.Alert{},
		&persistent.Setting{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func newTestScheduler(db *gorm.DB) *Scheduler {
	return New(persistent.NewDigestRepo(db), notifications.SMTPConfig{}, Config{CheckInterval: time.Minute})
}

func TestGenerateDigest_Empty(t *testing.T) {
	db := setupTestDB(t)
	s := newTestScheduler(db)

	digest, err := s.GenerateDigest(context.Background(), 1, "daily", time.Now())
	require.NoError(t, err)

	assert.Equal(t, "last 24 hours", digest.Period)
	assert.Equal(t, int64(0), digest.NewAlertsCount)
	assert.Equal(t, int64(0), digest.PackagesAnalyzed)
	assert.Empty(t, digest.ClassificationBreakdown)
	assert.Empty(t, digest.TopAlerts)
}

func TestGenerateDigest_WithData(t *testing.T) {
	db := setupTestDB(t)
	s := newTestScheduler(db)
	now := time.Now()

	// Create test data for org 1
	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 1}
	db.Create(&pkg)

	rel1 := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := persistent.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)

	diff := persistent.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"}
	db.Create(&diff)

	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)

	alert1 := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, WorkspaceID: 1, Severity: "critical", Status: "new", Message: "Critical alert"}
	db.Create(&alert1)
	alert2 := persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, WorkspaceID: 1, Severity: "medium", Status: "new", Message: "Medium alert"}
	db.Create(&alert2)

	digest, err := s.GenerateDigest(context.Background(), 1, "daily", now)
	require.NoError(t, err)

	assert.Equal(t, "last 24 hours", digest.Period)
	assert.Equal(t, int64(2), digest.NewAlertsCount)
	assert.Equal(t, int64(1), digest.PackagesAnalyzed) // 1 package with releases
	assert.NotEmpty(t, digest.ClassificationBreakdown)
	assert.Equal(t, int64(1), digest.ClassificationBreakdown["malicious"])
	assert.Len(t, digest.TopAlerts, 2)
	// Critical should be first
	assert.Equal(t, "critical", digest.TopAlerts[0].Severity)
	assert.Equal(t, "requests", digest.TopAlerts[0].PackageName)
}

func TestGenerateDigest_WeeklyPeriod(t *testing.T) {
	db := setupTestDB(t)
	s := newTestScheduler(db)

	digest, err := s.GenerateDigest(context.Background(), 1, "weekly", time.Now())
	require.NoError(t, err)

	assert.Equal(t, "last 7 days", digest.Period)
}

func TestGenerateDigest_WorkspaceScoping(t *testing.T) {
	db := setupTestDB(t)
	s := newTestScheduler(db)
	now := time.Now()

	// Create alerts for org 1 and org 2
	pkg1 := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 1}
	db.Create(&pkg1)
	pkg2 := persistent.Package{Name: "express", Ecosystem: "npm", WorkspaceID: 2}
	db.Create(&pkg2)

	rel1 := persistent.Release{PackageID: pkg1.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := persistent.Release{PackageID: pkg2.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel2)

	diff1 := persistent.Diff{ReleaseID: rel1.ID, DiffContent: "d1"}
	db.Create(&diff1)
	diff2 := persistent.Diff{ReleaseID: rel2.ID, DiffContent: "d2"}
	db.Create(&diff2)

	analysis1 := persistent.Analysis{DiffID: diff1.ID, Classification: "malicious", Confidence: 0.9, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis1)
	analysis2 := persistent.Analysis{DiffID: diff2.ID, Classification: "benign", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis2)

	db.Create(&persistent.Alert{AnalysisID: analysis1.ID, PackageID: pkg1.ID, WorkspaceID: 1, Severity: "critical", Status: "new", Message: "Org 1 alert"})
	db.Create(&persistent.Alert{AnalysisID: analysis2.ID, PackageID: pkg2.ID, WorkspaceID: 2, Severity: "low", Status: "new", Message: "Org 2 alert"})

	// Digest for org 1 should only see org 1's data
	digest1, err := s.GenerateDigest(context.Background(), 1, "daily", now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), digest1.NewAlertsCount)
	assert.Equal(t, "critical", digest1.TopAlerts[0].Severity)

	// Digest for org 2 should only see org 2's data
	digest2, err := s.GenerateDigest(context.Background(), 2, "daily", now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), digest2.NewAlertsCount)
	assert.Equal(t, "low", digest2.TopAlerts[0].Severity)
}

func TestGenerateDigest_TopAlertsLimit(t *testing.T) {
	db := setupTestDB(t)
	s := newTestScheduler(db)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", WorkspaceID: 1}
	db.Create(&pkg)
	rel := persistent.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel)
	diff := persistent.Diff{ReleaseID: rel.ID, DiffContent: "d"}
	db.Create(&diff)
	analysis := persistent.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.9, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)

	// Create 7 alerts
	for i := range 7 {
		severity := "medium"
		if i < 3 {
			severity = "critical"
		}
		db.Create(&persistent.Alert{AnalysisID: analysis.ID, PackageID: pkg.ID, WorkspaceID: 1, Severity: persistent.AlertSeverity(severity), Status: "new", Message: "Alert"})
	}

	digest, err := s.GenerateDigest(context.Background(), 1, "daily", time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(7), digest.NewAlertsCount)
	assert.Len(t, digest.TopAlerts, 5) // Limited to 5
}

func TestFormatDigestText(t *testing.T) {
	d := &DigestContent{
		Period:           "last 24 hours",
		NewAlertsCount:   3,
		PackagesAnalyzed: 5,
		ClassificationBreakdown: map[string]int64{
			"malicious":  1,
			"suspicious": 2,
		},
		TopAlerts: []entity.DigestTopAlert{
			{ID: 1, PackageName: "requests", Severity: "critical", Message: "Malicious code detected", CreatedAt: time.Now()},
			{ID: 2, PackageName: "flask", Severity: "medium", Message: "Suspicious pattern found", CreatedAt: time.Now()},
		},
	}

	text := FormatDigestText(d)
	assert.Contains(t, text, "last 24 hours")
	assert.Contains(t, text, "New Alerts:        3")
	assert.Contains(t, text, "Packages Analyzed: 5")
	assert.Contains(t, text, "malicious")
	assert.Contains(t, text, "suspicious")
	assert.Contains(t, text, "[CRITICAL] requests")
	assert.Contains(t, text, "[MEDIUM] flask")
	assert.Contains(t, text, "Veilence-MX email digest")
}

func TestFormatDigestText_Empty(t *testing.T) {
	d := &DigestContent{
		Period:                  "last 24 hours",
		NewAlertsCount:          0,
		PackagesAnalyzed:        0,
		ClassificationBreakdown: map[string]int64{},
		TopAlerts:               nil,
	}

	text := FormatDigestText(d)
	assert.Contains(t, text, "No new alerts in this period")
	assert.Contains(t, text, "All clear!")
}

func TestIsDue(t *testing.T) {
	db := setupTestDB(t)
	s := newTestScheduler(db)
	now := time.Now()

	// Never sent — should be due
	assert.True(t, s.isDue(1, "daily", now))
	assert.True(t, s.isDue(1, "weekly", now))

	// Sent 12 hours ago — daily should not be due, weekly should not be due
	s.lastSentAt[1] = now.Add(-12 * time.Hour)
	assert.False(t, s.isDue(1, "daily", now))
	assert.False(t, s.isDue(1, "weekly", now))

	// Sent 25 hours ago — daily should be due, weekly should not be due
	s.lastSentAt[1] = now.Add(-25 * time.Hour)
	assert.True(t, s.isDue(1, "daily", now))
	assert.False(t, s.isDue(1, "weekly", now))

	// Sent 8 days ago — both should be due
	s.lastSentAt[1] = now.Add(-8 * 24 * time.Hour)
	assert.True(t, s.isDue(1, "daily", now))
	assert.True(t, s.isDue(1, "weekly", now))
}

func TestParseRecipients(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single", "a@b.com", []string{"a@b.com"}},
		{"multiple", "a@b.com,c@d.com", []string{"a@b.com", "c@d.com"}},
		{"with spaces", " a@b.com , c@d.com ", []string{"a@b.com", "c@d.com"}},
		{"empty entries", "a@b.com,,c@d.com,", []string{"a@b.com", "c@d.com"}},
		{"empty string", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRecipients(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "short", truncate("short", 10))
	assert.Equal(t, "1234567...", truncate("1234567890123", 10))
	assert.Equal(t, "", truncate("", 10))
}

func TestCapitalize(t *testing.T) {
	assert.Equal(t, "Daily", capitalize("daily"))
	assert.Equal(t, "Weekly", capitalize("weekly"))
	assert.Equal(t, "", capitalize(""))
}
