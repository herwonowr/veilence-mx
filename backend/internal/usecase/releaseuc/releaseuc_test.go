package releaseuc_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/releaseuc"
)

// ---------------------------------------------------------------------------
// Mock PackageRepository
// ---------------------------------------------------------------------------

type mockPackageRepo struct {
	findByIDResult *entity.Package
	findByIDErr    error
}

func (m *mockPackageRepo) FindByID(_ context.Context, _ uint) (*entity.Package, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	return m.findByIDResult, nil
}

// Unused interface methods — satisfy the interface.
func (m *mockPackageRepo) FindByWorkspaceID(context.Context, uint, int, int, string, entity.PackageFilters) ([]entity.Package, int64, error) {
	return nil, 0, nil
}
func (m *mockPackageRepo) FindActiveByWorkspaceID(context.Context, uint) ([]entity.Package, error) {
	return nil, nil
}
func (m *mockPackageRepo) FindByWorkspaceAndName(context.Context, uint, string, entity.Ecosystem) (*entity.Package, error) {
	return nil, nil
}
func (m *mockPackageRepo) Create(context.Context, *entity.Package) error   { return nil }
func (m *mockPackageRepo) Update(context.Context, *entity.Package) error   { return nil }
func (m *mockPackageRepo) BlockPackage(context.Context, uint, uint, string) error { return nil }
func (m *mockPackageRepo) UnblockPackage(context.Context, uint, uint) error      { return nil }
func (m *mockPackageRepo) RemovePackage(context.Context, uint, uint) error       { return nil }
func (m *mockPackageRepo) CountByWorkspace(context.Context, uint, *entity.Ecosystem) (int64, error) {
	return 0, nil
}
func (m *mockPackageRepo) ExistsByWorkspaceAndName(context.Context, uint, string, entity.Ecosystem) (bool, error) {
	return false, nil
}
func (m *mockPackageRepo) FindSuggestionsByWorkspaceID(context.Context, uint, int, int, string, entity.PackageFilters) ([]entity.Package, int64, error) {
	return nil, 0, nil
}
func (m *mockPackageRepo) ApprovePackage(context.Context, uint, uint) error          { return nil }
func (m *mockPackageRepo) RejectPackage(context.Context, uint, uint) error           { return nil }
func (m *mockPackageRepo) BulkApprovePackages(context.Context, uint, []uint) (int, error) {
	return 0, nil
}
func (m *mockPackageRepo) BulkApproveAllSuggestions(context.Context, uint) (int, error) {
	return 0, nil
}
func (m *mockPackageRepo) UpdateDownloadCounts(context.Context, uint, []entity.PackageDownloadUpdate) error {
	return nil
}
func (m *mockPackageRepo) FindStaleByWorkspaceID(context.Context, uint, time.Time) ([]entity.Package, error) {
	return nil, nil
}
func (m *mockPackageRepo) RemoveStaleByWorkspaceID(context.Context, uint, time.Time) (int, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock ReleaseRepository
// ---------------------------------------------------------------------------

type mockReleaseRepo struct {
	findByPackageIDResult []entity.Release
	findByPackageIDTotal  int64
	findByPackageIDErr    error

	findByIDWithPackageRelease *entity.Release
	findByIDWithPackagePkg     *entity.Package
	findByIDWithPackageErr     error

	findByPackageIDAllResult []entity.Release
	findByPackageIDAllErr    error

	updateStatusErr error
}

func (m *mockReleaseRepo) FindByID(context.Context, uint) (*entity.Release, error) { return nil, nil }
func (m *mockReleaseRepo) FindByIDWithPackage(_ context.Context, _ uint) (*entity.Release, *entity.Package, error) {
	if m.findByIDWithPackageErr != nil {
		return nil, nil, m.findByIDWithPackageErr
	}
	return m.findByIDWithPackageRelease, m.findByIDWithPackagePkg, nil
}
func (m *mockReleaseRepo) FindByPackageID(_ context.Context, _ uint, _, _ int) ([]entity.Release, int64, error) {
	if m.findByPackageIDErr != nil {
		return nil, 0, m.findByPackageIDErr
	}
	return m.findByPackageIDResult, m.findByPackageIDTotal, nil
}
func (m *mockReleaseRepo) FindByWorkspaceID(context.Context, uint, int, int, string, entity.ReleaseFilters) ([]entity.Release, int64, error) {
	return nil, 0, nil
}
func (m *mockReleaseRepo) FindByWorkspaceIDWithDetails(context.Context, uint, int, int, string, entity.ReleaseFilters) ([]entity.ReleaseWithDetails, int64, error) {
	return nil, 0, nil
}
func (m *mockReleaseRepo) FindByPackageIDAll(_ context.Context, _ uint) ([]entity.Release, error) {
	if m.findByPackageIDAllErr != nil {
		return nil, m.findByPackageIDAllErr
	}
	return m.findByPackageIDAllResult, nil
}
func (m *mockReleaseRepo) UpdateStatus(_ context.Context, _ uint, _ entity.ReleaseStatus) error {
	return m.updateStatusErr
}
func (m *mockReleaseRepo) Create(context.Context, *entity.Release) error { return nil }
func (m *mockReleaseRepo) Update(context.Context, *entity.Release) error { return nil }

// ---------------------------------------------------------------------------
// Mock DiffRepository
// ---------------------------------------------------------------------------

type mockDiffRepo struct {
	findFirstByReleaseIDResult *entity.Diff
	findFirstByReleaseIDErr    error

	findByReleaseIDsResult []entity.Diff
	findByReleaseIDsErr    error
}

func (m *mockDiffRepo) FindByID(context.Context, uint) (*entity.Diff, error) { return nil, nil }
func (m *mockDiffRepo) FindByReleaseID(context.Context, uint) ([]entity.Diff, error) {
	return nil, nil
}
func (m *mockDiffRepo) FindFirstByReleaseID(_ context.Context, _ uint) (*entity.Diff, error) {
	if m.findFirstByReleaseIDErr != nil {
		return nil, m.findFirstByReleaseIDErr
	}
	return m.findFirstByReleaseIDResult, nil
}
func (m *mockDiffRepo) FindByReleaseIDs(_ context.Context, _ []uint) ([]entity.Diff, error) {
	if m.findByReleaseIDsErr != nil {
		return nil, m.findByReleaseIDsErr
	}
	return m.findByReleaseIDsResult, nil
}
func (m *mockDiffRepo) Create(context.Context, *entity.Diff) error { return nil }

// ---------------------------------------------------------------------------
// Mock AnalysisRepository
// ---------------------------------------------------------------------------

type mockAnalysisRepo struct {
	findByDiffIDResult  []entity.Analysis
	findByDiffIDErr     error
	findByDiffIDsResult []entity.Analysis
	findByDiffIDsErr    error
}

func (m *mockAnalysisRepo) FindByID(context.Context, uint) (*entity.Analysis, error) {
	return nil, nil
}
func (m *mockAnalysisRepo) FindByDiffID(_ context.Context, _ uint) ([]entity.Analysis, error) {
	if m.findByDiffIDErr != nil {
		return nil, m.findByDiffIDErr
	}
	return m.findByDiffIDResult, nil
}
func (m *mockAnalysisRepo) FindByDiffIDs(_ context.Context, _ []uint) ([]entity.Analysis, error) {
	if m.findByDiffIDsErr != nil {
		return nil, m.findByDiffIDsErr
	}
	return m.findByDiffIDsResult, nil
}
func (m *mockAnalysisRepo) Create(context.Context, *entity.Analysis) error { return nil }
func (m *mockAnalysisRepo) CountByDiffID(context.Context, uint) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock QueueEnqueuer
// ---------------------------------------------------------------------------

type mockQueue struct {
	lastJobType    string
	lastRefID      uint
	enqueueResult  string
	enqueueErr     error
}

func (m *mockQueue) Enqueue(_ context.Context, jobType string, refID uint) (string, error) {
	m.lastJobType = jobType
	m.lastRefID = refID
	if m.enqueueErr != nil {
		return "", m.enqueueErr
	}
	return m.enqueueResult, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

const workspaceID = uint(10)

var now = time.Now()

func orgPackage(id uint) *entity.Package {
	return &entity.Package{ID: id, WorkspaceID: workspaceID, Name: "requests", Ecosystem: entity.EcosystemPython}
}

// ---------------------------------------------------------------------------
// ListByPackage
// ---------------------------------------------------------------------------

func TestListByPackage_Success(t *testing.T) {
	releases := []entity.Release{{ID: 1, Version: "1.0"}, {ID: 2, Version: "2.0"}}
	uc := releaseuc.New(
		&mockPackageRepo{findByIDResult: orgPackage(5)},
		&mockReleaseRepo{findByPackageIDResult: releases, findByPackageIDTotal: 2},
		&mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	result, total, err := uc.ListByPackage(context.Background(), workspaceID, 5, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, result, 2)
}

func TestListByPackage_PackageNotFound(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{findByIDErr: entity.ErrNotFound},
		&mockReleaseRepo{}, &mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, _, err := uc.ListByPackage(context.Background(), workspaceID, 999, 1, 20)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestListByPackage_WrongOrg(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{findByIDResult: &entity.Package{ID: 5, WorkspaceID: 999}},
		&mockReleaseRepo{}, &mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, _, err := uc.ListByPackage(context.Background(), workspaceID, 5, 1, 20)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestListByPackage_RepoError(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{findByIDResult: orgPackage(5)},
		&mockReleaseRepo{findByPackageIDErr: errors.New("db error")},
		&mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, _, err := uc.ListByPackage(context.Background(), workspaceID, 5, 1, 20)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ReleaseUseCase.ListByPackage")
}

// ---------------------------------------------------------------------------
// GetRelease
// ---------------------------------------------------------------------------

func TestGetRelease_WithDiffAndAnalysis(t *testing.T) {
	release := &entity.Release{ID: 1, Status: entity.ReleaseStatusCompleted}
	pkg := orgPackage(5)
	diff := &entity.Diff{ID: 10, ReleaseID: 1}
	analyses := []entity.Analysis{{ID: 20, DiffID: 10, Classification: entity.ClassificationBenign}}

	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageRelease: release, findByIDWithPackagePkg: pkg},
		&mockDiffRepo{findFirstByReleaseIDResult: diff},
		&mockAnalysisRepo{findByDiffIDResult: analyses},
		nil,
	)

	detail, err := uc.GetRelease(context.Background(), workspaceID, 1)
	require.NoError(t, err)
	assert.Equal(t, uint(1), detail.Release.ID)
	assert.NotNil(t, detail.Diff)
	assert.NotNil(t, detail.Analysis)
	assert.Equal(t, entity.ClassificationBenign, detail.Analysis.Classification)
	assert.False(t, detail.IsBaseline)
}

func TestGetRelease_BaselineNoDiff(t *testing.T) {
	release := &entity.Release{ID: 1, Status: entity.ReleaseStatusCompleted}
	pkg := orgPackage(5)

	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageRelease: release, findByIDWithPackagePkg: pkg},
		&mockDiffRepo{findFirstByReleaseIDErr: entity.ErrNotFound},
		&mockAnalysisRepo{},
		nil,
	)

	detail, err := uc.GetRelease(context.Background(), workspaceID, 1)
	require.NoError(t, err)
	assert.True(t, detail.IsBaseline)
	assert.Nil(t, detail.Diff)
	assert.Nil(t, detail.Analysis)
}

func TestGetRelease_NotFound(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageErr: entity.ErrNotFound},
		&mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, err := uc.GetRelease(context.Background(), workspaceID, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestGetRelease_WrongOrg(t *testing.T) {
	release := &entity.Release{ID: 1}
	pkg := &entity.Package{ID: 5, WorkspaceID: 999}

	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageRelease: release, findByIDWithPackagePkg: pkg},
		&mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, err := uc.GetRelease(context.Background(), workspaceID, 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

// ---------------------------------------------------------------------------
// ReanalyzeRelease
// ---------------------------------------------------------------------------

func TestReanalyzeRelease_NilQueue(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{}, &mockReleaseRepo{},
		&mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, _, err := uc.ReanalyzeRelease(context.Background(), workspaceID, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue not configured")
}

func TestReanalyzeRelease_NoDiff_EnqueuesDiffJob(t *testing.T) {
	release := &entity.Release{ID: 1, Status: entity.ReleaseStatusCompleted}
	pkg := orgPackage(5)
	q := &mockQueue{enqueueResult: "job-123"}

	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageRelease: release, findByIDWithPackagePkg: pkg},
		&mockDiffRepo{findFirstByReleaseIDErr: entity.ErrNotFound},
		&mockAnalysisRepo{},
		q,
	)

	msg, jobID, err := uc.ReanalyzeRelease(context.Background(), workspaceID, 1)
	require.NoError(t, err)
	assert.Contains(t, msg, "diffing")
	assert.Equal(t, "job-123", jobID)
	assert.Equal(t, "diff", q.lastJobType)
	assert.Equal(t, uint(1), q.lastRefID) // release ID
}

func TestReanalyzeRelease_HasDiff_EnqueuesAnalyzeJob(t *testing.T) {
	release := &entity.Release{ID: 1}
	pkg := orgPackage(5)
	diff := &entity.Diff{ID: 10, ReleaseID: 1}
	q := &mockQueue{enqueueResult: "job-456"}

	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageRelease: release, findByIDWithPackagePkg: pkg},
		&mockDiffRepo{findFirstByReleaseIDResult: diff},
		&mockAnalysisRepo{},
		q,
	)

	msg, jobID, err := uc.ReanalyzeRelease(context.Background(), workspaceID, 1)
	require.NoError(t, err)
	assert.Contains(t, msg, "analysis")
	assert.Equal(t, "job-456", jobID)
	assert.Equal(t, "analyze", q.lastJobType)
	assert.Equal(t, uint(10), q.lastRefID) // diff ID
}

