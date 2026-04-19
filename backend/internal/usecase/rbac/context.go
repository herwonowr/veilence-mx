package rbac

import (
	"context"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
)

type contextKey string

const (
	// ctxWorkspaceID is the context key for the current workspace ID.
	ctxWorkspaceID contextKey = "workspace_id"

	// ctxMemberRole is the context key for the user's role in the current workspace.
	ctxMemberRole contextKey = "member_role"
)

// WithWorkspaceID returns a new context with the workspace ID set.
func WithWorkspaceID(ctx context.Context, workspaceID uint) context.Context {
	return context.WithValue(ctx, ctxWorkspaceID, workspaceID)
}

// WorkspaceIDFromContext extracts the workspace ID from the request context.
// Returns 0 if not set.
func WorkspaceIDFromContext(ctx context.Context) uint {
	id, _ := ctx.Value(ctxWorkspaceID).(uint)
	return id
}

// UserIDFromContext extracts the user ID from the request context.
// This delegates to the auth package's UserIDFromContext to ensure
// compatibility with the auth middleware's context keys.
func UserIDFromContext(ctx context.Context) uint {
	return auth.UserIDFromContext(ctx)
}

// WithMemberRole returns a new context with the member's role name set.
func WithMemberRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxMemberRole, role)
}

// MemberRoleFromContext extracts the member's role name from the request context.
// Returns empty string if not set.
func MemberRoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(ctxMemberRole).(string)
	return role
}
