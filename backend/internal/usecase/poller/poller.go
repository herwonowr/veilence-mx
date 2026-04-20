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

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
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
// Each tick, the poller checks which workspaces are due for monitoring/discovery
// based on their per-workspace interval settings.
const BaseTickInterval = 30 * time.Second

// MaxWorkspaceConcurrency limits how many workspaces can be polled concurrently
// within a single poll cycle. This prevents resource exhaustion when
// many workspaces are due simultaneously.
const MaxWorkspaceConcurrency = 5

// JobTypeDiff is the queue job type for diff processing.
const JobTypeDiff = "diff"

// Poller polls package registries for new releases and discovers new packages.
type Poller struct {
	repo     PollerRepository
	python   usecase.Registry
	npm      usecase.Registry
	config   Config
	queue    usecase.QueueEnqueuer
	notifier usecase.NotificationDispatcher
	mu       sync.Mutex
	settings *settingsCache

	// lastPollAt tracks when each workspace was last polled/discovered.
	// Keys: "workspaceID:monitor" and "workspaceID:discover"
	// Protected by lastPollMu.
	lastPollMu sync.RWMutex
	lastPollAt map[string]time.Time
}

// New creates a new Poller instance.
func New(repo PollerRepository, python usecase.Registry, npm usecase.Registry, config Config, q usecase.QueueEnqueuer, notifier usecase.NotificationDispatcher) *Poller {
	if config.Concurrency <= 0 {
		config.Concurrency = 5
	}
	return &Poller{
		repo:       repo,
		python:     python,
		npm:        npm,
		config:     config,
		queue:      q,
		notifier:   notifier,
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

// TriggerDiscovery resets the discovery timer for a workspace, making it due
// on the next tick. Called by the settings handler when discovery_scan_depth changes.
func (p *Poller) TriggerDiscovery(workspaceID string) {
	key := workspaceID + ":discover"
	p.lastPollMu.Lock()
	delete(p.lastPollAt, key)
	p.lastPollMu.Unlock()
	slog.Info("discovery triggered for workspace", "workspace_id", workspaceID)
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
	// Initial run - all workspaces are immediately due since lastPollAt is empty.
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

// runMonitorCycle checks all workspaces and monitors due ones.
func (p *Poller) runMonitorCycle(ctx context.Context) {
	// Fetch distinct workspace IDs that have active packages
	wsIDs, err := p.repo.DistinctActiveWorkspaceIDs(ctx)
	if err != nil {
		slog.Error("failed to load workspace IDs for monitoring", "error", err)
		return
	}

	if len(wsIDs) == 0 {
		slog.Debug("no packages to monitor")
		return
	}

	// Filter to workspaces that are due for monitoring.
	var dueWorkspaceIDs []string
	for _, workspaceID := range wsIDs {
		interval := p.getWorkspaceMonitoringInterval(workspaceID)
		if p.isWorkspaceDue(workspaceID, "monitor", interval) {
			dueWorkspaceIDs = append(dueWorkspaceIDs, workspaceID)
		}
	}

	if len(dueWorkspaceIDs) == 0 {
		return
	}

	// Monitor due workspaces concurrently, bounded by MaxWorkspaceConcurrency.
	wsSem := make(chan struct{}, MaxWorkspaceConcurrency)
	var wg sync.WaitGroup
	var totalChecked int64

	for _, workspaceID := range dueWorkspaceIDs {
		wg.Add(1)
		go func(workspaceID string) {
			defer wg.Done()
			wsSem <- struct{}{}
			defer func() { <-wsSem }()

			n := p.monitorWorkspacePackages(ctx, workspaceID)
			atomic.AddInt64(&totalChecked, int64(n))
			p.markWorkspacePolled(workspaceID, "monitor")
		}(workspaceID)
	}

	wg.Wait()
	if totalChecked > 0 {
		slog.Info("monitoring complete", "workspaces_polled", len(dueWorkspaceIDs), "packages_checked", totalChecked)
	}
}

// monitorWorkspacePackages loads all active packages for a workspace (both ecosystems)
// and checks each for new releases. Returns the number of packages checked.
func (p *Poller) monitorWorkspacePackages(ctx context.Context, workspaceID string) int {
	packages, err := p.repo.FindActivePackagesByWorkspace(ctx, workspaceID)
	if err != nil {
		slog.Error("failed to load active packages for monitoring", "workspace_id", workspaceID, "error", err)
		return 0
	}

	if len(packages) == 0 {
		return 0
	}

	sem := make(chan struct{}, p.config.Concurrency)
	var wg sync.WaitGroup

	for _, pkg := range packages {
		wg.Add(1)
		go func(pkg entity.Package) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			reg := p.registryForEcosystem(pkg.Ecosystem)
			if reg == nil {
				slog.Error("unknown ecosystem for package", "package", pkg.Name, "ecosystem", pkg.Ecosystem)
				return
			}
			if err := p.checkPackageForNewReleases(ctx, reg, pkg); err != nil {
				slog.Error("failed to check package", "package", pkg.Name, "ecosystem", pkg.Ecosystem, "workspace_id", workspaceID, "error", err)
			}
		}(pkg)
	}

	wg.Wait()
	return len(packages)
}

// registryForEcosystem returns the appropriate registry client for an ecosystem.
func (p *Poller) registryForEcosystem(ecosystem entity.Ecosystem) usecase.Registry {
	switch ecosystem {
	case entity.EcosystemPython:
		return p.python
	case entity.EcosystemNPM:
		return p.npm
	default:
		return nil
	}
}

// checkPackageForNewReleases fetches registry info and creates releases
// for ALL versions published after the last known release (no version depth cap).
func (p *Poller) checkPackageForNewReleases(ctx context.Context, reg usecase.Registry, pkg entity.Package) error {
	info, err := reg.GetPackage(ctx, pkg.Name)
	if err != nil {
		return fmt.Errorf("fetching package info: %w", err)
	}

	// Update package metadata
	p.repo.UpdatePackageMetadata(ctx, pkg.ID, info.Version, info.Description)

	if len(info.Versions) == 0 {
		return nil
	}

	// Find the most recent release we already know about
	latestKnownRelease, _ := p.repo.FindLatestRelease(ctx, pkg.ID)

	var newVersions []entity.RegistryVersionInfo

	if latestKnownRelease != nil && latestKnownRelease.ID != "" {
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
		existing, err := p.repo.FindReleaseByPackageAndVersion(ctx, pkg.ID, v.Version)
		if err != nil {
			slog.Error("failed to check release", "package", pkg.Name, "version", v.Version, "error", err)
			continue
		}
		if existing != nil {
			continue
		}

		release := &entity.Release{
			PackageID:   pkg.ID,
			Version:     v.Version,
			PublishedAt: v.PublishedAt,
			TarballURL:  v.TarballURL,
			SHA256:      v.SHA256,
			Status:      entity.ReleaseStatusPending,
		}

		if err := p.repo.CreateRelease(ctx, release); err != nil {
			slog.Error("failed to create release", "package", pkg.Name, "version", v.Version, "error", err)
			continue
		}

		slog.Info("new release detected", "package", pkg.Name, "version", v.Version, "ecosystem", reg.Name())

		// Enqueue diff job via Redis queue
		if p.queue != nil {
			jobID, err := p.queue.Enqueue(ctx, JobTypeDiff, pkg.WorkspaceID, release.ID)
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
	// Initial run - all workspaces are immediately due since lastPollAt is empty.
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

// runDiscoveryCycle checks all workspaces and runs discovery for due ones.
func (p *Poller) runDiscoveryCycle(ctx context.Context) {
	allWorkspaceIDs, err := p.repo.DistinctWorkspaceIDs(ctx)
	if err != nil {
		slog.Error("failed to load workspace IDs for discovery", "error", err)
		return
	}

	if len(allWorkspaceIDs) == 0 {
		return
	}

	for _, workspaceID := range allWorkspaceIDs {
		interval := p.getWorkspaceDiscoveryInterval(workspaceID)
		if !p.isWorkspaceDue(workspaceID, "discover", interval) {
			continue
		}

		scanDepth := p.getDiscoveryScanDepth(workspaceID)
		if scanDepth <= 0 {
			p.markWorkspacePolled(workspaceID, "discover")
			continue
		}

		autoApprove := p.getDiscoveryAutoApprove(workspaceID)

		slog.Info("running discovery for workspace", "workspace_id", workspaceID, "scan_depth", scanDepth, "auto_approve", autoApprove)

		// Scan both ecosystems to the full depth
		if p.python != nil {
			p.discoverPackages(ctx, p.python, scanDepth, workspaceID, autoApprove)
		}
		if p.npm != nil {
			p.discoverPackages(ctx, p.npm, scanDepth, workspaceID, autoApprove)
		}

		// Auto-remove stale packages if configured
		staleMonths := p.getStaleAutoRemoveMonths(workspaceID)
		if staleMonths > 0 {
			p.removeStalePackages(ctx, workspaceID, staleMonths)
		}

		p.markWorkspacePolled(workspaceID, "discover")
	}
}

// discoverPackages fetches top packages from a registry and upserts them.
// Discovery is additive only - it never removes packages.
// When autoApprove is true, new packages are created with status 'active' (auto-approved).
// When autoApprove is false, new packages are created with status 'suggested' (pending admin approval).
// Existing active packages get their download metrics refreshed.
func (p *Poller) discoverPackages(ctx context.Context, reg usecase.Registry, scanDepth int, workspaceID string, autoApprove bool) {
	rankings, err := reg.GetTopPackages(ctx, scanDepth)
	if err != nil {
		slog.Error("failed to fetch top packages for discovery", "ecosystem", reg.Name(), "workspace_id", workspaceID, "error", err)
		return
	}

	p.upsertDiscoveredPackages(ctx, workspaceID, rankings, entity.Ecosystem(reg.Name()), autoApprove)

	slog.Info("discovery complete", "ecosystem", reg.Name(), "workspace_id", workspaceID, "scanned", len(rankings))
}

// upsertDiscoveredPackages processes a set of rankings for a given workspace and ecosystem.
// For each ranking:
//   - New package → create as 'active' (if autoApprove) or 'suggested'
//   - Existing 'active' → update rank + download metrics
//   - Existing 'suggested' → update rank + download metrics
//   - Existing 'blocked' → skip entirely
//   - Existing 'removed' → re-suggest for admin review (or auto-approve if enabled)
func (p *Poller) upsertDiscoveredPackages(ctx context.Context, workspaceID string, rankings []entity.PackageRanking, ecosystem entity.Ecosystem, autoApprove bool) {
	now := time.Now()
	var suggested int
	var downloadUpdates []entity.PackageDownloadUpdate

	newStatus := entity.PackageStatusSuggested
	if autoApprove {
		newStatus = entity.PackageStatusActive
	}

	for _, ranking := range rankings {
		rank := ranking.Rank
		existing, _ := p.repo.FindPackageByWorkspaceAndName(ctx, workspaceID, ranking.Name, ecosystem)

		if existing == nil {
			// New package - create with appropriate status
			pkg := &entity.Package{
				WorkspaceID:            workspaceID,
				Name:                   ranking.Name,
				Ecosystem:              ecosystem,
				Source:                 entity.PackageSourceDiscovered,
				Status:                 newStatus,
				Rank:                   &rank,
				DownloadCount:          ranking.DownloadCount,
				DownloadCountUpdatedAt: &now,
			}
			if err := p.repo.CreatePackage(ctx, pkg); err != nil {
				slog.Error("failed to create discovered package", "package", ranking.Name, "error", err)
				continue
			}
			suggested++
		} else {
			// Existing package - handle based on status
			switch existing.Status {
			case entity.PackageStatusActive:
				// Update rank; collect download data for batch update
				p.repo.UpdatePackageRank(ctx, existing.ID, rank)
				downloadUpdates = append(downloadUpdates, entity.PackageDownloadUpdate{
					PackageID:     existing.ID,
					DownloadCount: ranking.DownloadCount,
				})
			case entity.PackageStatusSuggested:
				// Already pending review - update rank and download data
				p.repo.UpdatePackageDiscoveryMetrics(ctx, existing.ID, map[string]interface{}{
					"rank":           &rank,
					"download_count": ranking.DownloadCount,

					"download_count_updated_at": now,
				})
			case entity.PackageStatusBlocked:
				// Skip entirely - do not update rank or download data
				continue
			case entity.PackageStatusRemoved:
				// Re-suggest for admin review (or auto-approve if enabled)
				p.repo.UpdatePackageDiscoveryMetrics(ctx, existing.ID, map[string]interface{}{
					"status":         string(newStatus),
					"rank":           &rank,
					"download_count": ranking.DownloadCount,

					"download_count_updated_at": now,
				})
				suggested++
			}
		}
	}

	// Batch update download counts for active packages
	if len(downloadUpdates) > 0 {
		if err := p.repo.UpdateDownloadCounts(ctx, workspaceID, downloadUpdates); err != nil {
			slog.Error("failed to batch update download counts", "workspace_id", workspaceID, "ecosystem", ecosystem, "error", err)
		}
	}

	if suggested > 0 {
		label := "suggested"
		if autoApprove {
			label = "auto-approved"
		}
		slog.Info("discovery added packages", "ecosystem", string(ecosystem), "workspace_id", workspaceID, label, suggested)
		if p.notifier != nil {
			p.notifier.DispatchEvent(ctx, workspaceID, entity.NotificationEvent{
				Severity:      "medium",
				EventType:     entity.NotifEventDiscoveryAdded,
				Title:         fmt.Sprintf("Discovery: %d new %s packages found", suggested, string(ecosystem)),
				Message:       fmt.Sprintf("Discovery scan found %d new %s packages (%s). Review them in the Packages page.", suggested, string(ecosystem), label),
				ReferenceID:   "",
				ReferenceType: "package",
			})
		}
	}
}

// SyncTopPackages is kept for backward compatibility with the API handler.
// It delegates to discoverPackages with auto-approve disabled (manual trigger = always suggest).
func (p *Poller) SyncTopPackages(ctx context.Context, reg usecase.Registry, limit int, workspaceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	slog.Info("syncing top packages (legacy)", "ecosystem", reg.Name(), "limit", limit, "workspace_id", workspaceID)
	p.discoverPackages(ctx, reg, limit, workspaceID, false)
	return nil
}

// ---------------------------------------------------------------------------
// Stale Package Removal
// ---------------------------------------------------------------------------

// removeStalePackages removes active packages that have had no updates for
// the given number of months. Called at the end of the discovery cycle.
func (p *Poller) removeStalePackages(ctx context.Context, workspaceID string, months int) {
	if months <= 0 {
		return
	}

	staleBefore := time.Now().AddDate(0, -months, 0)
	removed, err := p.repo.RemoveStalePackages(ctx, workspaceID, staleBefore)
	if err != nil {
		slog.Error("failed to remove stale packages", "workspace_id", workspaceID, "months", months, "error", err)
		return
	}

	if removed > 0 {
		slog.Info("auto-removed stale packages", "workspace_id", workspaceID, "months", months, "removed", removed)
		if p.notifier != nil {
			p.notifier.DispatchEvent(ctx, workspaceID, entity.NotificationEvent{
				Severity:      "medium",
				EventType:     entity.NotifEventStaleRemoved,
				Title:         fmt.Sprintf("%d stale packages auto-removed", removed),
				Message:       fmt.Sprintf("%d packages with no updates in %d months were automatically removed from monitoring.", removed, months),
				ReferenceID:   "",
				ReferenceType: "",
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// pollRegistryKey returns the lastPollAt map key for an (workspaceID, purpose) pair.
func pollRegistryKey(workspaceID string, purpose string) string {
	return workspaceID + ":" + purpose
}

// getWorkspaceMonitoringInterval returns the monitoring interval for a specific workspace.
// Falls back to the global config default if no per-workspace setting exists.
func (p *Poller) getWorkspaceMonitoringInterval(workspaceID string) time.Duration {
	intervalStr := p.getSetting(entity.SettingMonitoringInterval, p.config.MonitoringInterval.String(), workspaceID)
	if d, err := time.ParseDuration(intervalStr); err == nil && d > 0 {
		return d
	}
	return p.config.MonitoringInterval
}

// getWorkspaceDiscoveryInterval returns the discovery interval for a specific workspace.
func (p *Poller) getWorkspaceDiscoveryInterval(workspaceID string) time.Duration {
	intervalStr := p.getSetting(entity.SettingDiscoveryInterval, p.config.DiscoveryInterval.String(), workspaceID)
	if d, err := time.ParseDuration(intervalStr); err == nil && d > 0 {
		return d
	}
	return p.config.DiscoveryInterval
}

// getDiscoveryScanDepth returns the discovery scan depth for a specific workspace.
// A value of 0 means discovery is disabled for that workspace.
func (p *Poller) getDiscoveryScanDepth(workspaceID string) int {
	depthStr := p.getSetting(entity.SettingDiscoveryScanDepth, "50", workspaceID)
	if v, err := strconv.Atoi(depthStr); err == nil && v >= 0 {
		return v
	}
	return 50
}

// getDiscoveryAutoApprove returns whether newly discovered packages should be
// automatically approved (status=active) instead of suggested (pending review).
func (p *Poller) getDiscoveryAutoApprove(workspaceID string) bool {
	return p.getSetting(entity.SettingDiscoveryAutoApprove, "false", workspaceID) == "true"
}

// getStaleAutoRemoveMonths returns the number of months after which active
// packages with no updates are automatically removed. 0 means disabled.
func (p *Poller) getStaleAutoRemoveMonths(workspaceID string) int {
	monthsStr := p.getSetting(entity.SettingStaleAutoRemoveMonths, "0", workspaceID)
	if v, err := strconv.Atoi(monthsStr); err == nil && v >= 0 {
		return v
	}
	return 0
}

// isWorkspaceDue returns true if the given workspace is due for the given purpose
// (monitor or discover) based on its interval and last poll time.
func (p *Poller) isWorkspaceDue(workspaceID string, purpose string, interval time.Duration) bool {
	key := pollRegistryKey(workspaceID, purpose)

	p.lastPollMu.RLock()
	last, ok := p.lastPollAt[key]
	p.lastPollMu.RUnlock()

	if !ok {
		return true // Never polled - immediately due.
	}
	return time.Since(last) >= interval
}

// markWorkspacePolled records the current time as the last poll time for the workspace+purpose.
func (p *Poller) markWorkspacePolled(workspaceID string, purpose string) {
	key := pollRegistryKey(workspaceID, purpose)
	p.lastPollMu.Lock()
	p.lastPollAt[key] = time.Now()
	p.lastPollMu.Unlock()
}

// getSetting retrieves a setting value, using the cache when available.
// Falls back to a database lookup on cache miss and stores the result.
// The workspaceID parameter scopes settings to the requesting workspace.
func (p *Poller) getSetting(key, defaultValue string, workspaceID string) string {
	cacheKey := workspaceID + ":" + key
	if v, ok := p.settings.get(cacheKey); ok {
		return v
	}

	// Cache miss - read from database, scoped to workspace
	value, err := p.repo.GetSetting(ctx_bg(), workspaceID, key)
	if err == nil && value != "" {
		p.settings.set(cacheKey, value)
		return value
	}

	// Not found in DB, cache the default
	p.settings.set(cacheKey, defaultValue)
	return defaultValue
}

// ctx_bg returns a background context for settings lookups that don't
// have a context readily available. This is safe because settings reads
// are fast, non-blocking operations.
func ctx_bg() context.Context {
	return context.Background()
}
