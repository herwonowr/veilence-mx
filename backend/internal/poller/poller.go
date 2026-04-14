package poller

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/registry"
)

// Config holds configuration for the poller.
type Config struct {
	MonitoringInterval time.Duration
	DiscoveryInterval  time.Duration
	Concurrency        int
}

// settingsCache holds cached settings values with a TTL.
type settingsCache struct {
	mu        sync.RWMutex
	values    map[string]string
	expiresAt time.Time
	ttl       time.Duration
}

// newSettingsCache creates a new settings cache with the given TTL.
func newSettingsCache(ttl time.Duration) *settingsCache {
	return &settingsCache{
		values: make(map[string]string),
		ttl:    ttl,
	}
}

// get retrieves a cached setting value. Returns the value and true if found
// and not expired, or empty string and false otherwise.
func (c *settingsCache) get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if time.Now().After(c.expiresAt) {
		return "", false
	}
	v, ok := c.values[key]
	return v, ok
}

// set stores a setting value in the cache, refreshing the TTL.
func (c *settingsCache) set(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
	c.expiresAt = time.Now().Add(c.ttl)
}

// invalidate clears the cache, forcing the next read to hit the database.
func (c *settingsCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = make(map[string]string)
	c.expiresAt = time.Time{}
}

// SettingsCacheTTL is the default TTL for cached poller settings.
// Settings are re-read from the database after this duration.
const SettingsCacheTTL = 1 * time.Minute

// BaseTickInterval is the base polling interval used by both loops.
// Each tick, the poller checks which orgs are due for monitoring/discovery
// based on their per-org interval settings.
const BaseTickInterval = 30 * time.Second

// MaxOrgConcurrency limits how many orgs can be polled concurrently
// within a single poll cycle. This prevents resource exhaustion when
// many orgs are due simultaneously.
const MaxOrgConcurrency = 5

// Poller polls package registries for new releases and discovers new packages.
type Poller struct {
	db       *gorm.DB
	python   registry.Registry
	npm      registry.Registry
	config   Config
	queue    *queue.Queue
	mu       sync.Mutex
	settings *settingsCache

	// lastPollAt tracks when each org was last polled/discovered.
	// Keys: "orgID:monitor" and "orgID:discover"
	// Protected by lastPollMu.
	lastPollMu sync.RWMutex
	lastPollAt map[string]time.Time
}

// New creates a new Poller instance.
func New(db *gorm.DB, python registry.Registry, npm registry.Registry, config Config, q *queue.Queue) *Poller {
	if config.Concurrency <= 0 {
		config.Concurrency = 5
	}
	return &Poller{
		db:         db,
		python:     python,
		npm:        npm,
		config:     config,
		queue:      q,
		settings:   newSettingsCache(SettingsCacheTTL),
		lastPollAt: make(map[string]time.Time),
	}
}

// InvalidateSettingsCache clears the cached settings, forcing the next poll
// cycle to re-read them from the database. Call this after settings are updated.
func (p *Poller) InvalidateSettingsCache() {
	p.settings.invalidate()
	slog.Info("poller settings cache invalidated")
}

// TriggerDiscovery resets the discovery timer for an org, making it due
// on the next tick. Called by the settings handler when discovery_scan_depth changes.
func (p *Poller) TriggerDiscovery(orgID uint) {
	key := fmt.Sprintf("%d:discover", orgID)
	p.lastPollMu.Lock()
	delete(p.lastPollAt, key)
	p.lastPollMu.Unlock()
	slog.Info("discovery triggered for org", "org_id", orgID)
}

// Start begins the monitoring and discovery loops.
func (p *Poller) Start(ctx context.Context) {
	slog.Info("starting poller",
		"monitoring_interval", p.config.MonitoringInterval,
		"discovery_interval", p.config.DiscoveryInterval,
	)
	go p.monitorLoop(ctx)
	go p.discoveryLoop(ctx)
}

// ---------------------------------------------------------------------------
// Monitoring Loop
// ---------------------------------------------------------------------------

// monitorLoop is a single goroutine that checks ALL active packages
// (regardless of ecosystem) for new releases.
func (p *Poller) monitorLoop(ctx context.Context) {
	// Initial run — all orgs are immediately due since lastPollAt is empty.
	p.runMonitorCycle(ctx)

	ticker := time.NewTicker(BaseTickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("monitor loop shutting down")
			return
		case <-ticker.C:
			p.runMonitorCycle(ctx)
		}
	}
}

