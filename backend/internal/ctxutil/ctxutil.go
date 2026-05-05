// Package ctxutil provides shared context key types and accessor functions
// for values stored in request contexts by middleware. This package exists to
// break import cycles between sibling usecase sub-packages (e.g. audit
// importing auth or rbac).
package ctxutil

import (
	"context"
)

type userIDKey struct{}
type emailKey struct{}
type authMethodKey struct{}
type workspaceIDKey struct{}
type memberRoleKey struct{}

// WithUserID returns a new context with the user ID set.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromContext extracts the authenticated user's ID from the request context.
// Returns "" if no user is authenticated.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey{}).(string); ok {
		return v
	}
	return ""
}

// WithEmail returns a new context with the email set.
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey{}, email)
}

// EmailFromContext extracts the authenticated user's email from the request context.
// Returns an empty string if no user is authenticated.
func EmailFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(emailKey{}).(string); ok {
		return v
	}
	return ""
}

// WithAuthMethod returns a new context with the auth method set.
func WithAuthMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, authMethodKey{}, method)
}

// AuthMethodFromContext extracts the authentication method from the request context.
// Returns an empty string if no auth method is set.
func AuthMethodFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(authMethodKey{}).(string); ok {
		return v
	}
	return ""
}

// WithWorkspaceID returns a new context with the workspace ID set.
func WithWorkspaceID(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, workspaceIDKey{}, workspaceID)
}

// WorkspaceIDFromContext extracts the workspace ID from the request context.
// Returns "" if not set.
func WorkspaceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(workspaceIDKey{}).(string); ok {
		return v
	}
	return ""
}

// WithMemberRole returns a new context with the member's role name set.
func WithMemberRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, memberRoleKey{}, role)
}

// MemberRoleFromContext extracts the member's role name from the request context.
// Returns empty string if not set.
func MemberRoleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(memberRoleKey{}).(string); ok {
		return v
	}
	return ""
}
