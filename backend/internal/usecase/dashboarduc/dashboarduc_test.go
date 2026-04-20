package dashboarduc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/dashboarduc"
)

// ---------------------------------------------------------------------------
// Mock DashboardRepository
// ---------------------------------------------------------------------------

type mockDashboardRepo struct {
	getStatsResult *entity.DashboardStats
	getStatsErr    error

	releaseActivityResult    []entity.ReleaseActivityPoint
	releaseActivityErr       error

	classificationDistResult []entity.ClassificationCount
	classificationDistErr    error

	baselineCountResult int64
	baselineCountErr    error

	ecosystemDistResult []entity.EcosystemCount
	ecosystemDistErr    error

	alertsBySeverityResult []entity.AlertSeverityCount
	alertsBySeverityErr    error

	releaseStatusDistResult []entity.ReleaseStatusCount
	releaseStatusDistErr    error

	unanalyzedDiffIDsResult []uint
	unanalyzedDiffIDsErr    error
}

func (m *mockDashboardRepo) GetStats(_ context.Context, _ uint) (*entity.DashboardStats, error) {
	if m.getStatsErr != nil {
		return nil, m.getStatsErr
	}
	return m.getStatsResult, nil
}

func (m *mockDashboardRepo) GetReleaseActivity(_ context.Context, _ uint, _, _ time.Time) ([]entity.ReleaseActivityPoint, error) {
	if m.releaseActivityErr != nil {
		return nil, m.releaseActivityErr
	}
	return m.releaseActivityResult, nil
}

func (m *mockDashboardRepo) GetClassificationDistribution(_ context.Context, _ uint, _, _ time.Time) ([]entity.ClassificationCount, error) {
	if m.classificationDistErr != nil {
		return nil, m.classificationDistErr
	}
	return m.classificationDistResult, nil
}

func (m *mockDashboardRepo) GetBaselineCount(_ context.Context, _ uint, _, _ time.Time) (int64, error) {
	if m.baselineCountErr != nil {
		return 0, m.baselineCountErr
	}
	return m.baselineCountResult, nil
}

func (m *mockDashboardRepo) GetEcosystemDistribution(_ context.Context, _ uint) ([]entity.EcosystemCount, error) {
	if m.ecosystemDistErr != nil {
		return nil, m.ecosystemDistErr
	}
	return m.ecosystemDistResult, nil
}

func (m *mockDashboardRepo) GetAlertsBySeverity(_ context.Context, _ uint, _, _ time.Time) ([]entity.AlertSeverityCount, error) {
	if m.alertsBySeverityErr != nil {
		return nil, m.alertsBySeverityErr
	}
	return m.alertsBySeverityResult, nil
}

func (m *mockDashboardRepo) GetReleaseStatusDistribution(_ context.Context, _ uint, _, _ time.Time) ([]entity.ReleaseStatusCount, error) {
	if m.releaseStatusDistErr != nil {
		return nil, m.releaseStatusDistErr
	}
	return m.releaseStatusDistResult, nil
}

func (m *mockDashboardRepo) GetUnanalyzedDiffIDs(_ context.Context, _ uint) ([]uint, error) {
	if m.unanalyzedDiffIDsErr != nil {
		return nil, m.unanalyzedDiffIDsErr
	}
	return m.unanalyzedDiffIDsResult, nil
}

// ---------------------------------------------------------------------------
// Mock ReleaseRepository
// ---------------------------------------------------------------------------

type mockReleaseRepo struct {
	findByWorkspaceIDWithDetailsResult []entity.ReleaseWithDetails
	findByWorkspaceIDWithDetailsTotal  int64
	findByWorkspaceIDWithDetailsErr    error
}

