package api

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/veilence/veilence-mx/backend/internal/audit"
	"github.com/veilence/veilence-mx/backend/internal/auth"
	"github.com/veilence/veilence-mx/backend/internal/api/handlers"
	"github.com/veilence/veilence-mx/backend/internal/api/middleware"
	"github.com/veilence/veilence-mx/backend/internal/metrics"
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
	r.Use(middleware.Metrics)
	r.Use(middleware.CORS(frontendURL))

	// Rate limiters
	defaultLimiter := middleware.NewRateLimiter(100)  // 100 req/min
	authLimiter := middleware.NewRateLimiter(10)       // 10 req/min for auth endpoints

	// Global rate limit
	r.Use(defaultLimiter.Limit)

	// Routes
	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Get("/health", h.Health.HealthCheck)
		r.Get("/ready", h.Health.ReadinessCheck)
		r.Handle("/metrics", metrics.Handler())

		// Public auth routes (no authentication required, stricter rate limit)
		r.Route("/auth", func(r chi.Router) {
			r.Use(authLimiter.Limit)
			r.Post("/register", h.Auth.Register)
			r.Post("/login", h.Auth.Login)
			r.Post("/refresh", h.Auth.RefreshToken)
			r.Post("/forgot-password", h.Auth.ForgotPassword)
			r.Post("/reset-password", h.Auth.ResetPassword)
			r.Post("/verify-email", h.Auth.VerifyEmail)
		})

		// Public invitation info (no auth required, so frontend can show
		// "you've been invited to X" before the user logs in)
		r.Get("/invitations/{token}", h.Org.GetInvitationInfo)

		// Protected routes (authentication required)
		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(authService))

			// Protected auth routes
			r.Post("/auth/logout", h.Auth.Logout)
			r.Get("/auth/me", h.Auth.GetMe)
			r.Post("/auth/send-verification", h.Auth.SendVerificationEmail)
			r.Route("/auth/api-keys", func(r chi.Router) {
				r.Post("/", h.Auth.CreateAPIKey)
				r.Get("/", h.Auth.ListAPIKeys)
				r.Delete("/{id}", h.Auth.RevokeAPIKey)
			})

			// User notifications (not org-scoped, across all orgs)
			r.Get("/notifications", h.Notifications.ListUserNotifications)
			r.Get("/notifications/unread-count", h.Notifications.GetUnreadCount)
			r.Put("/notifications/read-all", h.Notifications.MarkAllNotificationsRead)
			r.Put("/notifications/{id}/read", h.Notifications.MarkNotificationRead)

			// Permissions (global, not org-scoped)
			r.Get("/permissions", h.Org.ListPermissions)

			// Org-scoped flat routes (org ID from X-Org-ID header or org_id query param)
			r.Group(func(r chi.Router) {
				r.Use(rbac.RequireOrg(rbacService))

				// Dashboard
				r.Get("/dashboard/stats", h.Dashboard.GetDashboardStats)
				r.Get("/dashboard/recent-releases", h.Dashboard.GetRecentReleases)
				r.Get("/dashboard/charts", h.Dashboard.GetChartData)

				// Packages
				r.Route("/packages", func(r chi.Router) {
					r.Get("/", h.Packages.ListPackages)
					r.Post("/", h.Packages.CreatePackage)
					r.Route("/{id}", func(r chi.Router) {
						r.Get("/", h.Packages.GetPackage)
						r.Delete("/", h.Packages.DeletePackage)
						r.Get("/releases", h.Packages.ListPackageReleases)
					})
				})

				// Releases
				r.Route("/releases", func(r chi.Router) {
					r.Get("/{id}", h.Packages.GetRelease)
				})

				// Alerts
				r.Route("/alerts", func(r chi.Router) {
					r.Get("/", h.Alerts.ListAlerts)
					r.Patch("/{id}", h.Alerts.UpdateAlert)
				})

				// Settings
				r.Get("/settings", h.Settings.GetSettings)
				r.Put("/settings", h.Settings.UpdateSettings)

				// Sync triggers
				r.Post("/sync/top-packages", h.Settings.SyncTopPackages)
				r.Post("/sync/reanalyze", h.Dashboard.ReanalyzeAll)

				// Queue monitoring (global data, but requires org membership)
				r.Get("/queue/stats", h.Queue.GetQueueStats)
				r.Get("/queue/dead", h.Queue.GetDeadJobs)
				r.Post("/queue/retry-dead", h.Queue.RetryDeadJobs)
			})

			// Organization routes
			r.Route("/orgs", func(r chi.Router) {
				r.Post("/", h.Org.CreateOrganization)
				r.Get("/", h.Org.ListOrganizations)

				// Invitation acceptance (requires auth but not org membership)
				r.Post("/{orgId}/invitations/{token}/accept", h.Org.AcceptInvitation)

				// Org-scoped routes (require membership + permissions)
				r.Route("/{orgId}", func(r chi.Router) {
					r.Use(rbac.RequireOrg(rbacService))

					r.With(rbac.RequirePermission(rbacService, "org", "read")).Get("/", h.Org.GetOrganization)
					r.With(rbac.RequirePermission(rbacService, "org", "write")).Put("/", h.Org.UpdateOrganization)
					r.With(rbac.RequirePermission(rbacService, "org", "delete")).Delete("/", h.Org.DeleteOrganization)

					// Members
					r.With(rbac.RequirePermission(rbacService, "members", "read")).Get("/members", h.Org.ListMembers)
					r.With(rbac.RequirePermission(rbacService, "members", "invite")).Post("/invitations", h.Org.InviteMember)
					r.With(rbac.RequirePermission(rbacService, "members", "read")).Get("/invitations", h.Org.ListPendingInvitations)
					r.With(rbac.RequirePermission(rbacService, "members", "invite")).Delete("/invitations/{id}", h.Org.RevokeInvitation)
					r.With(rbac.RequirePermission(rbacService, "members", "remove")).Delete("/members/{userId}", h.Org.RemoveMember)
					r.With(rbac.RequirePermission(rbacService, "members", "remove")).Put("/members/{userId}/role", h.Org.UpdateMemberRole)

					// Roles
					r.With(rbac.RequirePermission(rbacService, "roles", "read")).Get("/roles", h.Org.ListRoles)

					// Audit logs
					r.With(rbac.RequirePermission(rbacService, "audit", "read")).Get("/audit-logs", h.AuditLogs.ListAuditLogs)

					// Notification channels
					r.Route("/notification-channels", func(r chi.Router) {
						r.With(rbac.RequirePermission(rbacService, "notifications", "read")).Get("/", h.Notifications.ListNotificationChannels)
						r.With(rbac.RequirePermission(rbacService, "notifications", "create")).Post("/", h.Notifications.CreateNotificationChannel)
						r.With(rbac.RequirePermission(rbacService, "notifications", "update")).Put("/{id}", h.Notifications.UpdateNotificationChannel)
						r.With(rbac.RequirePermission(rbacService, "notifications", "delete")).Delete("/{id}", h.Notifications.DeleteNotificationChannel)
					})

					// Notification rules
					r.Route("/notification-rules", func(r chi.Router) {
						r.With(rbac.RequirePermission(rbacService, "notifications", "read")).Get("/", h.Notifications.ListNotificationRules)
						r.With(rbac.RequirePermission(rbacService, "notifications", "create")).Post("/", h.Notifications.CreateNotificationRule)
						r.With(rbac.RequirePermission(rbacService, "notifications", "delete")).Delete("/{id}", h.Notifications.DeleteNotificationRule)
					})
				})
			})
		})
	})

	return r
}
