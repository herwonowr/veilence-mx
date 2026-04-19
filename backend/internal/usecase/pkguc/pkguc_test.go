package pkguc_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase/pkguc"
)

// ---------------------------------------------------------------------------
// Mock PackageRepository
// ---------------------------------------------------------------------------

type mockPackageRepo struct {
	packages map[uint]*entity.Package // keyed by ID
	nextID   uint

	// Controls for injecting errors
	findByIDErr          error
	findByWorkspaceIDErr       error
	existsByOrgErr       error
	createErr            error
	blockErr             error
	unblockErr           error
	removeErr            error
	approveErr           error
	rejectErr            error
	bulkApproveErr       error
	findSuggestionsErr   error
	findStaleErr         error
	removeStaleErr       error

	// Track calls
	blockCalls       []blockCall
	unblockCalls     []unblockCall
	removeCalls      []removeCall
	approveCalls     []approveCall
	rejectCalls      []rejectCall
	removeStaleCalls []removeStalCall
}

type blockCall struct {
	workspaceID, pkgID uint
	reason       string
}
type unblockCall struct{ workspaceID, pkgID uint }
type removeCall struct{ workspaceID, pkgID uint }
type approveCall struct{ workspaceID, pkgID uint }
type rejectCall struct{ workspaceID, pkgID uint }
type removeStalCall struct {
	workspaceID uint
	staleBefore time.Time
}

func newMockRepo() *mockPackageRepo {
	return &mockPackageRepo{
		packages: make(map[uint]*entity.Package),
		nextID:   1,
	}
}

// seedPackage inserts a package into the mock store and returns it.
func (m *mockPackageRepo) seedPackage(workspaceID uint, name string, eco entity.Ecosystem, status entity.PackageStatus) *entity.Package {
	pkg := &entity.Package{
		ID:        m.nextID,
		WorkspaceID:     workspaceID,
		Name:      name,
		Ecosystem: eco,
		Status:    status,
		Source:    entity.PackageSourceManual,
	}
	m.packages[m.nextID] = pkg
	m.nextID++
	return pkg
}

func (m *mockPackageRepo) FindByID(_ context.Context, id uint) (*entity.Package, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	pkg, ok := m.packages[id]
	if !ok {
		return nil, entity.ErrNotFound
	}
	return pkg, nil
}

func (m *mockPackageRepo) FindByWorkspaceID(_ context.Context, workspaceID uint, page, limit int, _ string, _ entity.PackageFilters) ([]entity.Package, int64, error) {
	if m.findByWorkspaceIDErr != nil {
		return nil, 0, m.findByWorkspaceIDErr
	}
	var result []entity.Package
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID {
			result = append(result, *pkg)
		}
	}
	// Simple pagination
	start := (page - 1) * limit
	if start > len(result) {
		return nil, int64(len(result)), nil
	}
	end := start + limit
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], int64(len(result)), nil
}

func (m *mockPackageRepo) FindActiveByWorkspaceID(_ context.Context, workspaceID uint) ([]entity.Package, error) {
	var result []entity.Package
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID && pkg.Status == entity.PackageStatusActive {
			result = append(result, *pkg)
		}
	}
	return result, nil
}

func (m *mockPackageRepo) FindByWorkspaceAndName(_ context.Context, workspaceID uint, name string, eco entity.Ecosystem) (*entity.Package, error) {
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID && pkg.Name == name && pkg.Ecosystem == eco {
			return pkg, nil
		}
	}
	return nil, entity.ErrNotFound
}

