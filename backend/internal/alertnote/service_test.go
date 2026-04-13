package alertnote_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/alertnote"
	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.User{},
		&models.Package{},
		&models.Release{},
		&models.Diff{},
		&models.Analysis{},
		&models.Alert{},
		&models.AlertNote{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

func newService(db *gorm.DB) *alertnote.Service {
	return alertnote.NewService(
		repository.NewAlertNoteRepo(db),
		repository.NewAlertRepo(db),
		repository.NewUserRepo(db),
	)
}

// createTestAlert inserts a minimal alert chain (package → release → diff → analysis → alert).
func createTestAlert(t *testing.T, db *gorm.DB, orgID uint) models.Alert {
	t.Helper()
	pkg := models.Package{Name: "test-pkg", Ecosystem: "python", OrgID: orgID}
	db.Create(&pkg)
	rel1 := models.Release{PackageID: pkg.ID, Version: "1.0.0", Status: "completed"}
	db.Create(&rel1)
	rel2 := models.Release{PackageID: pkg.ID, Version: "1.1.0", Status: "completed"}
	db.Create(&rel2)
	diff := models.Diff{ReleaseID: rel2.ID, PrevReleaseID: rel1.ID, DiffContent: "diff"}
	db.Create(&diff)
	analysis := models.Analysis{DiffID: diff.ID, Classification: "malicious", Confidence: 0.95, ModelUsed: "test", AnalyzerType: "copilot"}
	db.Create(&analysis)
	alert := models.Alert{OrgID: orgID, AnalysisID: analysis.ID, PackageID: pkg.ID, Severity: "critical", Status: "new", Message: "test alert"}
	db.Create(&alert)
	return alert
}

// --- ListByAlert tests ---

func TestListByAlert_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create some notes
	db.Create(&models.AlertNote{AlertID: alert.ID, OrgID: 1, UserID: 10, UserEmail: "alice@test.com", Content: "First note"})
	db.Create(&models.AlertNote{AlertID: alert.ID, OrgID: 1, UserID: 11, UserEmail: "bob@test.com", Content: "Second note"})

	notes, err := svc.ListByAlert(context.Background(), 1, alert.ID)
	require.NoError(t, err)
	assert.Len(t, notes, 2)
	// Notes are returned newest-first (DESC order)
	assert.Equal(t, "Second note", notes[0].Content)
	assert.Equal(t, "First note", notes[1].Content)
}

func TestListByAlert_EmptyList(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	notes, err := svc.ListByAlert(context.Background(), 1, alert.ID)
	require.NoError(t, err)
	assert.Len(t, notes, 0)
}

func TestListByAlert_AlertNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	_, err := svc.ListByAlert(context.Background(), 1, 99999)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestListByAlert_CrossOrg_ReturnsNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a note on org 1's alert
	db.Create(&models.AlertNote{AlertID: alert.ID, OrgID: 1, UserID: 10, UserEmail: "alice@test.com", Content: "Note"})

	// Org 2 tries to access org 1's alert notes — should get not found
	_, err := svc.ListByAlert(context.Background(), 2, alert.ID)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// --- Create tests ---

func TestCreate_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a user for email resolution
	db.Create(&models.User{Email: "analyst@test.com"})
	var user models.User
	db.First(&user)

	note, err := svc.Create(context.Background(), 1, alert.ID, user.ID, "This looks suspicious")
	require.NoError(t, err)
	assert.NotZero(t, note.ID)
	assert.Equal(t, alert.ID, note.AlertID)
	assert.Equal(t, uint(1), note.OrgID)
	assert.Equal(t, user.ID, note.UserID)
	assert.Equal(t, "analyst@test.com", note.UserEmail)
	assert.Equal(t, "This looks suspicious", note.Content)
	assert.False(t, note.CreatedAt.IsZero())
}

