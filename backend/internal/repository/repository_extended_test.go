package repository_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"
)

// =========================================================================
// ReleaseRepo
// =========================================================================

func TestReleaseRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewReleaseRepo(db)

	// Prerequisite: create a package
	pkg := &models.Package{OrgID: 1, Name: "requests", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	release := &domain.Release{
		PackageID:   pkg.ID,
		Version:     "1.0.0",
		PublishedAt: time.Now(),
		TarballURL:  "https://example.com/requests-1.0.0.tar.gz",
		SHA256:      "abc123",
		Status:      domain.ReleaseStatusPending,
	}
	err := repo.Create(ctx, release)
	require.NoError(t, err)
	assert.NotZero(t, release.ID)

	found, err := repo.FindByID(ctx, release.ID)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", found.Version)
	assert.Equal(t, pkg.ID, found.PackageID)
	assert.Equal(t, domain.ReleaseStatusPending, found.Status)
}

func TestReleaseRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewReleaseRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReleaseRepo_FindByPackageID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewReleaseRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "flask", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	for i := 0; i < 3; i++ {
		r := &domain.Release{
			PackageID:   pkg.ID,
			Version:     "1.0." + string(rune('0'+i)),
			PublishedAt: time.Now(),
			Status:      domain.ReleaseStatusCompleted,
		}
		require.NoError(t, repo.Create(ctx, r))
	}

	releases, total, err := repo.FindByPackageID(ctx, pkg.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, releases, 3)
}

func TestReleaseRepo_FindByPackageID_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewReleaseRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "numpy", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	for i := 0; i < 5; i++ {
		r := &domain.Release{
			PackageID:   pkg.ID,
			Version:     "2.0." + string(rune('0'+i)),
			PublishedAt: time.Now(),
			Status:      domain.ReleaseStatusCompleted,
		}
		require.NoError(t, repo.Create(ctx, r))
	}

	releases, total, err := repo.FindByPackageID(ctx, pkg.ID, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, releases, 2)

	// Page 2
	releases2, _, err := repo.FindByPackageID(ctx, pkg.ID, 2, 2)
	require.NoError(t, err)
	assert.Len(t, releases2, 2)
}

func TestReleaseRepo_FindByPackageID_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewReleaseRepo(db)

	releases, total, err := repo.FindByPackageID(ctx, 999, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Len(t, releases, 0)
}

func TestReleaseRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewReleaseRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "django", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	release := &domain.Release{
		PackageID:   pkg.ID,
		Version:     "3.0.0",
		PublishedAt: time.Now(),
		Status:      domain.ReleaseStatusPending,
	}
	require.NoError(t, repo.Create(ctx, release))

	release.Status = domain.ReleaseStatusCompleted
	release.ErrorMessage = ""
	err := repo.Update(ctx, release)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, release.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ReleaseStatusCompleted, found.Status)
}

// =========================================================================
// AlertRepo
// =========================================================================

func TestAlertRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAlertRepo(db)

	alert := &domain.Alert{
		OrgID:    1,
		PackageID: 1,
		AnalysisID: 1,
		Severity: domain.AlertSeverityHigh,
		Status:   domain.AlertStatusNew,
		Message:  "Suspicious code injection detected",
	}
	err := repo.Create(ctx, alert)
	require.NoError(t, err)
	assert.NotZero(t, alert.ID)

	found, err := repo.FindByID(ctx, alert.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.AlertSeverityHigh, found.Severity)
	assert.Equal(t, domain.AlertStatusNew, found.Status)
	assert.Equal(t, "Suspicious code injection detected", found.Message)
}

func TestAlertRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAlertRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAlertRepo_FindByOrgID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAlertRepo(db)

	// Create prerequisite packages so the JOIN succeeds
	pkg1 := &models.Package{OrgID: 1, Name: "requests", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg1).Error)
	pkg2 := &models.Package{OrgID: 2, Name: "express", Ecosystem: "npm"}
	require.NoError(t, db.Create(pkg2).Error)

	// Create alerts for two orgs
	for i, orgID := range []uint{1, 1, 2} {
		pkgID := pkg1.ID
		if orgID == 2 {
			pkgID = pkg2.ID
		}
		_ = i
		a := &domain.Alert{
			OrgID: orgID, PackageID: pkgID, AnalysisID: 1,
			Severity: domain.AlertSeverityMedium,
			Status:   domain.AlertStatusNew,
			Message:  "test alert",
		}
		require.NoError(t, repo.Create(ctx, a))
	}

	alerts, total, err := repo.FindByOrgID(ctx, 1, 1, 10, "created_at DESC", domain.AlertFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, alerts, 2)
}

func TestAlertRepo_FindByOrgID_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAlertRepo(db)

	// Create prerequisite package so the JOIN succeeds
	pkg := &models.Package{OrgID: 1, Name: "requests", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	severities := []domain.AlertSeverity{domain.AlertSeverityLow, domain.AlertSeverityHigh, domain.AlertSeverityCritical}
	for _, sev := range severities {
		a := &domain.Alert{
			OrgID: 1, PackageID: pkg.ID, AnalysisID: 1,
			Severity: sev,
			Status:   domain.AlertStatusNew,
			Message:  "alert " + string(sev),
		}
		require.NoError(t, repo.Create(ctx, a))
	}

	// Filter by severity
	highSev := domain.AlertSeverityHigh
	alerts, total, err := repo.FindByOrgID(ctx, 1, 1, 10, "created_at DESC", domain.AlertFilters{Severity: &highSev})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, alerts, 1)
	assert.Equal(t, domain.AlertSeverityHigh, alerts[0].Severity)

	// Filter by status
	newStatus := domain.AlertStatusNew
	alerts, total, err = repo.FindByOrgID(ctx, 1, 1, 10, "created_at DESC", domain.AlertFilters{Status: &newStatus})
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
}

func TestAlertRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAlertRepo(db)

	alert := &domain.Alert{
		OrgID: 1, PackageID: 1, AnalysisID: 1,
		Severity: domain.AlertSeverityHigh,
		Status:   domain.AlertStatusNew,
		Message:  "original",
	}
	require.NoError(t, repo.Create(ctx, alert))

	alert.Status = domain.AlertStatusAcknowledged
	err := repo.Update(ctx, alert)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, alert.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.AlertStatusAcknowledged, found.Status)
}

func TestAlertRepo_CountByOrgAndStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAlertRepo(db)

	// Create alerts with mixed statuses
	statuses := []domain.AlertStatus{domain.AlertStatusNew, domain.AlertStatusNew, domain.AlertStatusAcknowledged, domain.AlertStatusResolved}
	for _, status := range statuses {
		a := &domain.Alert{
			OrgID: 1, PackageID: 1, AnalysisID: 1,
			Severity: domain.AlertSeverityMedium,
			Status:   status,
			Message:  "test",
		}
		require.NoError(t, repo.Create(ctx, a))
	}

	counts, err := repo.CountByOrgAndStatus(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), counts[domain.AlertStatusNew])
	assert.Equal(t, int64(1), counts[domain.AlertStatusAcknowledged])
	assert.Equal(t, int64(1), counts[domain.AlertStatusResolved])
}