func (m *mockPackageRepo) ExistsByWorkspaceAndName(_ context.Context, workspaceID uint, name string, eco entity.Ecosystem) (bool, error) {
	if m.existsByOrgErr != nil {
		return false, m.existsByOrgErr
	}
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID && pkg.Name == name && pkg.Ecosystem == eco {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockPackageRepo) Create(_ context.Context, pkg *entity.Package) error {
	if m.createErr != nil {
		return m.createErr
	}
	pkg.ID = m.nextID
	m.packages[m.nextID] = pkg
	m.nextID++
	return nil
}

func (m *mockPackageRepo) Update(_ context.Context, pkg *entity.Package) error {
	m.packages[pkg.ID] = pkg
	return nil
}

func (m *mockPackageRepo) BlockPackage(_ context.Context, workspaceID, pkgID uint, reason string) error {
	m.blockCalls = append(m.blockCalls, blockCall{workspaceID, pkgID, reason})
	if m.blockErr != nil {
		return m.blockErr
	}
	pkg, ok := m.packages[pkgID]
	if !ok || pkg.WorkspaceID != workspaceID {
		return entity.ErrNotFound
	}
	pkg.Status = entity.PackageStatusBlocked
	pkg.BlockedReason = reason
	return nil
}

func (m *mockPackageRepo) UnblockPackage(_ context.Context, workspaceID, pkgID uint) error {
	m.unblockCalls = append(m.unblockCalls, unblockCall{workspaceID, pkgID})
	if m.unblockErr != nil {
		return m.unblockErr
	}
	pkg, ok := m.packages[pkgID]
	if !ok || pkg.WorkspaceID != workspaceID {
		return entity.ErrNotFound
	}
	pkg.Status = entity.PackageStatusActive
	pkg.BlockedReason = ""
	return nil
}

func (m *mockPackageRepo) RemovePackage(_ context.Context, workspaceID, pkgID uint) error {
	m.removeCalls = append(m.removeCalls, removeCall{workspaceID, pkgID})
	if m.removeErr != nil {
		return m.removeErr
	}
	pkg, ok := m.packages[pkgID]
	if !ok || pkg.WorkspaceID != workspaceID {
		return entity.ErrNotFound
	}
	pkg.Status = entity.PackageStatusRemoved
	return nil
}

func (m *mockPackageRepo) CountByWorkspace(_ context.Context, workspaceID uint, eco *entity.Ecosystem) (int64, error) {
	var count int64
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID {
			if eco == nil || pkg.Ecosystem == *eco {
				count++
			}
		}
	}
	return count, nil
}

func (m *mockPackageRepo) FindSuggestionsByWorkspaceID(_ context.Context, workspaceID uint, page, limit int, _ string, _ entity.PackageFilters) ([]entity.Package, int64, error) {
	if m.findSuggestionsErr != nil {
		return nil, 0, m.findSuggestionsErr
	}
	var result []entity.Package
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID && pkg.Status == entity.PackageStatusSuggested {
			result = append(result, *pkg)
		}
	}
	total := int64(len(result))
	start := (page - 1) * limit
	if start > len(result) {
		return nil, total, nil
	}
	end := start + limit
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}

func (m *mockPackageRepo) ApprovePackage(_ context.Context, workspaceID, pkgID uint) error {
	m.approveCalls = append(m.approveCalls, approveCall{workspaceID, pkgID})
	if m.approveErr != nil {
		return m.approveErr
	}
	pkg, ok := m.packages[pkgID]
	if !ok || pkg.WorkspaceID != workspaceID {
		return entity.ErrNotFound
	}
	pkg.Status = entity.PackageStatusActive
	return nil
}

func (m *mockPackageRepo) RejectPackage(_ context.Context, workspaceID, pkgID uint) error {
	m.rejectCalls = append(m.rejectCalls, rejectCall{workspaceID, pkgID})
	if m.rejectErr != nil {
		return m.rejectErr
	}
	pkg, ok := m.packages[pkgID]
	if !ok || pkg.WorkspaceID != workspaceID {
		return entity.ErrNotFound
	}
	pkg.Status = entity.PackageStatusRemoved
	return nil
}

func (m *mockPackageRepo) BulkApprovePackages(_ context.Context, workspaceID uint, pkgIDs []uint) (int, error) {
	if m.bulkApproveErr != nil {
		return 0, m.bulkApproveErr
	}
	count := 0
	for _, id := range pkgIDs {
		pkg, ok := m.packages[id]
		if ok && pkg.WorkspaceID == workspaceID && pkg.Status == entity.PackageStatusSuggested {
			pkg.Status = entity.PackageStatusActive
			count++
		}
	}
	return count, nil
}

