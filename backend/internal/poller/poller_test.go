package poller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/registry"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = db.AutoMigrate(
		&models.Package{},
		&models.Release{},
		&models.Diff{},
		&models.Analysis{},
		&models.Alert{},
		&models.Setting{},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})
	return db
}

type mockRegistry struct {
	name        string
	packages    map[string]*registry.PackageInfo
	topPackages []string
	getErr      error
	topErr      error
}

func (m *mockRegistry) Name() string { return m.name }

func (m *mockRegistry) GetPackage(ctx context.Context, name string) (*registry.PackageInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if info, ok := m.packages[name]; ok {
		return info, nil
	}
	return nil, fmt.Errorf("package %s not found", name)
}

func (m *mockRegistry) GetTopPackages(ctx context.Context, limit int) ([]string, error) {
	if m.topErr != nil {
		return nil, m.topErr
	}
	if limit > len(m.topPackages) {
		return m.topPackages, nil
	}
	return m.topPackages[:limit], nil
}

func (m *mockRegistry) DownloadTarball(ctx context.Context, url string) (string, error) {
	return "", fmt.Errorf("not implemented in mock")
}

func TestNew_DefaultConcurrency(t *testing.T) {
	p := New(nil, nil, nil, Config{}, nil)
	assert.Equal(t, 5, p.config.Concurrency)
}

func TestNew_CustomConcurrency(t *testing.T) {
	p := New(nil, nil, nil, Config{Concurrency: 10}, nil)
	assert.Equal(t, 10, p.config.Concurrency)
}

func TestNew_ZeroConcurrency(t *testing.T) {
	p := New(nil, nil, nil, Config{Concurrency: 0}, nil)
	assert.Equal(t, 5, p.config.Concurrency)
}

func TestApplyVersionDepth_Default(t *testing.T) {
	db := setupTestDB(t)
	
	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	versions := []registry.VersionInfo{
		{Version: "3.0.0", PublishedAt: time.Now()},
		{Version: "2.0.0", PublishedAt: time.Now().Add(-1 * time.Hour)},
		{Version: "1.0.0", PublishedAt: time.Now().Add(-2 * time.Hour)},
	}

	result := p.applyVersionDepth(versions, 1)
	assert.Len(t, result, 1)
	assert.Equal(t, "3.0.0", result[0].Version)
}

func TestApplyVersionDepth_LatestMode(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthMode, Value: "latest"})


	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	versions := []registry.VersionInfo{
		{Version: "3.0.0"},
		{Version: "2.0.0"},
		{Version: "1.0.0"},
	}

	result := p.applyVersionDepth(versions, 1)
	assert.Len(t, result, 1)
	assert.Equal(t, "3.0.0", result[0].Version)
}

func TestApplyVersionDepth_CustomMode(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthMode, Value: "custom"})
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthCount, Value: "3"})


	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	versions := []registry.VersionInfo{
		{Version: "5.0.0"},
		{Version: "4.0.0"},
		{Version: "3.0.0"},
		{Version: "2.0.0"},
		{Version: "1.0.0"},
	}

	result := p.applyVersionDepth(versions, 1)
	assert.Len(t, result, 3)
	assert.Equal(t, "5.0.0", result[0].Version)
	assert.Equal(t, "3.0.0", result[2].Version)
}

func TestApplyVersionDepth_CustomMode_DefaultCount(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthMode, Value: "custom"})


	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	versions := make([]registry.VersionInfo, 10)
	for i := range 10 {
		versions[i] = registry.VersionInfo{Version: fmt.Sprintf("%d.0.0", 10-i)}
	}

	result := p.applyVersionDepth(versions, 1)
	assert.Len(t, result, 5)
}

func TestApplyVersionDepth_CustomMode_CountExceedsVersions(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthMode, Value: "custom"})
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthCount, Value: "5"})


	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	versions := []registry.VersionInfo{
		{Version: "2.0.0"},
		{Version: "1.0.0"},
	}

	result := p.applyVersionDepth(versions, 1)
	assert.Len(t, result, 2)
}

func TestApplyVersionDepth_EmptyVersions(t *testing.T) {
	db := setupTestDB(t)

	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	result := p.applyVersionDepth([]registry.VersionInfo{}, 1)
	assert.Empty(t, result)
}

func TestApplyVersionDepth_InvalidCount(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthMode, Value: "custom"})
	db.Create(&models.Setting{OrgID: 1, Key: models.SettingVersionDepthCount, Value: "invalid"})


	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	versions := make([]registry.VersionInfo, 10)
	for i := range 10 {
		versions[i] = registry.VersionInfo{Version: fmt.Sprintf("%d.0.0", 10-i)}
	}

	result := p.applyVersionDepth(versions, 1)
	assert.Len(t, result, 5)
}