func TestAlertRepo_CountByOrgAndStatus_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAlertRepo(db)

	counts, err := repo.CountByOrgAndStatus(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, counts)
}

// =========================================================================
// SessionRepo
// =========================================================================

func TestSessionRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	session := &domain.Session{
		UserID:     1,
		TokenHash:  "session-hash-123",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		LastActive: time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	err := repo.Create(ctx, session)
	require.NoError(t, err)
	assert.NotZero(t, session.ID)

	found, err := repo.FindByID(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, uint(1), found.UserID)
	assert.Equal(t, "192.168.1.1", found.IPAddress)
}

func TestSessionRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionRepo_FindByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	session := &domain.Session{
		UserID:     1,
		TokenHash:  "unique-token-hash",
		IPAddress:  "10.0.0.1",
		UserAgent:  "TestAgent",
		LastActive: time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, session))

	found, err := repo.FindByTokenHash(ctx, "unique-token-hash")
	require.NoError(t, err)
	assert.Equal(t, session.ID, found.ID)
}

func TestSessionRepo_FindByTokenHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	_, err := repo.FindByTokenHash(ctx, "nonexistent-hash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionRepo_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	// Create 2 active sessions and 1 expired session for user 1
	for i := 0; i < 2; i++ {
		s := &domain.Session{
			UserID:     1,
			TokenHash:  "active-hash-" + string(rune('a'+i)),
			IPAddress:  "10.0.0.1",
			LastActive: time.Now(),
			ExpiresAt:  time.Now().Add(24 * time.Hour),
		}
		require.NoError(t, repo.Create(ctx, s))
	}
	// Expired session
	expired := &domain.Session{
		UserID:     1,
		TokenHash:  "expired-hash",
		IPAddress:  "10.0.0.1",
		LastActive: time.Now().Add(-48 * time.Hour),
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, expired))

	sessions, err := repo.FindByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, sessions, 2, "should only return active (non-expired) sessions")
}

func TestSessionRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	session := &domain.Session{
		UserID:     1,
		TokenHash:  "delete-me-hash",
		IPAddress:  "10.0.0.1",
		LastActive: time.Now(),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, session))

	err := repo.Delete(ctx, session.ID)
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, session.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionRepo_UpdateLastActive(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	session := &domain.Session{
		UserID:     1,
		TokenHash:  "update-active-hash",
		IPAddress:  "10.0.0.1",
		LastActive: time.Now().Add(-1 * time.Hour),
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, session))

	newActive := time.Now()
	err := repo.UpdateLastActive(ctx, session.ID, newActive)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, session.ID)
	require.NoError(t, err)
	assert.WithinDuration(t, newActive, found.LastActive, time.Second)
}

func TestSessionRepo_DeleteExpired(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	// Create 1 active and 2 expired sessions
	active := &domain.Session{
		UserID: 1, TokenHash: "still-active",
		IPAddress: "10.0.0.1", LastActive: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, active))

	for i := 0; i < 2; i++ {
		expired := &domain.Session{
			UserID: 1, TokenHash: "expired-" + string(rune('a'+i)),
			IPAddress: "10.0.0.1", LastActive: time.Now().Add(-48 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, repo.Create(ctx, expired))
	}

	deleted, err := repo.DeleteExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	// Active session should still exist
	_, err = repo.FindByID(ctx, active.ID)
	require.NoError(t, err)
}

func TestSessionRepo_CountByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	for i := 0; i < 3; i++ {
		s := &domain.Session{
			UserID: 1, TokenHash: "count-hash-" + string(rune('a'+i)),
			IPAddress: "10.0.0.1", LastActive: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		require.NoError(t, repo.Create(ctx, s))
	}

	count, err := repo.CountByUserID(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestSessionRepo_DeleteOldestByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	// Create sessions with different creation times
	var firstID uint
	for i := 0; i < 3; i++ {
		s := &domain.Session{
			UserID: 1, TokenHash: "oldest-hash-" + string(rune('a'+i)),
			IPAddress: "10.0.0.1", LastActive: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		require.NoError(t, repo.Create(ctx, s))
		if i == 0 {
			firstID = s.ID
		}
	}

	err := repo.DeleteOldestByUserID(ctx, 1)
	require.NoError(t, err)

	// The oldest (first created) should be deleted
	_, err = repo.FindByID(ctx, firstID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSessionRepo_DeleteOldestByUserID_NoSessions(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSessionRepo(db)

	// Should not error when there are no sessions to delete
	err := repo.DeleteOldestByUserID(ctx, 999)
	require.NoError(t, err)
}

// =========================================================================
// DiffRepo
// =========================================================================

func TestDiffRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDiffRepo(db)

	// Create prerequisite package and releases
	pkg := &models.Package{OrgID: 1, Name: "lodash", Ecosystem: "npm"}
	require.NoError(t, db.Create(pkg).Error)

	r1 := &models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)

	diff := &domain.Diff{
		ReleaseID:        r2.ID,
		PrevReleaseID:    r1.ID,
		DiffContent:      "--- a/index.js\n+++ b/index.js\n@@ -1 +1 @@\n-old\n+new",
		FileChangesCount: 1,
		LinesAdded:       1,
		LinesRemoved:     1,
	}
	err := repo.Create(ctx, diff)
	require.NoError(t, err)
	assert.NotZero(t, diff.ID)

	found, err := repo.FindByID(ctx, diff.ID)
	require.NoError(t, err)
	assert.Equal(t, r2.ID, found.ReleaseID)
	assert.Equal(t, r1.ID, found.PrevReleaseID)
	assert.Equal(t, 1, found.FileChangesCount)
	assert.Equal(t, 1, found.LinesAdded)
}

func TestDiffRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDiffRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDiffRepo_FindByReleaseID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDiffRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "express", Ecosystem: "npm"}
	require.NoError(t, db.Create(pkg).Error)

	r1 := &models.Release{PackageID: pkg.ID, Version: "4.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "4.1.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)

	diff := &domain.Diff{
		ReleaseID:     r2.ID,
		PrevReleaseID: r1.ID,
		DiffContent:   "diff content",
	}
	require.NoError(t, repo.Create(ctx, diff))

	diffs, err := repo.FindByReleaseID(ctx, r2.ID)
	require.NoError(t, err)
	assert.Len(t, diffs, 1)
	assert.Equal(t, diff.ID, diffs[0].ID)
}

func TestDiffRepo_FindByReleaseID_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDiffRepo(db)

	diffs, err := repo.FindByReleaseID(ctx, 999)
	require.NoError(t, err)
	assert.Len(t, diffs, 0)
}

// =========================================================================
// AnalysisRepo
// =========================================================================

func TestAnalysisRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAnalysisRepo(db)

	// Create prerequisite chain: package → release → diff
	pkg := &models.Package{OrgID: 1, Name: "react", Ecosystem: "npm"}
	require.NoError(t, db.Create(pkg).Error)
	r1 := &models.Release{PackageID: pkg.ID, Version: "17.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "18.0.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)
	diff := &models.Diff{ReleaseID: r2.ID, PrevReleaseID: r1.ID, DiffContent: "changes"}
	require.NoError(t, db.Create(diff).Error)

	analysis := &domain.Analysis{
		DiffID:         diff.ID,
		Classification: domain.ClassificationBenign,
		Confidence:     0.95,
		Reasoning:      "No suspicious patterns detected",
		ModelUsed:      "claude-3-opus",
		AnalyzerType:   domain.AnalyzerTypeAPI,
		RawResponse:    `{"result":"benign"}`,
	}
	err := repo.Create(ctx, analysis)
	require.NoError(t, err)
	assert.NotZero(t, analysis.ID)

	found, err := repo.FindByID(ctx, analysis.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.ClassificationBenign, found.Classification)
	assert.Equal(t, 0.95, found.Confidence)
	assert.Equal(t, "claude-3-opus", found.ModelUsed)
	assert.Equal(t, domain.AnalyzerTypeAPI, found.AnalyzerType)
}

func TestAnalysisRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAnalysisRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAnalysisRepo_FindByDiffID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAnalysisRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "vue", Ecosystem: "npm"}
	require.NoError(t, db.Create(pkg).Error)
	r1 := &models.Release{PackageID: pkg.ID, Version: "2.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "3.0.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)
	diff := &models.Diff{ReleaseID: r2.ID, PrevReleaseID: r1.ID, DiffContent: "vue changes"}
	require.NoError(t, db.Create(diff).Error)

	// Create 2 analyses for same diff
	for _, class := range []domain.Classification{domain.ClassificationBenign, domain.ClassificationSuspicious} {
		a := &domain.Analysis{
			DiffID:         diff.ID,
			Classification: class,
			Confidence:     0.8,
			Reasoning:      "test",
			ModelUsed:      "test-model",
			AnalyzerType:   domain.AnalyzerTypeAPI,
		}
		require.NoError(t, repo.Create(ctx, a))
	}

	analyses, err := repo.FindByDiffID(ctx, diff.ID)
	require.NoError(t, err)
	assert.Len(t, analyses, 2)
}

func TestAnalysisRepo_FindByDiffID_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAnalysisRepo(db)

	analyses, err := repo.FindByDiffID(ctx, 999)
	require.NoError(t, err)
	assert.Len(t, analyses, 0)
}

func TestAnalysisRepo_CountByDiffID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAnalysisRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "angular", Ecosystem: "npm"}
	require.NoError(t, db.Create(pkg).Error)
	r1 := &models.Release{PackageID: pkg.ID, Version: "14.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "15.0.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)
	diff := &models.Diff{ReleaseID: r2.ID, PrevReleaseID: r1.ID, DiffContent: "angular changes"}
	require.NoError(t, db.Create(diff).Error)

	// Create 3 analyses
	for i := 0; i < 3; i++ {
		a := &domain.Analysis{
			DiffID: diff.ID, Classification: domain.ClassificationBenign,
			Confidence: 0.9, ModelUsed: "test", AnalyzerType: domain.AnalyzerTypeAPI,
		}
		require.NoError(t, repo.Create(ctx, a))
	}

	count, err := repo.CountByDiffID(ctx, diff.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestAnalysisRepo_CountByDiffID_Zero(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAnalysisRepo(db)

	count, err := repo.CountByDiffID(ctx, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// =========================================================================
// OrgMemberRepo
// =========================================================================

func TestOrgMemberRepo_CreateAndFindByUserAndOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrgMemberRepo(db)

	// Create prerequisite role
	role := &models.Role{OrgID: 1, Name: "owner", Description: "Organization owner", IsSystem: true}
	require.NoError(t, db.Create(role).Error)

	member := &domain.OrgMember{
		OrgID:    1,
		UserID:   42,
		RoleID:   role.ID,
		JoinedAt: time.Now(),
	}
	err := repo.Create(ctx, member)
	require.NoError(t, err)
	assert.NotZero(t, member.ID)

	found, err := repo.FindByUserAndOrg(ctx, 42, 1)
	require.NoError(t, err)
	assert.Equal(t, uint(42), found.UserID)
	assert.Equal(t, uint(1), found.OrgID)
	assert.Equal(t, role.ID, found.RoleID)
}

func TestOrgMemberRepo_FindByUserAndOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrgMemberRepo(db)

	_, err := repo.FindByUserAndOrg(ctx, 999, 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrgMemberRepo_FindByOrgID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrgMemberRepo(db)

	role := &models.Role{OrgID: 1, Name: "member", Description: "Member", IsSystem: true}
	require.NoError(t, db.Create(role).Error)

	for i := uint(1); i <= 3; i++ {
		m := &domain.OrgMember{OrgID: 1, UserID: i, RoleID: role.ID, JoinedAt: time.Now()}
		require.NoError(t, repo.Create(ctx, m))
	}

	members, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, members, 3)
}

func TestOrgMemberRepo_CountByUserAndOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrgMemberRepo(db)

	role := &models.Role{OrgID: 1, Name: "viewer", Description: "Viewer", IsSystem: true}
	require.NoError(t, db.Create(role).Error)

	m := &domain.OrgMember{OrgID: 1, UserID: 42, RoleID: role.ID, JoinedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, m))

	count, err := repo.CountByUserAndOrg(ctx, 42, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	count, err = repo.CountByUserAndOrg(ctx, 99, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestOrgMemberRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrgMemberRepo(db)

	role1 := &models.Role{OrgID: 1, Name: "member", Description: "Member", IsSystem: true}
	role2 := &models.Role{OrgID: 1, Name: "admin", Description: "Admin", IsSystem: true}
	require.NoError(t, db.Create(role1).Error)
	require.NoError(t, db.Create(role2).Error)

	member := &domain.OrgMember{OrgID: 1, UserID: 42, RoleID: role1.ID, JoinedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, member))

	// Promote to admin
	member.RoleID = role2.ID
	err := repo.Update(ctx, member)
	require.NoError(t, err)

	found, err := repo.FindByUserAndOrg(ctx, 42, 1)
	require.NoError(t, err)
	assert.Equal(t, role2.ID, found.RoleID)
}

