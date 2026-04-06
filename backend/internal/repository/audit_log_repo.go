package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// AuditLogRepo implements domain.AuditLogRepository using GORM.
type AuditLogRepo struct {
	db *gorm.DB
}

// NewAuditLogRepo creates a new AuditLogRepo.
func NewAuditLogRepo(db *gorm.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(ctx context.Context, entry *domain.AuditLog) error {
	m := auditLogToModel(entry)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating audit log: %w", err)
	}
	entry.ID = m.ID
	entry.CreatedAt = m.CreatedAt
	return nil
}

func (r *AuditLogRepo) FindByOrgID(ctx context.Context, orgID uint, filters domain.AuditLogFilters, page, limit int) ([]domain.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).Where("org_id = ?", orgID)

	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.Resource != "" {
		query = query.Where("resource = ?", filters.Resource)
	}
	if filters.UserID != 0 {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.FromDate != nil {
		query = query.Where("created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("created_at <= ?", *filters.ToDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting audit logs: %w", err)
	}

	var ms []models.AuditLog
	err := query.
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing audit logs: %w", err)
	}

	result := make([]domain.AuditLog, len(ms))
	for i := range ms {
		result[i] = *auditLogToDomain(&ms[i])
	}
	return result, total, nil
}

func (r *AuditLogRepo) FindByID(ctx context.Context, id uint) (*domain.AuditLog, error) {
	var m models.AuditLog
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("audit log %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("getting audit log: %w", err)
	}
	return auditLogToDomain(&m), nil
}

// --- Converters ---

func auditLogToDomain(m *models.AuditLog) *domain.AuditLog {
	return &domain.AuditLog{
		ID:            m.ID,
		UserID:        m.UserID,
		OrgID:         m.OrgID,
		Action:        m.Action,
		Resource:      m.Resource,
		ResourceID:    m.ResourceID,
		Details:       m.Details,
		IPAddress:     m.IPAddress,
		UserAgent:     m.UserAgent,
		CorrelationID: m.CorrelationID,
		CreatedAt:     m.CreatedAt,
	}
}

func auditLogToModel(d *domain.AuditLog) *models.AuditLog {
	return &models.AuditLog{
		ID:            d.ID,
		UserID:        d.UserID,
		OrgID:         d.OrgID,
		Action:        d.Action,
		Resource:      d.Resource,
		ResourceID:    d.ResourceID,
		Details:       d.Details,
		IPAddress:     d.IPAddress,
		UserAgent:     d.UserAgent,
		CorrelationID: d.CorrelationID,
		CreatedAt:     d.CreatedAt,
	}
}