func (m *mockPackageRepo) BulkApproveAllSuggestions(_ context.Context, workspaceID uint) (int, error) {
	if m.bulkApproveErr != nil {
		return 0, m.bulkApproveErr
	}
	count := 0
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID && pkg.Status == entity.PackageStatusSuggested {
			pkg.Status = entity.PackageStatusActive
			count++
		}
	}
	return count, nil
}

func (m *mockPackageRepo) UpdateDownloadCounts(_ context.Context, _ uint, _ []entity.PackageDownloadUpdate) error {
	return nil
}

func (m *mockPackageRepo) FindStaleByWorkspaceID(_ context.Context, workspaceID uint, _ time.Time) ([]entity.Package, error) {
	if m.findStaleErr != nil {
		return nil, m.findStaleErr
	}
	var result []entity.Package
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID && pkg.Status == entity.PackageStatusActive {
			result = append(result, *pkg)
		}
	}
	return result, nil
}

func (m *mockPackageRepo) RemoveStaleByWorkspaceID(_ context.Context, workspaceID uint, staleBefore time.Time) (int, error) {
	m.removeStaleCalls = append(m.removeStaleCalls, removeStalCall{workspaceID, staleBefore})
	if m.removeStaleErr != nil {
		return 0, m.removeStaleErr
	}
	count := 0
	for _, pkg := range m.packages {
		if pkg.WorkspaceID == workspaceID && pkg.Status == entity.PackageStatusActive && pkg.UpdatedAt.Before(staleBefore) {
			pkg.Status = entity.PackageStatusRemoved
			count++
		}
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// Mock AuditLogger
// ---------------------------------------------------------------------------

type auditEntry struct {
	action, resource string
	resourceID       uint
	details          string
}

type mockAuditLogger struct {
	entries []auditEntry
}

func (m *mockAuditLogger) LogAction(_ context.Context, action, resource string, resourceID uint, details string) {
	m.entries = append(m.entries, auditEntry{action, resource, resourceID, details})
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func setup() (*mockPackageRepo, *mockAuditLogger, *pkguc.UseCase) {
	repo := newMockRepo()
	audit := &mockAuditLogger{}
	uc := pkguc.New(repo, audit)
	return repo, audit, uc
}

// ===========================================================================
// ListPackages
// ===========================================================================

func TestListPackages_Success(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "requests", entity.EcosystemPython, entity.PackageStatusActive)
	repo.seedPackage(1, "flask", entity.EcosystemPython, entity.PackageStatusActive)
	repo.seedPackage(2, "express", entity.EcosystemNPM, entity.PackageStatusActive) // different org

	pkgs, total, err := uc.ListPackages(context.Background(), 1, 1, 20, "name ASC", entity.PackageFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, pkgs, 2)
}

func TestListPackages_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.findByWorkspaceIDErr = fmt.Errorf("db error")

	_, _, err := uc.ListPackages(context.Background(), 1, 1, 20, "", entity.PackageFilters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// GetPackage
// ===========================================================================

func TestGetPackage_Success(t *testing.T) {
	repo, _, uc := setup()
	seeded := repo.seedPackage(1, "requests", entity.EcosystemPython, entity.PackageStatusActive)

	pkg, err := uc.GetPackage(context.Background(), 1, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, "requests", pkg.Name)
}

func TestGetPackage_NotFound(t *testing.T) {
	_, _, uc := setup()

	_, err := uc.GetPackage(context.Background(), 1, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestGetPackage_WrongOrg(t *testing.T) {
	repo, _, uc := setup()
	seeded := repo.seedPackage(2, "requests", entity.EcosystemPython, entity.PackageStatusActive)

	_, err := uc.GetPackage(context.Background(), 1, seeded.ID) // org 1 can't see org 2's package
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestGetPackage_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.findByIDErr = fmt.Errorf("db error")

	_, err := uc.GetPackage(context.Background(), 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// CreatePackage
// ===========================================================================

func TestCreatePackage_Success(t *testing.T) {
	_, audit, uc := setup()

	pkg, err := uc.CreatePackage(context.Background(), 1, "requests", entity.EcosystemPython)
	require.NoError(t, err)
	assert.Equal(t, "requests", pkg.Name)
	assert.Equal(t, entity.EcosystemPython, pkg.Ecosystem)
	assert.Equal(t, entity.PackageSourceManual, pkg.Source)
	assert.Equal(t, entity.PackageStatusActive, pkg.Status)
	assert.Equal(t, uint(1), pkg.WorkspaceID)
	assert.NotZero(t, pkg.ID)

	// Verify audit log
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "create", audit.entries[0].action)
	assert.Equal(t, "package", audit.entries[0].resource)
	assert.Equal(t, pkg.ID, audit.entries[0].resourceID)
	assert.Contains(t, audit.entries[0].details, "requests")
	assert.Contains(t, audit.entries[0].details, "python")
}

func TestCreatePackage_AlreadyExists(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "requests", entity.EcosystemPython, entity.PackageStatusActive)

	_, err := uc.CreatePackage(context.Background(), 1, "requests", entity.EcosystemPython)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrConflict))
}

func TestCreatePackage_ExistsCheckError(t *testing.T) {
	repo, _, uc := setup()
	repo.existsByOrgErr = fmt.Errorf("db error")

	_, err := uc.CreatePackage(context.Background(), 1, "requests", entity.EcosystemPython)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checking existence")
}

func TestCreatePackage_RepoCreateError(t *testing.T) {
	repo, _, uc := setup()
	repo.createErr = fmt.Errorf("insert failed")

	_, err := uc.CreatePackage(context.Background(), 1, "requests", entity.EcosystemPython)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insert failed")
}

// ===========================================================================
// ImportPackages
// ===========================================================================

func TestImportPackages_Success(t *testing.T) {
	_, _, uc := setup()

	entries := []entity.ImportEntry{
		{Name: "requests", Ecosystem: entity.EcosystemPython},
		{Name: "flask", Ecosystem: entity.EcosystemPython},
	}

	result, err := uc.ImportPackages(context.Background(), 1, entries)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Imported)
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, result.Errors)
}

func TestImportPackages_SkipsDuplicates(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "requests", entity.EcosystemPython, entity.PackageStatusActive)

	entries := []entity.ImportEntry{
		{Name: "requests", Ecosystem: entity.EcosystemPython}, // existing
		{Name: "flask", Ecosystem: entity.EcosystemPython},    // new
	}

	result, err := uc.ImportPackages(context.Background(), 1, entries)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 1, result.Skipped)
}

