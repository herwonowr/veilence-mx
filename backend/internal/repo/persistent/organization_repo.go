package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// OrganizationRepo implements entity.OrganizationRepository using GORM.
type OrganizationRepo struct {
	db *gorm.DB
}

// NewOrganizationRepo creates a new OrganizationRepo.
func NewOrganizationRepo(db *gorm.DB) *OrganizationRepo {
	return &OrganizationRepo{db: db}
}

func (r *OrganizationRepo) FindByID(ctx context.Context, id uint) (*entity.Organization, error) {
	var m Organization
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("organization %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding organization: %w", err)
	}
	return orgToDomain(&m), nil
}

func (r *OrganizationRepo) FindBySlug(ctx context.Context, slug string) (*entity.Organization, error) {
	var m Organization
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("organization %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding organization by slug: %w", err)
	}
	return orgToDomain(&m), nil
}

func (r *OrganizationRepo) CountBySlug(ctx context.Context, slug string, excludeID *uint) (int64, error) {
	query := r.db.WithContext(ctx).Model(&Organization{}).Where("slug = ?", slug)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting organizations by slug: %w", err)
	}
	return count, nil
}

func (r *OrganizationRepo) Create(ctx context.Context, org *entity.Organization) error {
	m := orgToModel(org)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating organization: %w", err)
	}
	org.ID = m.ID
	org.CreatedAt = m.CreatedAt
	org.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *OrganizationRepo) Update(ctx context.Context, org *entity.Organization) error {
	m := orgToModel(org)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating organization: %w", err)
	}
	org.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *OrganizationRepo) SoftDelete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&Organization{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting organization: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("organization %w", entity.ErrNotFound)
	}
	return nil
}

func (r *OrganizationRepo) FindByUserID(ctx context.Context, userID uint) ([]entity.Organization, error) {
	var ms []Organization
	err := r.db.WithContext(ctx).
		Joins("JOIN org_members ON org_members.org_id = organizations.id").
		Where("org_members.user_id = ? AND organizations.deleted_at IS NULL", userID).
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("listing user organizations: %w", err)
	}
	result := make([]entity.Organization, len(ms))
	for i := range ms {
		result[i] = *orgToDomain(&ms[i])
	}
	return result, nil
}

// --- Converters ---

func orgToDomain(m *Organization) *entity.Organization {
	return &entity.Organization{
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

func orgToModel(d *entity.Organization) *Organization {
	return &Organization{
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