func TestOrgMemberRepo_DeleteByUserAndOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrgMemberRepo(db)

	role := &models.Role{OrgID: 1, Name: "member", Description: "Member", IsSystem: true}
	require.NoError(t, db.Create(role).Error)

	member := &domain.OrgMember{OrgID: 1, UserID: 42, RoleID: role.ID, JoinedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, member))

	err := repo.DeleteByUserAndOrg(ctx, 42, 1)
	require.NoError(t, err)

	_, err = repo.FindByUserAndOrg(ctx, 42, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrgMemberRepo_DeleteByUserAndOrg_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrgMemberRepo(db)

	err := repo.DeleteByUserAndOrg(ctx, 999, 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// =========================================================================
// PasswordResetTokenRepo
// =========================================================================

func TestPasswordResetTokenRepo_CreateAndFindByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPasswordResetTokenRepo(db)

	token := &domain.PasswordResetToken{
		UserID:    1,
		TokenHash: "reset-hash-abc",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := repo.Create(ctx, token)
	require.NoError(t, err)
	assert.NotZero(t, token.ID)

	found, err := repo.FindByTokenHash(ctx, "reset-hash-abc")
	require.NoError(t, err)
	assert.Equal(t, uint(1), found.UserID)
	assert.Nil(t, found.UsedAt)
}

func TestPasswordResetTokenRepo_FindByTokenHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPasswordResetTokenRepo(db)

	_, err := repo.FindByTokenHash(ctx, "nonexistent-hash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPasswordResetTokenRepo_MarkUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPasswordResetTokenRepo(db)

	token := &domain.PasswordResetToken{
		UserID:    1,
		TokenHash: "mark-used-hash",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	err := repo.MarkUsed(ctx, token.ID)
	require.NoError(t, err)

	found, err := repo.FindByTokenHash(ctx, "mark-used-hash")
	require.NoError(t, err)
	assert.NotNil(t, found.UsedAt)
}

func TestPasswordResetTokenRepo_MarkUsed_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPasswordResetTokenRepo(db)

	err := repo.MarkUsed(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPasswordResetTokenRepo_DeleteExpiredByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPasswordResetTokenRepo(db)

	// Create an expired token
	expired := &domain.PasswordResetToken{
		UserID:    1,
		TokenHash: "expired-reset-hash",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, expired))

	// Create an active token
	active := &domain.PasswordResetToken{
		UserID:    1,
		TokenHash: "active-reset-hash",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, active))

	err := repo.DeleteExpiredByUserID(ctx, 1)
	require.NoError(t, err)

	// Active token should still exist
	found, err := repo.FindByTokenHash(ctx, "active-reset-hash")
	require.NoError(t, err)
	assert.NotNil(t, found)

	// Expired token should be gone
	_, err = repo.FindByTokenHash(ctx, "expired-reset-hash")
	require.Error(t, err)
}

// =========================================================================
// EmailVerificationTokenRepo
// =========================================================================

func TestEmailVerificationTokenRepo_CreateAndFindByTokenHash(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewEmailVerificationTokenRepo(db)

	token := &domain.EmailVerificationToken{
		UserID:    1,
		TokenHash: "verify-hash-xyz",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := repo.Create(ctx, token)
	require.NoError(t, err)
	assert.NotZero(t, token.ID)

	found, err := repo.FindByTokenHash(ctx, "verify-hash-xyz")
	require.NoError(t, err)
	assert.Equal(t, uint(1), found.UserID)
}

func TestEmailVerificationTokenRepo_FindByTokenHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewEmailVerificationTokenRepo(db)

	_, err := repo.FindByTokenHash(ctx, "nonexistent-verify-hash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEmailVerificationTokenRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewEmailVerificationTokenRepo(db)

	token := &domain.EmailVerificationToken{
		UserID:    1,
		TokenHash: "delete-verify-hash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	err := repo.Delete(ctx, token.ID)
	require.NoError(t, err)

	_, err = repo.FindByTokenHash(ctx, "delete-verify-hash")
	require.Error(t, err)
}

func TestEmailVerificationTokenRepo_DeleteByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewEmailVerificationTokenRepo(db)

	// Create 2 tokens for same user
	for i := 0; i < 2; i++ {
		token := &domain.EmailVerificationToken{
			UserID:    1,
			TokenHash: "user-verify-hash-" + string(rune('a'+i)),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		require.NoError(t, repo.Create(ctx, token))
	}

	// Create 1 token for different user
	otherToken := &domain.EmailVerificationToken{
		UserID:    2,
		TokenHash: "other-user-verify-hash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, otherToken))

	err := repo.DeleteByUserID(ctx, 1)
	require.NoError(t, err)

	// User 2's token should still exist
	found, err := repo.FindByTokenHash(ctx, "other-user-verify-hash")
	require.NoError(t, err)
	assert.Equal(t, uint(2), found.UserID)
}

// =========================================================================
// RoleRepo
// =========================================================================

func TestRoleRepo_CreateAndFindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRoleRepo(db)

	// Create a permission first
	perm := &models.Permission{Resource: "packages", Action: "read"}
	require.NoError(t, db.Create(perm).Error)

	role := &domain.Role{
		OrgID:       1,
		Name:        "custom-role",
		Description: "A custom role",
		IsSystem:    false,
		Permissions: []domain.Permission{
			{ID: perm.ID, Resource: "packages", Action: "read"},
		},
	}
	err := repo.Create(ctx, role)
	require.NoError(t, err)
	assert.NotZero(t, role.ID)

	found, err := repo.FindByID(ctx, role.ID)
	require.NoError(t, err)
	assert.Equal(t, "custom-role", found.Name)
	assert.Equal(t, "A custom role", found.Description)
}

func TestRoleRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRoleRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRoleRepo_FindByIDAndOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRoleRepo(db)

	role := &domain.Role{OrgID: 1, Name: "org-role", Description: "Org specific", IsSystem: false}
	require.NoError(t, repo.Create(ctx, role))

	found, err := repo.FindByIDAndOrg(ctx, role.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "org-role", found.Name)

	// Cross-org should fail
	_, err = repo.FindByIDAndOrg(ctx, role.ID, 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRoleRepo_FindByOrgID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRoleRepo(db)

	for _, name := range []string{"owner", "admin", "member", "viewer"} {
		r := &domain.Role{OrgID: 1, Name: name, Description: name + " role", IsSystem: true}
		require.NoError(t, repo.Create(ctx, r))
	}

	roles, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, roles, 4)
}

func TestRoleRepo_FindByOrgID_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRoleRepo(db)

	roles, err := repo.FindByOrgID(ctx, 999)
	require.NoError(t, err)
	assert.Len(t, roles, 0)
}

// =========================================================================
// PermissionRepo
// =========================================================================

func TestPermissionRepo_FindOrCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPermissionRepo(db)

	perm := &domain.Permission{Resource: "packages", Action: "read"}
	err := repo.FindOrCreate(ctx, perm)
	require.NoError(t, err)
	assert.NotZero(t, perm.ID)

	// Finding the same permission should return same ID
	perm2 := &domain.Permission{Resource: "packages", Action: "read"}
	err = repo.FindOrCreate(ctx, perm2)
	require.NoError(t, err)
	assert.Equal(t, perm.ID, perm2.ID)
}

func TestPermissionRepo_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPermissionRepo(db)

	// Seed some permissions
	for _, sp := range []struct{ resource, action string }{
		{"packages", "read"}, {"packages", "write"}, {"alerts", "read"},
	} {
		p := &domain.Permission{Resource: sp.resource, Action: sp.action}
		require.NoError(t, repo.FindOrCreate(ctx, p))
	}

	perms, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, perms, 3)
}

