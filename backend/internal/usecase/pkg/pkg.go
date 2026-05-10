// Package pkguc implements the business logic for package lifecycle management.
package pkg

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
	repo              usecase.PackageRepository
	audit             usecase.AuditLogger
	registries        map[entity.Ecosystem]usecase.Registry
	monitoringTrigger usecase.MonitoringTrigger
}

// New creates a new package UseCase.
func New(repo usecase.PackageRepository, audit usecase.AuditLogger, registries map[entity.Ecosystem]usecase.Registry, monitoringTrigger usecase.MonitoringTrigger) *UseCase {
	return &UseCase{repo: repo, audit: audit, registries: registries, monitoringTrigger: monitoringTrigger}
}

// validatePackageExists checks that a package exists on its ecosystem registry.
func (uc *UseCase) validatePackageExists(ctx context.Context, name string, ecosystem entity.Ecosystem) error {
	if name == "" || len(name) > 200 {
		return &entity.ValidationError{Message: "invalid package name"}
	}
	reg, ok := uc.registries[ecosystem]
	if !ok {
		return nil
	}
	_, err := reg.GetPackage(ctx, name)
	if err != nil {
		return &entity.ValidationError{Message: fmt.Sprintf("package %q not found on %s registry", name, ecosystem)}
	}
	return nil
}

// ListPackages returns a paginated list of packages for a workspace with optional filters.
func (uc *UseCase) ListPackages(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error) {
	packages, total, err := uc.repo.FindByWorkspaceID(ctx, workspaceID, page, limit, sortClause, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("%w", err)
	}
	return packages, total, nil
}

// GetPackage returns a single package by ID, scoped to a workspace.
func (uc *UseCase) GetPackage(ctx context.Context, workspaceID, pkgID string) (*entity.Package, error) {
	pkg, err := uc.repo.FindByIDAndWorkspaceID(ctx, pkgID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("%w", err)
	}
	return pkg, nil
}

