package alertuc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/alertuc"
)

// ---------------------------------------------------------------------------
// Mock AlertRepository
// ---------------------------------------------------------------------------

type mockAlertRepo struct {
	findByWorkspaceIDWithPackageResult []entity.AlertWithPackage
	findByWorkspaceIDWithPackageTotal  int64
	findByWorkspaceIDWithPackageErr    error

	findByIDWithPackageAlert *entity.Alert
	findByIDWithPackagePkg   *entity.Package
	findByIDWithPackageErr   error

	findByIDResult *entity.Alert
	findByIDErr    error

	updateStatusErr error
}

func (m *mockAlertRepo) FindByID(_ context.Context, id string) (*entity.Alert, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDResult, nil
}

func (m *mockAlertRepo) FindByIDAndWorkspaceID(_ context.Context, _ string, _ string) (*entity.Alert, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	if m.findByIDResult != nil {
		return m.findByIDResult, nil
	}
	return nil, entity.ErrNotFound
}

func (m *mockAlertRepo) FindByIDWithPackage(_ context.Context, _, _ string) (*entity.Alert, *entity.Package, error) {
	if m.findByIDWithPackageErr != nil {
		return nil, nil, m.findByIDWithPackageErr
	}
	return m.findByIDWithPackageAlert, m.findByIDWithPackagePkg, nil
}

func (m *mockAlertRepo) FindByWorkspaceID(_ context.Context, _ string, _, _ int, _ string, _ entity.AlertFilters) ([]entity.Alert, int64, error) {
	return nil, 0, nil // unused
}

func (m *mockAlertRepo) FindByWorkspaceIDWithPackage(_ context.Context, _ string, _, _ int, _ string, _ entity.AlertFilters) ([]entity.AlertWithPackage, int64, error) {
	if m.findByWorkspaceIDWithPackageErr != nil {
		return nil, 0, m.findByWorkspaceIDWithPackageErr
	}
	return m.findByWorkspaceIDWithPackageResult, m.findByWorkspaceIDWithPackageTotal, nil
}

