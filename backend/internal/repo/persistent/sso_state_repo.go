package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// SSOStateRepo implements usecase.SSOStateRepository using GORM.
type SSOStateRepo struct {
	db *gorm.DB
}

// NewSSOStateRepo creates a new SSOStateRepo.
func NewSSOStateRepo(db *gorm.DB) *SSOStateRepo {
	return &SSOStateRepo{db: db}
}

func (r *SSOStateRepo) Create(ctx context.Context, state *entity.SSOState) error {
	m := ssoStateToModel(state)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("SSOStateRepo.Create: %w", err)
	}
	state.ID = m.ID
	state.CreatedAt = m.CreatedAt
	return nil
}

func (r *SSOStateRepo) FindByState(ctx context.Context, state string) (*entity.SSOState, error) {
	var m SSOState
	if err := r.db.WithContext(ctx).Where("state = ? AND expires_at > ?", state, time.Now()).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("sso state %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("SSOStateRepo.FindByState: %w", err)
	}
	return ssoStateToDomain(&m), nil
}

func (r *SSOStateRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&SSOState{}).Error; err != nil {
		return fmt.Errorf("SSOStateRepo.Delete: %w", err)
	}
	return nil
}

func (r *SSOStateRepo) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at <= ?", time.Now()).Delete(&SSOState{})
	if result.Error != nil {
		return 0, fmt.Errorf("SSOStateRepo.DeleteExpired: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// --- Converters ---

func ssoStateToDomain(m *SSOState) *entity.SSOState {
	return &entity.SSOState{
		ID:            m.ID,
		ConfigID:      m.ConfigID,
		State:         m.State,
		UserID:        m.UserID,
		Provider:      entity.SSOProvider(m.Provider),
		CallbackURL:   m.CallbackURL,
		Mode:          m.Mode,
		CodeVerifier:  m.CodeVerifier,
		SAMLRequestID: m.SAMLRequestID,
		ExpiresAt:     m.ExpiresAt,
		CreatedAt:     m.CreatedAt,
	}
}

func ssoStateToModel(d *entity.SSOState) *SSOState {
	return &SSOState{
		ID:            d.ID,
		ConfigID:      d.ConfigID,
		State:         d.State,
		UserID:        d.UserID,
		Provider:      string(d.Provider),
		CallbackURL:   d.CallbackURL,
		Mode:          d.Mode,
		CodeVerifier:  d.CodeVerifier,
		SAMLRequestID: d.SAMLRequestID,
		ExpiresAt:     d.ExpiresAt,
		CreatedAt:     d.CreatedAt,
	}
}
