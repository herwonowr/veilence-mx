package alertnote_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/alertnote"
)

// ---------------------------------------------------------------------------
// Mock AlertNoteRepository
// ---------------------------------------------------------------------------

type mockNoteRepo struct {
	findByAlertIDResult []entity.AlertNote
	findByAlertIDErr    error

	findByIDResult *entity.AlertNote
	findByIDErr    error

	createErr error
	updateErr error
	deleteErr error
}

func (m *mockNoteRepo) FindByAlertID(_ context.Context, _ uint) ([]entity.AlertNote, error) {
	if m.findByAlertIDErr != nil {
		return nil, m.findByAlertIDErr
	}
	return m.findByAlertIDResult, nil
}

func (m *mockNoteRepo) FindByID(_ context.Context, _ uint) (*entity.AlertNote, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDResult, nil
}

func (m *mockNoteRepo) Create(_ context.Context, _ *entity.AlertNote) error { return m.createErr }
func (m *mockNoteRepo) Update(_ context.Context, _ *entity.AlertNote) error { return m.updateErr }
func (m *mockNoteRepo) Delete(_ context.Context, _ uint) error              { return m.deleteErr }

// ---------------------------------------------------------------------------
// Mock AlertRepository (for ownership checks)
// ---------------------------------------------------------------------------

type mockAlertRepo struct {
	findByIDResult *entity.Alert
	findByIDErr    error
}

func (m *mockAlertRepo) FindByID(_ context.Context, _ uint) (*entity.Alert, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDResult, nil
}

func (m *mockAlertRepo) FindByIDWithPackage(context.Context, uint, uint) (*entity.Alert, *entity.Package, error) {
	return nil, nil, nil
}
func (m *mockAlertRepo) FindByOrgID(context.Context, uint, int, int, string, entity.AlertFilters) ([]entity.Alert, int64, error) {
	return nil, 0, nil
}
func (m *mockAlertRepo) FindByOrgIDWithPackage(context.Context, uint, int, int, string, entity.AlertFilters) ([]entity.AlertWithPackage, int64, error) {
	return nil, 0, nil
}
func (m *mockAlertRepo) Create(context.Context, *entity.Alert) error                          { return nil }
func (m *mockAlertRepo) Update(context.Context, *entity.Alert) error                          { return nil }
func (m *mockAlertRepo) UpdateStatus(context.Context, uint, entity.AlertStatus) error         { return nil }
func (m *mockAlertRepo) CountByOrgAndStatus(context.Context, uint) (map[entity.AlertStatus]int64, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock UserRepository (for resolveUserEmail)
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	findByIDResult *entity.User
	findByIDErr    error
}

func (m *mockUserRepo) FindByID(_ context.Context, _ uint) (*entity.User, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDResult, nil
}

func (m *mockUserRepo) FindByEmail(context.Context, string) (*entity.User, error) { return nil, nil }
func (m *mockUserRepo) Create(context.Context, *entity.User) error                { return nil }
func (m *mockUserRepo) Update(context.Context, *entity.User) error                { return nil }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

const (
	orgID   = uint(10)
	alertID = uint(1)
	noteID  = uint(100)
	userID  = uint(42)
)

func orgAlert() *entity.Alert {
	return &entity.Alert{ID: alertID, OrgID: orgID}
}

func ownedNote() *entity.AlertNote {
	return &entity.AlertNote{
		ID:      noteID,
		AlertID: alertID,
		OrgID:   orgID,
		UserID:  userID,
		Content: "original",
	}
}

// ---------------------------------------------------------------------------
// ListByAlert
// ---------------------------------------------------------------------------

