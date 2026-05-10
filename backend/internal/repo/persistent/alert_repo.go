package persistent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// AlertRepo implements entity.AlertRepository using GORM.
type AlertRepo struct {
	db *gorm.DB
}

// NewAlertRepo creates a new AlertRepo.
func NewAlertRepo(db *gorm.DB) *AlertRepo {
	return &AlertRepo{db: db}
}

func (r *AlertRepo) FindByIDAndWorkspaceID(ctx context.Context, id, workspaceID string) (*entity.Alert, error) {
	var m Alert
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("alert %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding alert: %w", err)
	}
	return alertToDomain(&m), nil
}

func (r *AlertRepo) Create(ctx context.Context, alert *entity.Alert) error {
	m := alertToModel(alert)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating alert: %w", err)
	}
	alert.ID = m.ID
	alert.CreatedAt = m.CreatedAt
	alert.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *AlertRepo) FindByIDWithPackage(ctx context.Context, id, workspaceID string) (*entity.Alert, *entity.Package, error) {
	var m Alert
	err := r.db.WithContext(ctx).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("alert %w", entity.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("finding alert: %w", err)
	}

	var pkg Package
	if err := r.db.WithContext(ctx).Where("id = ?", m.PackageID).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("finding alert package: %w", err)
	}

	return alertToDomain(&m), packageToDomain(&pkg), nil
}

func (r *AlertRepo) FindByWorkspaceIDWithPackage(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&Alert{}).
		Joins("JOIN packages ON packages.id = alerts.package_id").
		Where("alerts.workspace_id = ?", workspaceID)

	if filters.Search != nil && *filters.Search != "" {
		escapedSearch := escapeLikeRepo(*filters.Search)
		query = query.Where("(LOWER(packages.name) LIKE LOWER(?) OR LOWER(alerts.message) LIKE LOWER(?))",
			"%"+escapedSearch+"%", "%"+escapedSearch+"%")
	}
	if filters.Severity != nil {
		query = query.Where("alerts.severity = ?", string(*filters.Severity))
	}
	if filters.Status != nil {
		query = query.Where("alerts.status = ?", string(*filters.Status))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting alerts with package: %w", err)
	}

	if sortClause != "" && !strings.Contains(sortClause, ".") {
		sortClause = "alerts." + sortClause
	}

	type alertWithPkg struct {
		Alert
		PackageName      string `gorm:"column:package_name"`
		PackageEcosystem string `gorm:"column:package_ecosystem"`
	}
	var rows []alertWithPkg
	err := query.
		Select("alerts.*, packages.name as package_name, packages.ecosystem as package_ecosystem").
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing alerts with package: %w", err)
	}

	result := make([]entity.AlertWithPackage, len(rows))
	for i, row := range rows {
		result[i] = entity.AlertWithPackage{
			Alert:            *alertToDomain(&row.Alert),
			PackageName:      row.PackageName,
			PackageEcosystem: row.PackageEcosystem,
		}
	}
	return result, total, nil
}

func (r *AlertRepo) UpdateStatus(ctx context.Context, id, workspaceID string, status entity.AlertStatus) error {
	result := r.db.WithContext(ctx).
		Model(&Alert{}).
		Where("id = ? AND workspace_id = ?", id, workspaceID).
		Update("status", string(status))
	if result.Error != nil {
		return fmt.Errorf("updating alert status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("alert %w", entity.ErrNotFound)
	}
	return nil
}

func (r *AlertRepo) CountByWorkspaceAndStatus(ctx context.Context, workspaceID string) (map[entity.AlertStatus]int64, error) {
	type statusCount struct {
		Status string
		Count  int64
	}
	var results []statusCount
	err := r.db.WithContext(ctx).
		Model(&Alert{}).
		Select("status, count(*) as count").
		Where("workspace_id = ?", workspaceID).
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("counting alerts by status: %w", err)
	}

	counts := make(map[entity.AlertStatus]int64)
	for _, r := range results {
		counts[entity.AlertStatus(r.Status)] = r.Count
	}
	return counts, nil
}

// --- Converters ---

func alertToDomain(m *Alert) *entity.Alert {
	return &entity.Alert{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		AnalysisID:  derefStr(m.AnalysisID),
		ReleaseID:   derefStr(m.ReleaseID),
		PackageID:   m.PackageID,
		Severity:    entity.AlertSeverity(m.Severity),
		Status:      entity.AlertStatus(m.Status),
		Message:     m.Message,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func alertToModel(d *entity.Alert) *Alert {
	return &Alert{
		ID:          d.ID,
		WorkspaceID: d.WorkspaceID,
		AnalysisID:  strToNullableUUID(d.AnalysisID),
		ReleaseID:   strToNullableUUID(d.ReleaseID),
		PackageID:   d.PackageID,
		Severity:    AlertSeverity(d.Severity),
		Status:      AlertStatus(d.Status),
		Message:     d.Message,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

// escapeLikeRepo escapes LIKE special characters for safe queries.
func escapeLikeRepo(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
