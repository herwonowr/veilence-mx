package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// AuditLogRepo implements entity.AuditLogRepository using GORM.
type AuditLogRepo struct {
	db *gorm.DB
}

// NewAuditLogRepo creates a new AuditLogRepo.
func NewAuditLogRepo(db *gorm.DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

func (r *AuditLogRepo) Create(ctx context.Context, entry *entity.AuditLog) error {
	m := auditLogToModel(entry)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating audit log: %w", err)
	}
	entry.ID = m.ID
	entry.CreatedAt = m.CreatedAt
	return nil
}

func (r *AuditLogRepo) FindByWorkspaceID(ctx context.Context, workspaceID string, filters entity.AuditLogFilters, page, limit int) ([]entity.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&AuditLog{}).
		Joins("LEFT JOIN workspaces ON workspaces.id = audit_logs.workspace_id").
		Where("audit_logs.workspace_id = ?", workspaceID)

	if filters.Action != "" {
		query = query.Where("audit_logs.action ILIKE ?", "%"+filters.Action+"%")
	}
	if filters.Resource != "" {
		query = query.Where("audit_logs.resource ILIKE ?", "%"+filters.Resource+"%")
	}
	if filters.UserID != "" {
		query = query.Where("audit_logs.user_id = ?", filters.UserID)
	}
	if filters.FromDate != nil {
		query = query.Where("audit_logs.created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("audit_logs.created_at <= ?", *filters.ToDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting audit logs: %w", err)
	}

	var ms []auditLogWithWorkspace
	err := query.
		Select("audit_logs.*, workspaces.name as workspace_name").
		Order("audit_logs.created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing audit logs: %w", err)
	}

	result := make([]entity.AuditLog, len(ms))
	for i := range ms {
		d := auditLogToDomain(&ms[i].AuditLog)
		d.WorkspaceName = ms[i].WorkspaceName
		result[i] = *d
	}
	return result, total, nil
}

// auditLogWithWorkspace is a projection used when joining audit_logs with workspaces.
type auditLogWithWorkspace struct {
	AuditLog
	WorkspaceName string
}

func (r *AuditLogRepo) FindAll(ctx context.Context, filters entity.AuditLogFilters, page, limit int, sortClause string) ([]entity.AuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&AuditLog{}).
		Joins("LEFT JOIN workspaces ON workspaces.id = audit_logs.workspace_id")

	if filters.WorkspaceID != "" {
		query = query.Where("audit_logs.workspace_id = ?", filters.WorkspaceID)
	}
	if filters.WorkspaceName != "" {
		query = query.Where("workspaces.name ILIKE ?", "%"+filters.WorkspaceName+"%")
	}
	if filters.Action != "" {
		query = query.Where("audit_logs.action ILIKE ?", "%"+filters.Action+"%")
	}
	if filters.Resource != "" {
		query = query.Where("audit_logs.resource ILIKE ?", "%"+filters.Resource+"%")
	}
	if filters.UserID != "" {
		query = query.Where("audit_logs.user_id = ?", filters.UserID)
	}
	if filters.UserEmail != "" {
		query = query.Where("audit_logs.user_email ILIKE ?", "%"+filters.UserEmail+"%")
	}
	if filters.FromDate != nil {
		query = query.Where("audit_logs.created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("audit_logs.created_at <= ?", *filters.ToDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting audit logs: %w", err)
	}

	if sortClause == "" {
		sortClause = "audit_logs.created_at DESC"
	}

	var ms []auditLogWithWorkspace
	err := query.
		Select("audit_logs.*, workspaces.name as workspace_name").
		Order(sortClause).
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing audit logs: %w", err)
	}

	result := make([]entity.AuditLog, len(ms))
	for i := range ms {
		d := auditLogToDomain(&ms[i].AuditLog)
		d.WorkspaceName = ms[i].WorkspaceName
		result[i] = *d
	}
	return result, total, nil
}

func (r *AuditLogRepo) FindByID(ctx context.Context, id string) (*entity.AuditLog, error) {
	var m AuditLog
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("audit log %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("getting audit log: %w", err)
	}
	return auditLogToDomain(&m), nil
}

// --- Converters ---

func auditLogToDomain(m *AuditLog) *entity.AuditLog {
	return &entity.AuditLog{
		ID:            m.ID,
		UserID:        derefStr(m.UserID),
		UserEmail:     m.UserEmail,
		WorkspaceID:   derefStr(m.WorkspaceID),
		Action:        m.Action,
		Resource:      m.Resource,
		ResourceID:    derefStr(m.ResourceID),
		Details:       m.Details,
		IPAddress:     m.IPAddress,
		UserAgent:     m.UserAgent,
		CorrelationID: m.CorrelationID,
		CreatedAt:     m.CreatedAt,
	}
}

func auditLogToModel(d *entity.AuditLog) *AuditLog {
	return &AuditLog{
		ID:            d.ID,
		UserID:        strToNullableUUID(d.UserID),
		UserEmail:     d.UserEmail,
		WorkspaceID:   strToNullableUUID(d.WorkspaceID),
		Action:        d.Action,
		Resource:      d.Resource,
		ResourceID:    strToNullableUUID(d.ResourceID),
		Details:       d.Details,
		IPAddress:     d.IPAddress,
		UserAgent:     d.UserAgent,
		CorrelationID: d.CorrelationID,
		CreatedAt:     d.CreatedAt,
	}
}
