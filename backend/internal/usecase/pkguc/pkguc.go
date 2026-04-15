// Package pkguc implements the business logic for package lifecycle management.
package pkguc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// UseCase implements usecase.PackageService using a PackageRepository
// and usecase.AuditLogger for logging security-relevant actions.
type UseCase struct {
	repo  usecase.PackageRepository
	audit usecase.AuditLogger
}

// New creates a new package UseCase.
func New(repo usecase.PackageRepository, audit usecase.AuditLogger) *UseCase {
	return &UseCase{repo: repo, audit: audit}
}

// ListPackages returns a paginated list of packages for an org with optional filters.
func (uc *UseCase) ListPackages(ctx context.Context, orgID uint, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error) {
	packages, total, err := uc.repo.FindByOrgID(ctx, orgID, page, limit, sortClause, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("PackageUseCase.ListPackages: %w", err)
	}
	return packages, total, nil
}

// GetPackage returns a single package by ID, scoped to an org.
func (uc *UseCase) GetPackage(ctx context.Context, orgID, pkgID uint) (*entity.Package, error) {
	pkg, err := uc.repo.FindByID(ctx, pkgID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("PackageUseCase.GetPackage: %w", err)
	}
	if pkg.OrgID != orgID {
		return nil, entity.ErrNotFound
	}
	return pkg, nil
}

// CreatePackage adds a new manual package to monitoring within the given org.
func (uc *UseCase) CreatePackage(ctx context.Context, orgID uint, name string, ecosystem entity.Ecosystem) (*entity.Package, error) {
	exists, err := uc.repo.ExistsByOrgAndName(ctx, orgID, name, ecosystem)
	if err != nil {
		return nil, fmt.Errorf("PackageUseCase.CreatePackage: checking existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("package already monitored: %w", entity.ErrConflict)
	}

	pkg := &entity.Package{
		OrgID:     orgID,
		Name:      name,
		Ecosystem: ecosystem,
		Source:    entity.PackageSourceManual,
		Status:   entity.PackageStatusActive,
	}

	if err := uc.repo.Create(ctx, pkg); err != nil {
		return nil, fmt.Errorf("PackageUseCase.CreatePackage: %w", err)
	}

	uc.audit.LogAction(ctx, "create", "package", pkg.ID,
		fmt.Sprintf("added %s package %q to monitoring", ecosystem, name))

	return pkg, nil
}

// ImportPackages bulk-imports packages into monitoring for the given org.
func (uc *UseCase) ImportPackages(ctx context.Context, orgID uint, entries []entity.ImportEntry) (*entity.ImportResult, error) {
	result := &entity.ImportResult{}

	for _, entry := range entries {
		exists, err := uc.repo.ExistsByOrgAndName(ctx, orgID, entry.Name, entry.Ecosystem)
		if err != nil {
			slog.Error("failed to check package existence", "name", entry.Name, "error", err)
			result.Errors = append(result.Errors, entity.ImportErrorEntry{Name: entry.Name, Error: "failed to check existence"})
			continue
		}
		if exists {
			result.Skipped++
			continue
		}

		pkg := &entity.Package{
			OrgID:     orgID,
			Name:      entry.Name,
			Ecosystem: entry.Ecosystem,
			Source:    entity.PackageSourceImported,
			Status:   entity.PackageStatusActive,
		}

		if err := uc.repo.Create(ctx, pkg); err != nil {
			slog.Error("failed to import package", "name", entry.Name, "ecosystem", entry.Ecosystem, "error", err)
			result.Errors = append(result.Errors, entity.ImportErrorEntry{Name: entry.Name, Error: "failed to create package"})
			continue
		}
		result.Imported++
	}

	return result, nil
}

// BlockPackage sets a package's status to 'blocked' with a reason.
// Only active packages can be blocked.
func (uc *UseCase) BlockPackage(ctx context.Context, orgID, pkgID uint, reason string) (*entity.Package, error) {
	if err := uc.repo.BlockPackage(ctx, orgID, pkgID, reason); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("blocking package: %w", err)
	}

	// Fetch the updated package to return to the caller
	pkg, err := uc.repo.FindByID(ctx, pkgID)
	if err != nil {
		return nil, fmt.Errorf("fetching blocked package: %w", err)
	}

	detail := fmt.Sprintf("blocked package %q (%s)", pkg.Name, pkg.Ecosystem)
	if reason != "" {
		detail += ": " + reason
	}
	uc.audit.LogAction(ctx, "block", "package", pkgID, detail)

	return pkg, nil
}