func TestReanalyzeRelease_NotFound(t *testing.T) {
	q := &mockQueue{}
	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageErr: entity.ErrNotFound},
		&mockDiffRepo{}, &mockAnalysisRepo{}, q,
	)

	_, _, err := uc.ReanalyzeRelease(context.Background(), workspaceID, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestReanalyzeRelease_WrongOrg(t *testing.T) {
	release := &entity.Release{ID: 1}
	pkg := &entity.Package{ID: 5, WorkspaceID: 999}
	q := &mockQueue{}

	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageRelease: release, findByIDWithPackagePkg: pkg},
		&mockDiffRepo{}, &mockAnalysisRepo{}, q,
	)

	_, _, err := uc.ReanalyzeRelease(context.Background(), workspaceID, 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestReanalyzeRelease_EnqueueError(t *testing.T) {
	release := &entity.Release{ID: 1}
	pkg := orgPackage(5)
	diff := &entity.Diff{ID: 10, ReleaseID: 1}
	q := &mockQueue{enqueueErr: errors.New("queue full")}

	uc := releaseuc.New(
		&mockPackageRepo{},
		&mockReleaseRepo{findByIDWithPackageRelease: release, findByIDWithPackagePkg: pkg},
		&mockDiffRepo{findFirstByReleaseIDResult: diff},
		&mockAnalysisRepo{},
		q,
	)

	_, _, err := uc.ReanalyzeRelease(context.Background(), workspaceID, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enqueue")
}

// ---------------------------------------------------------------------------
// GetAnalysisHistory
// ---------------------------------------------------------------------------

func TestGetAnalysisHistory_Success(t *testing.T) {
	pubTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	analysisTime := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	releases := []entity.Release{
		{ID: 1, Version: "1.0.0", Status: entity.ReleaseStatusCompleted, PublishedAt: pubTime},
		{ID: 2, Version: "1.1.0", Status: entity.ReleaseStatusCompleted, PublishedAt: pubTime},
	}
	diffs := []entity.Diff{
		{ID: 10, ReleaseID: 2},
	}
	analyses := []entity.Analysis{
		{ID: 20, DiffID: 10, Classification: entity.ClassificationBenign, Confidence: 0.95, CreatedAt: analysisTime},
	}

	uc := releaseuc.New(
		&mockPackageRepo{findByIDResult: orgPackage(5)},
		&mockReleaseRepo{findByPackageIDAllResult: releases},
		&mockDiffRepo{findByReleaseIDsResult: diffs},
		&mockAnalysisRepo{findByDiffIDsResult: analyses},
		nil,
	)

	history, err := uc.GetAnalysisHistory(context.Background(), workspaceID, 5)
	require.NoError(t, err)
	require.Len(t, history, 2)

	// First release: no diff → baseline
	assert.Equal(t, "1.0.0", history[0].Version)
	assert.Equal(t, "baseline", history[0].Classification)

	// Second release: has diff + analysis
	assert.Equal(t, "1.1.0", history[1].Version)
	assert.Equal(t, "benign", history[1].Classification)
	assert.Equal(t, 0.95, history[1].Confidence)
}

func TestGetAnalysisHistory_NoReleases(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{findByIDResult: orgPackage(5)},
		&mockReleaseRepo{findByPackageIDAllResult: []entity.Release{}},
		&mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	history, err := uc.GetAnalysisHistory(context.Background(), workspaceID, 5)
	require.NoError(t, err)
	assert.Empty(t, history)
}

func TestGetAnalysisHistory_PackageNotFound(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{findByIDErr: entity.ErrNotFound},
		&mockReleaseRepo{}, &mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, err := uc.GetAnalysisHistory(context.Background(), workspaceID, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestGetAnalysisHistory_WrongOrg(t *testing.T) {
	uc := releaseuc.New(
		&mockPackageRepo{findByIDResult: &entity.Package{ID: 5, WorkspaceID: 999}},
		&mockReleaseRepo{}, &mockDiffRepo{}, &mockAnalysisRepo{}, nil,
	)

	_, err := uc.GetAnalysisHistory(context.Background(), workspaceID, 5)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestGetAnalysisHistory_DiffLoadError(t *testing.T) {
	releases := []entity.Release{{ID: 1, Version: "1.0"}}

	uc := releaseuc.New(
		&mockPackageRepo{findByIDResult: orgPackage(5)},
		&mockReleaseRepo{findByPackageIDAllResult: releases},
		&mockDiffRepo{findByReleaseIDsErr: errors.New("db error")},
		&mockAnalysisRepo{}, nil,
	)

	_, err := uc.GetAnalysisHistory(context.Background(), workspaceID, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading diffs")
}
