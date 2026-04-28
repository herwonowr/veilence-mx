package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/veilence/veilence-mx/backend/internal/entity"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// AuditLogFilters is an alias for entity.AuditLogFilters.
// Kept for backward compatibility with controller imports.
type AuditLogFilters = entity.AuditLogFilters

// AuditContext captures the before-state of a resource for audit logging.
// Use CaptureState to snapshot the current state before a mutation, then
// call LogChange to record both the before and after values.
type AuditContext struct {
	service    *Service
	ctx        context.Context
	resource   string
	resourceID string
	before     map[string]any
}

// Service provides audit logging operations.
type Service struct {
	repo usecase.AuditLogRepository
}

// NewService creates a new audit service.
func NewService(repo usecase.AuditLogRepository) *Service {
	return &Service{repo: repo}
}

// LogAction creates an audit log entry, extracting user, workspace, IP, user-agent,
// and correlation ID from the request context.
func (s *Service) LogAction(ctx context.Context, action, resource string, resourceID string, details string) {
	userID := auth.UserIDFromContext(ctx)
	workspaceID := rbac.WorkspaceIDFromContext(ctx)
	correlationID := CorrelationIDFromContext(ctx)

	// Extract IP and User-Agent from the context (set by middleware)
	ipAddress := IPAddressFromContext(ctx)
	userAgent := UserAgentFromContext(ctx)

	entry := &entity.AuditLog{
		UserID:        userID,
		WorkspaceID:   workspaceID,
		Action:        action,
		Resource:      resource,
		ResourceID:    resourceID,
		Details:       details,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		CorrelationID: correlationID,
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		slog.Error("failed to create audit log",
			"error", err,
			"action", action,
			"resource", resource,
			"resource_id", resourceID,
			"user_id", userID,
			"workspace_id", workspaceID,
		)
		return
	}

	slog.Debug("audit log created",
		"audit_id", entry.ID,
		"action", action,
		"resource", resource,
		"resource_id", resourceID,
		"user_id", userID,
		"workspace_id", workspaceID,
		"correlation_id", correlationID,
	)
}

// LogAuthEvent logs an authentication-related event (login, logout, failed login).
// This is similar to LogAction but uses the provided userID and email rather than
// extracting from context, since auth events may not have a user in context yet
// (e.g., failed login).
func (s *Service) LogAuthEvent(ctx context.Context, action string, userID string, details string) {
	correlationID := CorrelationIDFromContext(ctx)

	ipAddress := IPAddressFromContext(ctx)
	userAgent := UserAgentFromContext(ctx)

	entry := &entity.AuditLog{
		UserID:        userID,
		WorkspaceID:   "", // Auth events are not workspace-scoped
		Action:        action,
		Resource:      "auth",
		ResourceID:    userID,
		Details:       details,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		CorrelationID: correlationID,
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		slog.Error("failed to create auth audit log",
			"error", err,
			"action", action,
			"user_id", userID,
		)
		return
	}

	slog.Debug("auth audit log created",
		"action", action,
		"user_id", userID,
		"ip", ipAddress,
	)
}

// CaptureState creates an AuditContext by recording the current state of a resource.
// Pass a map of field names to current values. After making changes, call
// auditCtx.LogChange(action, afterValues) to record the diff.
func (s *Service) CaptureState(ctx context.Context, resource string, resourceID string, before map[string]any) *AuditContext {
	return &AuditContext{
		service:    s,
		ctx:        ctx,
		resource:   resource,
		resourceID: resourceID,
		before:     before,
	}
}

// LogChange records an audit log entry with before/after values.
// Only changed fields are included in the details.
func (ac *AuditContext) LogChange(action string, after map[string]any) {
	changes := make(map[string]any)
	for key, beforeVal := range ac.before {
		afterVal, exists := after[key]
		if !exists {
			continue
		}
		beforeStr := fmt.Sprintf("%v", beforeVal)
		afterStr := fmt.Sprintf("%v", afterVal)
		if beforeStr != afterStr {
			changes[key] = map[string]any{
				"before": beforeVal,
				"after":  afterVal,
			}
		}
	}

	if len(changes) == 0 {
		return // No changes to log
	}

	detailsJSON, err := json.Marshal(map[string]any{
		"action":  action,
		"changes": changes,
	})
	if err != nil {
		slog.Error("failed to marshal audit change details", "error", err)
		return
	}

	ac.service.LogAction(ac.ctx, action, ac.resource, ac.resourceID, string(detailsJSON))
}

// ListAuditLogs returns a paginated list of audit logs for a workspace
// with optional filters.
func (s *Service) ListAuditLogs(workspaceID string, filters AuditLogFilters, page, limit int) ([]entity.AuditLog, int64, error) {
	return s.repo.FindByWorkspaceID(context.Background(), workspaceID, filters, page, limit)
}

// GetAuditLog returns a single audit log entry by ID.
func (s *Service) GetAuditLog(id string) (*entity.AuditLog, error) {
	return s.repo.FindByID(context.Background(), id)
}
