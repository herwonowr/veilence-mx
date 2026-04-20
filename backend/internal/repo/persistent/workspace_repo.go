package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// WorkspaceRepo implements usecase.WorkspaceRepository using GORM.
type WorkspaceRepo struct {
	db *gorm.DB
}

// NewWorkspaceRepo creates a new WorkspaceRepo.
func NewWorkspaceRepo(db *gorm.DB) *WorkspaceRepo {
	return &WorkspaceRepo{db: db}
}

func (r *WorkspaceRepo) FindByID(ctx context.Context, id string) (*entity.Workspace, error) {
	var m Workspace
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workspace %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding workspace: %w", err)
	}
	return workspaceToDomain(&m), nil
}

func (r *WorkspaceRepo) FindBySlug(ctx context.Context, slug string) (*entity.Workspace, error) {
	var m Workspace
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workspace %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding workspace by slug: %w", err)
	}
	return workspaceToDomain(&m), nil
}

func (r *WorkspaceRepo) CountBySlug(ctx context.Context, slug string, excludeID *string) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Workspace{}).Where("slug = ?", slug)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting workspaces by slug: %w", err)
	}
	return count, nil
}

func (r *WorkspaceRepo) Create(ctx context.Context, ws *entity.Workspace) error {
	m := workspaceToModel(ws)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}
	ws.ID = m.ID
	ws.CreatedAt = m.CreatedAt
	ws.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *WorkspaceRepo) Update(ctx context.Context, ws *entity.Workspace) error {
	m := workspaceToModel(ws)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating workspace: %w", err)
	}
	ws.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *WorkspaceRepo) SoftDelete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Workspace{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting workspace: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("workspace %w", entity.ErrNotFound)
	}
	return nil
}

func (r *WorkspaceRepo) FindByUserID(ctx context.Context, userID string) ([]entity.Workspace, error) {
	var ms []Workspace
	err := r.db.WithContext(ctx).
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id").
		Where("workspace_members.user_id = ? AND workspaces.deleted_at IS NULL", userID).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing user workspaces: %w", err)
	}
	result := make([]entity.Workspace, len(ms))
	for i := range ms {
		result[i] = *workspaceToDomain(&ms[i])
	}
	return result, nil
}

// --- Converters ---

func workspaceToDomain(m *Workspace) *entity.Workspace {
	return &entity.Workspace{
		ID:          m.ID,
		Name:        m.Name,
		Slug:        m.Slug,
		Description: m.Description,
		OwnerID:     m.OwnerID,
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func workspaceToModel(d *entity.Workspace) *Workspace {
	return &Workspace{
		ID:          d.ID,
		Name:        d.Name,
		Slug:        d.Slug,
		Description: d.Description,
		OwnerID:     d.OwnerID,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
