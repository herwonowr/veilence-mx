package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/repo/persistent"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// AuditLogFilters holds the query parameters for filtering audit logs.
type AuditLogFilters struct {
	Action   string
	Resource string
	UserID   uint
	FromDate *time.Time
	ToDate   *time.Time
}

// AuditContext captures the before-state of a resource for audit logging.
// Use CaptureState to snapshot the current state before a mutation, then
// call LogChange to record both the before and after values.
type AuditContext struct {
	service    *Service
	ctx        context.Context
	resource   string
	resourceID uint
	before     map[string]any
}

// Service provides audit logging operations.
type Service struct {
	db *gorm.DB
}

// NewService creates a new audit service.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// LogAction creates an audit log entry, extracting user, org, IP, user-agent,
// and correlation ID from the request context.
func (s *Service) LogAction(ctx context.Context, action, resource string, resourceID uint, details string) {
	userID := auth.UserIDFromContext(ctx)
	orgID := rbac.OrgIDFromContext(ctx)
	correlationID := CorrelationIDFromContext(ctx)

	// Extract IP and User-Agent from the request if available
	var ipAddress, userAgent string
	if r, ok := ctx.Value(httpRequestKey).(*http.Request); ok {
		ipAddress = r.RemoteAddr
		userAgent = r.Header.Get("User-Agent")
	}

	entry := &persistent.AuditLog{
		UserID:        userID,
		OrgID:         orgID,
		Action:        action,
		Resource:      resource,
		ResourceID:    resourceID,
		Details:       details,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		CorrelationID: correlationID,
	}

	if err := s.db.Create(entry).Error; err != nil {
		slog.Error("failed to create audit log",
			"error", err,
			"action", action,
			"resource", resource,
			"resource_id", resourceID,
			"user_id", userID,
			"org_id", orgID,
		)
		return
	}

	slog.Debug("audit log created",
		"audit_id", entry.ID,
		"action", action,
		"resource", resource,
		"resource_id", resourceID,
		"user_id", userID,
		"org_id", orgID,
		"correlation_id", correlationID,
	)
}

// LogAuthEvent logs an authentication-related event (login, logout, failed login).
// This is similar to LogAction but uses the provided userID and email rather than
// extracting from context, since auth events may not have a user in context yet
// (e.g., failed login).
func (s *Service) LogAuthEvent(ctx context.Context, action string, userID uint, details string) {
	correlationID := CorrelationIDFromContext(ctx)

	var ipAddress, userAgent string
	if r, ok := ctx.Value(httpRequestKey).(*http.Request); ok {
		ipAddress = r.RemoteAddr
		userAgent = r.Header.Get("User-Agent")
	}

	entry := &persistent.AuditLog{
		UserID:        userID,
		OrgID:         0, // Auth events are not org-scoped
		Action:        action,
		Resource:      "auth",
		ResourceID:    userID,
		Details:       details,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		CorrelationID: correlationID,
	}

	if err := s.db.Create(entry).Error; err != nil {
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
func (s *Service) CaptureState(ctx context.Context, resource string, resourceID uint, before map[string]any) *AuditContext {
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

// ListAuditLogs returns a paginated list of audit logs for an organization
// with optional filters.
func (s *Service) ListAuditLogs(orgID uint, filters AuditLogFilters, page, limit int) ([]persistent.AuditLog, int64, error) {
	query := s.db.Model(&persistent.AuditLog{}).Where("org_id = ?", orgID)

	if filters.Action != "" {
		query = query.Where("action = ?", filters.Action)
	}
	if filters.Resource != "" {
		query = query.Where("resource = ?", filters.Resource)
	}
	if filters.UserID != 0 {
		query = query.Where("user_id = ?", filters.UserID)
	}
	if filters.FromDate != nil {
		query = query.Where("created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("created_at <= ?", *filters.ToDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("counting audit logs: %w", err)
	}

	var logs []persistent.AuditLog
	err := query.
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("listing audit logs: %w", err)
	}

	return logs, total, nil
}

// GetAuditLog returns a single audit log entry by ID.
func (s *Service) GetAuditLog(id uint) (*persistent.AuditLog, error) {
	var entry persistent.AuditLog
	if err := s.db.First(&entry, id).Error; err != nil {
		return nil, fmt.Errorf("getting audit log: %w", err)
	}
	return &entry, nil
}

// httpRequestKeyType is the context key type for storing the HTTP request.
type httpRequestKeyType string

const httpRequestKey httpRequestKeyType = "http_request"

// WithHTTPRequest returns a new context with the HTTP request stored in it.
// This is used by the audit middleware to capture IP and User-Agent.
func WithHTTPRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, httpRequestKey, r)
}

// RequestCaptureMiddleware is a Chi middleware that stores the HTTP request
// in the context so the audit service can extract IP address and User-Agent.
func RequestCaptureMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithHTTPRequest(r.Context(), r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