func TestPermissionRepo_FindAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPermissionRepo(db)

	perms, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, perms, 0)
}

func TestPermissionRepo_CheckUserPermission(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPermissionRepo(db)

	// Set up: permission → role → role_permissions → org_member
	perm := &models.Permission{Resource: "packages", Action: "read"}
	require.NoError(t, db.Create(perm).Error)

	role := &models.Role{OrgID: 1, Name: "member", Description: "Member", IsSystem: true,
		Permissions: []models.Permission{*perm}}
	require.NoError(t, db.Create(role).Error)

	member := &models.OrgMember{OrgID: 1, UserID: 42, RoleID: role.ID, JoinedAt: time.Now()}
	require.NoError(t, db.Create(member).Error)

	// Should have permission
	has, err := repo.CheckUserPermission(ctx, 42, 1, "packages", "read")
	require.NoError(t, err)
	assert.True(t, has)

	// Should NOT have ungranted permission
	has, err = repo.CheckUserPermission(ctx, 42, 1, "packages", "delete")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestPermissionRepo_CheckUserPermission_NonMember(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPermissionRepo(db)

	has, err := repo.CheckUserPermission(ctx, 999, 1, "packages", "read")
	require.NoError(t, err)
	assert.False(t, has)
}

// =========================================================================
// DashboardRepo
// =========================================================================

func TestDashboardRepo_GetStats(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	// Seed packages for org 1
	pkg1 := &models.Package{OrgID: 1, Name: "requests", Ecosystem: "python"}
	pkg2 := &models.Package{OrgID: 1, Name: "flask", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg1).Error)
	require.NoError(t, db.Create(pkg2).Error)

	// Seed a release
	rel := &models.Release{PackageID: pkg1.ID, Version: "1.0.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(rel).Error)

	// Seed an alert
	alert := &models.Alert{OrgID: 1, PackageID: pkg1.ID, AnalysisID: 0, Severity: "high", Status: "new", Message: "test"}
	require.NoError(t, db.Create(alert).Error)

	stats, err := repo.GetStats(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalPackages)
	assert.Equal(t, int64(1), stats.TotalReleases)
	assert.Equal(t, int64(1), stats.ActiveAlerts)
}

func TestDashboardRepo_GetStats_EmptyOrg(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	stats, err := repo.GetStats(ctx, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalPackages)
	assert.Equal(t, int64(0), stats.TotalReleases)
	assert.Equal(t, int64(0), stats.ActiveAlerts)
}

func TestDashboardRepo_GetEcosystemDistribution(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "requests", Ecosystem: "python"}).Error)
	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "flask", Ecosystem: "python"}).Error)
	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "lodash", Ecosystem: "npm"}).Error)

	dist, err := repo.GetEcosystemDistribution(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, dist, 2)

	// Check counts
	distMap := make(map[string]int64)
	for _, d := range dist {
		distMap[d.Ecosystem] = d.Count
	}
	assert.Equal(t, int64(2), distMap["python"])
	assert.Equal(t, int64(1), distMap["npm"])
}

func TestDashboardRepo_GetEcosystemDistribution_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	dist, err := repo.GetEcosystemDistribution(ctx, 999)
	require.NoError(t, err)
	assert.Len(t, dist, 0)
}

func TestDashboardRepo_GetReleaseActivity(t *testing.T) {
	// NOTE: GetReleaseActivity uses DATE() which returns string in SQLite
	// but time.Time in PostgreSQL. Skip in SQLite-based unit tests.
	// This method is covered by integration tests against PostgreSQL.
	t.Skip("DATE() SQL function returns string in SQLite, requires PostgreSQL")
}

func TestDashboardRepo_GetAlertsBySeverity(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	now := time.Now()
	severities := []string{"low", "high", "high", "critical"}
	for i, sev := range severities {
		a := &models.Alert{
			OrgID: 1, PackageID: 1, AnalysisID: 0,
			Severity: models.AlertSeverity(sev), Status: "new",
			Message: "alert " + string(rune('0'+i)),
		}
		require.NoError(t, db.Create(a).Error)
	}

	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)
	result, err := repo.GetAlertsBySeverity(ctx, 1, from, to)
	require.NoError(t, err)

	sevMap := make(map[string]int64)
	for _, r := range result {
		sevMap[r.Severity] = r.Count
	}
	assert.Equal(t, int64(1), sevMap["low"])
	assert.Equal(t, int64(2), sevMap["high"])
	assert.Equal(t, int64(1), sevMap["critical"])
}

func TestDashboardRepo_GetUnanalyzedDiffIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "test-unanalyzed", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	r1 := &models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)

	// Create a diff without an analysis
	diff := &models.Diff{ReleaseID: r2.ID, PrevReleaseID: r1.ID, DiffContent: "unanalyzed"}
	require.NoError(t, db.Create(diff).Error)

	ids, err := repo.GetUnanalyzedDiffIDs(ctx, 1)
	require.NoError(t, err)
	assert.Contains(t, ids, diff.ID)
}

func TestDashboardRepo_GetUnanalyzedDiffIDs_AllAnalyzed(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "test-analyzed", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	r1 := &models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)

	diff := &models.Diff{ReleaseID: r2.ID, PrevReleaseID: r1.ID, DiffContent: "analyzed"}
	require.NoError(t, db.Create(diff).Error)

	// Create analysis for the diff
	analysis := &models.Analysis{
		DiffID: diff.ID, Classification: "benign", Confidence: 0.9,
		ModelUsed: "test", AnalyzerType: "api",
	}
	require.NoError(t, db.Create(analysis).Error)

	ids, err := repo.GetUnanalyzedDiffIDs(ctx, 1)
	require.NoError(t, err)
	assert.NotContains(t, ids, diff.ID)
}

// =========================================================================
// Additional UserRepo tests (expand coverage)
// =========================================================================

func TestUserRepo_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewUserRepo(db)

	user1 := &domain.User{Email: "dupe@example.com", PasswordHash: "hash1", IsActive: true}
	require.NoError(t, repo.Create(ctx, user1))

	user2 := &domain.User{Email: "dupe@example.com", PasswordHash: "hash2", IsActive: true}
	err := repo.Create(ctx, user2)
	require.Error(t, err, "should fail for duplicate email")
}

// =========================================================================
// Additional RefreshTokenRepo tests (expand coverage)
// =========================================================================

func TestRefreshTokenRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRefreshTokenRepo(db)

	token := &domain.RefreshToken{
		UserID:    1,
		TokenHash: "new-token-hash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := repo.Create(ctx, token)
	require.NoError(t, err)
	assert.NotZero(t, token.ID)

	found, err := repo.FindByTokenHash(ctx, "new-token-hash")
	require.NoError(t, err)
	assert.Equal(t, uint(1), found.UserID)
}

