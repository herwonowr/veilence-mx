package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/domain"
)

// PackageService implements domain.PackageService using a PackageRepository
// and audit.Service for logging security-relevant actions.
type PackageService struct {
	repo  domain.PackageRepository
	audit *audit.Service
}

// NewPackageService creates a new PackageService.
func NewPackageService(repo domain.PackageRepository, audit *audit.Service) *PackageService {
	return &PackageService{repo: repo, audit: audit}
}

// BlockPackage sets a package's status to 'blocked' with a reason.
// Only active packages can be blocked.
func (s *PackageService) BlockPackage(ctx context.Context, orgID, pkgID uint, reason string) (*domain.Package, error) {
	if err := s.repo.BlockPackage(ctx, orgID, pkgID, reason); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("package %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("blocking package: %w", err)
	}

	// Fetch the updated package to return to the caller
	pkg, err := s.repo.FindByID(ctx, pkgID)
	if err != nil {
		return nil, fmt.Errorf("fetching blocked package: %w", err)
	}

	detail := fmt.Sprintf("blocked package %q (%s)", pkg.Name, pkg.Ecosystem)
	if reason != "" {
		detail += ": " + reason
	}
	s.audit.LogAction(ctx, "block", "package", pkgID, detail)

	return pkg, nil
}

// UnblockPackage sets a package's status back to 'active'.
// Only blocked packages can be unblocked.
func (s *PackageService) UnblockPackage(ctx context.Context, orgID, pkgID uint) (*domain.Package, error) {
	if err := s.repo.UnblockPackage(ctx, orgID, pkgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("package %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("unblocking package: %w", err)
	}

	pkg, err := s.repo.FindByID(ctx, pkgID)
	if err != nil {
		return nil, fmt.Errorf("fetching unblocked package: %w", err)
	}

	s.audit.LogAction(ctx, "unblock", "package", pkgID,
		fmt.Sprintf("unblocked package %q (%s)", pkg.Name, pkg.Ecosystem))

	return pkg, nil
}

// RemovePackage sets a package's status to 'removed'.
// Active or blocked packages can be removed.
func (s *PackageService) RemovePackage(ctx context.Context, orgID, pkgID uint) error {
	// Fetch the package first so we can log its name
	pkg, err := s.repo.FindByID(ctx, pkgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("package %w", domain.ErrNotFound)
		}
		return fmt.Errorf("fetching package: %w", err)
	}

	if err := s.repo.RemovePackage(ctx, orgID, pkgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("package %w", domain.ErrNotFound)
		}
		return fmt.Errorf("removing package: %w", err)
	}

	s.audit.LogAction(ctx, "remove", "package", pkgID,
		fmt.Sprintf("removed package %q (%s) from monitoring", pkg.Name, pkg.Ecosystem))

	return nil
}
