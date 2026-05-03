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
	IsSuperAdmin       bool       `json:"isSuperAdmin"`
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
		IsSuperAdmin:       u.IsSuperAdmin,
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

// AdminUserResponse is the JSON representation of a user for super-admin views.
// Includes fields not shown in the regular UserResponse (isSuperAdmin, deactivatedAt, authMethod).
type AdminUserResponse struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	FirstName          string     `json:"firstName"`
	LastName           string     `json:"lastName"`
	IsActive           bool       `json:"isActive"`
	IsSuperAdmin       bool       `json:"isSuperAdmin"`
	DeactivatedAt      *time.Time `json:"deactivatedAt,omitempty"`
	EmailVerified      bool       `json:"emailVerified"`
	MustChangePassword bool       `json:"mustChangePassword"`
	AuthMethod         string     `json:"authMethod"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

// AdminUserFromEntity maps a domain User to an admin response DTO.
func AdminUserFromEntity(u *entity.User) AdminUserResponse {
	authMethod := string(u.AuthProvider)
	if authMethod == "local" {
		authMethod = "password"
	}
	return AdminUserResponse{
		ID:                 u.ID,
		Email:              u.Email,
		FirstName:          u.FirstName,
		LastName:           u.LastName,
		IsActive:           u.IsActive,
		IsSuperAdmin:       u.IsSuperAdmin,
		DeactivatedAt:      u.DeactivatedAt,
		EmailVerified:      u.EmailVerified,
		MustChangePassword: u.MustChangePassword,
		AuthMethod:         authMethod,
		LastLoginAt:        u.LastLoginAt,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}

// AdminUsersFromEntities maps a slice of domain Users to admin response DTOs.
func AdminUsersFromEntities(users []entity.User) []AdminUserResponse {
	result := make([]AdminUserResponse, len(users))
	for i := range users {
		result[i] = AdminUserFromEntity(&users[i])
	}
	return result
}

// ListUsersResponse is the paginated response for GET /api/admin/users.
// This matches the frontend ListUsersResponse type exactly.
type ListUsersResponse struct {
	Users      []AdminUserResponse `json:"users"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	TotalPages int                 `json:"totalPages"`
}

// PlatformUserDetailResponse is the detailed admin view of a user,
// including linked SSO identities and workspace memberships.
type PlatformUserDetailResponse struct {
	AdminUserResponse
	Identities []LinkedIdentityResponse `json:"identities"`
	Workspaces []UserWorkspaceResponse  `json:"workspaces"`
}

// LinkedIdentityResponse is the JSON representation of a linked SSO identity.
type LinkedIdentityResponse struct {
	ID             string `json:"id"`
	Provider       string `json:"provider"`
	ProviderEmail  string `json:"providerEmail"`
	ProviderUserID string `json:"providerUserId"`
}

// UserWorkspaceResponse is the JSON representation of a user's workspace membership.
type UserWorkspaceResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}
