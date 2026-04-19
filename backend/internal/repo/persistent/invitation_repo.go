package persistent

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// InvitationRepo implements entity.InvitationRepository using GORM.
type InvitationRepo struct {
	db *gorm.DB
}

// NewInvitationRepo creates a new InvitationRepo.
func NewInvitationRepo(db *gorm.DB) *InvitationRepo {
	return &InvitationRepo{db: db}
}

func (r *InvitationRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*entity.Invitation, error) {
	var m Invitation
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invitation %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("finding invitation: %w", err)
	}
	return invitationToDomain(&m), nil
}

func (r *InvitationRepo) Create(ctx context.Context, invitation *entity.Invitation) error {
	m := invitationToModel(invitation)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating invitation: %w", err)
	}
	invitation.ID = m.ID
	invitation.CreatedAt = m.CreatedAt
	return nil
}

func (r *InvitationRepo) Update(ctx context.Context, invitation *entity.Invitation) error {
	m := invitationToModel(invitation)
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return fmt.Errorf("updating invitation: %w", err)
	}
	return nil
}

// --- Converters ---

func invitationToDomain(m *Invitation) *entity.Invitation {
	return &entity.Invitation{
		ID:         m.ID,
		WorkspaceID:      m.WorkspaceID,
		Email:      m.Email,
		RoleID:     m.RoleID,
		TokenHash:  m.TokenHash,
		InvitedBy:  m.InvitedBy,
		ExpiresAt:  m.ExpiresAt,
		AcceptedAt: m.AcceptedAt,
		CreatedAt:  m.CreatedAt,
	}
}

func invitationToModel(d *entity.Invitation) *Invitation {
	return &Invitation{
		ID:         d.ID,
		WorkspaceID:      d.WorkspaceID,
		Email:      d.Email,
		RoleID:     d.RoleID,
		TokenHash:  d.TokenHash,
		InvitedBy:  d.InvitedBy,
		ExpiresAt:  d.ExpiresAt,
		AcceptedAt: d.AcceptedAt,
		CreatedAt:  d.CreatedAt,
	}
}