func TestCheckPackage_NewRelease(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*registry.PackageInfo{
			"requests": {
				Name:        "requests",
				Version:     "2.31.0",
				Description: "HTTP library",
				Versions: []registry.VersionInfo{
					{Version: "2.31.0", PublishedAt: time.Now(), TarballURL: "https://example.com/2.31.0.tar.gz"},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := models.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)

	err := p.checkPackage(context.Background(), mock, pkg)
	require.NoError(t, err)

	var releases []models.Release
	db.Where("package_id = ?", pkg.ID).Find(&releases)
	assert.Len(t, releases, 1)
	assert.Equal(t, "2.31.0", releases[0].Version)
	assert.Equal(t, models.ReleaseStatusPending, releases[0].Status)

}

func TestCheckPackage_NewRelease_IncludesBaseline(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*registry.PackageInfo{
			"requests": {
				Name:        "requests",
				Version:     "2.31.0",
				Description: "HTTP library",
				Versions: []registry.VersionInfo{
					{Version: "2.31.0", PublishedAt: time.Now(), TarballURL: "https://example.com/2.31.0.tar.gz"},
					{Version: "2.30.0", PublishedAt: time.Now().Add(-24 * time.Hour), TarballURL: "https://example.com/2.30.0.tar.gz"},
					{Version: "2.29.0", PublishedAt: time.Now().Add(-48 * time.Hour), TarballURL: "https://example.com/2.29.0.tar.gz"},
				},
			},
		},
	}

	// "latest only" mode — applyVersionDepth returns [2.31.0]
	// but checkPackage should also include 2.30.0 as baseline
	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := models.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)

	err := p.checkPackage(context.Background(), mock, pkg)
	require.NoError(t, err)

	var releases []models.Release
	db.Where("package_id = ?", pkg.ID).Order("id ASC").Find(&releases)
	assert.Len(t, releases, 2)

	// Oldest first — baseline (2.30.0) should have lower ID
	assert.Equal(t, "2.30.0", releases[0].Version)
	assert.Equal(t, "2.31.0", releases[1].Version)

	// Both sent to channel: baseline first, then latest
}

func TestCheckPackage_ExistingRelease_NoBaseline(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*registry.PackageInfo{
			"requests": {
				Name:    "requests",
				Version: "2.32.0",
				Versions: []registry.VersionInfo{
					{Version: "2.32.0", PublishedAt: time.Now(), TarballURL: "https://example.com/2.32.0.tar.gz"},
					{Version: "2.31.0", PublishedAt: time.Now().Add(-24 * time.Hour)},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := models.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	// Already have a previous release — no baseline needed
	db.Create(&models.Release{PackageID: pkg.ID, Version: "2.31.0", Status: models.ReleaseStatusCompleted})

	err := p.checkPackage(context.Background(), mock, pkg)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Release{}).Where("package_id = ?", pkg.ID).Count(&count)
	// Only 2.32.0 was added (2.31.0 already existed), no baseline added
	assert.Equal(t, int64(2), count)

	// Only the new release sent to channel
}

func TestCheckPackage_ExistingRelease(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*registry.PackageInfo{
			"requests": {
				Name:    "requests",
				Version: "2.31.0",
				Versions: []registry.VersionInfo{
					{Version: "2.31.0"},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := models.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)
	db.Create(&models.Release{PackageID: pkg.ID, Version: "2.31.0", Status: models.ReleaseStatusCompleted})

	err := p.checkPackage(context.Background(), mock, pkg)
	require.NoError(t, err)

	var count int64
	db.Model(&models.Release{}).Where("package_id = ?", pkg.ID).Count(&count)
	assert.Equal(t, int64(1), count)

	assert.Empty(t, nil)
}

func TestCheckPackage_EcosystemError(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name:   "python",
		getErr: fmt.Errorf("network timeout"),
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := models.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)

	err := p.checkPackage(context.Background(), mock, pkg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

func TestCheckPackage_UpdatesMetadata(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*registry.PackageInfo{
			"requests": {
				Name:        "requests",
				Version:     "2.31.0",
				Description: "HTTP library for Python",
				Versions: []registry.VersionInfo{
					{Version: "2.31.0"},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := models.Package{Name: "requests", Ecosystem: "python"}
	db.Create(&pkg)

	p.checkPackage(context.Background(), mock, pkg)

	var updated models.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, "2.31.0", updated.LatestVersion)
	assert.Equal(t, "HTTP library for Python", updated.Description)
}

func TestSyncTopPackages_NewPackages(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name:        "python",
		topPackages: []string{"requests", "boto3", "flask"},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	err := p.SyncTopPackages(context.Background(), mock, 3, 1)
	require.NoError(t, err)

	var packages []models.Package
	db.Find(&packages)
	assert.Len(t, packages, 3)

	var pkg models.Package
	db.Where("name = ?", "requests").First(&pkg)
	assert.Equal(t, uint(1), *pkg.Rank)
	assert.False(t, pkg.IsCustom)

	var pkg2 models.Package
	db.Where("name = ?", "flask").First(&pkg2)
	assert.Equal(t, uint(3), *pkg2.Rank)
}

func TestSyncTopPackages_UpdateExisting(t *testing.T) {
	db := setupTestDB(t)
	

	rank := uint(10)
	db.Create(&models.Package{OrgID: 1, Name: "requests", Ecosystem: "python", Rank: &rank, IsCustom: true})

	mock := &mockRegistry{
		name:        "python",
		topPackages: []string{"requests"},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	err := p.SyncTopPackages(context.Background(), mock, 1, 1)
	require.NoError(t, err)

	var pkg models.Package
	db.Where("name = ?", "requests").First(&pkg)
	assert.Equal(t, uint(1), *pkg.Rank)
	assert.False(t, pkg.IsCustom)
}

func TestSyncTopPackages_EcosystemError(t *testing.T) {
	db := setupTestDB(t)
	

	mock := &mockRegistry{
		name:   "python",
		topErr: fmt.Errorf("API unavailable"),
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	err := p.SyncTopPackages(context.Background(), mock, 10, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API unavailable")
}
