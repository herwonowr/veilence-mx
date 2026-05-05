// Package alertuc implements the business logic for alert management.
package alert

import (
	"context"
	"errors"
	"fmt"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// UseCase implements usecase.AlertService.
type UseCase struct {
	alerts usecase.AlertRepository
	audit  usecase.AuditLogger
}

// New creates a new alert UseCase.
func New(alerts usecase.AlertRepository, audit usecase.AuditLogger) *UseCase {
	return &UseCase{alerts: alerts, audit: audit}
}

// ListAlerts returns a paginated list of alerts with package info for the given workspace.
func (uc *UseCase) ListAlerts(ctx context.Context, workspaceID string, page, limit int, sortClause string, filters entity.AlertFilters) ([]entity.AlertWithPackage, int64, error) {
	results, total, err := uc.alerts.FindByWorkspaceIDWithPackage(ctx, workspaceID, page, limit, sortClause, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("%w", err)
	}
	return results, total, nil
}

// GetAlert returns a single alert by ID with package info, scoped to the given workspace.
func (uc *UseCase) GetAlert(ctx context.Context, workspaceID, alertID string) (*entity.Alert, *entity.Package, error) {
	alert, pkg, err := uc.alerts.FindByIDWithPackage(ctx, alertID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, nil, entity.ErrNotFound
		}
		return nil, nil, fmt.Errorf("%w", err)
	}
	return alert, pkg, nil
}

// UpdateAlertStatus updates the status of an alert scoped to the given workspace.
func (uc *UseCase) UpdateAlertStatus(ctx context.Context, workspaceID, alertID string, status entity.AlertStatus) (*entity.Alert, error) {
	alert, err := uc.alerts.FindByIDAndWorkspaceID(ctx, alertID, workspaceID)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil, entity.ErrNotFound
		}
		return nil, fmt.Errorf("finding alert: %w", err)
	}

	if err := alert.ValidateStatusTransition(status); err != nil {
		return nil, err
	}

	if err := uc.alerts.UpdateStatus(ctx, alertID, workspaceID, status); err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	alert.Status = status

	uc.audit.LogAction(ctx, "update", "alert", alertID,
		fmt.Sprintf("updated alert status to %q", status))

	return alert, nil
}