// runMonitorCycle checks all orgs and monitors due ones.
func (p *Poller) runMonitorCycle(ctx context.Context) {
	// Fetch distinct org IDs that have active packages
	var orgIDs []uint
	if err := p.db.Model(&models.Package{}).
		Where("status = ?", models.PackageStatusActive).
		Distinct("org_id").
		Pluck("org_id", &orgIDs).Error; err != nil {
		slog.Error("failed to load org IDs for monitoring", "error", err)
		return
	}

	if len(orgIDs) == 0 {
		slog.Debug("no packages to monitor")
		return
	}

	// Filter to orgs that are due for monitoring.
	var dueOrgIDs []uint
	for _, orgID := range orgIDs {
		interval := p.getOrgMonitoringInterval(orgID)
		if p.isOrgDue(orgID, "monitor", interval) {
			dueOrgIDs = append(dueOrgIDs, orgID)
		}
	}

	if len(dueOrgIDs) == 0 {
		return
	}

	// Monitor due orgs concurrently, bounded by MaxOrgConcurrency.
	orgSem := make(chan struct{}, MaxOrgConcurrency)
	var wg sync.WaitGroup
	var totalChecked int64

	for _, orgID := range dueOrgIDs {
		wg.Add(1)
		go func(orgID uint) {
			defer wg.Done()
			orgSem <- struct{}{}
			defer func() { <-orgSem }()

			n := p.monitorOrgPackages(ctx, orgID)
			atomic.AddInt64(&totalChecked, int64(n))
			p.markOrgPolled(orgID, "monitor")
		}(orgID)
	}

	wg.Wait()
	if totalChecked > 0 {
		slog.Info("monitoring complete", "orgs_polled", len(dueOrgIDs), "packages_checked", totalChecked)
	}
}

// monitorOrgPackages loads all active packages for an org (both ecosystems)
// and checks each for new releases. Returns the number of packages checked.
func (p *Poller) monitorOrgPackages(ctx context.Context, orgID uint) int {
	var packages []models.Package
	if err := p.db.Where("org_id = ? AND status = ?", orgID, models.PackageStatusActive).
		Find(&packages).Error; err != nil {
		slog.Error("failed to load active packages for monitoring", "org_id", orgID, "error", err)
		return 0
	}

	if len(packages) == 0 {
		return 0
	}

	sem := make(chan struct{}, p.config.Concurrency)
	var wg sync.WaitGroup

	for _, pkg := range packages {
		wg.Add(1)
		go func(pkg models.Package) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			reg := p.registryForEcosystem(pkg.Ecosystem)
			if reg == nil {
				slog.Error("unknown ecosystem for package", "package", pkg.Name, "ecosystem", pkg.Ecosystem)
				return
			}
			if err := p.checkPackageForNewReleases(ctx, reg, pkg); err != nil {
				slog.Error("failed to check package", "package", pkg.Name, "ecosystem", pkg.Ecosystem, "org_id", orgID, "error", err)
			}
		}(pkg)
	}

	wg.Wait()
	return len(packages)
}

// registryForEcosystem returns the appropriate registry client for an ecosystem.
func (p *Poller) registryForEcosystem(ecosystem models.Ecosystem) registry.Registry {
	switch ecosystem {
	case models.EcosystemPython:
		return p.python
	case models.EcosystemNPM:
		return p.npm
	default:
		return nil
	}
}

