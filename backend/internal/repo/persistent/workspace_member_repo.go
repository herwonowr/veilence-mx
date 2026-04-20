package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// WorkspaceMemberRepo implements usecase.WorkspaceMemberRepository using GORM.
type WorkspaceMemberRepo struct {
	db *gorm.DB
}

// NewWorkspaceMemberRepo creates a new WorkspaceMemberRepo.
func NewWorkspaceMemberRepo(db *gorm.DB) *WorkspaceMemberRepo {
	return &WorkspaceMemberRepo{db: db}
}

func (r *WorkspaceMemberRepo) FindByWorkspaceID(ctx context.Context, workspaceID string) ([]entity.WorkspaceMember, error) {
	var ms []WorkspaceMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("workspace_id = ?", workspaceID).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing workspace members: %w", err)
	}
	result := make([]entity.WorkspaceMember, len(ms))
	for i := range ms {
		result[i] = *workspaceMemberToDomain(&ms[i])
	}
	return result, nil
}

func (r *WorkspaceMemberRepo) FindByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (*entity.WorkspaceMember, error) {
	var m WorkspaceMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ? AND workspace_id = ?", userID, workspaceID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("member %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding member: %w", err)
	}
	return workspaceMemberToDomain(&m), nil
}

func (r *WorkspaceMemberRepo) CountByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting membership: %w", err)
	}
	return count, nil
}

func (r *WorkspaceMemberRepo) Create(ctx context.Context, member *entity.WorkspaceMember) error {
	m := workspaceMemberToModel(member)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating membership: %w", err)
	}
	member.ID = m.ID
	member.CreatedAt = m.CreatedAt
	member.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *WorkspaceMemberRepo) Update(ctx context.Context, member *entity.WorkspaceMember) error {
	m := workspaceMemberToModel(member)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating membership: %w", err)
	}
	// Reload with role
	var reloaded WorkspaceMember
	r.db.WithContext(ctx).Preload("Role").Where("id = ?", m.ID).First(&reloaded)
	*member = *workspaceMemberToDomain(&reloaded)
	return nil
}

func (r *WorkspaceMemberRepo) DeleteByUserAndWorkspace(ctx context.Context, userID, workspaceID string) error {
	result := r.db.WithContext(ctx).Where("workspace_id = ? AND user_id = ?", workspaceID, userID).Delete(&WorkspaceMember{})
	if result.Error != nil {
		return fmt.Errorf("removing member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("member %w", entity.ErrNotFound)
	}
	return nil
}

// FindWorkspaceIDsByUserID returns the workspace IDs the user is a member of.
// Implements usecase.UserWorkspaceLister.
func (r *WorkspaceMemberRepo) FindWorkspaceIDsByUserID(ctx context.Context, userID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&WorkspaceMember{}).
		Where("user_id = ?", userID).
		Pluck("workspace_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("listing workspace IDs for user: %w", err)
	}
	return ids, nil
}

// --- Converters ---

func workspaceMemberToDomain(m *WorkspaceMember) *entity.WorkspaceMember {
	d := &entity.WorkspaceMember{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		UserID:      m.UserID,
		RoleID:      m.RoleID,
		JoinedAt:    m.JoinedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.Role.ID != "" {
		r := roleToDomain(&m.Role)
		d.Role = r
	}
	return d
}

func workspaceMemberToModel(d *entity.WorkspaceMember) *WorkspaceMember {
	return &WorkspaceMember{
		ID:          d.ID,
		WorkspaceID: d.WorkspaceID,
		UserID:      d.UserID,
		RoleID:      d.RoleID,
		JoinedAt:    d.JoinedAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
