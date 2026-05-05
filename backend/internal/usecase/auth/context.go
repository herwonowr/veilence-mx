package auth

import (
	"context"

	"github.com/veilence/veilence-mx/backend/internal/ctxutil"
	"github.com/veilence/veilence-mx/backend/internal/entity"
)

type contextKey string

const (
	// contextKeyAPIKeyRole is the context key for the API key role (only set for API key auth).
	contextKeyAPIKeyRole contextKey = "api_key_role"
	// contextKeyAPIKeyWorkspaceID is the context key for the API key's workspace ID (only set for API key auth).
	contextKeyAPIKeyWorkspaceID contextKey = "api_key_workspace_id"

	// AuthMethodJWT indicates authentication via JWT Bearer token.
	AuthMethodJWT = "jwt"
	// AuthMethodAPIKey indicates authentication via X-API-Key header.
	AuthMethodAPIKey = "api_key"
)

// WithUserID returns a new context with the user ID set.
func WithUserID(ctx context.Context, userID string) context.Context {
	return ctxutil.WithUserID(ctx, userID)
}

// WithEmail returns a new context with the email set.
func WithEmail(ctx context.Context, email string) context.Context {
	return ctxutil.WithEmail(ctx, email)
}

// WithAuthMethod returns a new context with the auth method set.
func WithAuthMethod(ctx context.Context, method string) context.Context {
	return ctxutil.WithAuthMethod(ctx, method)
}

// WithAPIKeyRole returns a new context with the API key role set.
func WithAPIKeyRole(ctx context.Context, role entity.APIKeyRole) context.Context {
	return context.WithValue(ctx, contextKeyAPIKeyRole, role)
}

// WithAPIKeyWorkspaceID returns a new context with the API key workspace ID set.
func WithAPIKeyWorkspaceID(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, contextKeyAPIKeyWorkspaceID, workspaceID)
}

// UserIDFromContext extracts the authenticated user's ID from the request context.
// Returns "" if no user is authenticated.
func UserIDFromContext(ctx context.Context) string {
	return ctxutil.UserIDFromContext(ctx)
}

// EmailFromContext extracts the authenticated user's email from the request context.
// Returns an empty string if no user is authenticated.
func EmailFromContext(ctx context.Context) string {
	return ctxutil.EmailFromContext(ctx)
}

// AuthMethodFromContext extracts the authentication method from the request context.
// Returns an empty string if no auth method is set (unauthenticated request).
func AuthMethodFromContext(ctx context.Context) string {
	return ctxutil.AuthMethodFromContext(ctx)
}

// APIKeyRoleFromContext extracts the API key role from the request context.
// Returns empty string if the request was not authenticated via API key.
func APIKeyRoleFromContext(ctx context.Context) entity.APIKeyRole {
	if v, ok := ctx.Value(contextKeyAPIKeyRole).(entity.APIKeyRole); ok {
		return v
	}
	return ""
}

// APIKeyWorkspaceIDFromContext extracts the API key's workspace ID from the request context.
// Returns "" if the request was not authenticated via API key.
func APIKeyWorkspaceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(contextKeyAPIKeyWorkspaceID).(string); ok {
		return v
	}
	return ""
}