func TestListByAlert_Success(t *testing.T) {
	notes := []entity.AlertNote{{ID: 1}, {ID: 2}}
	uc := alertnote.New(
		&mockNoteRepo{findByAlertIDResult: notes},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	result, err := uc.ListByAlert(context.Background(), orgID, alertID)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListByAlert_AlertNotFound(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDErr: entity.ErrNotFound},
		&mockUserRepo{},
	)

	_, err := uc.ListByAlert(context.Background(), orgID, alertID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestListByAlert_WrongOrg(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDResult: &entity.Alert{ID: alertID, OrgID: 999}},
		&mockUserRepo{},
	)

	_, err := uc.ListByAlert(context.Background(), orgID, alertID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestListByAlert_RepoError(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{findByAlertIDErr: errors.New("db error")},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	_, err := uc.ListByAlert(context.Background(), orgID, alertID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "listing notes")
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{findByIDResult: &entity.User{ID: userID, Email: "user@test.com"}},
	)

	note, err := uc.Create(context.Background(), orgID, alertID, userID, "test content")
	require.NoError(t, err)
	assert.Equal(t, "test content", note.Content)
	assert.Equal(t, "user@test.com", note.UserEmail)
	assert.Equal(t, alertID, note.AlertID)
	assert.Equal(t, orgID, note.OrgID)
	assert.Equal(t, userID, note.UserID)
}

func TestCreate_EmptyContent(t *testing.T) {
	uc := alertnote.New(&mockNoteRepo{}, &mockAlertRepo{}, &mockUserRepo{})

	_, err := uc.Create(context.Background(), orgID, alertID, userID, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}

func TestCreate_ContentTooLong(t *testing.T) {
	uc := alertnote.New(&mockNoteRepo{}, &mockAlertRepo{}, &mockUserRepo{})

	longContent := strings.Repeat("a", entity.MaxNoteLength+1)
	_, err := uc.Create(context.Background(), orgID, alertID, userID, longContent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most")
}

func TestCreate_AlertNotFound(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDErr: entity.ErrNotFound},
		&mockUserRepo{},
	)

	_, err := uc.Create(context.Background(), orgID, alertID, userID, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestCreate_WrongOrg(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDResult: &entity.Alert{ID: alertID, OrgID: 999}},
		&mockUserRepo{},
	)

	_, err := uc.Create(context.Background(), orgID, alertID, userID, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestCreate_UserEmailResolutionFailure_StillCreates(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{findByIDErr: errors.New("user not found")},
	)

	note, err := uc.Create(context.Background(), orgID, alertID, userID, "test")
	require.NoError(t, err)
	assert.Empty(t, note.UserEmail) // graceful fallback
	assert.Equal(t, "test", note.Content)
}

func TestCreate_RepoError(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{createErr: errors.New("db error")},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{findByIDResult: &entity.User{ID: userID, Email: "u@t.com"}},
	)

	_, err := uc.Create(context.Background(), orgID, alertID, userID, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating alert note")
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUpdate_Success(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: ownedNote()},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	note, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "updated content")
	require.NoError(t, err)
	assert.Equal(t, "updated content", note.Content)
}

func TestUpdate_EmptyContent(t *testing.T) {
	uc := alertnote.New(&mockNoteRepo{}, &mockAlertRepo{}, &mockUserRepo{})

	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content is required")
}

func TestUpdate_ContentTooLong(t *testing.T) {
	uc := alertnote.New(&mockNoteRepo{}, &mockAlertRepo{}, &mockUserRepo{})

	longContent := strings.Repeat("a", entity.MaxNoteLength+1)
	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, longContent)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most")
}

func TestUpdate_AlertNotFound(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDErr: entity.ErrNotFound},
		&mockUserRepo{},
	)

	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestUpdate_NoteNotFound(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{findByIDErr: entity.ErrNotFound},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestUpdate_NoteWrongOrg(t *testing.T) {
	wrongOrgNote := ownedNote()
	wrongOrgNote.OrgID = 999

	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: wrongOrgNote},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestUpdate_NoteWrongAlert(t *testing.T) {
	wrongAlertNote := ownedNote()
	wrongAlertNote.AlertID = 999

	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: wrongAlertNote},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestUpdate_NotAuthor(t *testing.T) {
	otherUserNote := ownedNote()
	otherUserNote.UserID = 999

	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: otherUserNote},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "test")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrForbidden))
}

func TestUpdate_RepoError(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: ownedNote(), updateErr: errors.New("db error")},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	_, err := uc.Update(context.Background(), orgID, alertID, noteID, userID, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updating alert note")
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete_Success(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: ownedNote()},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	err := uc.Delete(context.Background(), orgID, alertID, noteID, userID)
	require.NoError(t, err)
}

func TestDelete_AlertNotFound(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{},
		&mockAlertRepo{findByIDErr: entity.ErrNotFound},
		&mockUserRepo{},
	)

	err := uc.Delete(context.Background(), orgID, alertID, noteID, userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestDelete_NoteNotFound(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{findByIDErr: entity.ErrNotFound},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	err := uc.Delete(context.Background(), orgID, alertID, noteID, userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestDelete_NoteWrongOrg(t *testing.T) {
	wrongOrgNote := ownedNote()
	wrongOrgNote.OrgID = 999

	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: wrongOrgNote},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	err := uc.Delete(context.Background(), orgID, alertID, noteID, userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestDelete_NoteWrongAlert(t *testing.T) {
	wrongAlertNote := ownedNote()
	wrongAlertNote.AlertID = 999

	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: wrongAlertNote},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	err := uc.Delete(context.Background(), orgID, alertID, noteID, userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestDelete_NotAuthor(t *testing.T) {
	otherUserNote := ownedNote()
	otherUserNote.UserID = 999

	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: otherUserNote},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	err := uc.Delete(context.Background(), orgID, alertID, noteID, userID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrForbidden))
}

func TestDelete_RepoError(t *testing.T) {
	uc := alertnote.New(
		&mockNoteRepo{findByIDResult: ownedNote(), deleteErr: errors.New("db error")},
		&mockAlertRepo{findByIDResult: orgAlert()},
		&mockUserRepo{},
	)

	err := uc.Delete(context.Background(), orgID, alertID, noteID, userID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deleting alert note")
}