func TestImportPackages_CreateError(t *testing.T) {
	repo, _, uc := setup()
	repo.createErr = fmt.Errorf("insert failed")

	entries := []entity.ImportEntry{
		{Name: "requests", Ecosystem: entity.EcosystemPython},
	}

	result, err := uc.ImportPackages(context.Background(), 1, entries)
	require.NoError(t, err) // ImportPackages doesn't return error, it collects them
	assert.Equal(t, 0, result.Imported)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "requests", result.Errors[0].Name)
}

func TestImportPackages_ExistsCheckError(t *testing.T) {
	repo, _, uc := setup()
	repo.existsByOrgErr = fmt.Errorf("db error")

	entries := []entity.ImportEntry{
		{Name: "requests", Ecosystem: entity.EcosystemPython},
	}

	result, err := uc.ImportPackages(context.Background(), 1, entries)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
	assert.Len(t, result.Errors, 1)
}

func TestImportPackages_EmptyList(t *testing.T) {
	_, _, uc := setup()

	result, err := uc.ImportPackages(context.Background(), 1, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, result.Errors)
}

// ===========================================================================
// BlockPackage
// ===========================================================================

func TestBlockPackage_Success(t *testing.T) {
	repo, audit, uc := setup()
	seeded := repo.seedPackage(1, "evil-pkg", entity.EcosystemNPM, entity.PackageStatusActive)

	pkg, err := uc.BlockPackage(context.Background(), 1, seeded.ID, "supply chain attack")
	require.NoError(t, err)
	assert.Equal(t, entity.PackageStatusBlocked, pkg.Status)

	// Verify audit log
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "block", audit.entries[0].action)
	assert.Contains(t, audit.entries[0].details, "evil-pkg")
	assert.Contains(t, audit.entries[0].details, "supply chain attack")
}