func TestRefreshTokenRepo_FindByTokenHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRefreshTokenRepo(db)

	_, err := repo.FindByTokenHash(ctx, "nonexistent-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRefreshTokenRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRefreshTokenRepo(db)

	token := &domain.RefreshToken{
		UserID:    1,
		TokenHash: "delete-by-id-hash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, token))

	err := repo.Delete(ctx, token.ID)
	require.NoError(t, err)

	_, err = repo.FindByTokenHash(ctx, "delete-by-id-hash")
	require.Error(t, err)
}

// =========================================================================
// Additional APIKeyRepo tests (expand coverage)
// =========================================================================

func TestAPIKeyRepo_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	key := &models.APIKey{
		UserID: 1, Name: "find-by-id", KeyHash: "hash-find",
		KeyPrefix: "vmx_fb", IsActive: true,
	}
	require.NoError(t, db.Create(key).Error)

	found, err := repo.FindByID(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, "find-by-id", found.Name)
}

func TestAPIKeyRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAPIKeyRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	key := &models.APIKey{
		UserID: 1, Name: "update-me", KeyHash: "hash-upd",
		KeyPrefix: "vmx_up", IsActive: true,
	}
	require.NoError(t, db.Create(key).Error)

	found, err := repo.FindByID(ctx, key.ID)
	require.NoError(t, err)
	now := time.Now()
	found.LastUsedAt = &now
	err = repo.Update(ctx, found)
	require.NoError(t, err)

	reloaded, err := repo.FindByID(ctx, key.ID)
	require.NoError(t, err)
	assert.NotNil(t, reloaded.LastUsedAt)
}

func TestAPIKeyRepo_FindActiveByPrefix_InactiveExcluded(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	// Active key
	active := &models.APIKey{
		UserID: 1, Name: "active", KeyHash: "hash-a",
		KeyPrefix: "vmx_ia", IsActive: true,
	}
	require.NoError(t, db.Create(active).Error)

	// Inactive key with same prefix — must set IsActive=false after create
	// because GORM treats false (zero value) as "not set" and applies default:true
	inactive := &models.APIKey{
		UserID: 1, Name: "inactive", KeyHash: "hash-i",
		KeyPrefix: "vmx_ia", IsActive: true,
	}
	require.NoError(t, db.Create(inactive).Error)
	require.NoError(t, db.Model(inactive).Update("is_active", false).Error)

	found, err := repo.FindActiveByPrefix(ctx, "vmx_ia")
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "active", found[0].Name)
}

// =========================================================================
// Additional PackageRepo tests (expand coverage)
// =========================================================================

func TestPackageRepo_FindByOrgAndName(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "express", Ecosystem: "npm"}).Error)

	found, err := repo.FindByOrgAndName(ctx, 1, "express", domain.EcosystemNPM)
	require.NoError(t, err)
	assert.Equal(t, "express", found.Name)
	assert.Equal(t, domain.EcosystemNPM, found.Ecosystem)
}

func TestPackageRepo_FindByOrgAndName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	_, err := repo.FindByOrgAndName(ctx, 1, "nonexistent", domain.EcosystemPython)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPackageRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	pkg := &domain.Package{OrgID: 1, Name: "pandas", Ecosystem: domain.EcosystemPython, Description: "Data analysis"}
	err := repo.Create(ctx, pkg)
	require.NoError(t, err)
	assert.NotZero(t, pkg.ID)
}

func TestPackageRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	pkg := &domain.Package{OrgID: 1, Name: "scipy", Ecosystem: domain.EcosystemPython}
	require.NoError(t, repo.Create(ctx, pkg))

	pkg.LatestVersion = "1.12.0"
	pkg.Description = "Scientific computing"
	err := repo.Update(ctx, pkg)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, pkg.ID)
	require.NoError(t, err)
	assert.Equal(t, "1.12.0", found.LatestVersion)
	assert.Equal(t, "Scientific computing", found.Description)
}

func TestPackageRepo_CountByOrg_WithEcosystemFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "p1", Ecosystem: "python"}).Error)
	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "p2", Ecosystem: "python"}).Error)
	require.NoError(t, db.Create(&models.Package{OrgID: 1, Name: "p3", Ecosystem: "npm"}).Error)

	pypi := domain.EcosystemPython
	count, err := repo.CountByOrg(ctx, 1, &pypi)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	npm := domain.EcosystemNPM
	count, err = repo.CountByOrg(ctx, 1, &npm)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// =========================================================================
// Additional AuditLogRepo tests (expand coverage)
// =========================================================================

func TestAuditLogRepo_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAuditLogRepo(db)

	entry := &models.AuditLog{
		UserID: 1, OrgID: 1, Action: "update", Resource: "setting", ResourceID: 5,
		Details: "changed analyzer mode",
	}
	require.NoError(t, db.Create(entry).Error)

	found, err := repo.FindByID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Equal(t, "update", found.Action)
	assert.Equal(t, "setting", found.Resource)
}

func TestAuditLogRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAuditLogRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestAuditLogRepo_FindByOrgID_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAuditLogRepo(db)

	require.NoError(t, db.Create(&models.AuditLog{UserID: 1, OrgID: 1, Action: "create", Resource: "package", ResourceID: 1}).Error)
	require.NoError(t, db.Create(&models.AuditLog{UserID: 2, OrgID: 1, Action: "update", Resource: "package", ResourceID: 1}).Error)
	require.NoError(t, db.Create(&models.AuditLog{UserID: 1, OrgID: 1, Action: "create", Resource: "alert", ResourceID: 1}).Error)

	// Filter by resource
	logs, total, err := repo.FindByOrgID(ctx, 1, domain.AuditLogFilters{Resource: "package"}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)

	// Filter by action
	logs, total, err = repo.FindByOrgID(ctx, 1, domain.AuditLogFilters{Action: "create"}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, logs, 2)

	// Filter by userID
	logs, total, err = repo.FindByOrgID(ctx, 1, domain.AuditLogFilters{UserID: 2}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
}

func TestAuditLogRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAuditLogRepo(db)

	entry := &domain.AuditLog{
		UserID:   1,
		OrgID:    1,
		Action:   "delete",
		Resource: "package",
		ResourceID: 42,
		Details:  "removed package lodash",
	}
	err := repo.Create(ctx, entry)
	require.NoError(t, err)
	assert.NotZero(t, entry.ID)

	found, err := repo.FindByID(ctx, entry.ID)
	require.NoError(t, err)
	assert.Equal(t, "delete", found.Action)
}

// =========================================================================
// Additional SettingRepo tests (expand coverage)
// =========================================================================

func TestSettingRepo_FindByOrgID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSettingRepo(db)

	require.NoError(t, db.Create(&models.Setting{OrgID: 1, Key: "k1", Value: "v1"}).Error)
	require.NoError(t, db.Create(&models.Setting{OrgID: 1, Key: "k2", Value: "v2"}).Error)
	require.NoError(t, db.Create(&models.Setting{OrgID: 2, Key: "k3", Value: "v3"}).Error)

	settings, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, settings, 2)
}