// UnblockPackage sets a package's status back to 'active'.
// Only blocked packages can be unblocked.
func (uc *UseCase) UnblockPackage(ctx context.Context, orgID, pkgID uint) (*entity.Package, error) {
	if err := uc.repo.UnblockPackage(ctx, orgID, pkgID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("unblocking package: %w", err)
	}

	pkg, err := uc.repo.FindByID(ctx, pkgID)
	if err != nil {
		return nil, fmt.Errorf("fetching unblocked package: %w", err)
	}

	uc.audit.LogAction(ctx, "unblock", "package", pkgID,
		fmt.Sprintf("unblocked package %q (%s)", pkg.Name, pkg.Ecosystem))

	return pkg, nil
}

// RemovePackage sets a package's status to 'removed'.
// Active or blocked packages can be removed.
func (uc *UseCase) RemovePackage(ctx context.Context, orgID, pkgID uint) error {
	// Fetch the package first so we can log its name
	pkg, err := uc.repo.FindByID(ctx, pkgID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return fmt.Errorf("fetching package: %w", err)
	}

	if err := uc.repo.RemovePackage(ctx, orgID, pkgID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return fmt.Errorf("removing package: %w", err)
	}

	uc.audit.LogAction(ctx, "remove", "package", pkgID,
		fmt.Sprintf("removed package %q (%s) from monitoring", pkg.Name, pkg.Ecosystem))

	return nil
}

// ApprovePackage promotes a suggested package to active monitoring.
func (uc *UseCase) ApprovePackage(ctx context.Context, orgID, pkgID uint) (*entity.Package, error) {
	if err := uc.repo.ApprovePackage(ctx, orgID, pkgID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("PackageUseCase.ApprovePackage: %w", err)
	}

	pkg, err := uc.repo.FindByID(ctx, pkgID)
	if err != nil {
		return nil, fmt.Errorf("PackageUseCase.ApprovePackage: fetching approved package: %w", err)
	}

	uc.audit.LogAction(ctx, "approve", "package", pkgID,
		fmt.Sprintf("approved suggested package %q (%s) for monitoring", pkg.Name, pkg.Ecosystem))

	return pkg, nil
}

// RejectPackage rejects a suggested package, setting its status to removed.
func (uc *UseCase) RejectPackage(ctx context.Context, orgID, pkgID uint) error {
	// Fetch the package first so we can log its name
	pkg, err := uc.repo.FindByID(ctx, pkgID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return fmt.Errorf("PackageUseCase.RejectPackage: fetching package: %w", err)
	}

	if err := uc.repo.RejectPackage(ctx, orgID, pkgID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return fmt.Errorf("PackageUseCase.RejectPackage: %w", err)
	}

	uc.audit.LogAction(ctx, "reject", "package", pkgID,
		fmt.Sprintf("rejected suggested package %q (%s)", pkg.Name, pkg.Ecosystem))

	return nil
}

// BulkApprovePackages approves multiple suggested packages at once.
func (uc *UseCase) BulkApprovePackages(ctx context.Context, orgID uint, pkgIDs []uint) (int, error) {
	count, err := uc.repo.BulkApprovePackages(ctx, orgID, pkgIDs)
	if err != nil {
		return 0, fmt.Errorf("PackageUseCase.BulkApprovePackages: %w", err)
	}

	uc.audit.LogAction(ctx, "bulk_approve", "package", 0,
		fmt.Sprintf("bulk approved %d suggested packages", count))

	return count, nil
}

// ListSuggestions returns a paginated list of suggested packages for an org.
func (uc *UseCase) ListSuggestions(ctx context.Context, orgID uint, page, limit int) ([]entity.Package, int64, error) {
	packages, total, err := uc.repo.FindSuggestionsByOrgID(ctx, orgID, page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("PackageUseCase.ListSuggestions: %w", err)
	}
	return packages, total, nil
}

// ListStalePackages returns active packages with no releases since staleBefore.
func (uc *UseCase) ListStalePackages(ctx context.Context, orgID uint, staleBefore time.Time) ([]entity.Package, error) {
	packages, err := uc.repo.FindStaleByOrgID(ctx, orgID, staleBefore)
	if err != nil {
		return nil, fmt.Errorf("PackageUseCase.ListStalePackages: %w", err)
	}
	return packages, nil
}
