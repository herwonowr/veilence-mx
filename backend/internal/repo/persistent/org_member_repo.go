package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// OrgMemberRepo implements entity.OrgMemberRepository using GORM.
type OrgMemberRepo struct {
	db *gorm.DB
}

// NewOrgMemberRepo creates a new OrgMemberRepo.
func NewOrgMemberRepo(db *gorm.DB) *OrgMemberRepo {
	return &OrgMemberRepo{db: db}
}

func (r *OrgMemberRepo) FindByOrgID(ctx context.Context, orgID uint) ([]entity.OrgMember, error) {
	var ms []OrgMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("org_id = ?", orgID).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing org members: %w", err)
	}
	result := make([]entity.OrgMember, len(ms))
	for i := range ms {
		result[i] = *orgMemberToDomain(&ms[i])
	}
	return result, nil
}

func (r *OrgMemberRepo) FindByUserAndOrg(ctx context.Context, userID, orgID uint) (*entity.OrgMember, error) {
	var m OrgMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ? AND org_id = ?", userID, orgID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("member %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding member: %w", err)
	}
	return orgMemberToDomain(&m), nil
}

func (r *OrgMemberRepo) CountByUserAndOrg(ctx context.Context, userID, orgID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&OrgMember{}).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting membership: %w", err)
	}
	return count, nil
}

func (r *OrgMemberRepo) Create(ctx context.Context, member *entity.OrgMember) error {
	m := orgMemberToModel(member)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating membership: %w", err)
	}
	member.ID = m.ID
	member.CreatedAt = m.CreatedAt
	member.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *OrgMemberRepo) Update(ctx context.Context, member *entity.OrgMember) error {
	m := orgMemberToModel(member)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating membership: %w", err)
	}
	// Reload with role
	var reloaded OrgMember
	r.db.WithContext(ctx).Preload("Role").First(&reloaded, m.ID)
	*member = *orgMemberToDomain(&reloaded)
	return nil
}

func (r *OrgMemberRepo) DeleteByUserAndOrg(ctx context.Context, userID, orgID uint) error {
	result := r.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", orgID, userID).Delete(&OrgMember{})
	if result.Error != nil {
		return fmt.Errorf("removing member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("member %w", entity.ErrNotFound)
	}
	return nil
}

// --- Converters ---

func orgMemberToDomain(m *OrgMember) *entity.OrgMember {
	d := &entity.OrgMember{
		ID:        m.ID,
		OrgID:     m.OrgID,
		UserID:    m.UserID,
		RoleID:    m.RoleID,
		JoinedAt:  m.JoinedAt,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.Role.ID != 0 {
		r := roleToDomain(&m.Role)
		d.Role = r
	}
	return d
}

func orgMemberToModel(d *entity.OrgMember) *OrgMember {
	return &OrgMember{
		ID:        d.ID,
		OrgID:     d.OrgID,
		UserID:    d.UserID,
		RoleID:    d.RoleID,
		JoinedAt:  d.JoinedAt,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