func (m *mockAlertRepo) Create(_ context.Context, _ *entity.Alert) error   { return nil }
func (m *mockAlertRepo) Update(_ context.Context, _ *entity.Alert) error   { return nil }
func (m *mockAlertRepo) UpdateStatus(_ context.Context, _ string, _ string, _ entity.AlertStatus) error {
	return m.updateStatusErr
}
func (m *mockAlertRepo) CountByWorkspaceAndStatus(_ context.Context, _ string) (map[entity.AlertStatus]int64, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock AuditLogger
// ---------------------------------------------------------------------------

type mockAuditLogger struct {
	lastAction   string
	lastResource string
}

func (m *mockAuditLogger) LogAction(_ context.Context, action, resource string, _ string, _ string) {
	m.lastAction = action
	m.lastResource = resource
}

// ---------------------------------------------------------------------------
// ListAlerts
// ---------------------------------------------------------------------------

func TestListAlerts_Success(t *testing.T) {
	wsID := "01935d5a-0000-7000-8000-00000000000a"
	alerts := []entity.AlertWithPackage{
		{Alert: entity.Alert{ID: "01935d5a-0000-7000-8000-000000000001", WorkspaceID: wsID}, PackageName: "requests"},
		{Alert: entity.Alert{ID: "01935d5a-0000-7000-8000-000000000002", WorkspaceID: wsID}, PackageName: "flask"},
	}
	repo := &mockAlertRepo{
		findByWorkspaceIDWithPackageResult: alerts,
		findByWorkspaceIDWithPackageTotal:  2,
	}
	uc := alertuc.New(repo, &mockAuditLogger{})

	result, total, err := uc.ListAlerts(context.Background(), wsID, 1, 20, "", entity.AlertFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)
	assert.Equal(t, "requests", result[0].PackageName)
}

func TestListAlerts_RepoError(t *testing.T) {
	repo := &mockAlertRepo{
		findByWorkspaceIDWithPackageErr: errors.New("db error"),
	}
	uc := alertuc.New(repo, &mockAuditLogger{})

	_, _, err := uc.ListAlerts(context.Background(), "01935d5a-0000-7000-8000-00000000000a", 1, 20, "", entity.AlertFilters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AlertUseCase.ListAlerts")
}

// ---------------------------------------------------------------------------
// GetAlert
// ---------------------------------------------------------------------------

func TestGetAlert_Success(t *testing.T) {
	wsID := "01935d5a-0000-7000-8000-00000000000a"
	alertID := "01935d5a-0000-7000-8000-000000000001"
	alert := &entity.Alert{ID: alertID, WorkspaceID: wsID}
	pkg := &entity.Package{ID: "01935d5a-0000-7000-8000-000000000005", Name: "requests"}
	repo := &mockAlertRepo{
		findByIDWithPackageAlert: alert,
		findByIDWithPackagePkg:   pkg,
	}
	uc := alertuc.New(repo, &mockAuditLogger{})

	gotAlert, gotPkg, err := uc.GetAlert(context.Background(), wsID, alertID)
	require.NoError(t, err)
	assert.Equal(t, alertID, gotAlert.ID)
	assert.Equal(t, "requests", gotPkg.Name)
}

func TestGetAlert_NotFound(t *testing.T) {
	repo := &mockAlertRepo{
		findByIDWithPackageErr: entity.ErrNotFound,
	}
	uc := alertuc.New(repo, &mockAuditLogger{})

	_, _, err := uc.GetAlert(context.Background(), "01935d5a-0000-7000-8000-00000000000a", "01935d5a-0000-7000-8000-0000000003e7")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestGetAlert_RepoError(t *testing.T) {
	repo := &mockAlertRepo{
		findByIDWithPackageErr: errors.New("db error"),
	}
	uc := alertuc.New(repo, &mockAuditLogger{})

	_, _, err := uc.GetAlert(context.Background(), "01935d5a-0000-7000-8000-00000000000a", "01935d5a-0000-7000-8000-000000000001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AlertUseCase.GetAlert")
	assert.False(t, errors.Is(err, entity.ErrNotFound))
}

// ---------------------------------------------------------------------------
// UpdateAlertStatus
// ---------------------------------------------------------------------------

func TestUpdateAlertStatus_Success(t *testing.T) {
	wsID := "01935d5a-0000-7000-8000-00000000000a"
	alertID := "01935d5a-0000-7000-8000-000000000001"
	alert := &entity.Alert{ID: alertID, WorkspaceID: wsID, Status: entity.AlertStatusNew}
	audit := &mockAuditLogger{}
	repo := &mockAlertRepo{findByIDResult: alert}
	uc := alertuc.New(repo, audit)

	updated, err := uc.UpdateAlertStatus(context.Background(), wsID, alertID, entity.AlertStatusAcknowledged)
	require.NoError(t, err)
	assert.Equal(t, entity.AlertStatusAcknowledged, updated.Status)
	assert.Equal(t, "update", audit.lastAction)
	assert.Equal(t, "alert", audit.lastResource)
}

func TestUpdateAlertStatus_AlertNotFound(t *testing.T) {
	repo := &mockAlertRepo{findByIDErr: entity.ErrNotFound}
	uc := alertuc.New(repo, &mockAuditLogger{})

	_, err := uc.UpdateAlertStatus(context.Background(), "01935d5a-0000-7000-8000-00000000000a", "01935d5a-0000-7000-8000-0000000003e7", entity.AlertStatusResolved)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestUpdateAlertStatus_WrongOrg(t *testing.T) {
	repo := &mockAlertRepo{findByIDErr: entity.ErrNotFound}
	uc := alertuc.New(repo, &mockAuditLogger{})

	_, err := uc.UpdateAlertStatus(context.Background(), "01935d5a-0000-7000-8000-000000000063", "01935d5a-0000-7000-8000-000000000001", entity.AlertStatusResolved)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestUpdateAlertStatus_FindRepoError(t *testing.T) {
	repo := &mockAlertRepo{findByIDErr: errors.New("db error")}
	uc := alertuc.New(repo, &mockAuditLogger{})

	_, err := uc.UpdateAlertStatus(context.Background(), "01935d5a-0000-7000-8000-00000000000a", "01935d5a-0000-7000-8000-000000000001", entity.AlertStatusResolved)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AlertUseCase.UpdateAlertStatus")
}

func TestUpdateAlertStatus_UpdateRepoError(t *testing.T) {
	wsID := "01935d5a-0000-7000-8000-00000000000a"
	alertID := "01935d5a-0000-7000-8000-000000000001"
	alert := &entity.Alert{ID: alertID, WorkspaceID: wsID, Status: entity.AlertStatusNew}
	repo := &mockAlertRepo{
		findByIDResult:  alert,
		updateStatusErr: errors.New("db error"),
	}
	uc := alertuc.New(repo, &mockAuditLogger{})

	_, err := uc.UpdateAlertStatus(context.Background(), wsID, alertID, entity.AlertStatusResolved)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AlertUseCase.UpdateAlertStatus")
}
