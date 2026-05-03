package persistent

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// DigestRepo implements digest.DigestRepository using GORM.
type DigestRepo struct {
	db *gorm.DB
}

// NewDigestRepo creates a new DigestRepo.
func NewDigestRepo(db *gorm.DB) *DigestRepo {
	return &DigestRepo{db: db}
}

// FindEnabledDigestConfigs returns workspace IDs that have email_digest_enabled=true,
// along with their frequency and recipients settings.
func (r *DigestRepo) FindEnabledDigestConfigs(ctx context.Context) ([]entity.DigestOrgConfig, error) {
	var enabledSettings []Setting
	if err := r.db.WithContext(ctx).
		Where("key = ? AND value = ?", entity.SettingEmailDigestEnabled, "true").
		Find(&enabledSettings).Error; err != nil {
		return nil, fmt.Errorf("DigestRepo.FindEnabledDigestConfigs: %w", err)
	}

	var configs []entity.DigestOrgConfig
	for _, setting := range enabledSettings {
		wsID := derefString(setting.WorkspaceID)

		var frequency, recipients string
		var freqSetting, recipSetting Setting

		if err := r.db.WithContext(ctx).
			Where("workspace_id = ? AND key = ?", wsID, entity.SettingEmailDigestFrequency).
			First(&freqSetting).Error; err != nil {
			frequency = "daily"
		} else {
			frequency = freqSetting.Value
		}

		if err := r.db.WithContext(ctx).
			Where("workspace_id = ? AND key = ?", wsID, entity.SettingEmailDigestRecipients).
			First(&recipSetting).Error; err != nil {
			continue
		} else {
			recipients = recipSetting.Value
		}

		if recipients == "" {
			continue
		}

		configs = append(configs, entity.DigestOrgConfig{
			WorkspaceID: wsID,
			Frequency:   frequency,
			Recipients:  recipients,
		})
	}

	return configs, nil
}

// CountAlertsSince counts alerts for a workspace created since the given time.
func (r *DigestRepo) CountAlertsSince(ctx context.Context, workspaceID string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&Alert{}).
		Where("workspace_id = ? AND created_at >= ?", workspaceID, since).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("DigestRepo.CountAlertsSince: %w", err)
	}
	return count, nil
}

// CountPackagesAnalyzedSince counts distinct packages with releases created since the given time.
func (r *DigestRepo) CountPackagesAnalyzedSince(ctx context.Context, workspaceID string, since time.Time) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&Release{}).
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.workspace_id = ? AND releases.created_at >= ?", workspaceID, since).
		Distinct("releases.package_id").
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("DigestRepo.CountPackagesAnalyzedSince: %w", err)
	}
	return count, nil
}

// GetClassificationBreakdownSince returns classification -> count for analyses since the given time.
func (r *DigestRepo) GetClassificationBreakdownSince(ctx context.Context, workspaceID string, since time.Time) (map[string]int64, error) {
	type classCount struct {
		Classification string
		Count          int64
	}
	var rows []classCount
	if err := r.db.WithContext(ctx).
		Model(&Analysis{}).
		Select("analyses.classification, COUNT(*) as count").
		Joins("JOIN diffs ON diffs.id = analyses.diff_id").
		Joins("JOIN releases ON releases.id = diffs.release_id").
		Joins("JOIN packages ON packages.id = releases.package_id").
		Where("packages.workspace_id = ? AND analyses.created_at >= ?", workspaceID, since).
		Group("analyses.classification").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("DigestRepo.GetClassificationBreakdownSince: %w", err)
	}

	result := make(map[string]int64)
	for _, row := range rows {
		result[row.Classification] = row.Count
	}
	return result, nil
}

// GetTopAlertsSince returns the top N alerts by severity for a workspace since the given time.
func (r *DigestRepo) GetTopAlertsSince(ctx context.Context, workspaceID string, since time.Time, limit int) ([]entity.DigestTopAlert, error) {
	type alertRow struct {
		ID          string
		PackageName string
		Severity    string
		Message     string
		CreatedAt   time.Time
	}
	var rows []alertRow
	if err := r.db.WithContext(ctx).
		Model(&Alert{}).
		Select("alerts.id, packages.name as package_name, alerts.severity, alerts.message, alerts.created_at").
		Joins("JOIN packages ON packages.id = alerts.package_id").
		Where("alerts.workspace_id = ? AND alerts.created_at >= ?", workspaceID, since).
		Order("CASE alerts.severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 WHEN 'low' THEN 3 END ASC, alerts.created_at DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("DigestRepo.GetTopAlertsSince: %w", err)
	}

	var alerts []entity.DigestTopAlert
	for _, row := range rows {
		alerts = append(alerts, entity.DigestTopAlert{
			ID:          row.ID,
			PackageName: row.PackageName,
			Severity:    row.Severity,
			Message:     row.Message,
			CreatedAt:   row.CreatedAt,
		})
	}
	return alerts, nil
}