func TestSettingRepo_FindByKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSettingRepo(db)

	_, err := repo.FindByKey(ctx, 1, "nonexistent_key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSettingRepo_FindOrCreateByKey(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewSettingRepo(db)

	// First call creates the setting
	setting, err := repo.FindOrCreateByKey(ctx, "new_key", "default_value")
	require.NoError(t, err)
	assert.Equal(t, "default_value", setting.Value)

	// Second call finds the existing setting
	setting2, err := repo.FindOrCreateByKey(ctx, "new_key", "other_default")
	require.NoError(t, err)
	assert.Equal(t, "default_value", setting2.Value, "should return existing value, not the new default")
}

// =========================================================================
// Additional OrganizationRepo tests (expand coverage)
// =========================================================================

func TestOrgRepo_FindBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	org := &domain.Organization{Name: "Slug Corp", Slug: "slug-corp", OwnerID: 1, IsActive: true}
	require.NoError(t, repo.Create(ctx, org))

	found, err := repo.FindBySlug(ctx, "slug-corp")
	require.NoError(t, err)
	assert.Equal(t, "Slug Corp", found.Name)
}

func TestOrgRepo_FindBySlug_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	_, err := repo.FindBySlug(ctx, "nonexistent-slug")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrgRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrgRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	org := &domain.Organization{Name: "Old Name", Slug: "old-name", OwnerID: 1, IsActive: true}
	require.NoError(t, repo.Create(ctx, org))

	org.Name = "New Name"
	err := repo.Update(ctx, org)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, org.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", found.Name)
}

func TestOrgRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	org := &domain.Organization{Name: "Delete Me", Slug: "delete-me", OwnerID: 1, IsActive: true}
	require.NoError(t, repo.Create(ctx, org))

	err := repo.SoftDelete(ctx, org.ID)
	require.NoError(t, err)

	// Should not be findable after soft delete
	_, err = repo.FindByID(ctx, org.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrgRepo_SoftDelete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	err := repo.SoftDelete(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOrgRepo_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	org1 := &domain.Organization{Name: "Org1", Slug: "org1", OwnerID: 1, IsActive: true}
	org2 := &domain.Organization{Name: "Org2", Slug: "org2", OwnerID: 2, IsActive: true}
	require.NoError(t, repo.Create(ctx, org1))
	require.NoError(t, repo.Create(ctx, org2))

	// Create membership for user 42 in both orgs
	role := &models.Role{OrgID: org1.ID, Name: "member", IsSystem: true}
	require.NoError(t, db.Create(role).Error)

	require.NoError(t, db.Create(&models.OrgMember{OrgID: org1.ID, UserID: 42, RoleID: role.ID, JoinedAt: time.Now()}).Error)
	require.NoError(t, db.Create(&models.OrgMember{OrgID: org2.ID, UserID: 42, RoleID: role.ID, JoinedAt: time.Now()}).Error)

	orgs, err := repo.FindByUserID(ctx, 42)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

func TestOrgRepo_FindByUserID_NoMemberships(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewOrganizationRepo(db)

	orgs, err := repo.FindByUserID(ctx, 999)
	require.NoError(t, err)
	assert.Len(t, orgs, 0)
}

// =========================================================================
// Additional InvitationRepo tests (expand coverage)
// =========================================================================

func TestInvitationRepo_FindByTokenHash_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewInvitationRepo(db)

	_, err := repo.FindByTokenHash(ctx, "nonexistent-invite-hash")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInvitationRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewInvitationRepo(db)

	inv := &domain.Invitation{
		OrgID: 1, Email: "invite@example.com", RoleID: 1,
		TokenHash: "update-invite-hash", InvitedBy: 1,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.Create(ctx, inv))

	now := time.Now()
	inv.AcceptedAt = &now
	err := repo.Update(ctx, inv)
	require.NoError(t, err)

	found, err := repo.FindByTokenHash(ctx, "update-invite-hash")
	require.NoError(t, err)
	assert.NotNil(t, found.AcceptedAt)
}

// =========================================================================
// Additional NotificationChannel tests (expand coverage)
// =========================================================================

func TestNotifChannelRepo_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationChannelRepo(db)

	ch := &models.NotificationChannel{OrgID: 1, Name: "Slack", Type: "slack", Config: `{"url":"https://hooks.slack.com/test"}`, IsActive: true}
	require.NoError(t, db.Create(ch).Error)

	found, err := repo.FindByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, "Slack", found.Name)
}

func TestNotifChannelRepo_FindByIDAndOrg_CrossTenant(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationChannelRepo(db)

	ch := &models.NotificationChannel{OrgID: 1, Name: "Private", Type: "email", Config: `{}`, IsActive: true}
	require.NoError(t, db.Create(ch).Error)

	// Same org should find it
	found, err := repo.FindByIDAndOrg(ctx, ch.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "Private", found.Name)

	// Different org should not find it
	_, err = repo.FindByIDAndOrg(ctx, ch.ID, 999)
	require.Error(t, err)
}

func TestNotifChannelRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationChannelRepo(db)

	ch := &models.NotificationChannel{OrgID: 1, Name: "Old Channel", Type: "email", Config: `{}`, IsActive: true}
	require.NoError(t, db.Create(ch).Error)

	found, err := repo.FindByID(ctx, ch.ID)
	require.NoError(t, err)

	found.Name = "Updated Channel"
	err = repo.Update(ctx, found)
	require.NoError(t, err)

	reloaded, err := repo.FindByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Channel", reloaded.Name)
}

func TestNotifChannelRepo_DeleteByIDAndOrg_CrossTenant(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationChannelRepo(db)

	ch := &models.NotificationChannel{OrgID: 1, Name: "Protected", Type: "webhook", Config: `{}`, IsActive: true}
	require.NoError(t, db.Create(ch).Error)

	// Cross-tenant delete should affect 0 rows
	affected, err := repo.DeleteByIDAndOrg(ctx, ch.ID, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected)

	// Channel should still exist
	_, err = repo.FindByID(ctx, ch.ID)
	require.NoError(t, err)
}

// =========================================================================
// Additional NotificationRepo tests (expand coverage)
// =========================================================================

