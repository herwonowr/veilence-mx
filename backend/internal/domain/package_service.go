package domain

import "context"

// PackageService defines the business logic operations for package lifecycle management.
// New operations (block/unblock/remove) go through this service layer rather than
// accessing the database directly from handlers — clean architecture pattern.
type PackageService interface {
	// BlockPackage sets a package's status to 'blocked' with an optional reason.
	// Returns the updated package. Audit-logged.
	BlockPackage(ctx context.Context, orgID, pkgID uint, reason string) (*Package, error)

	// UnblockPackage sets a package's status back to 'active', clearing block fields.
	// Returns the updated package. Audit-logged.
	UnblockPackage(ctx context.Context, orgID, pkgID uint) (*Package, error)

	// RemovePackage sets a package's status to 'removed'.
	// Audit-logged.
	RemovePackage(ctx context.Context, orgID, pkgID uint) error
}
