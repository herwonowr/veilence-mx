package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// OrgMemberRepo implements domain.OrgMemberRepository using GORM.
type OrgMemberRepo struct {
	db *gorm.DB
}

// NewOrgMemberRepo creates a new OrgMemberRepo.
func NewOrgMemberRepo(db *gorm.DB) *OrgMemberRepo {
	return &OrgMemberRepo{db: db}
}

func (r *OrgMemberRepo) FindByOrgID(ctx context.Context, orgID uint) ([]domain.OrgMember, error) {
	var ms []models.OrgMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("org_id = ?", orgID).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing org members: %w", err)
	}
	result := make([]domain.OrgMember, len(ms))
	for i := range ms {
		result[i] = *orgMemberToDomain(&ms[i])
	}
	return result, nil
}

func (r *OrgMemberRepo) FindByUserAndOrg(ctx context.Context, userID, orgID uint) (*domain.OrgMember, error) {
	var m models.OrgMember
	err := r.db.WithContext(ctx).
		Preload("Role").
		Where("user_id = ? AND org_id = ?", userID, orgID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("member %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding member: %w", err)
	}
	return orgMemberToDomain(&m), nil
}

func (r *OrgMemberRepo) CountByUserAndOrg(ctx context.Context, userID, orgID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.OrgMember{}).
		Where("org_id = ? AND user_id = ?", orgID, userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("counting membership: %w", err)
	}
	return count, nil
}

func (r *OrgMemberRepo) Create(ctx context.Context, member *domain.OrgMember) error {
	m := orgMemberToModel(member)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating membership: %w", err)
	}
	member.ID = m.ID
	member.CreatedAt = m.CreatedAt
	member.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *OrgMemberRepo) Update(ctx context.Context, member *domain.OrgMember) error {
	m := orgMemberToModel(member)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating membership: %w", err)
	}
	// Reload with role
	var reloaded models.OrgMember
	r.db.WithContext(ctx).Preload("Role").First(&reloaded, m.ID)
	*member = *orgMemberToDomain(&reloaded)
	return nil
}

func (r *OrgMemberRepo) DeleteByUserAndOrg(ctx context.Context, userID, orgID uint) error {
	result := r.db.WithContext(ctx).Where("org_id = ? AND user_id = ?", orgID, userID).Delete(&models.OrgMember{})
	if result.Error != nil {
		return fmt.Errorf("removing member: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("member %w", domain.ErrNotFound)
	}
	return nil
}

// --- Converters ---

func orgMemberToDomain(m *models.OrgMember) *domain.OrgMember {
	d := &domain.OrgMember{
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

func orgMemberToModel(d *domain.OrgMember) *models.OrgMember {
	return &models.OrgMember{
		ID:        d.ID,
		OrgID:     d.OrgID,
		UserID:    d.UserID,
		RoleID:    d.RoleID,
		JoinedAt:  d.JoinedAt,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
