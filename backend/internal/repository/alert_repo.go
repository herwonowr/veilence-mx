package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// AlertRepo implements domain.AlertRepository using GORM.
type AlertRepo struct {
	db *gorm.DB
}

// NewAlertRepo creates a new AlertRepo.
func NewAlertRepo(db *gorm.DB) *AlertRepo {
	return &AlertRepo{db: db}
}

func (r *AlertRepo) FindByID(ctx context.Context, id uint) (*domain.Alert, error) {
	var m models.Alert
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("alert %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding alert: %w", err)
	}
	return alertToDomain(&m), nil
}

func (r *AlertRepo) FindByOrgID(ctx context.Context, orgID uint, page, limit int, sortClause string, filters domain.AlertFilters) ([]domain.Alert, int64, error) {
	var total int64
	query := r.db.WithContext(ctx).Model(&models.Alert{}).
		Where("alerts.org_id = ?", orgID)

	// Only JOIN packages when we need to search
	needsJoin := filters.Search != nil && *filters.Search != ""
	if needsJoin {
		query = query.Joins("JOIN packages ON packages.id = alerts.package_id")
		escapedSearch := escapeLikeRepo(*filters.Search)
		query = query.Where("(packages.name ILIKE ? OR alerts.message ILIKE ?)",
			"%"+escapedSearch+"%", "%"+escapedSearch+"%")
	}

	if filters.Severity != nil {
		query = query.Where("alerts.severity = ?", string(*filters.Severity))
	}
	if filters.Status != nil {
		query = query.Where("alerts.status = ?", string(*filters.Status))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting alerts: %w", err)
	}

	// If sortClause doesn't contain a table qualifier, prefix with "alerts."
	// to avoid ambiguity when a JOIN is present.
	if sortClause != "" && !strings.Contains(sortClause, ".") {
		sortClause = "alerts." + sortClause
	}

	var ms []models.Alert
	err := query.
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing alerts: %w", err)
	}

	result := make([]domain.Alert, len(ms))
	for i := range ms {
		result[i] = *alertToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *AlertRepo) Create(ctx context.Context, alert *domain.Alert) error {
	m := alertToModel(alert)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating alert: %w", err)
	}
	alert.ID = m.ID
	alert.CreatedAt = m.CreatedAt
	alert.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *AlertRepo) Update(ctx context.Context, alert *domain.Alert) error {
	m := alertToModel(alert)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating alert: %w", err)
	}
	alert.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *AlertRepo) CountByOrgAndStatus(ctx context.Context, orgID uint) (map[domain.AlertStatus]int64, error) {
	type statusCount struct {
		Status string
		Count  int64
	}
	var results []statusCount
	err := r.db.WithContext(ctx).
		Model(&models.Alert{}).
		Select("status, count(*) as count").
		Where("org_id = ?", orgID).
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("counting alerts by status: %w", err)
	}

	counts := make(map[domain.AlertStatus]int64)
	for _, r := range results {
		counts[domain.AlertStatus(r.Status)] = r.Count
	}
	return counts, nil
}

// --- Converters ---

func alertToDomain(m *models.Alert) *domain.Alert {
	return &domain.Alert{
		ID:         m.ID,
		OrgID:      m.OrgID,
		AnalysisID: m.AnalysisID,
		ReleaseID:  m.ReleaseID,
		PackageID:  m.PackageID,
		Severity:   domain.AlertSeverity(m.Severity),
		Status:     domain.AlertStatus(m.Status),
		Message:    m.Message,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func alertToModel(d *domain.Alert) *models.Alert {
	return &models.Alert{
		ID:         d.ID,
		OrgID:      d.OrgID,
		AnalysisID: d.AnalysisID,
		ReleaseID:  d.ReleaseID,
		PackageID:  d.PackageID,
		Severity:   models.AlertSeverity(d.Severity),
		Status:     models.AlertStatus(d.Status),
		Message:    d.Message,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}

// escapeLikeRepo escapes LIKE/ILIKE special characters for safe queries.
func escapeLikeRepo(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