func TestBlockPackage_WithEmptyReason(t *testing.T) {
	repo, audit, uc := setup()
	seeded := repo.seedPackage(1, "suspicious-pkg", entity.EcosystemPython, entity.PackageStatusActive)

	pkg, err := uc.BlockPackage(context.Background(), 1, seeded.ID, "")
	require.NoError(t, err)
	assert.Equal(t, entity.PackageStatusBlocked, pkg.Status)

	require.Len(t, audit.entries, 1)
	assert.NotContains(t, audit.entries[0].details, ": ") // no trailing ": " when reason is empty
}

func TestBlockPackage_NotFound(t *testing.T) {
	_, _, uc := setup()

	_, err := uc.BlockPackage(context.Background(), 1, 999, "reason")
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestBlockPackage_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "pkg", entity.EcosystemNPM, entity.PackageStatusActive)
	repo.blockErr = fmt.Errorf("db error")

	_, err := uc.BlockPackage(context.Background(), 1, 1, "reason")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// UnblockPackage
// ===========================================================================

func TestUnblockPackage_Success(t *testing.T) {
	repo, audit, uc := setup()
	seeded := repo.seedPackage(1, "unblocked-pkg", entity.EcosystemPython, entity.PackageStatusBlocked)

	pkg, err := uc.UnblockPackage(context.Background(), 1, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.PackageStatusActive, pkg.Status)

	require.Len(t, audit.entries, 1)
	assert.Equal(t, "unblock", audit.entries[0].action)
	assert.Contains(t, audit.entries[0].details, "unblocked-pkg")
}