// CreatePackage adds a new manual package to monitoring within the given workspace.
func (uc *UseCase) CreatePackage(ctx context.Context, workspaceID string, name string, ecosystem entity.Ecosystem) (*entity.Package, error) {
	if err := uc.validatePackageExists(ctx, name, ecosystem); err != nil {
		return nil, err
	}

	exists, err := uc.repo.ExistsByWorkspaceAndName(ctx, workspaceID, name, ecosystem)
	if err != nil {
		return nil, fmt.Errorf("checking existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("package already monitored: %w", entity.ErrConflict)
	}

	pkg := &entity.Package{
		WorkspaceID: workspaceID,
		Name:        name,
		Ecosystem:   ecosystem,
		Source:      entity.PackageSourceManual,
		Status:      entity.PackageStatusActive,
	}

	if err := uc.repo.Create(ctx, pkg); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	uc.audit.LogAction(ctx, "create", "package", pkg.ID,
		fmt.Sprintf("added %s package %q to monitoring", ecosystem, name))

	return pkg, nil
}

// ImportPackages bulk-imports packages into monitoring for the given org.
func (uc *UseCase) ImportPackages(ctx context.Context, workspaceID string, entries []entity.ImportEntry) (*entity.ImportResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	result := &entity.ImportResult{}

	for _, entry := range entries {
		exists, err := uc.repo.ExistsByWorkspaceAndName(ctx, workspaceID, entry.Name, entry.Ecosystem)
		if err != nil {
			slog.Error("failed to check package existence", "name", entry.Name, "error", err)
			result.Errors = append(result.Errors, entity.ImportErrorEntry{Name: entry.Name, Error: "failed to check existence"})
			continue
		}
		if exists {
			result.Skipped++
			continue
		}

		if err := uc.validatePackageExists(ctx, entry.Name, entry.Ecosystem); err != nil {
			result.Errors = append(result.Errors, entity.ImportErrorEntry{Name: entry.Name, Error: fmt.Sprintf("not found on %s registry", entry.Ecosystem)})
			continue
		}

		pkg := &entity.Package{
			WorkspaceID: workspaceID,
			Name:        entry.Name,
			Ecosystem:   entry.Ecosystem,
			Source:      entity.PackageSourceImported,
			Status:      entity.PackageStatusActive,
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
func (uc *UseCase) BlockPackage(ctx context.Context, workspaceID, pkgID string, reason string) (*entity.Package, error) {
	if err := uc.repo.BlockPackage(ctx, workspaceID, pkgID, reason); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("blocking package: %w", err)
	}

	// Fetch the updated package to return to the caller
	pkg, err := uc.repo.FindByIDAndWorkspaceID(ctx, pkgID, workspaceID)
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
func (uc *UseCase) UnblockPackage(ctx context.Context, workspaceID, pkgID string) (*entity.Package, error) {
	if err := uc.repo.UnblockPackage(ctx, workspaceID, pkgID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("unblocking package: %w", err)
	}

	pkg, err := uc.repo.FindByIDAndWorkspaceID(ctx, pkgID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("fetching unblocked package: %w", err)
	}

	uc.audit.LogAction(ctx, "unblock", "package", pkgID,
		fmt.Sprintf("unblocked package %q (%s)", pkg.Name, pkg.Ecosystem))

	return pkg, nil
}

// RemovePackage sets a package's status to 'removed'.
// Active or blocked packages can be removed.
func (uc *UseCase) RemovePackage(ctx context.Context, workspaceID, pkgID string) error {
	// Fetch the package first so we can log its name
	pkg, err := uc.repo.FindByIDAndWorkspaceID(ctx, pkgID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return fmt.Errorf("fetching package: %w", err)
	}

	if err := uc.repo.RemovePackage(ctx, workspaceID, pkgID); err != nil {
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
func (uc *UseCase) ApprovePackage(ctx context.Context, workspaceID, pkgID string) (*entity.Package, error) {
	if err := uc.repo.ApprovePackage(ctx, workspaceID, pkgID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return nil, fmt.Errorf("%w", err)
	}

	pkg, err := uc.repo.FindByIDAndWorkspaceID(ctx, pkgID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("fetching approved package: %w", err)
	}

	uc.audit.LogAction(ctx, "approve", "package", pkgID,
		fmt.Sprintf("approved suggested package %q (%s) for monitoring", pkg.Name, pkg.Ecosystem))

	if uc.monitoringTrigger != nil {
		uc.monitoringTrigger.TriggerMonitoring(workspaceID)
	}

	return pkg, nil
}

// RejectPackage rejects a suggested package, setting its status to removed.
func (uc *UseCase) RejectPackage(ctx context.Context, workspaceID, pkgID string) error {
	// Fetch the package first so we can log its name
	pkg, err := uc.repo.FindByIDAndWorkspaceID(ctx, pkgID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return fmt.Errorf("fetching package: %w", err)
	}

	if err := uc.repo.RejectPackage(ctx, workspaceID, pkgID); err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return fmt.Errorf("package %w", entity.ErrNotFound)
		}
		return fmt.Errorf("%w", err)
	}

	uc.audit.LogAction(ctx, "reject", "package", pkgID,
		fmt.Sprintf("rejected suggested package %q (%s)", pkg.Name, pkg.Ecosystem))

	return nil
}

// BulkApprovePackages approves multiple suggested packages at once.
func (uc *UseCase) BulkApprovePackages(ctx context.Context, workspaceID string, pkgIDs []string) (int, error) {
	count, err := uc.repo.BulkApprovePackages(ctx, workspaceID, pkgIDs)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	uc.audit.LogAction(ctx, "bulk_approve", "package", "",
		fmt.Sprintf("bulk approved %d suggested packages", count))

	if count > 0 && uc.monitoringTrigger != nil {
		uc.monitoringTrigger.TriggerMonitoring(workspaceID)
	}

	return count, nil
}

// BulkApproveAllSuggestions approves all pending suggestions for a workspace.
func (uc *UseCase) BulkApproveAllSuggestions(ctx context.Context, workspaceID string) (int, error) {
	count, err := uc.repo.BulkApproveAllSuggestions(ctx, workspaceID)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	uc.audit.LogAction(ctx, "bulk_approve_all", "package", "",
		fmt.Sprintf("bulk approved all %d suggested packages", count))

	if count > 0 && uc.monitoringTrigger != nil {
		uc.monitoringTrigger.TriggerMonitoring(workspaceID)
	}

	return count, nil
}

// ListSuggestions returns a paginated list of suggested packages for a workspace.
func (uc *UseCase) ListSuggestions(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.PackageFilters) ([]entity.Package, int64, error) {
	packages, total, err := uc.repo.FindSuggestionsByWorkspaceID(ctx, workspaceID, page, limit, sortClause, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("%w", err)
	}
	return packages, total, nil
}

// ListStalePackages returns active packages with no releases since staleBefore.
func (uc *UseCase) ListStalePackages(ctx context.Context, workspaceID string, staleBefore time.Time) ([]entity.Package, error) {
	packages, err := uc.repo.FindStaleByWorkspaceID(ctx, workspaceID, staleBefore)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	return packages, nil
}

// CountPackages returns the total number of packages in a workspace (all ecosystems).
func (uc *UseCase) CountPackages(ctx context.Context, workspaceID string) (int64, error) {
	count, err := uc.repo.CountByWorkspace(ctx, workspaceID, nil)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}
	return count, nil
}

// RemoveStalePackages removes active packages that have had no updates for
// the given number of months. Returns the number of packages removed.
// A value of 0 means auto-removal is disabled.
func (uc *UseCase) RemoveStalePackages(ctx context.Context, workspaceID string, months int) (int, error) {
	if months <= 0 {
		return 0, nil
	}

	staleBefore := time.Now().AddDate(0, -months, 0)
	count, err := uc.repo.RemoveStaleByWorkspaceID(ctx, workspaceID, staleBefore)
	if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	if count > 0 {
		uc.audit.LogAction(ctx, "auto_remove_stale", "package", "",
			fmt.Sprintf("auto-removed %d stale packages (no updates in %d months)", count, months))
	}

	return count, nil
}
