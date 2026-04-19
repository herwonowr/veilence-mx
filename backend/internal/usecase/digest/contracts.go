package digest

import "github.com/veilence/veilence-mx/backend/internal/usecase"

// DigestRepository is a type alias for the interface defined in the shared
// usecase package. This avoids import cycles: repo/persistent can import
// usecase (not usecase/digest) to reference the interface.
type DigestRepository = usecase.DigestRepository
