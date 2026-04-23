package rbac

import "github.com/veilence/veilence-mx/backend/internal/usecase"

// RBACRepository is a type alias for the interface defined in the shared
// usecase package. This avoids import cycles: repo/persistent can import
// usecase (not usecase/rbac) to reference the interface.
type RBACRepository = usecase.RBACRepository

// InvitationEmailSender is a type alias for the interface defined in the shared
// usecase package. Used by the RBAC service to send invitation emails.
type InvitationEmailSender = usecase.InvitationEmailSender

// UserEmailResolver is a type alias for the interface defined in the shared
// usecase package. Used to resolve user IDs to emails for invitation emails.
type UserEmailResolver = usecase.UserEmailResolver