func TestCreate_AlertNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	_, err := svc.Create(context.Background(), 1, 99999, 10, "note content")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCreate_CrossOrg_ReturnsNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Org 2 tries to create a note on org 1's alert
	_, err := svc.Create(context.Background(), 2, alert.ID, 10, "malicious note from other org")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Verify no note was created
	var count int64
	db.Model(&models.AlertNote{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestCreate_EmptyContent(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	_, err := svc.Create(context.Background(), 1, alert.ID, 10, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}

func TestCreate_ContentTooLong(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	longContent := strings.Repeat("a", domain.MaxNoteLength+1)
	_, err := svc.Create(context.Background(), 1, alert.ID, 10, longContent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most")
}

func TestCreate_UserNotFound_StillCreatesNote(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// User 999 doesn't exist — note should still be created with empty email
	note, err := svc.Create(context.Background(), 1, alert.ID, 999, "note from unknown user")
	require.NoError(t, err)
	assert.Equal(t, "", note.UserEmail)
	assert.Equal(t, "note from unknown user", note.Content)
}

func TestCreate_MaxLengthContent_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Exactly at the limit should succeed
	maxContent := strings.Repeat("x", domain.MaxNoteLength)
	note, err := svc.Create(context.Background(), 1, alert.ID, 10, maxContent)
	require.NoError(t, err)
	assert.Equal(t, domain.MaxNoteLength, len(note.Content))
}

// --- Update tests ---

func TestUpdate_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a note as user 10
	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "original content")
	require.NoError(t, err)

	// Update by the same user
	updated, err := svc.Update(context.Background(), 1, alert.ID, note.ID, 10, "updated content")
	require.NoError(t, err)
	assert.Equal(t, note.ID, updated.ID)
	assert.Equal(t, "updated content", updated.Content)
	assert.False(t, updated.UpdatedAt.IsZero())
}

func TestUpdate_NotOwner_ReturnsForbidden(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a note as user 10
	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "original content")
	require.NoError(t, err)

	// User 20 tries to update — should get forbidden
	_, err = svc.Update(context.Background(), 1, alert.ID, note.ID, 20, "hacked content")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)

	// Verify content unchanged
	notes, err := svc.ListByAlert(context.Background(), 1, alert.ID)
	require.NoError(t, err)
	assert.Equal(t, "original content", notes[0].Content)
}

func TestUpdate_NoteNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	_, err := svc.Update(context.Background(), 1, alert.ID, 99999, 10, "new content")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdate_AlertNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	// Non-existent alert ID
	_, err := svc.Update(context.Background(), 1, 99999, 1, 10, "new content")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdate_CrossOrg_ReturnsNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a note in org 1
	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "org 1 note")
	require.NoError(t, err)

	// Org 2 tries to update — should get not found (tenant isolation via alert check)
	_, err = svc.Update(context.Background(), 2, alert.ID, note.ID, 10, "hacked from org 2")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdate_NoteOnDifferentAlert(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert1 := createTestAlert(t, db, 1)
	alert2 := createTestAlert(t, db, 1)

	// Create a note on alert1
	note, err := svc.Create(context.Background(), 1, alert1.ID, 10, "note on alert 1")
	require.NoError(t, err)

	// Try to update via alert2's URL — note belongs to alert1, should get not found
	_, err = svc.Update(context.Background(), 1, alert2.ID, note.ID, 10, "sneaky update")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdate_EmptyContent(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "original")
	require.NoError(t, err)

	_, err = svc.Update(context.Background(), 1, alert.ID, note.ID, 10, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}

func TestUpdate_ContentTooLong(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "original")
	require.NoError(t, err)

	longContent := strings.Repeat("a", domain.MaxNoteLength+1)
	_, err = svc.Update(context.Background(), 1, alert.ID, note.ID, 10, longContent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most")
}

// --- Delete tests ---

func TestDelete_Success(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a note as user 10
	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "to be deleted")
	require.NoError(t, err)

	// Delete by the same user
	err = svc.Delete(context.Background(), 1, alert.ID, note.ID, 10)
	require.NoError(t, err)

	// Verify note is gone
	notes, err := svc.ListByAlert(context.Background(), 1, alert.ID)
	require.NoError(t, err)
	assert.Len(t, notes, 0)
}

func TestDelete_NotOwner_ReturnsForbidden(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a note as user 10
	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "my note")
	require.NoError(t, err)

	// User 20 tries to delete — should get forbidden
	err = svc.Delete(context.Background(), 1, alert.ID, note.ID, 20)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrForbidden)

	// Verify note still exists
	notes, err := svc.ListByAlert(context.Background(), 1, alert.ID)
	require.NoError(t, err)
	assert.Len(t, notes, 1)
}

func TestDelete_NoteNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	err := svc.Delete(context.Background(), 1, alert.ID, 99999, 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDelete_AlertNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)

	// Non-existent alert ID
	err := svc.Delete(context.Background(), 1, 99999, 1, 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDelete_CrossOrg_ReturnsNotFound(t *testing.T) {
	db := setupTestDB(t)
	svc := newService(db)
	alert := createTestAlert(t, db, 1)

	// Create a note in org 1
	note, err := svc.Create(context.Background(), 1, alert.ID, 10, "org 1 note")
	require.NoError(t, err)

	// Org 2 tries to delete — should get not found (tenant isolation via alert check)
	err = svc.Delete(context.Background(), 2, alert.ID, note.ID, 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Verify note still exists
	var count int64
	db.Model(&models.AlertNote{}).Count(&count)
	assert.Equal(t, int64(1), count)
}
