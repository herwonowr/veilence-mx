package persistent

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// DashboardRepo implements entity.DashboardRepository using GORM.
type DashboardRepo struct {
	db *gorm.DB
}

// NewDashboardRepo creates a new DashboardRepo.
func NewDashboardRepo(db *gorm.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

func (r *DashboardRepo) GetStats(ctx context.Context, orgID uint) (*entity.DashboardStats, error) {
	var stats entity.DashboardStats

	if err := r.db.WithContext(ctx).Model(&Package{}).
		Where("org_id = ? AND status = ?", orgID, PackageStatusActive).
		Count(&stats.TotalPackages).Error; err != nil {
		return nil, fmt.Errorf("counting packages: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND packages.status = ?", orgID, PackageStatusActive).
		Count(&stats.TotalReleases).Error; err != nil {
		return nil, fmt.Errorf("counting releases: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND packages.status = ? AND releases.status IN ?", orgID, PackageStatusActive, []string{"pending", "diffing", "analyzing"}).
		Count(&stats.PendingAnalyses).Error; err != nil {
		return nil, fmt.Errorf("counting pending analyses: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&Alert{}).
		Where("org_id = ? AND status = ?", orgID, "new").
		Count(&stats.ActiveAlerts).Error; err != nil {
		return nil, fmt.Errorf("counting active alerts: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&Analysis{}).
		Joins("JOIN diffs ON diffs.id = analyses.diff_id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND packages.status = ? AND analyses.classification = ?", orgID, PackageStatusActive, "malicious").
		Count(&stats.RecentMalicious).Error; err != nil {
		return nil, fmt.Errorf("counting malicious analyses: %w", err)
	}

	return &stats, nil
}

func (r *DashboardRepo) GetReleaseActivity(ctx context.Context, orgID uint, from, to time.Time) ([]entity.ReleaseActivityPoint, error) {
	var rows []struct {
		Date  time.Time
		Count int64
	}

	err := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Select("DATE(releases.created_at) as date, COUNT(*) as count").
		Where("packages.org_id = ? AND packages.status = ? AND releases.created_at >= ? AND releases.created_at <= ?", orgID, PackageStatusActive, from, to).
		Group("DATE(releases.created_at)").
		Order("date ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("querying release activity: %w", err)
	}

	result := make([]entity.ReleaseActivityPoint, len(rows))
	for i, row := range rows {
		result[i] = entity.ReleaseActivityPoint{Date: row.Date, Count: row.Count}
	}
	return result, nil
}

func (r *DashboardRepo) GetClassificationDistribution(ctx context.Context, orgID uint, from, to time.Time) ([]entity.ClassificationCount, error) {
	var rows []struct {
		Classification string
		Count          int64
	}

	err := r.db.WithContext(ctx).Model(&Analysis{}).
		Joins("JOIN diffs ON diffs.id = analyses.diff_id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Select("analyses.classification, COUNT(*) as count").
		Where("packages.org_id = ? AND packages.status = ? AND analyses.created_at >= ? AND analyses.created_at <= ?", orgID, PackageStatusActive, from, to).
		Group("analyses.classification").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("querying classification distribution: %w", err)
	}

	result := make([]entity.ClassificationCount, len(rows))
	for i, row := range rows {
		result[i] = entity.ClassificationCount{Classification: row.Classification, Count: row.Count}
	}
	return result, nil
}

func (r *DashboardRepo) GetBaselineCount(ctx context.Context, orgID uint, from, to time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.org_id = ? AND packages.status = ? AND releases.status = ? AND releases.id NOT IN (SELECT release_id FROM diffs) AND releases.created_at >= ? AND releases.created_at <= ?",
			orgID, PackageStatusActive, ReleaseStatusCompleted, from, to).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting baselines: %w", err)
	}
	return count, nil
}

func (r *DashboardRepo) GetEcosystemDistribution(ctx context.Context, orgID uint) ([]entity.EcosystemCount, error) {
	var rows []struct {
		Ecosystem string
		Count     int64
	}

	err := r.db.WithContext(ctx).Model(&Package{}).
		Select("ecosystem, COUNT(*) as count").
		Where("org_id = ? AND status = ?", orgID, PackageStatusActive).
		Group("ecosystem").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("querying ecosystem distribution: %w", err)
	}

	result := make([]entity.EcosystemCount, len(rows))
	for i, row := range rows {
		result[i] = entity.EcosystemCount{Ecosystem: row.Ecosystem, Count: row.Count}
	}
	return result, nil
}

func (r *DashboardRepo) GetAlertsBySeverity(ctx context.Context, orgID uint, from, to time.Time) ([]entity.AlertSeverityCount, error) {
	var rows []struct {
		Severity string
		Count    int64
	}

	err := r.db.WithContext(ctx).Model(&Alert{}).
		Select("severity, COUNT(*) as count").
		Where("org_id = ? AND created_at >= ? AND created_at <= ?", orgID, from, to).
		Group("severity").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("querying alerts by severity: %w", err)
	}

	result := make([]entity.AlertSeverityCount, len(rows))
	for i, row := range rows {
		result[i] = entity.AlertSeverityCount{Severity: row.Severity, Count: row.Count}
	}
	return result, nil
}

func (r *DashboardRepo) GetReleaseStatusDistribution(ctx context.Context, orgID uint, from, to time.Time) ([]entity.ReleaseStatusCount, error) {
	var rows []struct {
		Status string
		Count  int64
	}

	err := r.db.WithContext(ctx).Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Select("releases.status, COUNT(*) as count").
		Where("packages.org_id = ? AND packages.status = ? AND releases.created_at >= ? AND releases.created_at <= ?", orgID, PackageStatusActive, from, to).
		Group("releases.status").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("querying release status distribution: %w", err)
	}

	result := make([]entity.ReleaseStatusCount, len(rows))
	for i, row := range rows {
		result[i] = entity.ReleaseStatusCount{Status: row.Status, Count: row.Count}
	}
	return result, nil
}

func (r *DashboardRepo) GetUnanalyzedDiffIDs(ctx context.Context, orgID uint) ([]uint, error) {
	var ids []uint
	err := r.db.WithContext(ctx).Model(&Diff{}).
		Select("diffs.id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Joins("LEFT JOIN analyses ON analyses.diff_id = diffs.id").
		Where("packages.org_id = ? AND packages.status = ? AND analyses.id IS NULL", orgID, PackageStatusActive).
		Pluck("diffs.id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("querying unanalyzed diff IDs: %w", err)
	}
	return ids, nil
}