func TestUnblockPackage_NotFound(t *testing.T) {
	_, _, uc := setup()

	_, err := uc.UnblockPackage(context.Background(), 1, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestUnblockPackage_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "pkg", entity.EcosystemNPM, entity.PackageStatusBlocked)
	repo.unblockErr = fmt.Errorf("db error")

	_, err := uc.UnblockPackage(context.Background(), 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// RemovePackage
// ===========================================================================

func TestRemovePackage_Success(t *testing.T) {
	repo, audit, uc := setup()
	seeded := repo.seedPackage(1, "removed-pkg", entity.EcosystemNPM, entity.PackageStatusActive)

	err := uc.RemovePackage(context.Background(), 1, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.PackageStatusRemoved, repo.packages[seeded.ID].Status)

	require.Len(t, audit.entries, 1)
	assert.Equal(t, "remove", audit.entries[0].action)
	assert.Contains(t, audit.entries[0].details, "removed-pkg")
}

func TestRemovePackage_NotFound_FindByID(t *testing.T) {
	_, _, uc := setup()

	err := uc.RemovePackage(context.Background(), 1, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestRemovePackage_NotFound_Remove(t *testing.T) {
	repo, _, uc := setup()
	seeded := repo.seedPackage(1, "pkg", entity.EcosystemPython, entity.PackageStatusActive)
	repo.removeErr = entity.ErrNotFound

	err := uc.RemovePackage(context.Background(), 1, seeded.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestRemovePackage_RepoError(t *testing.T) {
	repo, _, uc := setup()
	seeded := repo.seedPackage(1, "pkg", entity.EcosystemPython, entity.PackageStatusActive)
	repo.removeErr = fmt.Errorf("db error")

	err := uc.RemovePackage(context.Background(), 1, seeded.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// ApprovePackage
// ===========================================================================

func TestApprovePackage_Success(t *testing.T) {
	repo, audit, uc := setup()
	seeded := repo.seedPackage(1, "suggested-pkg", entity.EcosystemNPM, entity.PackageStatusSuggested)

	pkg, err := uc.ApprovePackage(context.Background(), 1, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.PackageStatusActive, pkg.Status)

	require.Len(t, audit.entries, 1)
	assert.Equal(t, "approve", audit.entries[0].action)
	assert.Contains(t, audit.entries[0].details, "suggested-pkg")
}

func TestApprovePackage_NotFound(t *testing.T) {
	_, _, uc := setup()

	_, err := uc.ApprovePackage(context.Background(), 1, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestApprovePackage_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "pkg", entity.EcosystemPython, entity.PackageStatusSuggested)
	repo.approveErr = fmt.Errorf("db error")

	_, err := uc.ApprovePackage(context.Background(), 1, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// RejectPackage
// ===========================================================================

func TestRejectPackage_Success(t *testing.T) {
	repo, audit, uc := setup()
	seeded := repo.seedPackage(1, "rejected-pkg", entity.EcosystemPython, entity.PackageStatusSuggested)

	err := uc.RejectPackage(context.Background(), 1, seeded.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.PackageStatusRemoved, repo.packages[seeded.ID].Status)

	require.Len(t, audit.entries, 1)
	assert.Equal(t, "reject", audit.entries[0].action)
	assert.Contains(t, audit.entries[0].details, "rejected-pkg")
}

func TestRejectPackage_NotFound_Fetch(t *testing.T) {
	_, _, uc := setup()

	err := uc.RejectPackage(context.Background(), 1, 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestRejectPackage_NotFound_Reject(t *testing.T) {
	repo, _, uc := setup()
	seeded := repo.seedPackage(1, "pkg", entity.EcosystemPython, entity.PackageStatusSuggested)
	repo.rejectErr = entity.ErrNotFound

	err := uc.RejectPackage(context.Background(), 1, seeded.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, entity.ErrNotFound))
}

func TestRejectPackage_RepoError(t *testing.T) {
	repo, _, uc := setup()
	seeded := repo.seedPackage(1, "pkg", entity.EcosystemNPM, entity.PackageStatusSuggested)
	repo.rejectErr = fmt.Errorf("db error")

	err := uc.RejectPackage(context.Background(), 1, seeded.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// BulkApprovePackages
// ===========================================================================

func TestBulkApprovePackages_Success(t *testing.T) {
	repo, audit, uc := setup()
	s1 := repo.seedPackage(1, "pkg-a", entity.EcosystemPython, entity.PackageStatusSuggested)
	s2 := repo.seedPackage(1, "pkg-b", entity.EcosystemPython, entity.PackageStatusSuggested)
	repo.seedPackage(1, "pkg-c", entity.EcosystemNPM, entity.PackageStatusActive) // already active

	count, err := uc.BulkApprovePackages(context.Background(), 1, []uint{s1.ID, s2.ID, 999})
	require.NoError(t, err)
	assert.Equal(t, 2, count) // only 2 suggested, 999 doesn't exist

	require.Len(t, audit.entries, 1)
	assert.Equal(t, "bulk_approve", audit.entries[0].action)
	assert.Contains(t, audit.entries[0].details, "2")
}

func TestBulkApprovePackages_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.bulkApproveErr = fmt.Errorf("db error")

	_, err := uc.BulkApprovePackages(context.Background(), 1, []uint{1, 2})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestBulkApprovePackages_EmptyList(t *testing.T) {
	_, audit, uc := setup()

	count, err := uc.BulkApprovePackages(context.Background(), 1, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.Len(t, audit.entries, 1)
	assert.Contains(t, audit.entries[0].details, "0")
}

// ===========================================================================
// ListSuggestions
// ===========================================================================

func TestListSuggestions_Success(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "suggested-a", entity.EcosystemPython, entity.PackageStatusSuggested)
	repo.seedPackage(1, "suggested-b", entity.EcosystemNPM, entity.PackageStatusSuggested)
	repo.seedPackage(1, "active-pkg", entity.EcosystemPython, entity.PackageStatusActive) // not suggested

	pkgs, total, err := uc.ListSuggestions(context.Background(), 1, 1, 20, "", entity.PackageFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, pkgs, 2)
}

func TestListSuggestions_EmptyOrg(t *testing.T) {
	_, _, uc := setup()

	pkgs, total, err := uc.ListSuggestions(context.Background(), 99, 1, 20, "", entity.PackageFilters{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, pkgs)
}

func TestListSuggestions_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.findSuggestionsErr = fmt.Errorf("db error")

	_, _, err := uc.ListSuggestions(context.Background(), 1, 1, 20, "", entity.PackageFilters{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// ListStalePackages
// ===========================================================================

func TestListStalePackages_Success(t *testing.T) {
	repo, _, uc := setup()
	repo.seedPackage(1, "stale-pkg", entity.EcosystemPython, entity.PackageStatusActive)
	repo.seedPackage(1, "blocked-pkg", entity.EcosystemNPM, entity.PackageStatusBlocked) // not active

	pkgs, err := uc.ListStalePackages(context.Background(), 1, time.Now().Add(-6*30*24*time.Hour))
	require.NoError(t, err)
	assert.Len(t, pkgs, 1) // only active packages
	assert.Equal(t, "stale-pkg", pkgs[0].Name)
}

func TestListStalePackages_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.findStaleErr = fmt.Errorf("db error")

	_, err := uc.ListStalePackages(context.Background(), 1, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// ===========================================================================
// RemoveStalePackages
// ===========================================================================

func TestRemoveStalePackages_Success(t *testing.T) {
	repo, audit, uc := setup()
	// Seed a stale package (updated long ago)
	stale := repo.seedPackage(1, "stale-lib", entity.EcosystemPython, entity.PackageStatusActive)
	stale.UpdatedAt = time.Now().AddDate(0, -7, 0) // 7 months ago

	count, err := uc.RemoveStalePackages(context.Background(), 1, 6)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, entity.PackageStatusRemoved, repo.packages[stale.ID].Status)

	// Verify audit log
	require.Len(t, audit.entries, 1)
	assert.Equal(t, "auto_remove_stale", audit.entries[0].action)
	assert.Contains(t, audit.entries[0].details, "1 stale packages")
	assert.Contains(t, audit.entries[0].details, "6 months")
}

func TestRemoveStalePackages_DisabledWhenZero(t *testing.T) {
	_, audit, uc := setup()

	count, err := uc.RemoveStalePackages(context.Background(), 1, 0)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, audit.entries) // No audit log when disabled
}

func TestRemoveStalePackages_DisabledWhenNegative(t *testing.T) {
	_, audit, uc := setup()

	count, err := uc.RemoveStalePackages(context.Background(), 1, -1)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, audit.entries)
}

func TestRemoveStalePackages_NoStalePackages(t *testing.T) {
	repo, audit, uc := setup()
	// Seed a fresh package (updated recently)
	fresh := repo.seedPackage(1, "fresh-lib", entity.EcosystemNPM, entity.PackageStatusActive)
	fresh.UpdatedAt = time.Now() // updated just now

	count, err := uc.RemoveStalePackages(context.Background(), 1, 6)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Equal(t, entity.PackageStatusActive, repo.packages[fresh.ID].Status)
	assert.Empty(t, audit.entries) // No audit log when nothing removed
}

func TestRemoveStalePackages_RepoError(t *testing.T) {
	repo, _, uc := setup()
	repo.removeStaleErr = fmt.Errorf("db error")

	_, err := uc.RemoveStalePackages(context.Background(), 1, 6)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestRemoveStalePackages_OnlyRemovesWorkspaceScoped(t *testing.T) {
	repo, _, uc := setup()
	// Org 1 stale
	stale1 := repo.seedPackage(1, "stale-a", entity.EcosystemPython, entity.PackageStatusActive)
	stale1.UpdatedAt = time.Now().AddDate(0, -13, 0)
	// Org 2 stale (should NOT be removed when calling for org 1)
	stale2 := repo.seedPackage(2, "stale-b", entity.EcosystemNPM, entity.PackageStatusActive)
	stale2.UpdatedAt = time.Now().AddDate(0, -13, 0)

	count, err := uc.RemoveStalePackages(context.Background(), 1, 12)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, entity.PackageStatusRemoved, repo.packages[stale1.ID].Status)
	assert.Equal(t, entity.PackageStatusActive, repo.packages[stale2.ID].Status) // org 2 untouched
}