func TestNotifRepo_MarkAllRead(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRepo(db)

	for i := 0; i < 3; i++ {
		n := &models.Notification{
			OrgID: 1, UserID: 42, ChannelID: 1,
			Title: "Notif", Message: "msg", IsRead: false, SentAt: time.Now(),
		}
		require.NoError(t, db.Create(n).Error)
	}

	affected, err := repo.MarkAllRead(ctx, 1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(3), affected)

	count, err := repo.CountUnread(ctx, 1, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestNotifRepo_FindByUserAndOrg_OnlyUnread(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRepo(db)

	require.NoError(t, db.Create(&models.Notification{
		OrgID: 1, UserID: 42, ChannelID: 1,
		Title: "Read", Message: "msg", IsRead: true, SentAt: time.Now(),
	}).Error)
	require.NoError(t, db.Create(&models.Notification{
		OrgID: 1, UserID: 42, ChannelID: 1,
		Title: "Unread", Message: "msg", IsRead: false, SentAt: time.Now(),
	}).Error)

	notifs, err := repo.FindByUserAndOrg(ctx, 1, 42, true)
	require.NoError(t, err)
	assert.Len(t, notifs, 1)
	assert.Equal(t, "Unread", notifs[0].Title)
}

func TestNotifRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRepo(db)

	notif := &domain.Notification{
		OrgID:     1,
		UserID:    42,
		ChannelID: 1,
		Title:     "New Alert",
		Message:   "A critical vulnerability detected",
		IsRead:    false,
		SentAt:    time.Now(),
	}
	err := repo.Create(ctx, notif)
	require.NoError(t, err)
	assert.NotZero(t, notif.ID)
}

// =========================================================================
// Additional NotificationRuleRepo tests (expand coverage)
// =========================================================================

func TestNotifRuleRepo_FindActiveByOrgID_InactiveExcluded(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRuleRepo(db)

	require.NoError(t, db.Create(&models.NotificationRule{OrgID: 1, ChannelID: 1, Severity: "high", IsActive: true}).Error)
	// Create rule then mark inactive — GORM treats false as zero-value and applies default
	inactiveRule := &models.NotificationRule{OrgID: 1, ChannelID: 1, Severity: "low", IsActive: true}
	require.NoError(t, db.Create(inactiveRule).Error)
	require.NoError(t, db.Model(inactiveRule).Update("is_active", false).Error)

	active, err := repo.FindActiveByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, active, 1)
	assert.Equal(t, "high", active[0].Severity)
}

func TestNotifRuleRepo_DeleteByIDAndOrg_CrossTenant(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRuleRepo(db)

	rule := &models.NotificationRule{OrgID: 1, ChannelID: 1, Severity: "critical", IsActive: true}
	require.NoError(t, db.Create(rule).Error)

	// Cross-tenant delete should affect 0 rows
	affected, err := repo.DeleteByIDAndOrg(ctx, rule.ID, 999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected)

	// Rule should still exist
	rules, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, rules, 1)
}

// =========================================================================
// Domain-level Create methods (cover repo.Create paths)
// =========================================================================

func TestAPIKeyRepo_Create_ViaDomain(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewAPIKeyRepo(db)

	key := &domain.APIKey{
		UserID:    1,
		Name:      "domain-key",
		KeyHash:   "domain-hash-123",
		KeyPrefix: "vmx_dk",
		Scope:     domain.APIKeyScopeWrite,
		IsActive:  true,
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)
	assert.NotZero(t, key.ID)

	found, err := repo.FindByID(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, "domain-key", found.Name)
	assert.Equal(t, domain.APIKeyScopeWrite, found.Scope)
}

func TestNotifChannelRepo_Create_ViaDomain(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationChannelRepo(db)

	ch := &domain.NotificationChannel{
		OrgID:    1,
		Name:     "Webhook Channel",
		Type:     domain.NotificationChannelWebhook,
		Config:   `{"url":"https://hooks.example.com/callback"}`,
		IsActive: true,
	}
	err := repo.Create(ctx, ch)
	require.NoError(t, err)
	assert.NotZero(t, ch.ID)

	found, err := repo.FindByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, "Webhook Channel", found.Name)
	assert.Equal(t, domain.NotificationChannelWebhook, found.Type)
}

func TestNotifRuleRepo_Create_ViaDomain(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationRuleRepo(db)

	rule := &domain.NotificationRule{
		OrgID:     1,
		ChannelID: 1,
		Severity:  "critical",
		IsActive:  true,
	}
	err := repo.Create(ctx, rule)
	require.NoError(t, err)
	assert.NotZero(t, rule.ID)

	rules, err := repo.FindByOrgID(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, "critical", rules[0].Severity)
}

// =========================================================================
// Dashboard: additional methods testable with SQLite
// =========================================================================

func TestDashboardRepo_GetClassificationDistribution(t *testing.T) {
	// NOTE: Uses DATE()-like aggregation with time range which works differently in SQLite
	// but the basic GROUP BY classification still works
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "test-class", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)
	r1 := &models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed", PublishedAt: time.Now()}
	r2 := &models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(r1).Error)
	require.NoError(t, db.Create(r2).Error)
	diff := &models.Diff{ReleaseID: r2.ID, PrevReleaseID: r1.ID, DiffContent: "test"}
	require.NoError(t, db.Create(diff).Error)

	require.NoError(t, db.Create(&models.Analysis{DiffID: diff.ID, Classification: "benign", Confidence: 0.9, ModelUsed: "test", AnalyzerType: "api"}).Error)
	require.NoError(t, db.Create(&models.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "api"}).Error)

	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)

	dist, err := repo.GetClassificationDistribution(ctx, 1, from, to)
	require.NoError(t, err)
	assert.Len(t, dist, 2)

	classMap := make(map[string]int64)
	for _, d := range dist {
		classMap[d.Classification] = d.Count
	}
	assert.Equal(t, int64(1), classMap["benign"])
	assert.Equal(t, int64(1), classMap["malicious"])
}

func TestDashboardRepo_GetReleaseStatusDistribution(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "test-status-dist", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	statuses := []string{"pending", "completed", "completed", "error"}
	for i, status := range statuses {
		r := &models.Release{PackageID: pkg.ID, Version: "1.0." + string(rune('0'+i)), Status: models.ReleaseStatus(status), PublishedAt: time.Now()}
		require.NoError(t, db.Create(r).Error)
	}

	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)

	dist, err := repo.GetReleaseStatusDistribution(ctx, 1, from, to)
	require.NoError(t, err)

	statusMap := make(map[string]int64)
	for _, d := range dist {
		statusMap[d.Status] = d.Count
	}
	assert.Equal(t, int64(1), statusMap["pending"])
	assert.Equal(t, int64(2), statusMap["completed"])
	assert.Equal(t, int64(1), statusMap["error"])
}

func TestDashboardRepo_GetBaselineCount(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewDashboardRepo(db)

	pkg := &models.Package{OrgID: 1, Name: "test-baseline", Ecosystem: "python"}
	require.NoError(t, db.Create(pkg).Error)

	// Create a completed release WITHOUT a diff (should count as baseline)
	baseline := &models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(baseline).Error)

	// Create a completed release WITH a diff (should NOT count as baseline)
	withDiff := &models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed", PublishedAt: time.Now()}
	require.NoError(t, db.Create(withDiff).Error)
	require.NoError(t, db.Create(&models.Diff{ReleaseID: withDiff.ID, PrevReleaseID: baseline.ID, DiffContent: "test"}).Error)

	now := time.Now()
	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)

	count, err := repo.GetBaselineCount(ctx, 1, from, to)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "only the release without a diff should be counted as baseline")
}

// =========================================================================
// PackageRepo: FindByID via domain repo (cover the error branch)
// =========================================================================

func TestPackageRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPackageRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// =========================================================================
// NotificationChannelRepo: FindByID_NotFound
// =========================================================================

func TestNotifChannelRepo_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewNotificationChannelRepo(db)

	_, err := repo.FindByID(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
