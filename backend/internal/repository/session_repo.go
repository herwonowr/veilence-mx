package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/models"
)

// SessionRepo implements domain.SessionRepository using GORM.
type SessionRepo struct {
	db *gorm.DB
}

// NewSessionRepo creates a new SessionRepo.
func NewSessionRepo(db *gorm.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) FindByID(ctx context.Context, id uint) (*domain.Session, error) {
	var m models.Session
	if err := r.db.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding session: %w", err)
	}
	return sessionToDomain(&m), nil
}

func (r *SessionRepo) FindByUserID(ctx context.Context, userID uint) ([]domain.Session, error) {
	var ms []models.Session
	if err := r.db.WithContext(ctx).Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Order("last_active DESC").Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	result := make([]domain.Session, len(ms))
	for i := range ms {
		result[i] = *sessionToDomain(&ms[i])
	}
	return result, nil
}

func (r *SessionRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	var m models.Session
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("session %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("finding session by token hash: %w", err)
	}
	return sessionToDomain(&m), nil
}

func (r *SessionRepo) Create(ctx context.Context, session *domain.Session) error {
	m := sessionToModel(session)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("creating session: %w", err)
	}
	session.ID = m.ID
	session.CreatedAt = m.CreatedAt
	return nil
}

func (r *SessionRepo) UpdateLastActive(ctx context.Context, id uint, lastActive time.Time) error {
	result := r.db.WithContext(ctx).Model(&models.Session{}).Where("id = ?", id).Update("last_active", lastActive)
	if result.Error != nil {
		return fmt.Errorf("updating session last_active: %w", result.Error)
	}
	return nil
}

func (r *SessionRepo) UpdateTokenHash(ctx context.Context, id uint, tokenHash string) error {
	result := r.db.WithContext(ctx).Model(&models.Session{}).Where("id = ?", id).Update("token_hash", tokenHash)
	if result.Error != nil {
		return fmt.Errorf("updating session token_hash: %w", result.Error)
	}
	return nil
}

func (r *SessionRepo) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Session{}, id).Error; err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("expires_at <= ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return 0, fmt.Errorf("deleting expired sessions: %w", result.Error)
	}
	return result.RowsAffected, nil
}

func (r *SessionRepo) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Session{}).
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("counting sessions: %w", err)
	}
	return count, nil
}

func (r *SessionRepo) DeleteOldestByUserID(ctx context.Context, userID uint) error {
	// Find the oldest session for the user
	var oldest models.Session
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at ASC").First(&oldest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // no sessions to delete
		}
		return fmt.Errorf("finding oldest session: %w", err)
	}
	if err := r.db.WithContext(ctx).Delete(&models.Session{}, oldest.ID).Error; err != nil {
		return fmt.Errorf("deleting oldest session: %w", err)
	}
	return nil
}

// --- Converters ---

func sessionToDomain(m *models.Session) *domain.Session {
	return &domain.Session{
		ID:         m.ID,
		UserID:     m.UserID,
		TokenHash:  m.TokenHash,
		IPAddress:  m.IPAddress,
		UserAgent:  m.UserAgent,
		CreatedAt:  m.CreatedAt,
		LastActive: m.LastActive,
		ExpiresAt:  m.ExpiresAt,
	}
}

func sessionToModel(d *domain.Session) *models.Session {
	return &models.Session{
		ID:         d.ID,
		UserID:     d.UserID,
		TokenHash:  d.TokenHash,
		IPAddress:  d.IPAddress,
		UserAgent:  d.UserAgent,
		CreatedAt:  d.CreatedAt,
		LastActive: d.LastActive,
		ExpiresAt:  d.ExpiresAt,
	}
}
