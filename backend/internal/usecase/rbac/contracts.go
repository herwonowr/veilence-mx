package rbac

import "github.com/veilence/veilence-mx/backend/internal/usecase"

// RBACRepository is a type alias for the interface defined in the shared
// usecase package. This avoids import cycles: repo/persistent can import
// usecase (not usecase/rbac) to reference the interface.
type RBACRepository = usecase.RBACRepository