// checkPackageForNewReleases fetches registry info and creates releases
// for ALL versions published after the last known release (no version depth cap).
func (p *Poller) checkPackageForNewReleases(ctx context.Context, reg registry.Registry, pkg models.Package) error {
	info, err := reg.GetPackage(ctx, pkg.Name)
	if err != nil {
		return fmt.Errorf("fetching package info: %w", err)
	}

	// Update package metadata
	p.db.Model(&pkg).Updates(map[string]any{
		"latest_version": info.Version,
		"description":    info.Description,
	})

	if len(info.Versions) == 0 {
		return nil
	}

	// Find the most recent release we already know about
	var latestKnownRelease models.Release
	p.db.Where("package_id = ?", pkg.ID).Order("published_at DESC").Limit(1).Find(&latestKnownRelease)

	var newVersions []registry.VersionInfo

	if latestKnownRelease.ID > 0 {
		// Find all versions published AFTER our latest known release
		for _, v := range info.Versions {
			if v.PublishedAt.After(latestKnownRelease.PublishedAt) && v.Version != latestKnownRelease.Version {
				newVersions = append(newVersions, v)
			}
		}
	} else {
		// First time: take latest + one baseline (same as current baseline logic)
		count := min(2, len(info.Versions))
		newVersions = info.Versions[:count]
	}

	if len(newVersions) == 0 {
		return nil
	}

	// Sort oldest-first so older versions get lower IDs,
	// allowing the differ to find them as "previous release".
	sort.Slice(newVersions, func(i, j int) bool {
		return newVersions[i].PublishedAt.Before(newVersions[j].PublishedAt)
	})

	for _, v := range newVersions {
		// Double-check: don't create if already exists (idempotency)
		var existingRelease models.Release
		result := p.db.Where("package_id = ? AND version = ?", pkg.ID, v.Version).Limit(1).Find(&existingRelease)
		if result.Error != nil {
			slog.Error("failed to check release", "package", pkg.Name, "version", v.Version, "error", result.Error)
			continue
		}
		if result.RowsAffected > 0 {
			continue
		}

		release := models.Release{
			PackageID:   pkg.ID,
			Version:     v.Version,
			PublishedAt: v.PublishedAt,
			TarballURL:  v.TarballURL,
			SHA256:      v.SHA256,
			Status:      models.ReleaseStatusPending,
		}

		if err := p.db.Create(&release).Error; err != nil {
			slog.Error("failed to create release", "package", pkg.Name, "version", v.Version, "error", err)
			continue
		}

		slog.Info("new release detected", "package", pkg.Name, "version", v.Version, "ecosystem", reg.Name())

		// Enqueue diff job via Redis queue
		if p.queue != nil {
			jobID, err := p.queue.Enqueue(ctx, queue.JobTypeDiff, release.ID)
			if err != nil {
				slog.Error("failed to enqueue diff job", "release_id", release.ID, "error", err)
				continue
			}
			slog.Info("diff job enqueued", "job_id", jobID, "release_id", release.ID)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Discovery Loop
// ---------------------------------------------------------------------------

// discoveryLoop is a single goroutine that periodically scans registry
// popularity rankings and adds new packages to monitoring.
func (p *Poller) discoveryLoop(ctx context.Context) {
	// Initial run — all orgs are immediately due since lastPollAt is empty.
	p.runDiscoveryCycle(ctx)

	ticker := time.NewTicker(BaseTickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("discovery loop shutting down")
			return
		case <-ticker.C:
			p.runDiscoveryCycle(ctx)
		}
	}
}

// runDiscoveryCycle checks all orgs and runs discovery for due ones.
func (p *Poller) runDiscoveryCycle(ctx context.Context) {
	// Get all org IDs that have settings (i.e., are set up)
	var orgIDs []uint
	if err := p.db.Model(&models.Setting{}).
		Where("org_id > 0").
		Distinct("org_id").
		Pluck("org_id", &orgIDs).Error; err != nil {
		slog.Error("failed to load org IDs for discovery", "error", err)
		return
	}

	// Also include orgs that have packages but might not have settings yet
	var pkgOrgIDs []uint
	if err := p.db.Model(&models.Package{}).
		Distinct("org_id").
		Pluck("org_id", &pkgOrgIDs).Error; err != nil {
		slog.Error("failed to load package org IDs for discovery", "error", err)
		return
	}

	// Merge unique org IDs
	seen := make(map[uint]bool, len(orgIDs)+len(pkgOrgIDs))
	for _, id := range orgIDs {
		seen[id] = true
	}
	for _, id := range pkgOrgIDs {
		seen[id] = true
	}

	var allOrgIDs []uint
	for id := range seen {
		allOrgIDs = append(allOrgIDs, id)
	}

	if len(allOrgIDs) == 0 {
		return
	}

	for _, orgID := range allOrgIDs {
		interval := p.getOrgDiscoveryInterval(orgID)
		if !p.isOrgDue(orgID, "discover", interval) {
			continue
		}

		scanDepth := p.getDiscoveryScanDepth(orgID)
		if scanDepth <= 0 {
			p.markOrgPolled(orgID, "discover")
			continue
		}

		slog.Info("running discovery for org", "org_id", orgID, "scan_depth", scanDepth)

		// Scan both ecosystems to the full depth
		if p.python != nil {
			p.discoverPackages(ctx, p.python, scanDepth, orgID)
		}
		if p.npm != nil {
			p.discoverPackages(ctx, p.npm, scanDepth, orgID)
		}

		p.markOrgPolled(orgID, "discover")
	}
}

// discoverPackages fetches top packages from a registry and upserts them.
// Discovery is additive only — it never removes packages.
func (p *Poller) discoverPackages(ctx context.Context, reg registry.Registry, scanDepth int, orgID uint) {
	names, err := reg.GetTopPackages(ctx, scanDepth)
	if err != nil {
		slog.Error("failed to fetch top packages for discovery", "ecosystem", reg.Name(), "org_id", orgID, "error", err)
		return
	}

	var added int
	for i, name := range names {
		rank := uint(i + 1)
		var existing models.Package
		result := p.db.Where("org_id = ? AND name = ? AND ecosystem = ?", orgID, name, reg.Name()).Limit(1).Find(&existing)

		if result.RowsAffected == 0 {
			// New package — create it
			pkg := models.Package{
				OrgID:     orgID,
				Name:      name,
				Ecosystem: models.Ecosystem(reg.Name()),
				Source:    models.PackageSourceDiscovered,
				Status:   models.PackageStatusActive,
				Rank:     &rank,
			}
			if err := p.db.Create(&pkg).Error; err != nil {
				slog.Error("failed to create discovered package", "package", name, "error", err)
				continue
			}
			added++
		} else {
			// Existing package — handle based on status
			switch existing.Status {
			case models.PackageStatusActive:
				// Update rank only
				p.db.Model(&existing).Update("rank", &rank)
			case models.PackageStatusBlocked:
				// Skip entirely — do not update rank
				continue
			case models.PackageStatusRemoved:
				// Re-add: set status back to active with updated rank
				p.db.Model(&existing).Updates(map[string]any{
					"status": models.PackageStatusActive,
					"rank":   &rank,
				})
				added++
			}
		}
	}

	slog.Info("discovery complete", "ecosystem", reg.Name(), "org_id", orgID, "scanned", len(names), "new_packages", added)
}

// SyncTopPackages is kept for backward compatibility with the API handler.
// It delegates to discoverPackages.
func (p *Poller) SyncTopPackages(ctx context.Context, reg registry.Registry, limit int, orgID uint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	slog.Info("syncing top packages (legacy)", "ecosystem", reg.Name(), "limit", limit, "org_id", orgID)
	p.discoverPackages(ctx, reg, limit, orgID)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// pollRegistryKey returns the lastPollAt map key for an (orgID, purpose) pair.
func pollRegistryKey(orgID uint, purpose string) string {
	return fmt.Sprintf("%d:%s", orgID, purpose)
}

// getOrgMonitoringInterval returns the monitoring interval for a specific org.
// Falls back to the global config default if no per-org setting exists.
func (p *Poller) getOrgMonitoringInterval(orgID uint) time.Duration {
	intervalStr := p.getSetting(models.SettingMonitoringInterval, p.config.MonitoringInterval.String(), orgID)
	if d, err := time.ParseDuration(intervalStr); err == nil && d > 0 {
		return d
	}
	return p.config.MonitoringInterval
}

// getOrgDiscoveryInterval returns the discovery interval for a specific org.
func (p *Poller) getOrgDiscoveryInterval(orgID uint) time.Duration {
	intervalStr := p.getSetting(models.SettingDiscoveryInterval, p.config.DiscoveryInterval.String(), orgID)
	if d, err := time.ParseDuration(intervalStr); err == nil && d > 0 {
		return d
	}
	return p.config.DiscoveryInterval
}

// getDiscoveryScanDepth returns the discovery scan depth for a specific org.
// A value of 0 means discovery is disabled for that org.
func (p *Poller) getDiscoveryScanDepth(orgID uint) int {
	depthStr := p.getSetting(models.SettingDiscoveryScanDepth, "50", orgID)
	if v, err := strconv.Atoi(depthStr); err == nil && v >= 0 {
		return v
	}
	return 50
}

// isOrgDue returns true if the given org is due for the given purpose
// (monitor or discover) based on its interval and last poll time.
func (p *Poller) isOrgDue(orgID uint, purpose string, interval time.Duration) bool {
	key := pollRegistryKey(orgID, purpose)

	p.lastPollMu.RLock()
	last, ok := p.lastPollAt[key]
	p.lastPollMu.RUnlock()

	if !ok {
		return true // Never polled — immediately due.
	}
	return time.Since(last) >= interval
}

// markOrgPolled records the current time as the last poll time for the org+purpose.
func (p *Poller) markOrgPolled(orgID uint, purpose string) {
	key := pollRegistryKey(orgID, purpose)
	p.lastPollMu.Lock()
	p.lastPollAt[key] = time.Now()
	p.lastPollMu.Unlock()
}

// getSetting retrieves a setting value, using the cache when available.
// Falls back to a database lookup on cache miss and stores the result.
// The orgID parameter scopes settings to the requesting organization.
func (p *Poller) getSetting(key, defaultValue string, orgID uint) string {
	cacheKey := fmt.Sprintf("%d:%s", orgID, key)
	if v, ok := p.settings.get(cacheKey); ok {
		return v
	}

	// Cache miss — read from database, scoped to org
	var setting models.Setting
	if tx := p.db.Where("key = ? AND org_id = ?", key, orgID).Limit(1).Find(&setting); tx.RowsAffected > 0 {
		p.settings.set(cacheKey, setting.Value)
		return setting.Value
	}

	// Not found in DB, cache the default
	p.settings.set(cacheKey, defaultValue)
	return defaultValue
}
