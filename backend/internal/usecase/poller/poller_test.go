package poller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

type mockRegistry struct {
	name        string
	packages    map[string]*entity.RegistryPackageInfo
	topPackages []entity.PackageRanking
	getErr      error
	topErr      error
}

func (m *mockRegistry) Name() string { return m.name }

func (m *mockRegistry) GetPackage(ctx context.Context, name string) (*entity.RegistryPackageInfo, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if info, ok := m.packages[name]; ok {
		return info, nil
	}
	return nil, fmt.Errorf("package %s not found", name)
}

func (m *mockRegistry) GetTopPackages(ctx context.Context, limit int) ([]entity.PackageRanking, error) {
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

// ---------------------------------------------------------------------------
// Config / Constructor Tests
// ---------------------------------------------------------------------------

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

func TestNew_ConfigFields(t *testing.T) {
	p := New(nil, nil, nil, Config{
		MonitoringInterval: 10 * time.Minute,
		DiscoveryInterval:  6 * time.Hour,
		Concurrency:        3,
	}, nil)
	assert.Equal(t, 10*time.Minute, p.config.MonitoringInterval)
	assert.Equal(t, 6*time.Hour, p.config.DiscoveryInterval)
	assert.Equal(t, 3, p.config.Concurrency)
}

// ---------------------------------------------------------------------------
// checkPackageForNewReleases Tests (replaces old checkPackage tests)
// ---------------------------------------------------------------------------

func TestCheckPackageForNewReleases_NewRelease(t *testing.T) {
	db := setupTestDB(t)

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {
				Name:        "requests",
				Version:     "2.31.0",
				Description: "HTTP library",
				Versions: []entity.RegistryVersionInfo{
					{Version: "2.31.0", PublishedAt: time.Now(), TarballURL: "https://example.com/2.31.0.tar.gz"},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	err := p.checkPackageForNewReleases(context.Background(), mock, pkg)
	require.NoError(t, err)

	var releases []persistent.Release
	db.Where("package_id = ?", pkg.ID).Find(&releases)
	assert.Len(t, releases, 1)
	assert.Equal(t, "2.31.0", releases[0].Version)
	assert.Equal(t, persistent.ReleaseStatusPending, releases[0].Status)
}

func TestCheckPackageForNewReleases_FirstTime_IncludesBaseline(t *testing.T) {
	db := setupTestDB(t)

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {
				Name:        "requests",
				Version:     "2.31.0",
				Description: "HTTP library",
				Versions: []entity.RegistryVersionInfo{
					{Version: "2.31.0", PublishedAt: time.Now(), TarballURL: "https://example.com/2.31.0.tar.gz"},
					{Version: "2.30.0", PublishedAt: time.Now().Add(-24 * time.Hour), TarballURL: "https://example.com/2.30.0.tar.gz"},
					{Version: "2.29.0", PublishedAt: time.Now().Add(-48 * time.Hour), TarballURL: "https://example.com/2.29.0.tar.gz"},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	err := p.checkPackageForNewReleases(context.Background(), mock, pkg)
	require.NoError(t, err)

	var releases []persistent.Release
	db.Where("package_id = ?", pkg.ID).Order("id ASC").Find(&releases)
	// First time: takes latest + one baseline = 2 releases
	assert.Len(t, releases, 2)

	// Oldest first — baseline (2.30.0) should have lower ID
	assert.Equal(t, "2.30.0", releases[0].Version)
	assert.Equal(t, "2.31.0", releases[1].Version)
}

func TestCheckPackageForNewReleases_ExistingRelease_NoBaseline(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now()
	mock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {
				Name:    "requests",
				Version: "2.32.0",
				Versions: []entity.RegistryVersionInfo{
					{Version: "2.32.0", PublishedAt: now, TarballURL: "https://example.com/2.32.0.tar.gz"},
					{Version: "2.31.0", PublishedAt: now.Add(-24 * time.Hour)},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)
	// Already have a previous release — no baseline needed
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "2.31.0", PublishedAt: now.Add(-24 * time.Hour), Status: persistent.ReleaseStatusCompleted})

	err := p.checkPackageForNewReleases(context.Background(), mock, pkg)
	require.NoError(t, err)

	var count int64
	db.Model(&persistent.Release{}).Where("package_id = ?", pkg.ID).Count(&count)
	// 2.31.0 existed + 2.32.0 added = 2 total
	assert.Equal(t, int64(2), count)
}

func TestCheckPackageForNewReleases_ExistingRelease_NoNew(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now()
	mock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {
				Name:    "requests",
				Version: "2.31.0",
				Versions: []entity.RegistryVersionInfo{
					{Version: "2.31.0", PublishedAt: now},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "2.31.0", PublishedAt: now, Status: persistent.ReleaseStatusCompleted})

	err := p.checkPackageForNewReleases(context.Background(), mock, pkg)
	require.NoError(t, err)

	var count int64
	db.Model(&persistent.Release{}).Where("package_id = ?", pkg.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestCheckPackageForNewReleases_EcosystemError(t *testing.T) {
	db := setupTestDB(t)

	mock := &mockRegistry{
		name:   "python",
		getErr: fmt.Errorf("network timeout"),
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	err := p.checkPackageForNewReleases(context.Background(), mock, pkg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

func TestCheckPackageForNewReleases_UpdatesMetadata(t *testing.T) {
	db := setupTestDB(t)

	mock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {
				Name:        "requests",
				Version:     "2.31.0",
				Description: "HTTP library for Python",
				Versions: []entity.RegistryVersionInfo{
					{Version: "2.31.0", PublishedAt: time.Now()},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)

	p.checkPackageForNewReleases(context.Background(), mock, pkg)

	var updated persistent.Package
	db.First(&updated, pkg.ID)
	assert.Equal(t, "2.31.0", updated.LatestVersion)
	assert.Equal(t, "HTTP library for Python", updated.Description)
}

// TestCheckPackageForNewReleases_AllMissedReleases verifies that when multiple
// releases are published since the last known, ALL of them are created — not
// just the latest one (the key behavioral change from the old applyVersionDepth).
func TestCheckPackageForNewReleases_AllMissedReleases(t *testing.T) {
	db := setupTestDB(t)

	baseTime := time.Now().Add(-72 * time.Hour) // 3 days ago
	mock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {
				Name:    "requests",
				Version: "2.35.0",
				Versions: []entity.RegistryVersionInfo{
					{Version: "2.35.0", PublishedAt: baseTime.Add(48 * time.Hour), TarballURL: "https://example.com/2.35.0.tar.gz"},
					{Version: "2.34.0", PublishedAt: baseTime.Add(36 * time.Hour), TarballURL: "https://example.com/2.34.0.tar.gz"},
					{Version: "2.33.0", PublishedAt: baseTime.Add(24 * time.Hour), TarballURL: "https://example.com/2.33.0.tar.gz"},
					{Version: "2.32.0", PublishedAt: baseTime.Add(12 * time.Hour), TarballURL: "https://example.com/2.32.0.tar.gz"},
					{Version: "2.31.0", PublishedAt: baseTime}, // This is the known release
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)
	// We already know 2.31.0
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "2.31.0", PublishedAt: baseTime, Status: persistent.ReleaseStatusCompleted})

	err := p.checkPackageForNewReleases(context.Background(), mock, pkg)
	require.NoError(t, err)

	var releases []persistent.Release
	db.Where("package_id = ?", pkg.ID).Order("published_at ASC").Find(&releases)
	// 2.31.0 (existing) + 2.32.0 + 2.33.0 + 2.34.0 + 2.35.0 = 5 total
	assert.Len(t, releases, 5)
	assert.Equal(t, "2.31.0", releases[0].Version)
	assert.Equal(t, "2.32.0", releases[1].Version)
	assert.Equal(t, "2.33.0", releases[2].Version)
	assert.Equal(t, "2.34.0", releases[3].Version)
	assert.Equal(t, "2.35.0", releases[4].Version)
}

func TestCheckPackageForNewReleases_Idempotent(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now()
	mock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {
				Name:    "requests",
				Version: "2.32.0",
				Versions: []entity.RegistryVersionInfo{
					{Version: "2.32.0", PublishedAt: now, TarballURL: "https://example.com/2.32.0.tar.gz"},
					{Version: "2.31.0", PublishedAt: now.Add(-24 * time.Hour)},
				},
			},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	pkg := persistent.Package{Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered}
	db.Create(&pkg)
	db.Create(&persistent.Release{PackageID: pkg.ID, Version: "2.31.0", PublishedAt: now.Add(-24 * time.Hour), Status: persistent.ReleaseStatusCompleted})

	// Run twice — should not create duplicate releases
	err := p.checkPackageForNewReleases(context.Background(), mock, pkg)
	require.NoError(t, err)
	err = p.checkPackageForNewReleases(context.Background(), mock, pkg)
	require.NoError(t, err)

	var count int64
	db.Model(&persistent.Release{}).Where("package_id = ?", pkg.ID).Count(&count)
	assert.Equal(t, int64(2), count)
}

// ---------------------------------------------------------------------------
// Discovery Tests (replaces old SyncTopPackages tests)
// ---------------------------------------------------------------------------

func TestDiscoverPackages_NewPackages(t *testing.T) {
	db := setupTestDB(t)

	mock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "requests", Rank: 1},
			{Name: "boto3", Rank: 2},
			{Name: "flask", Rank: 3},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	p.discoverPackages(context.Background(), mock, 3, 1)

	var packages []persistent.Package
	db.Find(&packages)
	assert.Len(t, packages, 3)

	var pkg persistent.Package
	db.Where("name = ?", "requests").First(&pkg)
	assert.Equal(t, uint(1), *pkg.Rank)
	assert.Equal(t, persistent.PackageSourceDiscovered, pkg.Source)
	assert.Equal(t, persistent.PackageStatusActive, pkg.Status)

	var pkg2 persistent.Package
	db.Where("name = ?", "flask").First(&pkg2)
	assert.Equal(t, uint(3), *pkg2.Rank)
}

func TestDiscoverPackages_UpdateExistingRank(t *testing.T) {
	db := setupTestDB(t)

	rank := uint(10)
	db.Create(&persistent.Package{OrgID: 1, Name: "requests", Ecosystem: "python", Rank: &rank, Source: persistent.PackageSourceDiscovered, Status: persistent.PackageStatusActive})

	mock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "requests", Rank: 1},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	p.discoverPackages(context.Background(), mock, 1, 1)

	var pkg persistent.Package
	db.Where("name = ?", "requests").First(&pkg)
	assert.Equal(t, uint(1), *pkg.Rank)
	// Source should remain unchanged
	assert.Equal(t, persistent.PackageSourceDiscovered, pkg.Source)
}

func TestDiscoverPackages_SkipsBlockedPackages(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now()
	rank := uint(10)
	db.Create(&persistent.Package{
		OrgID:         1,
		Name:          "malicious-pkg",
		Ecosystem:     "python",
		Rank:          &rank,
		Source:        persistent.PackageSourceDiscovered,
		Status:        persistent.PackageStatusBlocked,
		BlockedAt:     &now,
		BlockedReason: "known malware",
	})

	mock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "malicious-pkg", Rank: 1},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	p.discoverPackages(context.Background(), mock, 1, 1)

	var pkg persistent.Package
	db.Where("name = ?", "malicious-pkg").First(&pkg)
	// Status remains blocked
	assert.Equal(t, persistent.PackageStatusBlocked, pkg.Status)
	// Rank NOT updated for blocked packages
	assert.Equal(t, uint(10), *pkg.Rank)
}

func TestDiscoverPackages_ReAddsRemovedPackages(t *testing.T) {
	db := setupTestDB(t)

	rank := uint(5)
	db.Create(&persistent.Package{
		OrgID:     1,
		Name:      "requests",
		Ecosystem: "python",
		Rank:      &rank,
		Source:    persistent.PackageSourceDiscovered,
		Status:    persistent.PackageStatusRemoved,
	})

	mock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "requests", Rank: 1},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	p.discoverPackages(context.Background(), mock, 1, 1)

	var pkg persistent.Package
	db.Where("name = ?", "requests").First(&pkg)
	// Re-added: status back to active with updated rank
	assert.Equal(t, persistent.PackageStatusActive, pkg.Status)
	assert.Equal(t, uint(1), *pkg.Rank)
}

func TestDiscoverPackages_RegistryError(t *testing.T) {
	db := setupTestDB(t)

	mock := &mockRegistry{
		name:   "python",
		topErr: fmt.Errorf("API unavailable"),
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	// Should not panic — just logs the error
	p.discoverPackages(context.Background(), mock, 10, 1)

	var count int64
	db.Model(&persistent.Package{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDiscoverPackages_AdditiveOnly(t *testing.T) {
	db := setupTestDB(t)

	// Pre-existing packages that are NOT in the new top list
	rank := uint(1)
	db.Create(&persistent.Package{OrgID: 1, Name: "old-pkg", Ecosystem: "python", Rank: &rank, Source: persistent.PackageSourceDiscovered, Status: persistent.PackageStatusActive})

	mock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "new-pkg", Rank: 1},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	p.discoverPackages(context.Background(), mock, 1, 1)

	var count int64
	db.Model(&persistent.Package{}).Count(&count)
	// old-pkg still exists + new-pkg added = 2
	assert.Equal(t, int64(2), count)

	var oldPkg persistent.Package
	db.Where("name = ?", "old-pkg").First(&oldPkg)
	// old-pkg NOT removed — discovery is additive only
	assert.Equal(t, persistent.PackageStatusActive, oldPkg.Status)
}

// TestSyncTopPackages_BackwardCompat verifies the legacy SyncTopPackages wrapper works.
func TestSyncTopPackages_BackwardCompat(t *testing.T) {
	db := setupTestDB(t)

	mock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "requests", Rank: 1},
			{Name: "flask", Rank: 2},
		},
	}

	p := New(db, mock, nil, Config{Concurrency: 1}, nil)

	err := p.SyncTopPackages(context.Background(), mock, 2, 1)
	require.NoError(t, err)

	var count int64
	db.Model(&persistent.Package{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

// ---------------------------------------------------------------------------
// Settings & Interval Tests
// ---------------------------------------------------------------------------

func TestGetOrgMonitoringInterval_DefaultFallback(t *testing.T) {
	db := setupTestDB(t)

	p := New(db, nil, nil, Config{
		MonitoringInterval: 15 * time.Minute,
		Concurrency:        1,
	}, nil)

	interval := p.getOrgMonitoringInterval(1)
	assert.Equal(t, 15*time.Minute, interval)
}

func TestGetOrgMonitoringInterval_OrgOverride(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&persistent.Setting{OrgID: 1, Key: persistent.SettingMonitoringInterval, Value: "30m"})

	p := New(db, nil, nil, Config{
		MonitoringInterval: 15 * time.Minute,
		Concurrency:        1,
	}, nil)

	interval := p.getOrgMonitoringInterval(1)
	assert.Equal(t, 30*time.Minute, interval)
}

func TestGetOrgDiscoveryInterval_DefaultFallback(t *testing.T) {
	db := setupTestDB(t)

	p := New(db, nil, nil, Config{
		DiscoveryInterval: 24 * time.Hour,
		Concurrency:       1,
	}, nil)

	interval := p.getOrgDiscoveryInterval(1)
	assert.Equal(t, 24*time.Hour, interval)
}

func TestGetOrgDiscoveryInterval_OrgOverride(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&persistent.Setting{OrgID: 1, Key: persistent.SettingDiscoveryInterval, Value: "12h"})

	p := New(db, nil, nil, Config{
		DiscoveryInterval: 24 * time.Hour,
		Concurrency:       1,
	}, nil)

	interval := p.getOrgDiscoveryInterval(1)
	assert.Equal(t, 12*time.Hour, interval)
}

func TestGetDiscoveryScanDepth_Default(t *testing.T) {
	db := setupTestDB(t)

	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	depth := p.getDiscoveryScanDepth(1)
	assert.Equal(t, 50, depth)
}

func TestGetDiscoveryScanDepth_OrgOverride(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&persistent.Setting{OrgID: 1, Key: persistent.SettingDiscoveryScanDepth, Value: "200"})

	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	depth := p.getDiscoveryScanDepth(1)
	assert.Equal(t, 200, depth)
}

func TestGetDiscoveryScanDepth_InvalidValue(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&persistent.Setting{OrgID: 1, Key: persistent.SettingDiscoveryScanDepth, Value: "invalid"})

	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	depth := p.getDiscoveryScanDepth(1)
	assert.Equal(t, 50, depth) // Falls back to default
}

// ---------------------------------------------------------------------------
// Org Due / Polling Timer Tests
// ---------------------------------------------------------------------------

func TestIsOrgDue_NeverPolled(t *testing.T) {
	p := New(nil, nil, nil, Config{Concurrency: 1}, nil)
	// Never polled — should be immediately due
	assert.True(t, p.isOrgDue(1, "monitor", 5*time.Minute))
}

func TestIsOrgDue_RecentlyPolled(t *testing.T) {
	p := New(nil, nil, nil, Config{Concurrency: 1}, nil)
	p.markOrgPolled(1, "monitor")
	// Just polled — should NOT be due yet
	assert.False(t, p.isOrgDue(1, "monitor", 5*time.Minute))
}

func TestIsOrgDue_SeparatePurposes(t *testing.T) {
	p := New(nil, nil, nil, Config{Concurrency: 1}, nil)
	p.markOrgPolled(1, "monitor")
	// Monitor is marked, but discover is never polled — should be due
	assert.True(t, p.isOrgDue(1, "discover", 5*time.Minute))
	assert.False(t, p.isOrgDue(1, "monitor", 5*time.Minute))
}

// ---------------------------------------------------------------------------
// TriggerDiscovery Tests
// ---------------------------------------------------------------------------

func TestTriggerDiscovery_MakesOrgDue(t *testing.T) {
	p := New(nil, nil, nil, Config{Concurrency: 1}, nil)
	// Mark org as recently discovered
	p.markOrgPolled(1, "discover")
	assert.False(t, p.isOrgDue(1, "discover", 24*time.Hour))

	// Trigger discovery resets the timer
	p.TriggerDiscovery(1)
	assert.True(t, p.isOrgDue(1, "discover", 24*time.Hour))
}

func TestTriggerDiscovery_DoesNotAffectMonitor(t *testing.T) {
	p := New(nil, nil, nil, Config{Concurrency: 1}, nil)
	p.markOrgPolled(1, "monitor")
	p.markOrgPolled(1, "discover")

	p.TriggerDiscovery(1)

	// Monitor should still be "not due"
	assert.False(t, p.isOrgDue(1, "monitor", 24*time.Hour))
	// Discover should be due after trigger
	assert.True(t, p.isOrgDue(1, "discover", 24*time.Hour))
}

// ---------------------------------------------------------------------------
// Registry Selection Tests
// ---------------------------------------------------------------------------

func TestRegistryForEcosystem(t *testing.T) {
	pyMock := &mockRegistry{name: "python"}
	npmMock := &mockRegistry{name: "npm"}
	p := New(nil, pyMock, npmMock, Config{Concurrency: 1}, nil)

	assert.Equal(t, pyMock, p.registryForEcosystem(persistent.EcosystemPython))
	assert.Equal(t, npmMock, p.registryForEcosystem(persistent.EcosystemNPM))
	assert.Nil(t, p.registryForEcosystem("unknown"))
}

// ---------------------------------------------------------------------------
// Monitor Org Packages Tests
// ---------------------------------------------------------------------------

func TestMonitorOrgPackages_ChecksActivePackages(t *testing.T) {
	db := setupTestDB(t)

	now := time.Now()
	pyMock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {Name: "requests", Version: "1.0.0", Versions: []entity.RegistryVersionInfo{
				{Version: "1.0.0", PublishedAt: now},
			}},
		},
	}
	npmMock := &mockRegistry{
		name: "npm",
		packages: map[string]*entity.RegistryPackageInfo{
			"express": {Name: "express", Version: "4.0.0", Versions: []entity.RegistryVersionInfo{
				{Version: "4.0.0", PublishedAt: now},
			}},
		},
	}

	p := New(db, pyMock, npmMock, Config{Concurrency: 5}, nil)

	// Active packages — both ecosystems in same org
	db.Create(&persistent.Package{OrgID: 1, Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered})
	db.Create(&persistent.Package{OrgID: 1, Name: "express", Ecosystem: "npm", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered})

	// Blocked package — should NOT be loaded
	blockedAt := time.Now()
	db.Create(&persistent.Package{OrgID: 1, Name: "blocked-pkg", Ecosystem: "python", Status: persistent.PackageStatusBlocked, Source: persistent.PackageSourceDiscovered, BlockedAt: &blockedAt})

	// Removed package — should NOT be loaded
	db.Create(&persistent.Package{OrgID: 1, Name: "removed-pkg", Ecosystem: "python", Status: persistent.PackageStatusRemoved, Source: persistent.PackageSourceDiscovered})

	checked := p.monitorOrgPackages(context.Background(), 1)
	// Only active packages are checked
	assert.Equal(t, 2, checked)
}

// ---------------------------------------------------------------------------
// Run Monitor Cycle Tests
// ---------------------------------------------------------------------------

func TestRunMonitorCycle_SkipsNonDueOrgs(t *testing.T) {
	db := setupTestDB(t)

	pyMock := &mockRegistry{
		name: "python",
		packages: map[string]*entity.RegistryPackageInfo{
			"requests": {Name: "requests", Version: "1.0.0", Versions: []entity.RegistryVersionInfo{
				{Version: "1.0.0", PublishedAt: time.Now()},
			}},
		},
	}

	p := New(db, pyMock, nil, Config{
		MonitoringInterval: 1 * time.Hour,
		Concurrency:        1,
	}, nil)

	// Create an active package so the org shows up
	db.Create(&persistent.Package{OrgID: 1, Name: "requests", Ecosystem: "python", Status: persistent.PackageStatusActive, Source: persistent.PackageSourceDiscovered})

	// First cycle — org is due (never polled)
	p.runMonitorCycle(context.Background())

	var count1 int64
	db.Model(&persistent.Release{}).Count(&count1)
	assert.Greater(t, count1, int64(0))

	// Second cycle immediately — org should NOT be due yet
	p.runMonitorCycle(context.Background())

	var count2 int64
	db.Model(&persistent.Release{}).Count(&count2)
	// No additional releases — org was skipped
	assert.Equal(t, count1, count2)
}

// ---------------------------------------------------------------------------
// Run Discovery Cycle Tests
// ---------------------------------------------------------------------------

func TestRunDiscoveryCycle_DiscoversForDueOrgs(t *testing.T) {
	db := setupTestDB(t)

	pyMock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "requests", Rank: 1},
			{Name: "flask", Rank: 2},
		},
	}
	npmMock := &mockRegistry{
		name:        "npm",
		topPackages: []entity.PackageRanking{
			{Name: "express", Rank: 1},
			{Name: "lodash", Rank: 2},
		},
	}

	p := New(db, pyMock, npmMock, Config{
		DiscoveryInterval: 1 * time.Hour,
		Concurrency:       1,
	}, nil)

	// Create a setting so org 1 shows up
	db.Create(&persistent.Setting{OrgID: 1, Key: persistent.SettingDiscoveryScanDepth, Value: "2"})

	p.runDiscoveryCycle(context.Background())

	var count int64
	db.Model(&persistent.Package{}).Count(&count)
	// 2 python + 2 npm = 4 packages discovered
	assert.Equal(t, int64(4), count)
}

