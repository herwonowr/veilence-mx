package response

import (
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
)

// UserResponse is the JSON representation of a user.
// Replaces direct serialization of entity.User.
type UserResponse struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	FirstName          string     `json:"firstName"`
	LastName           string     `json:"lastName"`
	IsActive           bool       `json:"isActive"`
	EmailVerified      bool       `json:"emailVerified"`
	MustChangePassword bool       `json:"mustChangePassword"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// UserFromEntity maps a domain User to a response DTO.
// PasswordHash is intentionally excluded.
func UserFromEntity(u *entity.User) UserResponse {
	return UserResponse{
		ID:                 u.ID,
		Email:              u.Email,
		FirstName:          u.FirstName,
		LastName:           u.LastName,
		IsActive:           u.IsActive,
		EmailVerified:      u.EmailVerified,
		MustChangePassword: u.MustChangePassword,
		LastLoginAt:        u.LastLoginAt,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}

// APIKeyResponse is the JSON representation of an API key.
// Replaces direct serialization of entity.APIKey.
type APIKeyResponse struct {
	ID          string            `json:"id"`
	UserID      string            `json:"userId"`
	WorkspaceID string            `json:"workspaceId"`
	Name        string            `json:"name"`
	KeyPrefix   string            `json:"keyPrefix"`
	Role        entity.APIKeyRole `json:"role"`
	LastUsedAt  *time.Time        `json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time        `json:"expiresAt,omitempty"`
	IsActive    bool              `json:"isActive"`
	CreatedAt   time.Time         `json:"createdAt"`
}

// APIKeyFromEntity maps a domain APIKey to a response DTO.
// KeyHash is intentionally excluded.
func APIKeyFromEntity(k *entity.APIKey) APIKeyResponse {
	return APIKeyResponse{
		ID:          k.ID,
		UserID:      k.UserID,
		WorkspaceID: k.WorkspaceID,
		Name:        k.Name,
		KeyPrefix:   k.KeyPrefix,
		Role:        k.Role,
		LastUsedAt:  k.LastUsedAt,
		ExpiresAt:   k.ExpiresAt,
		IsActive:    k.IsActive,
		CreatedAt:   k.CreatedAt,
	}
}

// APIKeysFromEntities maps a slice of domain APIKeys to response DTOs.
func APIKeysFromEntities(keys []entity.APIKey) []APIKeyResponse {
	result := make([]APIKeyResponse, len(keys))
	for i := range keys {
		result[i] = APIKeyFromEntity(&keys[i])
	}
	return result
}

// APIKeyCreatedResponse is returned only at API key creation time.
// It includes the raw key string which is never persisted.
type APIKeyCreatedResponse struct {
	APIKey APIKeyResponse `json:"apiKey"`
	Key    string         `json:"key"`
}
