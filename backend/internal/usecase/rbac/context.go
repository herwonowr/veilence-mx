package rbac

import (
	"context"

	"github.com/veilence/veilence-mx/backend/internal/ctxutil"
)

// WithWorkspaceID returns a new context with the workspace ID set.
func WithWorkspaceID(ctx context.Context, workspaceID string) context.Context {
	return ctxutil.WithWorkspaceID(ctx, workspaceID)
}

// WorkspaceIDFromContext extracts the workspace ID from the request context.
// Returns "" if not set.
func WorkspaceIDFromContext(ctx context.Context) string {
	return ctxutil.WorkspaceIDFromContext(ctx)
}

// UserIDFromContext extracts the user ID from the request context.
// This delegates to the ctxutil package's UserIDFromContext to ensure
// compatibility with the auth middleware's context keys.
func UserIDFromContext(ctx context.Context) string {
	return ctxutil.UserIDFromContext(ctx)
}

// WithMemberRole returns a new context with the member's role name set.
func WithMemberRole(ctx context.Context, role string) context.Context {
	return ctxutil.WithMemberRole(ctx, role)
}

// MemberRoleFromContext extracts the member's role name from the request context.
// Returns empty string if not set.
func MemberRoleFromContext(ctx context.Context) string {
	return ctxutil.MemberRoleFromContext(ctx)
}