func TestRunDiscoveryCycle_ZeroScanDepth_Skips(t *testing.T) {
	db := setupTestDB(t)

	pyMock := &mockRegistry{
		name:        "python",
		topPackages: []entity.PackageRanking{
			{Name: "requests", Rank: 1},
		},
	}

	p := New(db, pyMock, nil, Config{
		DiscoveryInterval: 1 * time.Hour,
		Concurrency:       1,
	}, nil)

	// Set scan depth to 0 — should skip discovery
	db.Create(&persistent.Setting{OrgID: 1, Key: persistent.SettingDiscoveryScanDepth, Value: "0"})

	p.runDiscoveryCycle(context.Background())

	var count int64
	db.Model(&persistent.Package{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

// ---------------------------------------------------------------------------
// Settings Cache Tests
// ---------------------------------------------------------------------------

func TestSettingsCache_Basic(t *testing.T) {
	cache := newSettingsCache(5 * time.Minute)

	// Miss
	_, ok := cache.get("key")
	assert.False(t, ok)

	// Set and hit
	cache.set("key", "value")
	v, ok := cache.get("key")
	assert.True(t, ok)
	assert.Equal(t, "value", v)
}

func TestSettingsCache_Invalidate(t *testing.T) {
	cache := newSettingsCache(5 * time.Minute)
	cache.set("key", "value")

	cache.invalidate()

	_, ok := cache.get("key")
	assert.False(t, ok)
}

func TestInvalidateSettingsCache(t *testing.T) {
	db := setupTestDB(t)
	p := New(db, nil, nil, Config{Concurrency: 1}, nil)

	// Populate the cache
	p.settings.set("1:monitoring_interval", "5m")
	v, ok := p.settings.get("1:monitoring_interval")
	assert.True(t, ok)
	assert.Equal(t, "5m", v)

	// Invalidate
	p.InvalidateSettingsCache()

	_, ok = p.settings.get("1:monitoring_interval")
	assert.False(t, ok)
}