func (m *mockReleaseRepo) FindByID(context.Context, uint) (*entity.Release, error) { return nil, nil }
func (m *mockReleaseRepo) FindByIDWithPackage(context.Context, uint) (*entity.Release, *entity.Package, error) {
	return nil, nil, nil
}
func (m *mockReleaseRepo) FindByIDWithPackageAndWorkspace(context.Context, uint, uint) (*entity.Release, *entity.Package, error) {
	return nil, nil, nil
}
func (m *mockReleaseRepo) FindByPackageID(context.Context, uint, int, int) ([]entity.Release, int64, error) {
	return nil, 0, nil
}
func (m *mockReleaseRepo) FindByPackageIDAndWorkspace(context.Context, uint, uint, int, int) ([]entity.Release, int64, error) {
	return nil, 0, nil
}
func (m *mockReleaseRepo) FindByWorkspaceID(context.Context, uint, int, int, string, entity.ReleaseFilters) ([]entity.Release, int64, error) {
	return nil, 0, nil
}
func (m *mockReleaseRepo) FindByWorkspaceIDWithDetails(_ context.Context, _ uint, _, _ int, _ string, _ entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error) {
	if m.findByWorkspaceIDWithDetailsErr != nil {
		return nil, 0, m.findByWorkspaceIDWithDetailsErr
	}
	return m.findByWorkspaceIDWithDetailsResult, m.findByWorkspaceIDWithDetailsTotal, nil
}
func (m *mockReleaseRepo) FindByPackageIDAll(context.Context, uint) ([]entity.Release, error) {
	return nil, nil
}
func (m *mockReleaseRepo) UpdateStatus(context.Context, uint, entity.ReleaseStatus) error {
	return nil
}
func (m *mockReleaseRepo) Create(context.Context, *entity.Release) error { return nil }
func (m *mockReleaseRepo) Update(context.Context, *entity.Release) error { return nil }

// ---------------------------------------------------------------------------
// Mock DiffRepository
// ---------------------------------------------------------------------------

type mockDiffRepo struct{}

func (m *mockDiffRepo) FindByID(context.Context, uint) (*entity.Diff, error)           { return nil, nil }
func (m *mockDiffRepo) FindByIDAndWorkspace(context.Context, uint, uint) (*entity.Diff, error) {
	return nil, nil
}
func (m *mockDiffRepo) FindByReleaseID(context.Context, uint) ([]entity.Diff, error)   { return nil, nil }
func (m *mockDiffRepo) FindByReleaseIDAndWorkspace(context.Context, uint, uint) ([]entity.Diff, error) {
	return nil, nil
}
func (m *mockDiffRepo) FindFirstByReleaseID(context.Context, uint) (*entity.Diff, error) {
	return nil, nil
}
func (m *mockDiffRepo) FindByReleaseIDs(context.Context, []uint) ([]entity.Diff, error) {
	return nil, nil
}
func (m *mockDiffRepo) Create(context.Context, *entity.Diff) error { return nil }

// ---------------------------------------------------------------------------
// Mock AnalysisRepository
// ---------------------------------------------------------------------------

type mockAnalysisRepo struct{}

func (m *mockAnalysisRepo) FindByID(context.Context, uint) (*entity.Analysis, error) {
	return nil, nil
}
func (m *mockAnalysisRepo) FindByDiffID(context.Context, uint) ([]entity.Analysis, error) {
	return nil, nil
}
func (m *mockAnalysisRepo) FindByDiffIDs(context.Context, []uint) ([]entity.Analysis, error) {
	return nil, nil
}
func (m *mockAnalysisRepo) Create(context.Context, *entity.Analysis) error  { return nil }
func (m *mockAnalysisRepo) CountByDiffID(context.Context, uint) (int64, error) { return 0, nil }

// ---------------------------------------------------------------------------
// Mock QueueEnqueuer
// ---------------------------------------------------------------------------

type mockQueue struct {
	enqueuedIDs []uint
	enqueueErr  error
}

func (m *mockQueue) Enqueue(_ context.Context, _ string, _ uint, refID uint) (string, error) {
	if m.enqueueErr != nil {
		return "", m.enqueueErr
	}
	m.enqueuedIDs = append(m.enqueuedIDs, refID)
	return "job-id", nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

const workspaceID = uint(10)

var (
	from = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to   = time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC) // 3 days
)

func newUC(dash *mockDashboardRepo, q *mockQueue) *dashboarduc.UseCase {
	releases := &mockReleaseRepo{}
	if q == nil {
		return dashboarduc.New(dash, releases, &mockDiffRepo{}, &mockAnalysisRepo{}, nil)
	}
	return dashboarduc.New(dash, releases, &mockDiffRepo{}, &mockAnalysisRepo{}, q)
}

// ---------------------------------------------------------------------------
// GetStats
// ---------------------------------------------------------------------------

func TestGetStats_Success(t *testing.T) {
	stats := &entity.DashboardStats{TotalPackages: 10, ActiveAlerts: 3}
	uc := newUC(&mockDashboardRepo{getStatsResult: stats}, nil)

	result, err := uc.GetStats(context.Background(), workspaceID)
	require.NoError(t, err)
	assert.Equal(t, int64(10), result.TotalPackages)
	assert.Equal(t, int64(3), result.ActiveAlerts)
}

