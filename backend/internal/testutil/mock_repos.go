package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/domain"
)

// ---------------------------------------------------------------------------
// MockUserRepository
// ---------------------------------------------------------------------------

// MockUserRepository is a test double for domain.UserRepository.
type MockUserRepository struct {
	mu    sync.Mutex
	Users map[uint]*domain.User
	nextID uint
	// Errors lets tests inject specific errors for each method.
	Errors struct {
		FindByID    error
		FindByEmail error
		Create      error
		Update      error
	}
	// Calls tracks the number of times each method was invoked.
	Calls struct {
		FindByID    int
		FindByEmail int
		Create      int
		Update      int
	}
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{Users: make(map[uint]*domain.User), nextID: 1}
}

func (m *MockUserRepository) FindByID(_ context.Context, id uint) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	u, ok := m.Users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	clone := *u
	return &clone, nil
}

func (m *MockUserRepository) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByEmail++
	if m.Errors.FindByEmail != nil {
		return nil, m.Errors.FindByEmail
	}
	for _, u := range m.Users {
		if u.Email == email {
			clone := *u
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockUserRepository) Create(_ context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	user.ID = m.nextID
	m.nextID++
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	clone := *user
	m.Users[user.ID] = &clone
	return nil
}

func (m *MockUserRepository) Update(_ context.Context, user *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.Users[user.ID]; !ok {
		return fmt.Errorf("user not found")
	}
	user.UpdatedAt = time.Now()
	clone := *user
	m.Users[user.ID] = &clone
	return nil
}

// ---------------------------------------------------------------------------
// MockRefreshTokenRepository
// ---------------------------------------------------------------------------

type MockRefreshTokenRepository struct {
	mu     sync.Mutex
	tokens map[uint]*domain.RefreshToken
	nextID uint
	Errors struct {
		FindByTokenHash   error
		Create            error
		Delete            error
		DeleteByTokenHash error
	}
	Calls struct {
		FindByTokenHash   int
		Create            int
		Delete            int
		DeleteByTokenHash int
	}
}

func NewMockRefreshTokenRepository() *MockRefreshTokenRepository {
	return &MockRefreshTokenRepository{tokens: make(map[uint]*domain.RefreshToken), nextID: 1}
}

func (m *MockRefreshTokenRepository) FindByTokenHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByTokenHash++
	if m.Errors.FindByTokenHash != nil {
		return nil, m.Errors.FindByTokenHash
	}
	for _, t := range m.tokens {
		if t.TokenHash == hash {
			clone := *t
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("refresh token not found")
}

func (m *MockRefreshTokenRepository) Create(_ context.Context, token *domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	token.ID = m.nextID
	m.nextID++
	token.CreatedAt = time.Now()
	clone := *token
	m.tokens[token.ID] = &clone
	return nil
}

func (m *MockRefreshTokenRepository) Delete(_ context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Delete++
	if m.Errors.Delete != nil {
		return m.Errors.Delete
	}
	delete(m.tokens, id)
	return nil
}

func (m *MockRefreshTokenRepository) DeleteByTokenHash(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteByTokenHash++
	if m.Errors.DeleteByTokenHash != nil {
		return m.Errors.DeleteByTokenHash
	}
	for id, t := range m.tokens {
		if t.TokenHash == hash {
			delete(m.tokens, id)
			return nil
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// MockAPIKeyRepository
// ---------------------------------------------------------------------------

type MockAPIKeyRepository struct {
	mu     sync.Mutex
	keys   map[uint]*domain.APIKey
	nextID uint
	Errors struct {
		FindByID           error
		FindActiveByPrefix error
		FindByUserID       error
		Create             error
		Update             error
		SoftDelete         error
	}
	Calls struct {
		FindByID           int
		FindActiveByPrefix int
		FindByUserID       int
		Create             int
		Update             int
		SoftDelete         int
	}
}

func NewMockAPIKeyRepository() *MockAPIKeyRepository {
	return &MockAPIKeyRepository{keys: make(map[uint]*domain.APIKey), nextID: 1}
}

func (m *MockAPIKeyRepository) FindByID(_ context.Context, id uint) (*domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	k, ok := m.keys[id]
	if !ok {
		return nil, fmt.Errorf("api key not found")
	}
	clone := *k
	return &clone, nil
}

func (m *MockAPIKeyRepository) FindActiveByPrefix(_ context.Context, prefix string) ([]domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindActiveByPrefix++
	if m.Errors.FindActiveByPrefix != nil {
		return nil, m.Errors.FindActiveByPrefix
	}
	var result []domain.APIKey
	for _, k := range m.keys {
		if k.KeyPrefix == prefix && k.IsActive {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (m *MockAPIKeyRepository) FindByUserID(_ context.Context, userID uint) ([]domain.APIKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByUserID++
	if m.Errors.FindByUserID != nil {
		return nil, m.Errors.FindByUserID
	}
	var result []domain.APIKey
	for _, k := range m.keys {
		if k.UserID == userID && k.IsActive {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (m *MockAPIKeyRepository) Create(_ context.Context, key *domain.APIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	key.ID = m.nextID
	m.nextID++
	key.CreatedAt = time.Now()
	clone := *key
	m.keys[key.ID] = &clone
	return nil
}

func (m *MockAPIKeyRepository) Update(_ context.Context, key *domain.APIKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.keys[key.ID]; !ok {
		return fmt.Errorf("api key not found")
	}
	clone := *key
	m.keys[key.ID] = &clone
	return nil
}

func (m *MockAPIKeyRepository) SoftDelete(_ context.Context, userID, keyID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.SoftDelete++
	if m.Errors.SoftDelete != nil {
		return m.Errors.SoftDelete
	}
	k, ok := m.keys[keyID]
	if !ok || k.UserID != userID {
		return fmt.Errorf("api key not found")
	}
	k.IsActive = false
	return nil
}

// ---------------------------------------------------------------------------
// MockPackageRepository
// ---------------------------------------------------------------------------

type MockPackageRepository struct {
	mu       sync.Mutex
	packages map[uint]*domain.Package
	nextID   uint
	Errors   struct {
		FindByID       error
		FindByOrgID    error
		FindByOrgAndName error
		Create         error
		Update         error
		SoftDelete     error
		CountByOrg     error
	}
	Calls struct {
		FindByID       int
		FindByOrgID    int
		FindByOrgAndName int
		Create         int
		Update         int
		SoftDelete     int
		CountByOrg     int
	}
}

func NewMockPackageRepository() *MockPackageRepository {
	return &MockPackageRepository{packages: make(map[uint]*domain.Package), nextID: 1}
}

func (m *MockPackageRepository) FindByID(_ context.Context, id uint) (*domain.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	p, ok := m.packages[id]
	if !ok {
		return nil, fmt.Errorf("package not found")
	}
	clone := *p
	return &clone, nil
}

func (m *MockPackageRepository) FindByOrgID(_ context.Context, orgID uint, page, limit int, _ string) ([]domain.Package, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, 0, m.Errors.FindByOrgID
	}
	var result []domain.Package
	for _, p := range m.packages {
		if p.OrgID == orgID {
			result = append(result, *p)
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

func (m *MockPackageRepository) FindByOrgAndName(_ context.Context, orgID uint, name string, ecosystem domain.Ecosystem) (*domain.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgAndName++
	if m.Errors.FindByOrgAndName != nil {
		return nil, m.Errors.FindByOrgAndName
	}
	for _, p := range m.packages {
		if p.OrgID == orgID && p.Name == name && p.Ecosystem == ecosystem {
			clone := *p
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("package not found")
}

func (m *MockPackageRepository) Create(_ context.Context, pkg *domain.Package) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	pkg.ID = m.nextID
	m.nextID++
	pkg.CreatedAt = time.Now()
	pkg.UpdatedAt = time.Now()
	clone := *pkg
	m.packages[pkg.ID] = &clone
	return nil
}

func (m *MockPackageRepository) Update(_ context.Context, pkg *domain.Package) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.packages[pkg.ID]; !ok {
		return fmt.Errorf("package not found")
	}
	pkg.UpdatedAt = time.Now()
	clone := *pkg
	m.packages[pkg.ID] = &clone
	return nil
}

func (m *MockPackageRepository) SoftDelete(_ context.Context, orgID, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.SoftDelete++
	if m.Errors.SoftDelete != nil {
		return m.Errors.SoftDelete
	}
	p, ok := m.packages[id]
	if !ok || p.OrgID != orgID {
		return fmt.Errorf("package not found")
	}
	delete(m.packages, id)
	return nil
}

func (m *MockPackageRepository) CountByOrg(_ context.Context, orgID uint, ecosystem *domain.Ecosystem) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CountByOrg++
	if m.Errors.CountByOrg != nil {
		return 0, m.Errors.CountByOrg
	}
	var count int64
	for _, p := range m.packages {
		if p.OrgID == orgID {
			if ecosystem == nil || p.Ecosystem == *ecosystem {
				count++
			}
		}
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// MockReleaseRepository
// ---------------------------------------------------------------------------

type MockReleaseRepository struct {
	mu       sync.Mutex
	releases map[uint]*domain.Release
	nextID   uint
	Errors   struct {
		FindByID        error
		FindByPackageID error
		Create          error
		Update          error
	}
	Calls struct {
		FindByID        int
		FindByPackageID int
		Create          int
		Update          int
	}
}

func NewMockReleaseRepository() *MockReleaseRepository {
	return &MockReleaseRepository{releases: make(map[uint]*domain.Release), nextID: 1}
}

func (m *MockReleaseRepository) FindByID(_ context.Context, id uint) (*domain.Release, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	r, ok := m.releases[id]
	if !ok {
		return nil, fmt.Errorf("release not found")
	}
	clone := *r
	return &clone, nil
}

func (m *MockReleaseRepository) FindByPackageID(_ context.Context, packageID uint, page, limit int) ([]domain.Release, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByPackageID++
	if m.Errors.FindByPackageID != nil {
		return nil, 0, m.Errors.FindByPackageID
	}
	var result []domain.Release
	for _, r := range m.releases {
		if r.PackageID == packageID {
			result = append(result, *r)
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

func (m *MockReleaseRepository) Create(_ context.Context, release *domain.Release) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	release.ID = m.nextID
	m.nextID++
	release.CreatedAt = time.Now()
	clone := *release
	m.releases[release.ID] = &clone
	return nil
}

func (m *MockReleaseRepository) Update(_ context.Context, release *domain.Release) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.releases[release.ID]; !ok {
		return fmt.Errorf("release not found")
	}
	clone := *release
	m.releases[release.ID] = &clone
	return nil
}

// ---------------------------------------------------------------------------
// MockDiffRepository
// ---------------------------------------------------------------------------

type MockDiffRepository struct {
	mu     sync.Mutex
	diffs  map[uint]*domain.Diff
	nextID uint
	Errors struct {
		FindByID        error
		FindByReleaseID error
		Create          error
	}
	Calls struct {
		FindByID        int
		FindByReleaseID int
		Create          int
	}
}

func NewMockDiffRepository() *MockDiffRepository {
	return &MockDiffRepository{diffs: make(map[uint]*domain.Diff), nextID: 1}
}

func (m *MockDiffRepository) FindByID(_ context.Context, id uint) (*domain.Diff, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	d, ok := m.diffs[id]
	if !ok {
		return nil, fmt.Errorf("diff not found")
	}
	clone := *d
	return &clone, nil
}

func (m *MockDiffRepository) FindByReleaseID(_ context.Context, releaseID uint) ([]domain.Diff, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByReleaseID++
	if m.Errors.FindByReleaseID != nil {
		return nil, m.Errors.FindByReleaseID
	}
	var result []domain.Diff
	for _, d := range m.diffs {
		if d.ReleaseID == releaseID {
			result = append(result, *d)
		}
	}
	return result, nil
}

func (m *MockDiffRepository) Create(_ context.Context, diff *domain.Diff) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	diff.ID = m.nextID
	m.nextID++
	diff.CreatedAt = time.Now()
	clone := *diff
	m.diffs[diff.ID] = &clone
	return nil
}

// ---------------------------------------------------------------------------
// MockAnalysisRepository
// ---------------------------------------------------------------------------

type MockAnalysisRepository struct {
	mu       sync.Mutex
	analyses map[uint]*domain.Analysis
	nextID   uint
	Errors   struct {
		FindByID      error
		FindByDiffID  error
		Create        error
		CountByDiffID error
	}
	Calls struct {
		FindByID      int
		FindByDiffID  int
		Create        int
		CountByDiffID int
	}
}

func NewMockAnalysisRepository() *MockAnalysisRepository {
	return &MockAnalysisRepository{analyses: make(map[uint]*domain.Analysis), nextID: 1}
}

func (m *MockAnalysisRepository) FindByID(_ context.Context, id uint) (*domain.Analysis, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	a, ok := m.analyses[id]
	if !ok {
		return nil, fmt.Errorf("analysis not found")
	}
	clone := *a
	return &clone, nil
}

func (m *MockAnalysisRepository) FindByDiffID(_ context.Context, diffID uint) ([]domain.Analysis, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByDiffID++
	if m.Errors.FindByDiffID != nil {
		return nil, m.Errors.FindByDiffID
	}
	var result []domain.Analysis
	for _, a := range m.analyses {
		if a.DiffID == diffID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *MockAnalysisRepository) Create(_ context.Context, analysis *domain.Analysis) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	analysis.ID = m.nextID
	m.nextID++
	analysis.CreatedAt = time.Now()
	clone := *analysis
	m.analyses[analysis.ID] = &clone
	return nil
}

func (m *MockAnalysisRepository) CountByDiffID(_ context.Context, diffID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CountByDiffID++
	if m.Errors.CountByDiffID != nil {
		return 0, m.Errors.CountByDiffID
	}
	var count int64
	for _, a := range m.analyses {
		if a.DiffID == diffID {
			count++
		}
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// MockAlertRepository
// ---------------------------------------------------------------------------

type MockAlertRepository struct {
	mu     sync.Mutex
	alerts map[uint]*domain.Alert
	nextID uint
	Errors struct {
		FindByID           error
		FindByOrgID        error
		Create             error
		Update             error
		CountByOrgAndStatus error
	}
	Calls struct {
		FindByID           int
		FindByOrgID        int
		Create             int
		Update             int
		CountByOrgAndStatus int
	}
}

func NewMockAlertRepository() *MockAlertRepository {
	return &MockAlertRepository{alerts: make(map[uint]*domain.Alert), nextID: 1}
}

func (m *MockAlertRepository) FindByID(_ context.Context, id uint) (*domain.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	a, ok := m.alerts[id]
	if !ok {
		return nil, fmt.Errorf("alert not found")
	}
	clone := *a
	return &clone, nil
}

func (m *MockAlertRepository) FindByOrgID(_ context.Context, orgID uint, page, limit int, _ string, filters domain.AlertFilters) ([]domain.Alert, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, 0, m.Errors.FindByOrgID
	}
	var result []domain.Alert
	for _, a := range m.alerts {
		if a.OrgID != orgID {
			continue
		}
		if filters.Severity != nil && a.Severity != *filters.Severity {
			continue
		}
		if filters.Status != nil && a.Status != *filters.Status {
			continue
		}
		result = append(result, *a)
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

func (m *MockAlertRepository) Create(_ context.Context, alert *domain.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	alert.ID = m.nextID
	m.nextID++
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()
	clone := *alert
	m.alerts[alert.ID] = &clone
	return nil
}

func (m *MockAlertRepository) Update(_ context.Context, alert *domain.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.alerts[alert.ID]; !ok {
		return fmt.Errorf("alert not found")
	}
	alert.UpdatedAt = time.Now()
	clone := *alert
	m.alerts[alert.ID] = &clone
	return nil
}

func (m *MockAlertRepository) CountByOrgAndStatus(_ context.Context, orgID uint) (map[domain.AlertStatus]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CountByOrgAndStatus++
	if m.Errors.CountByOrgAndStatus != nil {
		return nil, m.Errors.CountByOrgAndStatus
	}
	counts := make(map[domain.AlertStatus]int64)
	for _, a := range m.alerts {
		if a.OrgID == orgID {
			counts[a.Status]++
		}
	}
	return counts, nil
}

// ---------------------------------------------------------------------------
// MockSettingRepository
// ---------------------------------------------------------------------------

type MockSettingRepository struct {
	mu       sync.Mutex
	settings map[string]*domain.Setting // key = "orgID:key"
	nextID   uint
	Errors   struct {
		FindByOrgID      error
		FindByKey        error
		Upsert           error
		FindOrCreateByKey error
	}
	Calls struct {
		FindByOrgID      int
		FindByKey        int
		Upsert           int
		FindOrCreateByKey int
	}
}

func NewMockSettingRepository() *MockSettingRepository {
	return &MockSettingRepository{settings: make(map[string]*domain.Setting), nextID: 1}
}

func settingKey(orgID uint, key string) string {
	return fmt.Sprintf("%d:%s", orgID, key)
}

func (m *MockSettingRepository) FindByOrgID(_ context.Context, orgID uint) ([]domain.Setting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, m.Errors.FindByOrgID
	}
	var result []domain.Setting
	for _, s := range m.settings {
		if s.OrgID == orgID || s.OrgID == 0 {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *MockSettingRepository) FindByKey(_ context.Context, orgID uint, key string) (*domain.Setting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByKey++
	if m.Errors.FindByKey != nil {
		return nil, m.Errors.FindByKey
	}
	s, ok := m.settings[settingKey(orgID, key)]
	if !ok {
		return nil, fmt.Errorf("setting not found")
	}
	clone := *s
	return &clone, nil
}

func (m *MockSettingRepository) Upsert(_ context.Context, setting *domain.Setting) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Upsert++
	if m.Errors.Upsert != nil {
		return m.Errors.Upsert
	}
	k := settingKey(setting.OrgID, setting.Key)
	if existing, ok := m.settings[k]; ok {
		existing.Value = setting.Value
		existing.UpdatedAt = time.Now()
		setting.ID = existing.ID
	} else {
		setting.ID = m.nextID
		m.nextID++
		setting.CreatedAt = time.Now()
		setting.UpdatedAt = time.Now()
		clone := *setting
		m.settings[k] = &clone
	}
	return nil
}

func (m *MockSettingRepository) FindOrCreateByKey(_ context.Context, key, defaultValue string) (*domain.Setting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindOrCreateByKey++
	if m.Errors.FindOrCreateByKey != nil {
		return nil, m.Errors.FindOrCreateByKey
	}
	k := settingKey(0, key)
	if s, ok := m.settings[k]; ok {
		clone := *s
		return &clone, nil
	}
	s := &domain.Setting{
		ID:    m.nextID,
		Key:   key,
		Value: defaultValue,
	}
	m.nextID++
	clone := *s
	m.settings[k] = &clone
	return s, nil
}

// ---------------------------------------------------------------------------
// MockOrganizationRepository
// ---------------------------------------------------------------------------

type MockOrganizationRepository struct {
	mu     sync.Mutex
	orgs   map[uint]*domain.Organization
	nextID uint
	Errors struct {
		FindByID     error
		FindBySlug   error
		CountBySlug  error
		Create       error
		Update       error
		SoftDelete   error
		FindByUserID error
	}
	Calls struct {
		FindByID     int
		FindBySlug   int
		CountBySlug  int
		Create       int
		Update       int
		SoftDelete   int
		FindByUserID int
	}
}

func NewMockOrganizationRepository() *MockOrganizationRepository {
	return &MockOrganizationRepository{orgs: make(map[uint]*domain.Organization), nextID: 1}
}

func (m *MockOrganizationRepository) FindByID(_ context.Context, id uint) (*domain.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	o, ok := m.orgs[id]
	if !ok {
		return nil, fmt.Errorf("organization not found")
	}
	clone := *o
	return &clone, nil
}

func (m *MockOrganizationRepository) FindBySlug(_ context.Context, slug string) (*domain.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindBySlug++
	if m.Errors.FindBySlug != nil {
		return nil, m.Errors.FindBySlug
	}
	for _, o := range m.orgs {
		if o.Slug == slug {
			clone := *o
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("organization not found")
}

func (m *MockOrganizationRepository) CountBySlug(_ context.Context, slug string, excludeID *uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CountBySlug++
	if m.Errors.CountBySlug != nil {
		return 0, m.Errors.CountBySlug
	}
	var count int64
	for _, o := range m.orgs {
		if o.Slug == slug {
			if excludeID != nil && o.ID == *excludeID {
				continue
			}
			count++
		}
	}
	return count, nil
}

func (m *MockOrganizationRepository) Create(_ context.Context, org *domain.Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	org.ID = m.nextID
	m.nextID++
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()
	clone := *org
	m.orgs[org.ID] = &clone
	return nil
}

func (m *MockOrganizationRepository) Update(_ context.Context, org *domain.Organization) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.orgs[org.ID]; !ok {
		return fmt.Errorf("organization not found")
	}
	org.UpdatedAt = time.Now()
	clone := *org
	m.orgs[org.ID] = &clone
	return nil
}

func (m *MockOrganizationRepository) SoftDelete(_ context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.SoftDelete++
	if m.Errors.SoftDelete != nil {
		return m.Errors.SoftDelete
	}
	if _, ok := m.orgs[id]; !ok {
		return fmt.Errorf("organization not found")
	}
	delete(m.orgs, id)
	return nil
}

func (m *MockOrganizationRepository) FindByUserID(_ context.Context, _ uint) ([]domain.Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByUserID++
	if m.Errors.FindByUserID != nil {
		return nil, m.Errors.FindByUserID
	}
	// Return all orgs (simplified — real impl checks membership)
	var result []domain.Organization
	for _, o := range m.orgs {
		result = append(result, *o)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// MockOrgMemberRepository
// ---------------------------------------------------------------------------

type MockOrgMemberRepository struct {
	mu      sync.Mutex
	members map[string]*domain.OrgMember // key = "userID:orgID"
	nextID  uint
	Errors  struct {
		FindByOrgID        error
		FindByUserAndOrg   error
		CountByUserAndOrg  error
		Create             error
		Update             error
		DeleteByUserAndOrg error
	}
	Calls struct {
		FindByOrgID        int
		FindByUserAndOrg   int
		CountByUserAndOrg  int
		Create             int
		Update             int
		DeleteByUserAndOrg int
	}
}

func NewMockOrgMemberRepository() *MockOrgMemberRepository {
	return &MockOrgMemberRepository{members: make(map[string]*domain.OrgMember), nextID: 1}
}

func memberKey(userID, orgID uint) string {
	return fmt.Sprintf("%d:%d", userID, orgID)
}

func (m *MockOrgMemberRepository) FindByOrgID(_ context.Context, orgID uint) ([]domain.OrgMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, m.Errors.FindByOrgID
	}
	var result []domain.OrgMember
	for _, mem := range m.members {
		if mem.OrgID == orgID {
			result = append(result, *mem)
		}
	}
	return result, nil
}

func (m *MockOrgMemberRepository) FindByUserAndOrg(_ context.Context, userID, orgID uint) (*domain.OrgMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByUserAndOrg++
	if m.Errors.FindByUserAndOrg != nil {
		return nil, m.Errors.FindByUserAndOrg
	}
	mem, ok := m.members[memberKey(userID, orgID)]
	if !ok {
		return nil, fmt.Errorf("membership not found")
	}
	clone := *mem
	return &clone, nil
}

func (m *MockOrgMemberRepository) CountByUserAndOrg(_ context.Context, userID, orgID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CountByUserAndOrg++
	if m.Errors.CountByUserAndOrg != nil {
		return 0, m.Errors.CountByUserAndOrg
	}
	if _, ok := m.members[memberKey(userID, orgID)]; ok {
		return 1, nil
	}
	return 0, nil
}

func (m *MockOrgMemberRepository) Create(_ context.Context, mem *domain.OrgMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	mem.ID = m.nextID
	m.nextID++
	mem.CreatedAt = time.Now()
	mem.UpdatedAt = time.Now()
	clone := *mem
	m.members[memberKey(mem.UserID, mem.OrgID)] = &clone
	return nil
}

func (m *MockOrgMemberRepository) Update(_ context.Context, mem *domain.OrgMember) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	k := memberKey(mem.UserID, mem.OrgID)
	if _, ok := m.members[k]; !ok {
		return fmt.Errorf("membership not found")
	}
	mem.UpdatedAt = time.Now()
	clone := *mem
	m.members[k] = &clone
	return nil
}

func (m *MockOrgMemberRepository) DeleteByUserAndOrg(_ context.Context, userID, orgID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteByUserAndOrg++
	if m.Errors.DeleteByUserAndOrg != nil {
		return m.Errors.DeleteByUserAndOrg
	}
	delete(m.members, memberKey(userID, orgID))
	return nil
}

// ---------------------------------------------------------------------------
// MockNotificationChannelRepository
// ---------------------------------------------------------------------------

type MockNotificationChannelRepository struct {
	mu       sync.Mutex
	channels map[uint]*domain.NotificationChannel
	nextID   uint
	Errors   struct {
		FindByID        error
		FindByIDAndOrg  error
		FindByOrgID     error
		Create          error
		Update          error
		DeleteByIDAndOrg error
	}
	Calls struct {
		FindByID        int
		FindByIDAndOrg  int
		FindByOrgID     int
		Create          int
		Update          int
		DeleteByIDAndOrg int
	}
}

func NewMockNotificationChannelRepository() *MockNotificationChannelRepository {
	return &MockNotificationChannelRepository{channels: make(map[uint]*domain.NotificationChannel), nextID: 1}
}

func (m *MockNotificationChannelRepository) FindByID(_ context.Context, id uint) (*domain.NotificationChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	ch, ok := m.channels[id]
	if !ok {
		return nil, fmt.Errorf("notification channel not found")
	}
	clone := *ch
	return &clone, nil
}

func (m *MockNotificationChannelRepository) FindByIDAndOrg(_ context.Context, id, orgID uint) (*domain.NotificationChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByIDAndOrg++
	if m.Errors.FindByIDAndOrg != nil {
		return nil, m.Errors.FindByIDAndOrg
	}
	ch, ok := m.channels[id]
	if !ok || ch.OrgID != orgID {
		return nil, fmt.Errorf("notification channel not found")
	}
	clone := *ch
	return &clone, nil
}

func (m *MockNotificationChannelRepository) FindByOrgID(_ context.Context, orgID uint) ([]domain.NotificationChannel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, m.Errors.FindByOrgID
	}
	var result []domain.NotificationChannel
	for _, ch := range m.channels {
		if ch.OrgID == orgID {
			result = append(result, *ch)
		}
	}
	return result, nil
}

func (m *MockNotificationChannelRepository) Create(_ context.Context, channel *domain.NotificationChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	channel.ID = m.nextID
	m.nextID++
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()
	clone := *channel
	m.channels[channel.ID] = &clone
	return nil
}

func (m *MockNotificationChannelRepository) Update(_ context.Context, channel *domain.NotificationChannel) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.channels[channel.ID]; !ok {
		return fmt.Errorf("notification channel not found")
	}
	channel.UpdatedAt = time.Now()
	clone := *channel
	m.channels[channel.ID] = &clone
	return nil
}

func (m *MockNotificationChannelRepository) DeleteByIDAndOrg(_ context.Context, id, orgID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteByIDAndOrg++
	if m.Errors.DeleteByIDAndOrg != nil {
		return 0, m.Errors.DeleteByIDAndOrg
	}
	ch, ok := m.channels[id]
	if !ok || ch.OrgID != orgID {
		return 0, nil
	}
	delete(m.channels, id)
	return 1, nil
}

// ---------------------------------------------------------------------------
// MockNotificationRuleRepository
// ---------------------------------------------------------------------------

type MockNotificationRuleRepository struct {
	mu     sync.Mutex
	rules  map[uint]*domain.NotificationRule
	nextID uint
	Errors struct {
		FindByOrgID       error
		FindActiveByOrgID error
		Create            error
		DeleteByIDAndOrg  error
	}
	Calls struct {
		FindByOrgID       int
		FindActiveByOrgID int
		Create            int
		DeleteByIDAndOrg  int
	}
}

func NewMockNotificationRuleRepository() *MockNotificationRuleRepository {
	return &MockNotificationRuleRepository{rules: make(map[uint]*domain.NotificationRule), nextID: 1}
}

func (m *MockNotificationRuleRepository) FindByOrgID(_ context.Context, orgID uint) ([]domain.NotificationRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, m.Errors.FindByOrgID
	}
	var result []domain.NotificationRule
	for _, r := range m.rules {
		if r.OrgID == orgID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *MockNotificationRuleRepository) FindActiveByOrgID(_ context.Context, orgID uint) ([]domain.NotificationRule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindActiveByOrgID++
	if m.Errors.FindActiveByOrgID != nil {
		return nil, m.Errors.FindActiveByOrgID
	}
	var result []domain.NotificationRule
	for _, r := range m.rules {
		if r.OrgID == orgID && r.IsActive {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *MockNotificationRuleRepository) Create(_ context.Context, rule *domain.NotificationRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	rule.ID = m.nextID
	m.nextID++
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	clone := *rule
	m.rules[rule.ID] = &clone
	return nil
}

func (m *MockNotificationRuleRepository) DeleteByIDAndOrg(_ context.Context, id, orgID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteByIDAndOrg++
	if m.Errors.DeleteByIDAndOrg != nil {
		return 0, m.Errors.DeleteByIDAndOrg
	}
	r, ok := m.rules[id]
	if !ok || r.OrgID != orgID {
		return 0, nil
	}
	delete(m.rules, id)
	return 1, nil
}

// ---------------------------------------------------------------------------
// MockNotificationRepository
// ---------------------------------------------------------------------------

type MockNotificationRepository struct {
	mu            sync.Mutex
	notifications map[uint]*domain.Notification
	nextID        uint
	Errors        struct {
		Create       error
		FindByUserAndOrg error
		MarkRead     error
		MarkAllRead  error
		CountUnread  error
	}
	Calls struct {
		Create       int
		FindByUserAndOrg int
		MarkRead     int
		MarkAllRead  int
		CountUnread  int
	}
}

func NewMockNotificationRepository() *MockNotificationRepository {
	return &MockNotificationRepository{notifications: make(map[uint]*domain.Notification), nextID: 1}
}

func (m *MockNotificationRepository) Create(_ context.Context, n *domain.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	n.ID = m.nextID
	m.nextID++
	n.CreatedAt = time.Now()
	clone := *n
	m.notifications[n.ID] = &clone
	return nil
}

func (m *MockNotificationRepository) FindByUserAndOrg(_ context.Context, orgID, userID uint, onlyUnread bool) ([]domain.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByUserAndOrg++
	if m.Errors.FindByUserAndOrg != nil {
		return nil, m.Errors.FindByUserAndOrg
	}
	var result []domain.Notification
	for _, n := range m.notifications {
		if orgID != 0 && n.OrgID != orgID {
			continue
		}
		if n.UserID != userID && n.UserID != 0 {
			continue
		}
		if onlyUnread && n.IsRead {
			continue
		}
		result = append(result, *n)
	}
	return result, nil
}

func (m *MockNotificationRepository) MarkRead(_ context.Context, id, userID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.MarkRead++
	if m.Errors.MarkRead != nil {
		return 0, m.Errors.MarkRead
	}
	n, ok := m.notifications[id]
	if !ok {
		return 0, nil
	}
	if n.UserID != userID && n.UserID != 0 {
		return 0, nil
	}
	n.IsRead = true
	return 1, nil
}

func (m *MockNotificationRepository) MarkAllRead(_ context.Context, orgID, userID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.MarkAllRead++
	if m.Errors.MarkAllRead != nil {
		return 0, m.Errors.MarkAllRead
	}
	var count int64
	for _, n := range m.notifications {
		if orgID != 0 && n.OrgID != orgID {
			continue
		}
		if n.UserID != userID && n.UserID != 0 {
			continue
		}
		if !n.IsRead {
			n.IsRead = true
			count++
		}
	}
	return count, nil
}

func (m *MockNotificationRepository) CountUnread(_ context.Context, orgID, userID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CountUnread++
	if m.Errors.CountUnread != nil {
		return 0, m.Errors.CountUnread
	}
	var count int64
	for _, n := range m.notifications {
		if orgID != 0 && n.OrgID != orgID {
			continue
		}
		if n.UserID != userID && n.UserID != 0 {
			continue
		}
		if !n.IsRead {
			count++
		}
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// MockSessionRepository
// ---------------------------------------------------------------------------

type MockSessionRepository struct {
	mu       sync.Mutex
	sessions map[uint]*domain.Session
	nextID   uint
	Errors   struct {
		FindByID              error
		FindByUserID          error
		FindByTokenHash       error
		Create                error
		UpdateLastActive      error
		UpdateTokenHash       error
		Delete                error
		DeleteExpired         error
		CountByUserID         error
		DeleteOldestByUserID  error
	}
	Calls struct {
		FindByID              int
		FindByUserID          int
		FindByTokenHash       int
		Create                int
		UpdateLastActive      int
		UpdateTokenHash       int
		Delete                int
		DeleteExpired         int
		CountByUserID         int
		DeleteOldestByUserID  int
	}
}

func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{sessions: make(map[uint]*domain.Session), nextID: 1}
}

func (m *MockSessionRepository) FindByID(_ context.Context, id uint) (*domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	clone := *s
	return &clone, nil
}

func (m *MockSessionRepository) FindByUserID(_ context.Context, userID uint) ([]domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByUserID++
	if m.Errors.FindByUserID != nil {
		return nil, m.Errors.FindByUserID
	}
	var result []domain.Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *MockSessionRepository) FindByTokenHash(_ context.Context, hash string) (*domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByTokenHash++
	if m.Errors.FindByTokenHash != nil {
		return nil, m.Errors.FindByTokenHash
	}
	for _, s := range m.sessions {
		if s.TokenHash == hash {
			clone := *s
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("session not found")
}

func (m *MockSessionRepository) Create(_ context.Context, session *domain.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	session.ID = m.nextID
	m.nextID++
	session.CreatedAt = time.Now()
	clone := *session
	m.sessions[session.ID] = &clone
	return nil
}

func (m *MockSessionRepository) UpdateLastActive(_ context.Context, id uint, lastActive time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.UpdateLastActive++
	if m.Errors.UpdateLastActive != nil {
		return m.Errors.UpdateLastActive
	}
	s, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}
	s.LastActive = lastActive
	return nil
}

func (m *MockSessionRepository) UpdateTokenHash(_ context.Context, id uint, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.UpdateTokenHash++
	if m.Errors.UpdateTokenHash != nil {
		return m.Errors.UpdateTokenHash
	}
	s, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}
	s.TokenHash = tokenHash
	return nil
}

func (m *MockSessionRepository) Delete(_ context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Delete++
	if m.Errors.Delete != nil {
		return m.Errors.Delete
	}
	delete(m.sessions, id)
	return nil
}

func (m *MockSessionRepository) DeleteExpired(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteExpired++
	if m.Errors.DeleteExpired != nil {
		return 0, m.Errors.DeleteExpired
	}
	now := time.Now()
	var count int64
	for id, s := range m.sessions {
		if s.ExpiresAt.Before(now) {
			delete(m.sessions, id)
			count++
		}
	}
	return count, nil
}

func (m *MockSessionRepository) CountByUserID(_ context.Context, userID uint) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CountByUserID++
	if m.Errors.CountByUserID != nil {
		return 0, m.Errors.CountByUserID
	}
	var count int64
	for _, s := range m.sessions {
		if s.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *MockSessionRepository) DeleteOldestByUserID(_ context.Context, userID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteOldestByUserID++
	if m.Errors.DeleteOldestByUserID != nil {
		return m.Errors.DeleteOldestByUserID
	}
	var oldestID uint
	var oldestTime time.Time
	first := true
	for _, s := range m.sessions {
		if s.UserID == userID {
			if first || s.CreatedAt.Before(oldestTime) {
				oldestID = s.ID
				oldestTime = s.CreatedAt
				first = false
			}
		}
	}
	if !first {
		delete(m.sessions, oldestID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// MockAuditLogRepository
// ---------------------------------------------------------------------------

type MockAuditLogRepository struct {
	mu     sync.Mutex
	logs   map[uint]*domain.AuditLog
	nextID uint
	Errors struct {
		Create      error
		FindByOrgID error
		FindByID    error
	}
	Calls struct {
		Create      int
		FindByOrgID int
		FindByID    int
	}
}

func NewMockAuditLogRepository() *MockAuditLogRepository {
	return &MockAuditLogRepository{logs: make(map[uint]*domain.AuditLog), nextID: 1}
}

func (m *MockAuditLogRepository) Create(_ context.Context, entry *domain.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	entry.ID = m.nextID
	m.nextID++
	entry.CreatedAt = time.Now()
	clone := *entry
	m.logs[entry.ID] = &clone
	return nil
}

func (m *MockAuditLogRepository) FindByOrgID(_ context.Context, orgID uint, _ domain.AuditLogFilters, page, limit int) ([]domain.AuditLog, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, 0, m.Errors.FindByOrgID
	}
	var result []domain.AuditLog
	for _, l := range m.logs {
		if l.OrgID == orgID {
			result = append(result, *l)
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

func (m *MockAuditLogRepository) FindByID(_ context.Context, id uint) (*domain.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	l, ok := m.logs[id]
	if !ok {
		return nil, fmt.Errorf("audit log not found")
	}
	clone := *l
	return &clone, nil
}

// ---------------------------------------------------------------------------
// MockRoleRepository
// ---------------------------------------------------------------------------

type MockRoleRepository struct {
	mu     sync.Mutex
	roles  map[uint]*domain.Role
	nextID uint
	Errors struct {
		FindByID       error
		FindByIDAndOrg error
		FindByOrgID    error
		Create         error
	}
	Calls struct {
		FindByID       int
		FindByIDAndOrg int
		FindByOrgID    int
		Create         int
	}
}

func NewMockRoleRepository() *MockRoleRepository {
	return &MockRoleRepository{roles: make(map[uint]*domain.Role), nextID: 1}
}

func (m *MockRoleRepository) FindByID(_ context.Context, id uint) (*domain.Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByID++
	if m.Errors.FindByID != nil {
		return nil, m.Errors.FindByID
	}
	r, ok := m.roles[id]
	if !ok {
		return nil, fmt.Errorf("role not found")
	}
	clone := *r
	return &clone, nil
}

func (m *MockRoleRepository) FindByIDAndOrg(_ context.Context, id, orgID uint) (*domain.Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByIDAndOrg++
	if m.Errors.FindByIDAndOrg != nil {
		return nil, m.Errors.FindByIDAndOrg
	}
	r, ok := m.roles[id]
	if !ok || r.OrgID != orgID {
		return nil, fmt.Errorf("role not found")
	}
	clone := *r
	return &clone, nil
}

func (m *MockRoleRepository) FindByOrgID(_ context.Context, orgID uint) ([]domain.Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByOrgID++
	if m.Errors.FindByOrgID != nil {
		return nil, m.Errors.FindByOrgID
	}
	var result []domain.Role
	for _, r := range m.roles {
		if r.OrgID == orgID {
			result = append(result, *r)
		}
	}
	return result, nil
}

func (m *MockRoleRepository) Create(_ context.Context, role *domain.Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	role.ID = m.nextID
	m.nextID++
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	clone := *role
	m.roles[role.ID] = &clone
	return nil
}

// ---------------------------------------------------------------------------
// MockPermissionRepository
// ---------------------------------------------------------------------------

type MockPermissionRepository struct {
	mu          sync.Mutex
	permissions map[uint]*domain.Permission
	nextID      uint
	// UserPermissions maps "userID:orgID:resource:action" to allowed.
	UserPermissions map[string]bool
	Errors struct {
		FindAll             error
		FindOrCreate        error
		CheckUserPermission error
	}
	Calls struct {
		FindAll             int
		FindOrCreate        int
		CheckUserPermission int
	}
}

func NewMockPermissionRepository() *MockPermissionRepository {
	return &MockPermissionRepository{
		permissions:     make(map[uint]*domain.Permission),
		UserPermissions: make(map[string]bool),
		nextID:          1,
	}
}

func (m *MockPermissionRepository) FindAll(_ context.Context) ([]domain.Permission, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindAll++
	if m.Errors.FindAll != nil {
		return nil, m.Errors.FindAll
	}
	var result []domain.Permission
	for _, p := range m.permissions {
		result = append(result, *p)
	}
	return result, nil
}

func (m *MockPermissionRepository) FindOrCreate(_ context.Context, perm *domain.Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindOrCreate++
	if m.Errors.FindOrCreate != nil {
		return m.Errors.FindOrCreate
	}
	for _, p := range m.permissions {
		if p.Resource == perm.Resource && p.Action == perm.Action {
			perm.ID = p.ID
			return nil
		}
	}
	perm.ID = m.nextID
	m.nextID++
	clone := *perm
	m.permissions[perm.ID] = &clone
	return nil
}

func (m *MockPermissionRepository) CheckUserPermission(_ context.Context, userID, orgID uint, resource, action string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.CheckUserPermission++
	if m.Errors.CheckUserPermission != nil {
		return false, m.Errors.CheckUserPermission
	}
	key := fmt.Sprintf("%d:%d:%s:%s", userID, orgID, resource, action)
	return m.UserPermissions[key], nil
}

// ---------------------------------------------------------------------------
// MockInvitationRepository
// ---------------------------------------------------------------------------

type MockInvitationRepository struct {
	mu          sync.Mutex
	invitations map[uint]*domain.Invitation
	nextID      uint
	Errors      struct {
		FindByTokenHash error
		Create          error
		Update          error
	}
	Calls struct {
		FindByTokenHash int
		Create          int
		Update          int
	}
}

func NewMockInvitationRepository() *MockInvitationRepository {
	return &MockInvitationRepository{invitations: make(map[uint]*domain.Invitation), nextID: 1}
}

func (m *MockInvitationRepository) FindByTokenHash(_ context.Context, hash string) (*domain.Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByTokenHash++
	if m.Errors.FindByTokenHash != nil {
		return nil, m.Errors.FindByTokenHash
	}
	for _, inv := range m.invitations {
		if inv.TokenHash == hash {
			clone := *inv
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("invitation not found")
}

func (m *MockInvitationRepository) Create(_ context.Context, inv *domain.Invitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	inv.ID = m.nextID
	m.nextID++
	inv.CreatedAt = time.Now()
	clone := *inv
	m.invitations[inv.ID] = &clone
	return nil
}

func (m *MockInvitationRepository) Update(_ context.Context, inv *domain.Invitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Update++
	if m.Errors.Update != nil {
		return m.Errors.Update
	}
	if _, ok := m.invitations[inv.ID]; !ok {
		return fmt.Errorf("invitation not found")
	}
	clone := *inv
	m.invitations[inv.ID] = &clone
	return nil
}

// ---------------------------------------------------------------------------
// MockPasswordResetTokenRepository
// ---------------------------------------------------------------------------

type MockPasswordResetTokenRepository struct {
	mu     sync.Mutex
	tokens map[uint]*domain.PasswordResetToken
	nextID uint
	Errors struct {
		FindByTokenHash        error
		Create                 error
		MarkUsed               error
		DeleteExpiredByUserID  error
	}
	Calls struct {
		FindByTokenHash        int
		Create                 int
		MarkUsed               int
		DeleteExpiredByUserID  int
	}
}

func NewMockPasswordResetTokenRepository() *MockPasswordResetTokenRepository {
	return &MockPasswordResetTokenRepository{tokens: make(map[uint]*domain.PasswordResetToken), nextID: 1}
}

func (m *MockPasswordResetTokenRepository) FindByTokenHash(_ context.Context, hash string) (*domain.PasswordResetToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByTokenHash++
	if m.Errors.FindByTokenHash != nil {
		return nil, m.Errors.FindByTokenHash
	}
	for _, t := range m.tokens {
		if t.TokenHash == hash {
			clone := *t
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("password reset token not found")
}

func (m *MockPasswordResetTokenRepository) Create(_ context.Context, token *domain.PasswordResetToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	token.ID = m.nextID
	m.nextID++
	token.CreatedAt = time.Now()
	clone := *token
	m.tokens[token.ID] = &clone
	return nil
}

func (m *MockPasswordResetTokenRepository) MarkUsed(_ context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.MarkUsed++
	if m.Errors.MarkUsed != nil {
		return m.Errors.MarkUsed
	}
	t, ok := m.tokens[id]
	if !ok {
		return fmt.Errorf("password reset token not found")
	}
	now := time.Now()
	t.UsedAt = &now
	return nil
}

func (m *MockPasswordResetTokenRepository) DeleteExpiredByUserID(_ context.Context, userID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteExpiredByUserID++
	if m.Errors.DeleteExpiredByUserID != nil {
		return m.Errors.DeleteExpiredByUserID
	}
	now := time.Now()
	for id, t := range m.tokens {
		if t.UserID == userID && (t.ExpiresAt.Before(now) || t.UsedAt != nil) {
			delete(m.tokens, id)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// MockEmailVerificationTokenRepository
// ---------------------------------------------------------------------------

type MockEmailVerificationTokenRepository struct {
	mu     sync.Mutex
	tokens map[uint]*domain.EmailVerificationToken
	nextID uint
	Errors struct {
		FindByTokenHash error
		Create          error
		Delete          error
		DeleteByUserID  error
	}
	Calls struct {
		FindByTokenHash int
		Create          int
		Delete          int
		DeleteByUserID  int
	}
}

func NewMockEmailVerificationTokenRepository() *MockEmailVerificationTokenRepository {
	return &MockEmailVerificationTokenRepository{tokens: make(map[uint]*domain.EmailVerificationToken), nextID: 1}
}

func (m *MockEmailVerificationTokenRepository) FindByTokenHash(_ context.Context, hash string) (*domain.EmailVerificationToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.FindByTokenHash++
	if m.Errors.FindByTokenHash != nil {
		return nil, m.Errors.FindByTokenHash
	}
	for _, t := range m.tokens {
		if t.TokenHash == hash {
			clone := *t
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("email verification token not found")
}

func (m *MockEmailVerificationTokenRepository) Create(_ context.Context, token *domain.EmailVerificationToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Create++
	if m.Errors.Create != nil {
		return m.Errors.Create
	}
	token.ID = m.nextID
	m.nextID++
	token.CreatedAt = time.Now()
	clone := *token
	m.tokens[token.ID] = &clone
	return nil
}

func (m *MockEmailVerificationTokenRepository) Delete(_ context.Context, id uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.Delete++
	if m.Errors.Delete != nil {
		return m.Errors.Delete
	}
	delete(m.tokens, id)
	return nil
}

func (m *MockEmailVerificationTokenRepository) DeleteByUserID(_ context.Context, userID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls.DeleteByUserID++
	if m.Errors.DeleteByUserID != nil {
		return m.Errors.DeleteByUserID
	}
	for id, t := range m.tokens {
		if t.UserID == userID {
			delete(m.tokens, id)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Compile-time interface compliance checks
// ---------------------------------------------------------------------------

var (
	_ domain.UserRepository                   = (*MockUserRepository)(nil)
	_ domain.RefreshTokenRepository           = (*MockRefreshTokenRepository)(nil)
	_ domain.APIKeyRepository                 = (*MockAPIKeyRepository)(nil)
	_ domain.PackageRepository                = (*MockPackageRepository)(nil)
	_ domain.ReleaseRepository                = (*MockReleaseRepository)(nil)
	_ domain.DiffRepository                   = (*MockDiffRepository)(nil)
	_ domain.AnalysisRepository               = (*MockAnalysisRepository)(nil)
	_ domain.AlertRepository                  = (*MockAlertRepository)(nil)
	_ domain.SettingRepository                = (*MockSettingRepository)(nil)
	_ domain.OrganizationRepository           = (*MockOrganizationRepository)(nil)
	_ domain.OrgMemberRepository              = (*MockOrgMemberRepository)(nil)
	_ domain.NotificationChannelRepository    = (*MockNotificationChannelRepository)(nil)
	_ domain.NotificationRuleRepository       = (*MockNotificationRuleRepository)(nil)
	_ domain.NotificationRepository           = (*MockNotificationRepository)(nil)
	_ domain.SessionRepository                = (*MockSessionRepository)(nil)
	_ domain.AuditLogRepository               = (*MockAuditLogRepository)(nil)
	_ domain.RoleRepository                   = (*MockRoleRepository)(nil)
	_ domain.PermissionRepository             = (*MockPermissionRepository)(nil)
	_ domain.InvitationRepository             = (*MockInvitationRepository)(nil)
	_ domain.PasswordResetTokenRepository     = (*MockPasswordResetTokenRepository)(nil)
	_ domain.EmailVerificationTokenRepository = (*MockEmailVerificationTokenRepository)(nil)
)
