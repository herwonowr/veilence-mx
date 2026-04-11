package poller

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/models"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/registry"
)

// Config holds configuration for the poller.
type Config struct {
	PyPIInterval        time.Duration
	NPMInterval         time.Duration
	Concurrency         int
	TopNRefreshInterval time.Duration
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

// Poller polls package registries for new releases.
type Poller struct {
	db       *gorm.DB
	pypi     registry.Registry
	npm      registry.Registry
	config   Config
	queue    *queue.Queue
	mu       sync.Mutex
	settings *settingsCache
}

// New creates a new Poller instance.
func New(db *gorm.DB, pypi registry.Registry, npm registry.Registry, config Config, q *queue.Queue) *Poller {
	if config.Concurrency <= 0 {
		config.Concurrency = 5
	}
	return &Poller{
		db:       db,
		pypi:     pypi,
		npm:      npm,
		config:   config,
		queue:    q,
		settings: newSettingsCache(SettingsCacheTTL),
	}
}

// InvalidateSettingsCache clears the cached settings, forcing the next poll
// cycle to re-read them from the database. Call this after settings are updated.
func (p *Poller) InvalidateSettingsCache() {
	p.settings.invalidate()
	slog.Info("poller settings cache invalidated")
}

// Start begins the polling loops for both registries.
func (p *Poller) Start(ctx context.Context) {
	slog.Info("starting poller", "pypi_interval", p.config.PyPIInterval, "npm_interval", p.config.NPMInterval)
	go p.pollLoop(ctx, p.pypi, p.config.PyPIInterval)
	go p.pollLoop(ctx, p.npm, p.config.NPMInterval)
}

func (p *Poller) pollLoop(ctx context.Context, reg registry.Registry, interval time.Duration) {
	p.pollRegistry(ctx, reg)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("poller shutting down", "registry", reg.Name())
			return
		case <-ticker.C:
			p.pollRegistry(ctx, reg)
		}
	}
}

func (p *Poller) pollRegistry(ctx context.Context, reg registry.Registry) {
	slog.Info("polling registry", "registry", reg.Name())

	// Fetch distinct org IDs that have packages in this registry
	var orgIDs []uint
	if err := p.db.Model(&models.Package{}).
		Where("registry = ?", reg.Name()).
		Distinct("org_id").
		Pluck("org_id", &orgIDs).Error; err != nil {
		slog.Error("failed to load org IDs for registry", "registry", reg.Name(), "error", err)
		return
	}

	if len(orgIDs) == 0 {
		slog.Info("no packages to poll", "registry", reg.Name())
		return
	}

	totalChecked := 0
	for _, orgID := range orgIDs {
		var packages []models.Package
		if err := p.db.Where("registry = ? AND org_id = ?", reg.Name(), orgID).Find(&packages).Error; err != nil {
			slog.Error("failed to load packages", "registry", reg.Name(), "org_id", orgID, "error", err)
			continue
		}

		if len(packages) == 0 {
			continue
		}

		sem := make(chan struct{}, p.config.Concurrency)
		var wg sync.WaitGroup

		for _, pkg := range packages {
			wg.Add(1)
			go func(pkg models.Package) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				if err := p.checkPackage(ctx, reg, pkg); err != nil {
					slog.Error("failed to check package", "package", pkg.Name, "registry", reg.Name(), "org_id", orgID, "error", err)
				}
			}(pkg)
		}

		wg.Wait()
		totalChecked += len(packages)
	}

	slog.Info("polling complete", "registry", reg.Name(), "packages_checked", totalChecked)
}

func (p *Poller) checkPackage(ctx context.Context, reg registry.Registry, pkg models.Package) error {
	info, err := reg.GetPackage(ctx, pkg.Name)
	if err != nil {
		return fmt.Errorf("fetching package info: %w", err)
	}

	// Update only this specific package record (scoped by primary key)
	p.db.Model(&pkg).Updates(map[string]any{
		"latest_version": info.Version,
		"description":    info.Description,
	})

	versions := p.applyVersionDepth(info.Versions, pkg.OrgID)

	// If no releases exist yet for this package and there are older versions
	// available, include one additional older version as a diff baseline.
	// This ensures the latest version can be compared even in "latest only" mode.
	var existingCount int64
	p.db.Model(&models.Release{}).Where("package_id = ?", pkg.ID).Count(&existingCount)
	if existingCount == 0 && len(versions) < len(info.Versions) {
		versions = append(versions, info.Versions[len(versions)])
	}

	// Process oldest-first so older versions get lower IDs,
	// allowing the differ to find them as "previous release".
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
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

		slog.Info("new release detected", "package", pkg.Name, "version", v.Version, "registry", reg.Name())

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

// applyVersionDepth limits versions based on the version_depth_mode setting.
// Versions must be sorted newest-first (both registry clients do this).
// "latest" (default) = only the newest version, "custom" = up to version_depth_count (1-5).
// Settings are read from an in-memory cache to avoid repeated DB lookups during each poll cycle.
// The orgID parameter scopes settings to the requesting organization.
func (p *Poller) applyVersionDepth(versions []registry.VersionInfo, orgID uint) []registry.VersionInfo {
	if len(versions) == 0 {
		return versions
	}

	mode := p.getSetting(models.SettingVersionDepthMode, "latest", orgID)

	switch mode {
	case "latest":
		return versions[:1]
	case "custom":
		count := 5 // default custom depth
		countStr := p.getSetting(models.SettingVersionDepthCount, "5", orgID)
		if v, err := strconv.Atoi(countStr); err == nil && v >= 1 && v <= 5 {
			count = v
		}
		if count > len(versions) {
			count = len(versions)
		}
		return versions[:count]
	default:
		return versions
	}
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

// SyncTopPackages fetches and upserts the top-N packages for a registry, scoped to the given org.
func (p *Poller) SyncTopPackages(ctx context.Context, reg registry.Registry, limit int, orgID uint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	slog.Info("syncing top packages", "registry", reg.Name(), "limit", limit, "org_id", orgID)

	names, err := reg.GetTopPackages(ctx, limit)
	if err != nil {
		return fmt.Errorf("fetching top packages: %w", err)
	}

	for i, name := range names {
		rank := uint(i + 1)
		var existing models.Package
		result := p.db.Where("org_id = ? AND name = ? AND registry = ?", orgID, name, reg.Name()).Limit(1).Find(&existing)
		if result.RowsAffected == 0 {
			pkg := models.Package{
				OrgID:    orgID,
				Name:     name,
				Registry: models.Registry(reg.Name()),
				IsCustom: false,
				Rank:     &rank,
			}
			if err := p.db.Create(&pkg).Error; err != nil {
				slog.Error("failed to create top package", "package", name, "error", err)
				continue
			}
		} else {
			p.db.Model(&existing).Updates(map[string]any{
				"rank":      &rank,
				"is_custom": false,
			})
		}
	}

	slog.Info("top packages synced", "registry", reg.Name(), "count", len(names), "org_id", orgID)
	return nil
}
