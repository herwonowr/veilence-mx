package v1

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	"github.com/veilence/veilence-mx/backend/internal/usecase/notifications"
	"github.com/veilence/veilence-mx/backend/internal/usecase/poller"
	"github.com/veilence/veilence-mx/backend/pkg/queue"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
	"github.com/veilence/veilence-mx/backend/internal/usecase/setup"
	"github.com/veilence/veilence-mx/backend/internal/usecase"
)

// Handlers aggregates all handler groups and exposes them to the router.
// It exists as a single struct to simplify router wiring; individual handler
// groups depend only on the services they need.
type Handlers struct {
	Auth          *AuthHandlers
	Sessions      *SessionHandlers
	Notifications *NotificationHandlers
	Workspace     *WorkspaceHandlers
	AuditLogs     *AuditHandlers
	Packages      *PackageHandlers
	Alerts        *AlertHandlers
	Settings      *SettingsHandlers
	Dashboard     *DashboardHandlers
	Queue         *QueueHandlers
	Health        *HealthHandlers
	Setup         *SetupHandlers
	Config        *ConfigHandlers
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

// WorkspaceHandlers handles workspace, member, and role management endpoints.
type WorkspaceHandlers struct {
	RBAC   *rbac.Service
	Audit  *audit.Service
	PkgSvc usecase.PackageService
}

// AuditHandlers handles audit log listing endpoints.
type AuditHandlers struct {
	Audit *audit.Service
}

// PackageHandlers handles package and release CRUD endpoints.
// All operations delegate to usecase services (clean architecture).
type PackageHandlers struct {
	PkgSvc     usecase.PackageService
	ReleaseSvc usecase.ReleaseService
	Audit      *audit.Service
}

// AlertHandlers handles alert listing, status update, and note endpoints.
type AlertHandlers struct {
	AlertSvc usecase.AlertService
	Notes    usecase.AlertNoteService
	Audit    *audit.Service
}

// SettingsHandlers handles settings CRUD and sync trigger endpoints.
type SettingsHandlers struct {
	SettingSvc usecase.SettingService
	Poller     *poller.Poller
	Python     usecase.Registry
	NPM        usecase.Registry
	Audit      *audit.Service
}

// DashboardHandlers handles dashboard statistics, recent releases, charts, and reanalysis endpoints.
type DashboardHandlers struct {
	DashboardSvc usecase.DashboardService
}

// QueueHandlers handles queue monitoring endpoints.
type QueueHandlers struct {
	Queue *queue.Queue
}

// HealthHandlers handles the health check endpoint.
type HealthHandlers struct {
	HealthSvc usecase.HealthService
}

// NewHandlers creates a Handlers aggregate from individual dependencies.
func NewHandlers(
	authService *auth.Service,
	rbacService *rbac.Service,
	auditService *audit.Service,
	notificationService *notifications.Service,
	pollerService *poller.Poller,
	pythonClient usecase.Registry,
	npmClient usecase.Registry,
	jobQueue *queue.Queue,
	alertNoteService usecase.AlertNoteService,
	packageService usecase.PackageService,
	alertService usecase.AlertService,
	releaseService usecase.ReleaseService,
	settingService usecase.SettingService,
	dashboardService usecase.DashboardService,
	healthService usecase.HealthService,
	setupService *setup.Service,
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
		Workspace: &WorkspaceHandlers{
			RBAC:   rbacService,
			Audit:  auditService,
			PkgSvc: packageService,
		},
		AuditLogs: &AuditHandlers{
			Audit: auditService,
		},
		Packages: &PackageHandlers{
			PkgSvc:     packageService,
			ReleaseSvc: releaseService,
			Audit:      auditService,
		},
		Alerts: &AlertHandlers{
			AlertSvc: alertService,
			Notes:    alertNoteService,
			Audit:    auditService,
		},
		Settings: &SettingsHandlers{
			SettingSvc: settingService,
			Poller:     pollerService,
			Python:     pythonClient,
			NPM:        npmClient,
			Audit:      auditService,
		},
		Dashboard: &DashboardHandlers{
			DashboardSvc: dashboardService,
		},
		Queue: &QueueHandlers{
			Queue: jobQueue,
		},
		Health: &HealthHandlers{
			HealthSvc: healthService,
		},
		Setup: &SetupHandlers{
			Setup: setupService,
		},
		Config: &ConfigHandlers{
			Setup: setupService,
			Auth:  authService,
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
func respondAppError(w http.ResponseWriter, err *Error) {
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

// parseUUID extracts a URL parameter by name and validates it as a UUID.
// Returns the UUID string and true on success, or empty string and false on failure.
func parseUUID(r *http.Request, param string) (string, bool) {
	raw := chi.URLParam(r, param)
	if _, err := uuid.Parse(raw); err != nil {
		return "", false
	}
	return raw, true
}
