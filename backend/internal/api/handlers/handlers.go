package handlers

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/veilence/veilence-mx/backend/internal/apperror"
	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/domain"
	"github.com/veilence/veilence-mx/backend/internal/notifications"
	"github.com/veilence/veilence-mx/backend/internal/poller"
	"github.com/veilence/veilence-mx/backend/internal/queue"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
	"github.com/veilence/veilence-mx/backend/internal/registry"
	"gorm.io/gorm"
)

// Handlers aggregates all handler groups and exposes them to the router.
// It exists as a single struct to simplify router wiring; individual handler
// groups depend only on the services they need.
type Handlers struct {
	Auth          *AuthHandlers
	Sessions      *SessionHandlers
	Notifications *NotificationHandlers
	Org           *OrgHandlers
	AuditLogs     *AuditHandlers
	Packages      *PackageHandlers
	Alerts        *AlertHandlers
	Settings      *SettingsHandlers
	Dashboard     *DashboardHandlers
	Queue         *QueueHandlers
	Health        *HealthHandlers
}

// AuthHandlers handles authentication and API key endpoints.
type AuthHandlers struct {
	Auth  *auth.Service
	Audit *audit.Service
}

// NotificationHandlers handles notification channel, rule, and user notification endpoints.
type NotificationHandlers struct {
	Notifications *notifications.Service
	Audit         *audit.Service
}

// OrgHandlers handles organization, member, and role management endpoints.
type OrgHandlers struct {
	RBAC  *rbac.Service
	Audit *audit.Service
}

// AuditHandlers handles audit log listing endpoints.
type AuditHandlers struct {
	Audit *audit.Service
}

// PackageHandlers handles package and release CRUD endpoints.
// NOTE: DB will be replaced by a service interface in a future sprint.
type PackageHandlers struct {
	DB    *gorm.DB
	Queue queue.Enqueuer
	Audit *audit.Service
}

// AlertHandlers handles alert listing and status update endpoints.
// NOTE: DB will be replaced by a service interface in a future sprint.
type AlertHandlers struct {
	DB         *gorm.DB
	AlertNotes domain.AlertNoteRepository
	Audit      *audit.Service
}

// SettingsHandlers handles settings CRUD and sync trigger endpoints.
// NOTE: DB will be replaced by a service interface in a future sprint.
type SettingsHandlers struct {
	DB     *gorm.DB
	Poller *poller.Poller
	Python registry.Registry
	NPM    registry.Registry
	Audit  *audit.Service
}

// DashboardHandlers handles dashboard statistics, recent releases, charts, and reanalysis endpoints.
type DashboardHandlers struct {
	DB        *gorm.DB
	Dashboard domain.DashboardRepository
	Queue     *queue.Queue
}

// QueueHandlers handles queue monitoring endpoints.
type QueueHandlers struct {
	Queue *queue.Queue
}

// HealthHandlers handles the health check endpoint.
type HealthHandlers struct {
	DB    *gorm.DB
	Queue *queue.Queue
}

// NewHandlers creates a Handlers aggregate from individual dependencies.
func NewHandlers(
	db *gorm.DB,
	authService *auth.Service,
	rbacService *rbac.Service,
	auditService *audit.Service,
	notificationService *notifications.Service,
	pollerService *poller.Poller,
	pythonClient registry.Registry,
	npmClient registry.Registry,
	jobQueue *queue.Queue,
	dashboardRepo domain.DashboardRepository,
	alertNoteRepo domain.AlertNoteRepository,
) *Handlers {
	return &Handlers{
		Auth: &AuthHandlers{
			Auth:  authService,
			Audit: auditService,
		},
		Sessions: &SessionHandlers{
			Auth: authService,
		},
		Notifications: &NotificationHandlers{
			Notifications: notificationService,
			Audit:         auditService,
		},
		Org: &OrgHandlers{
			RBAC:  rbacService,
			Audit: auditService,
		},
		AuditLogs: &AuditHandlers{
			Audit: auditService,
		},
		Packages: &PackageHandlers{
			DB:    db,
			Queue: jobQueue,
			Audit: auditService,
		},
		Alerts: &AlertHandlers{
			DB:         db,
			AlertNotes: alertNoteRepo,
			Audit:      auditService,
		},
		Settings: &SettingsHandlers{
			DB:     db,
			Poller: pollerService,
			Python: pythonClient,
			NPM:    npmClient,
			Audit:  auditService,
		},
		Dashboard: &DashboardHandlers{
			DB:        db,
			Dashboard: dashboardRepo,
			Queue:     jobQueue,
		},
		Queue: &QueueHandlers{
			Queue: jobQueue,
		},
		Health: &HealthHandlers{
			DB:    db,
			Queue: jobQueue,
		},
	}
}

// APIResponse is the standard JSON response envelope.
type APIResponse struct {
	Data  any     `json:"data"`
	Error *string `json:"error"`
	Meta  *Meta   `json:"meta,omitempty"`
}

// Meta holds pagination metadata.
type Meta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

func respondJSON(w http.ResponseWriter, status int, data any, meta *Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Data: data, Meta: meta})
}

func respondError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Error: &msg})
}

// respondAppError responds with a structured apperror, using its HTTP status and message.
func respondAppError(w http.ResponseWriter, err *apperror.Error) {
	respondError(w, err.HTTPStatus, err.Message)
}

func parsePagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}

// parseSort extracts sort_by and sort_dir from query params.
// Only columns in the allowedColumns set are accepted to prevent SQL injection.
func parseSort(r *http.Request, allowedColumns map[string]string, defaultSort string) string {
	sortBy := r.URL.Query().Get("sort_by")
	sortDir := r.URL.Query().Get("sort_dir")

	col, ok := allowedColumns[sortBy]
	if !ok {
		return defaultSort
	}

	if !slices.Contains([]string{"asc", "desc"}, sortDir) {
		sortDir = "asc"
	}

	return col + " " + sortDir
}

// escapeLike escapes LIKE special characters (%, _) to prevent
// wildcard injection when building LIKE queries from user input.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}