func TestGetStats_RepoError(t *testing.T) {
	uc := newUC(&mockDashboardRepo{getStatsErr: errors.New("db error")}, nil)

	_, err := uc.GetStats(context.Background(), workspaceID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DashboardUseCase.GetStats")
}

// ---------------------------------------------------------------------------
// GetRecentReleases
// ---------------------------------------------------------------------------

func TestGetRecentReleases_Success(t *testing.T) {
	releases := []entity.ReleaseWithDetails{
		{Release: entity.Release{ID: 1}, PackageName: "requests"},
	}
	relRepo := &mockReleaseRepo{
		findByWorkspaceIDWithDetailsResult: releases,
		findByWorkspaceIDWithDetailsTotal:  1,
	}
	uc := dashboarduc.New(&mockDashboardRepo{}, relRepo, &mockDiffRepo{}, &mockAnalysisRepo{}, nil)

	result, total, err := uc.GetRecentReleases(context.Background(), workspaceID, 1, 20, "", entity.ReleaseFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "requests", result[0].PackageName)
}

func TestGetRecentReleases_RepoError(t *testing.T) {
	relRepo := &mockReleaseRepo{findByWorkspaceIDWithDetailsErr: errors.New("db error")}
	uc := dashboarduc.New(&mockDashboardRepo{}, relRepo, &mockDiffRepo{}, &mockAnalysisRepo{}, nil)

	_, _, err := uc.GetRecentReleases(context.Background(), workspaceID, 1, 20, "", entity.ReleaseFilters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DashboardUseCase.GetRecentReleases")
}

// ---------------------------------------------------------------------------
// GetChartData
// ---------------------------------------------------------------------------

func TestGetChartData_FullSuccess(t *testing.T) {
	dash := &mockDashboardRepo{
		releaseActivityResult: []entity.ReleaseActivityPoint{
			{Date: from, Count: 5},
		},
		classificationDistResult: []entity.ClassificationCount{
			{Classification: "benign", Count: 10},
		},
		baselineCountResult: 3,
		ecosystemDistResult: []entity.EcosystemCount{
			{Ecosystem: "python", Count: 8},
		},
		alertsBySeverityResult: []entity.AlertSeverityCount{
			{Severity: "high", Count: 2},
		},
		releaseStatusDistResult: []entity.ReleaseStatusCount{
			{Status: "completed", Count: 7},
		},
	}
	uc := newUC(dash, nil)

	data, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.NoError(t, err)

	// 3 days in range (Jan 1, 2, 3)
	assert.Len(t, data.ReleaseActivity, 3)
	assert.Equal(t, int64(5), data.ReleaseActivity[0].Releases) // day with data
	assert.Equal(t, int64(0), data.ReleaseActivity[1].Releases) // day without data

	// Classifications: 1 row + 1 baseline
	assert.Len(t, data.Classifications, 2)

	// Ecosystems
	assert.Len(t, data.Ecosystems, 1)
	assert.Equal(t, "python", data.Ecosystems[0].Ecosystem)

	// Alerts by severity: always 4 (critical, high, medium, low)
	assert.Len(t, data.AlertsBySeverity, 4)

	// Release statuses
	assert.Len(t, data.ReleaseStatuses, 1)
}

func TestGetChartData_EmptyResults(t *testing.T) {
	dash := &mockDashboardRepo{
		releaseActivityResult:    []entity.ReleaseActivityPoint{},
		classificationDistResult: []entity.ClassificationCount{},
		baselineCountResult:      0,
		ecosystemDistResult:      []entity.EcosystemCount{},
		alertsBySeverityResult:   []entity.AlertSeverityCount{},
		releaseStatusDistResult:  []entity.ReleaseStatusCount{},
	}
	uc := newUC(dash, nil)

	data, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.NoError(t, err)

	// Empty classifications/ecosystems/statuses are empty slices (not nil)
	assert.NotNil(t, data.Classifications)
	assert.Empty(t, data.Classifications)
	assert.NotNil(t, data.Ecosystems)
	assert.Empty(t, data.Ecosystems)
	assert.NotNil(t, data.ReleaseStatuses)
	assert.Empty(t, data.ReleaseStatuses)
}

func TestGetChartData_ReleaseActivityError(t *testing.T) {
	dash := &mockDashboardRepo{releaseActivityErr: errors.New("db error")}
	uc := newUC(dash, nil)

	_, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "release activity")
}

func TestGetChartData_ClassificationDistError(t *testing.T) {
	dash := &mockDashboardRepo{
		releaseActivityResult: []entity.ReleaseActivityPoint{},
		classificationDistErr: errors.New("db error"),
	}
	uc := newUC(dash, nil)

	_, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "classification dist")
}

