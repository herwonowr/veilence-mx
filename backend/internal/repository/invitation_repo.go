package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// InvitationRepo implements domain.InvitationRepository using GORM.
type InvitationRepo struct {
	db *gorm.DB
}

// NewInvitationRepo creates a new InvitationRepo.
func NewInvitationRepo(db *gorm.DB) *InvitationRepo {
	return &InvitationRepo{db: db}
}

func (r *InvitationRepo) FindByToken(ctx context.Context, token string) (*domain.Invitation, error) {
	var m models.Invitation
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invitation %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding invitation: %w", err)
	}
	return invitationToDomain(&m), nil
}

func (r *InvitationRepo) Create(ctx context.Context, invitation *domain.Invitation) error {
	m := invitationToModel(invitation)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating invitation: %w", err)
	}
	invitation.ID = m.ID
	invitation.CreatedAt = m.CreatedAt
	return nil
}

func (r *InvitationRepo) Update(ctx context.Context, invitation *domain.Invitation) error {
	m := invitationToModel(invitation)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating invitation: %w", err)
	}
	return nil
}

// --- Converters ---

func invitationToDomain(m *models.Invitation) *domain.Invitation {
	return &domain.Invitation{
		ID:         m.ID,
		OrgID:      m.OrgID,
		Email:      m.Email,
		RoleID:     m.RoleID,
		Token:      m.Token,
		InvitedBy:  m.InvitedBy,
		ExpiresAt:  m.ExpiresAt,
		AcceptedAt: m.AcceptedAt,
		CreatedAt:  m.CreatedAt,
	}
}

func invitationToModel(d *domain.Invitation) *models.Invitation {
	return &models.Invitation{
		ID:         d.ID,
		OrgID:      d.OrgID,
		Email:      d.Email,
		RoleID:     d.RoleID,
		Token:      d.Token,
		InvitedBy:  d.InvitedBy,
		ExpiresAt:  d.ExpiresAt,
		AcceptedAt: d.AcceptedAt,
		CreatedAt:  d.CreatedAt,
	}
}
