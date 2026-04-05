package api

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/api/handlers"
	"github.com/veilence/veilence-mx/backend/internal/api/middleware"
	"github.com/veilence/veilence-mx/backend/internal/rbac"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(h *handlers.Handlers, frontendURL string, authService *auth.Service, rbacService *rbac.Service) chi.Router {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.BodySizeLimit(middleware.DefaultMaxBodySize))
	r.Use(audit.CorrelationMiddleware)
	r.Use(audit.RequestCaptureMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS(frontendURL))

	// Rate limiters
	defaultLimiter := middleware.NewRateLimiter(100)  // 100 req/min
	authLimiter := middleware.NewRateLimiter(10)       // 10 req/min for auth endpoints

	// Global rate limit
	r.Use(defaultLimiter.Limit)

	// Routes
	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Get("/health", h.HealthCheck)

		// Public auth routes (no authentication required, stricter rate limit)
		r.Route("/auth", func(r chi.Router) {
			r.Use(authLimiter.Limit)
			r.Post("/register", h.Register)
			r.Post("/login", h.Login)
			r.Post("/refresh", h.RefreshToken)
		})

		// Protected routes (authentication required)
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(authService))

			// Protected auth routes
			r.Post("/auth/logout", h.Logout)
			r.Get("/auth/me", h.GetMe)
			r.Route("/auth/api-keys", func(r chi.Router) {
				r.Post("/", h.CreateAPIKey)
				r.Get("/", h.ListAPIKeys)
				r.Delete("/{id}", h.RevokeAPIKey)
			})

			// User notifications (not org-scoped, across all orgs)
			r.Get("/notifications", h.ListUserNotifications)
			r.Get("/notifications/unread-count", h.GetUnreadCount)
			r.Put("/notifications/{id}/read", h.MarkNotificationRead)

			// Permissions (global, not org-scoped)
			r.Get("/permissions", h.ListPermissions)

			// Org-scoped flat routes (org ID from X-Org-ID header or org_id query param)
			r.Group(func(r chi.Router) {
				r.Use(rbac.RequireOrg(rbacService))

				// Dashboard
				r.Get("/dashboard/stats", h.GetDashboardStats)
				r.Get("/dashboard/recent-releases", h.GetRecentReleases)
				r.Get("/dashboard/charts", h.GetChartData)

				// Packages
				r.Route("/packages", func(r chi.Router) {
					r.Get("/", h.ListPackages)
					r.Post("/", h.CreatePackage)
					r.Route("/{id}", func(r chi.Router) {
						r.Get("/", h.GetPackage)
						r.Delete("/", h.DeletePackage)
						r.Get("/releases", h.ListPackageReleases)
					})
				})

				// Releases
				r.Route("/releases", func(r chi.Router) {
					r.Get("/{id}", h.GetRelease)
				})

				// Alerts
				r.Route("/alerts", func(r chi.Router) {
					r.Get("/", h.ListAlerts)
					r.Patch("/{id}", h.UpdateAlert)
				})

				// Settings
				r.Get("/settings", h.GetSettings)
				r.Put("/settings", h.UpdateSettings)

				// Sync triggers
				r.Post("/sync/top-packages", h.SyncTopPackages)
				r.Post("/sync/reanalyze", h.ReanalyzeAll)

				// Queue monitoring (global data, but requires org membership)
				r.Get("/queue/stats", h.GetQueueStats)
				r.Get("/queue/dead", h.GetDeadJobs)
				r.Post("/queue/retry-dead", h.RetryDeadJobs)
			})

			// Organization routes
			r.Route("/orgs", func(r chi.Router) {
				r.Post("/", h.CreateOrganization)
				r.Get("/", h.ListOrganizations)

				// Invitation acceptance (requires auth but not org membership)
				r.Post("/{orgId}/invitations/{token}/accept", h.AcceptInvitation)

				// Org-scoped routes (require membership + permissions)
				r.Route("/{orgId}", func(r chi.Router) {
					r.Use(rbac.RequireOrg(rbacService))

					r.With(rbac.RequirePermission(rbacService, "org", "read")).Get("/", h.GetOrganization)
					r.With(rbac.RequirePermission(rbacService, "org", "write")).Put("/", h.UpdateOrganization)
					r.With(rbac.RequirePermission(rbacService, "org", "delete")).Delete("/", h.DeleteOrganization)

					// Members
					r.With(rbac.RequirePermission(rbacService, "members", "read")).Get("/members", h.ListMembers)
					r.With(rbac.RequirePermission(rbacService, "members", "invite")).Post("/invitations", h.InviteMember)
					r.With(rbac.RequirePermission(rbacService, "members", "remove")).Delete("/members/{userId}", h.RemoveMember)
					r.With(rbac.RequirePermission(rbacService, "members", "remove")).Put("/members/{userId}/role", h.UpdateMemberRole)

					// Roles
					r.With(rbac.RequirePermission(rbacService, "roles", "read")).Get("/roles", h.ListRoles)

					// Audit logs
					r.With(rbac.RequirePermission(rbacService, "audit", "read")).Get("/audit-logs", h.ListAuditLogs)

					// Notification channels
					r.Route("/notification-channels", func(r chi.Router) {
						r.With(rbac.RequirePermission(rbacService, "notifications", "read")).Get("/", h.ListNotificationChannels)
						r.With(rbac.RequirePermission(rbacService, "notifications", "create")).Post("/", h.CreateNotificationChannel)
						r.With(rbac.RequirePermission(rbacService, "notifications", "update")).Put("/{id}", h.UpdateNotificationChannel)
						r.With(rbac.RequirePermission(rbacService, "notifications", "delete")).Delete("/{id}", h.DeleteNotificationChannel)
					})

					// Notification rules
					r.Route("/notification-rules", func(r chi.Router) {
						r.With(rbac.RequirePermission(rbacService, "notifications", "read")).Get("/", h.ListNotificationRules)
						r.With(rbac.RequirePermission(rbacService, "notifications", "create")).Post("/", h.CreateNotificationRule)
						r.With(rbac.RequirePermission(rbacService, "notifications", "delete")).Delete("/{id}", h.DeleteNotificationRule)
					})
				})
			})
		})
	})

	return r
}