func TestGetChartData_BaselineCountError(t *testing.T) {
	dash := &mockDashboardRepo{
		releaseActivityResult:    []entity.ReleaseActivityPoint{},
		classificationDistResult: []entity.ClassificationCount{},
		baselineCountErr:         errors.New("db error"),
	}
	uc := newUC(dash, nil)

	_, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "baseline count")
}

func TestGetChartData_EcosystemDistError(t *testing.T) {
	dash := &mockDashboardRepo{
		releaseActivityResult:    []entity.ReleaseActivityPoint{},
		classificationDistResult: []entity.ClassificationCount{},
		baselineCountResult:      0,
		ecosystemDistErr:         errors.New("db error"),
	}
	uc := newUC(dash, nil)

	_, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ecosystem dist")
}

func TestGetChartData_AlertsBySeverityError(t *testing.T) {
	dash := &mockDashboardRepo{
		releaseActivityResult:    []entity.ReleaseActivityPoint{},
		classificationDistResult: []entity.ClassificationCount{},
		baselineCountResult:      0,
		ecosystemDistResult:      []entity.EcosystemCount{},
		alertsBySeverityErr:      errors.New("db error"),
	}
	uc := newUC(dash, nil)

	_, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alerts by severity")
}

func TestGetChartData_ReleaseStatusDistError(t *testing.T) {
	dash := &mockDashboardRepo{
		releaseActivityResult:    []entity.ReleaseActivityPoint{},
		classificationDistResult: []entity.ClassificationCount{},
		baselineCountResult:      0,
		ecosystemDistResult:      []entity.EcosystemCount{},
		alertsBySeverityResult:   []entity.AlertSeverityCount{},
		releaseStatusDistErr:     errors.New("db error"),
	}
	uc := newUC(dash, nil)

	_, err := uc.GetChartData(context.Background(), workspaceID, from, to)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "release statuses")
}

// ---------------------------------------------------------------------------
// ReanalyzeAll
// ---------------------------------------------------------------------------

func TestReanalyzeAll_NilQueue(t *testing.T) {
	uc := newUC(&mockDashboardRepo{}, nil)

	_, err := uc.ReanalyzeAll(context.Background(), workspaceID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue not configured")
}

func TestReanalyzeAll_Success(t *testing.T) {
	dash := &mockDashboardRepo{unanalyzedDiffIDsResult: []uint{1, 2, 3}}
	q := &mockQueue{}
	uc := newUC(dash, q)

	count, err := uc.ReanalyzeAll(context.Background(), workspaceID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.Equal(t, []uint{1, 2, 3}, q.enqueuedIDs)
}

func TestReanalyzeAll_NoDiffs(t *testing.T) {
	dash := &mockDashboardRepo{unanalyzedDiffIDsResult: []uint{}}
	q := &mockQueue{}
	uc := newUC(dash, q)

	count, err := uc.ReanalyzeAll(context.Background(), workspaceID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestReanalyzeAll_GetDiffIDsError(t *testing.T) {
	dash := &mockDashboardRepo{unanalyzedDiffIDsErr: errors.New("db error")}
	q := &mockQueue{}
	uc := newUC(dash, q)

	_, err := uc.ReanalyzeAll(context.Background(), workspaceID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DashboardUseCase.ReanalyzeAll")
}

func TestReanalyzeAll_EnqueueError_ReturnsPartialCount(t *testing.T) {
	// Make the queue fail after the first call
	callCount := 0
	q := &failAfterNQueue{n: 1, callCount: &callCount}
	uc := dashboarduc.New(&mockDashboardRepo{unanalyzedDiffIDsResult: []uint{1, 2, 3}}, &mockReleaseRepo{}, &mockDiffRepo{}, &mockAnalysisRepo{}, q)

	count, err := uc.ReanalyzeAll(context.Background(), workspaceID)
	require.Error(t, err)
	assert.Equal(t, 1, count) // only 1 succeeded before failure
	assert.Contains(t, err.Error(), "enqueue diff")
}

// failAfterNQueue succeeds for the first n calls, then returns an error.
type failAfterNQueue struct {
	n         int
	callCount *int
}

func (q *failAfterNQueue) Enqueue(_ context.Context, _ string, _ uint, _ uint) (string, error) {
	*q.callCount++
	if *q.callCount > q.n {
		return "", errors.New("queue full")
	}
	return "job-id", nil
}
