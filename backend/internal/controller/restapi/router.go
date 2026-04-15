package restapi

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/veilence/veilence-mx/backend/internal/usecase/audit"
	"github.com/veilence/veilence-mx/backend/internal/usecase/auth"
	v1 "github.com/veilence/veilence-mx/backend/internal/controller/restapi/v1"
	"github.com/veilence/veilence-mx/backend/internal/controller/restapi/middleware"
	"github.com/veilence/veilence-mx/backend/pkg/metrics"
	"github.com/veilence/veilence-mx/backend/internal/usecase/rbac"
)

// NewRouter creates and configures the HTTP router.
func NewRouter(h *v1.Handlers, frontendURL string, authService *auth.Service, rbacService *rbac.Service) chi.Router {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(middleware.SecurityHeadersWithConfig(middleware.SecurityHeadersConfig{
		FrontendURL: frontendURL,
	}))
	r.Use(middleware.BodySizeLimit(middleware.DefaultMaxBodySize))
	r.Use(middleware.Sanitize)
	r.Use(audit.CorrelationMiddleware)
	r.Use(audit.RequestCaptureMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Metrics)
	r.Use(middleware.CORS(frontendURL))

	// Rate limiter group with per-category limits
	rateLimitGroup := middleware.NewRateLimiterGroup(nil) // uses DefaultLimits

	// Global rate limit (API category: 100 req/min)
	r.Use(rateLimitGroup.ForCategory(middleware.CategoryAPI))

	// CSRF protection (applied globally; safe methods get a cookie, state-changing
	// methods require double-submit. API key auth is exempt.)
	r.Use(middleware.CSRF(middleware.CSRFConfig{
		Secure: frontendURL != "" && frontendURL != "http://localhost:3000",
	}))

	// Routes
	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Get("/health", h.Health.HealthCheck)
		r.Get("/ready", h.Health.ReadinessCheck)
		r.Handle("/metrics", metrics.Handler())

		// Public auth routes (no authentication required, stricter rate limit)
		r.Route("/auth", func(r chi.Router) {
			r.Use(rateLimitGroup.ForCategory(middleware.CategoryAuth))
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
			r.Put("/auth/me", h.Auth.UpdateProfile)
			r.Post("/auth/change-password", h.Auth.ChangePassword)
			r.Post("/auth/send-verification", h.Auth.SendVerificationEmail)
			r.Route("/auth/api-keys", func(r chi.Router) {
				r.Post("/", h.Auth.CreateAPIKey)
				r.Get("/", h.Auth.ListAPIKeys)
				r.Delete("/{id}", h.Auth.RevokeAPIKey)
			})

			// Session management
			r.Route("/auth/sessions", func(r chi.Router) {
				r.Get("/", h.Sessions.ListSessions)
				r.Delete("/{id}", h.Sessions.RevokeSession)
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

				// Dashboard (read-only, any org member can view)
				r.Get("/dashboard/stats", h.Dashboard.GetDashboardStats)
				r.Get("/dashboard/recent-releases", h.Dashboard.GetRecentReleases)
				r.Get("/dashboard/charts", h.Dashboard.GetChartData)

				// Packages
				r.Route("/packages", func(r chi.Router) {
					r.With(rbac.RequirePermission(rbacService, "packages", "read")).Get("/", h.Packages.ListPackages)
					r.With(rbac.RequirePermission(rbacService, "packages", "write")).Post("/", h.Packages.CreatePackage)
					r.With(rbac.RequirePermission(rbacService, "packages", "write")).Post("/bulk-import", h.Packages.ImportPackages)

					// Static routes MUST be registered before /{id} to avoid Chi matching
					// "suggestions", "stale", "bulk-approve" as an {id} parameter.
					r.With(rbac.RequirePermission(rbacService, "packages", "read")).Get("/suggestions", h.Packages.ListSuggestions)
					r.With(rbac.RequirePermission(rbacService, "packages", "read")).Get("/stale", h.Packages.ListStalePackages)
					r.With(rbac.RequirePermission(rbacService, "packages", "write")).Post("/bulk-approve", h.Packages.BulkApprovePackages)

					r.Route("/{id}", func(r chi.Router) {
						r.With(rbac.RequirePermission(rbacService, "packages", "read")).Get("/", h.Packages.GetPackage)
						r.With(rbac.RequirePermission(rbacService, "packages", "delete")).Delete("/", h.Packages.DeletePackage)
						r.With(rbac.RequirePermission(rbacService, "packages", "write")).Post("/block", h.Packages.BlockPackage)
						r.With(rbac.RequirePermission(rbacService, "packages", "write")).Post("/unblock", h.Packages.UnblockPackage)
						r.With(rbac.RequirePermission(rbacService, "packages", "write")).Post("/approve", h.Packages.ApprovePackage)
						r.With(rbac.RequirePermission(rbacService, "packages", "write")).Post("/reject", h.Packages.RejectPackage)
						r.With(rbac.RequirePermission(rbacService, "releases", "read")).Get("/releases", h.Packages.ListPackageReleases)
						r.With(rbac.RequirePermission(rbacService, "packages", "read")).Get("/analysis-history", h.Packages.GetAnalysisHistory)
					})
				})

				// Releases
				r.Route("/releases", func(r chi.Router) {
					r.With(rbac.RequirePermission(rbacService, "releases", "read")).Get("/{id}", h.Packages.GetRelease)
					r.With(rbac.RequirePermission(rbacService, "settings", "write")).Post("/{id}/reanalyze", h.Packages.ReanalyzeRelease)
				})

				// Alerts
				r.Route("/alerts", func(r chi.Router) {
					r.With(rbac.RequirePermission(rbacService, "alerts", "read")).Get("/", h.Alerts.ListAlerts)
					r.Route("/{id}", func(r chi.Router) {
						r.With(rbac.RequirePermission(rbacService, "alerts", "read")).Get("/", h.Alerts.GetAlert)
						r.With(rbac.RequirePermission(rbacService, "alerts", "write")).Patch("/", h.Alerts.UpdateAlert)
						r.With(rbac.RequirePermission(rbacService, "alerts", "read")).Get("/notes", h.Alerts.ListAlertNotes)
						r.With(rbac.RequirePermission(rbacService, "alerts", "write")).Post("/notes", h.Alerts.CreateAlertNote)
						r.With(rbac.RequirePermission(rbacService, "alerts", "write")).Put("/notes/{noteId}", h.Alerts.UpdateAlertNote)
						r.With(rbac.RequirePermission(rbacService, "alerts", "write")).Delete("/notes/{noteId}", h.Alerts.DeleteAlertNote)
					})
				})

				// Settings
				r.With(rbac.RequirePermission(rbacService, "settings", "read")).Get("/settings", h.Settings.GetSettings)
				r.With(rbac.RequirePermission(rbacService, "settings", "write")).Put("/settings", h.Settings.UpdateSettings)

				// Sync triggers (stricter rate limit + write permission)
				r.Group(func(r chi.Router) {
					r.Use(rateLimitGroup.ForCategory(middleware.CategorySync))
					r.With(rbac.RequirePermission(rbacService, "settings", "write")).Post("/sync/discover", h.Settings.DiscoverPackages)
					r.With(rbac.RequirePermission(rbacService, "settings", "write")).Post("/sync/top-packages", h.Settings.SyncTopPackages) // Deprecated alias
					r.With(rbac.RequirePermission(rbacService, "settings", "write")).Post("/sync/reanalyze", h.Dashboard.ReanalyzeAll)
				})

				// Queue monitoring (global data, but requires org membership)
				r.With(rbac.RequirePermission(rbacService, "settings", "read")).Get("/queue/stats", h.Queue.GetQueueStats)
				r.With(rbac.RequirePermission(rbacService, "settings", "read")).Get("/queue/jobs", h.Queue.GetQueueJobs)
				r.With(rbac.RequirePermission(rbacService, "settings", "read")).Get("/queue/dead", h.Queue.GetDeadJobs) // Deprecated: use GET /queue/jobs?status=dead
				r.With(rbac.RequirePermission(rbacService, "settings", "write")).Post("/queue/retry-dead", h.Queue.RetryDeadJobs)
				r.With(rbac.RequirePermission(rbacService, "settings", "write")).Post("/queue/dead/{jobId}/retry", h.Queue.RetryDeadJob)
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
						r.Route("/{id}", func(r chi.Router) {
							r.With(rbac.RequirePermission(rbacService, "notifications", "update")).Put("/", h.Notifications.UpdateNotificationChannel)
							r.With(rbac.RequirePermission(rbacService, "notifications", "delete")).Delete("/", h.Notifications.DeleteNotificationChannel)
							r.With(rbac.RequirePermission(rbacService, "notifications", "update")).Post("/test", h.Notifications.TestNotificationChannel)
						})
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
