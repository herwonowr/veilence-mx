# Changelog

All notable changes to Veilence-MX are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-04-11

### Added

#### Core Pipeline
- **Supply chain monitoring pipeline**: Poller -> Differ -> Analyzer (LLM-powered) -> Alerts -> Dashboard
- **PyPI and npm registry polling** with configurable intervals and concurrency
- **Top-N package sync** for both registries with automatic ranking
- **Custom package monitoring** — add any PyPI or npm package to monitor
- **LLM-powered release analysis** using configurable AI model (default: Claude Sonnet 4.6)
- **Automatic diff generation** between release versions with configurable size limits
- **Version depth control** — analyze latest only or up to 5 most recent versions per package

#### Multi-Tenant Architecture
- **Organization-based multi-tenancy** with full data isolation between orgs
- **Role-Based Access Control (RBAC)** with 4 default roles: Owner, Admin, Member, Viewer
- **Granular permissions** for packages, alerts, settings, members, notifications, audit, org, and roles
- **Invitation system** with hashed tokens, email matching, and expiration
- **Per-org settings** for polling intervals, top-N counts, analysis configuration

#### Authentication & Security
- **JWT authentication** with access/refresh token pair and automatic refresh
- **JWT secret rotation** support via `JWT_SECRET_PREVIOUS`
- **API key authentication** with prefix-based lookup, scoped permissions (read/write/admin), and optional expiration
- **Session management** — list active sessions, revoke individual sessions
- **Password reset flow** with token-based email verification
- **Email verification** with token-based confirmation
- **CSRF protection** via double-submit cookie pattern with constant-time token comparison and automatic rotation
- **Input sanitization middleware** applied globally to all request bodies
- **Security headers** (CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy)
- **Rate limiting** with per-category limits: API (100/min), Auth (10/min), Sync (5/min)
- **Request body size limits** to prevent denial of service
- **CORS configuration** tied to `FRONTEND_URL`

#### Alerts & Notifications
- **Automated alert generation** from LLM analysis with severity classification (critical, high, medium, low)
- **Alert triage workflow** with status transitions: new -> acknowledged -> resolved
- **Email notification channels** with real SMTP support
- **Slack webhook notifications** with formatted alert messages
- **Generic webhook notifications** with HMAC signature verification
- **Notification rules engine** — route alerts to channels based on severity thresholds
- **In-app notification bell** with unread count and mark-all-read

#### Dashboard & Reporting
- **Dashboard statistics** — total packages, releases, alerts by severity, analysis status
- **Recent releases view** with classification filtering and pagination
- **Chart data endpoint** for visualizing trends over time
- **Org-scoped dashboard** — all stats filtered to the current organization

#### Frontend
- **Next.js 16 App Router** with TypeScript strict mode
- **Feature-based architecture** with 8 feature modules: auth, dashboard, packages, alerts, settings, notifications, organizations, queue
- **React Query (TanStack Query)** for server state management with 12+ custom hooks
- **shadcn/ui component library** with TailwindCSS styling
- **Zod form validation** for all user input forms
- **Error boundaries** with fallback UI for 404 and 500 errors
- **Dark/light theme toggle**
- **Responsive sidebar navigation** with mobile support
- **Protected routes** with automatic redirect to login
- **Organization selector** in the sidebar
- **Account page** — profile editing, password change
- **Settings page** — all org-scoped settings in a single form
- **Members management UI** — invite, list, remove members, change roles
- **Notification channels/rules UI** — create, update, delete channels and severity rules
- **Queue monitoring dashboard** — view stats, dead jobs, retry actions
- **Audit log viewer** — paginated, filterable audit trail
- **Package delete confirmation dialog** for destructive actions
- **Search debouncing** on package and release lists (300ms)

#### Infrastructure
- **Docker Compose** configuration with PostgreSQL 16, Redis 7, backend, and frontend services
- **Production Docker config** (`docker-compose.prod.yml`) with resource limits, log rotation, restart policies
- **Multi-stage Dockerfile** for backend with minimal Alpine runtime image
- **Container health checks** for all services (PostgreSQL, Redis, backend)
- **Versioned database migrations** (5 migration files) via golang-migrate with embedded SQL
- **Redis job queue** with exponential backoff retry (30s * 2^n), dead-letter queue, and stuck job recovery
- **Prometheus metrics** endpoint at `/api/metrics`
- **Health check** (`/api/health`) and **readiness probe** (`/api/ready`) endpoints
- **Structured logging** with Go `log/slog` key-value pairs

#### Testing
- **Backend unit tests** for repository and service layers (45%+ coverage)
- **Backend integration tests** for critical API flows
- **Backend benchmark tests** for performance-critical paths
- **Frontend component tests** with Vitest and React Testing Library (30%+ coverage, 27 test files)
- **E2E tests** with Playwright for critical user journeys (login, dashboard, alert status change)
- **CI/CD pipeline** with lint, test, build, and coverage gates
- **Vulnerability scanning** with `govulncheck` (Go) and `npm audit` (Node.js) in CI

#### Documentation
- **OpenAPI 3.1 specification** (`openapi.yaml`) with Swagger UI at `/api/docs` (development mode)
- **Production deployment guide** (`docs/deployment.md`) covering Docker Compose, Kubernetes, TLS, backups
- **Admin runbook** (`docs/runbook.md`) with operational procedures, API examples, troubleshooting
- **Changelog** following Keep a Changelog format

### Changed

#### Architecture Refactors (Sprint 1)
- **Clean Architecture refactor** — introduced domain layer (19 entities), repository layer (23 implementations), and service layer with dependency injection
- **Handler decomposition** — split monolithic handler into 11 focused handler groups: Auth, Sessions, Notifications, Org, AuditLogs, Packages, Alerts, Settings, Dashboard, Queue, Health
- **Auth and notification services** refactored to use repository interfaces instead of direct database access
- **API key validation** optimized with prefix-based database lookup (<10ms)

#### Frontend Architecture Refactors (Sprint 3a)
- **React Query migration** — replaced manual `useState`/`useEffect` data fetching with TanStack Query hooks across all features
- **Feature-based directory structure** — reorganized from flat layout to 8 feature modules with barrel exports
- **Removed duplicate legacy hooks directory** — consolidated all data hooks under feature modules

#### Security Hardening (Sprint 4)
- CSRF middleware now exempts pre-authentication routes (login, register, forgot-password, reset-password) to prevent fresh-session lockout
- API key authenticated requests exempt from CSRF validation
- Rate limit categories tuned for different endpoint sensitivities
- Audit logging expanded to cover all state-changing operations

### Fixed

#### Critical Fixes (Sprint 6)
- **Multi-tenant isolation in poller** — poller now queries packages per-org instead of across all orgs; settings reads scoped by org_id
- **CSRF blocking login on fresh sessions** — pre-auth routes exempt from CSRF validation
- **Production build failure** — test fixture files excluded from TypeScript production compilation
- **Login/register null data crash** — added null guard before destructuring auth response data
- **Non-JSON response crash** — `fetchApi` now catches `SyntaxError` from HTML error pages (e.g., nginx 502)
- **Duplicate error sanitization** — consolidated to single implementation in `error-sanitizer.ts`
- **GetRecentReleases classification filter** — filtering now applied before pagination for correct totals
- **Goroutine leak in CLIClient** — replaced `time.Tick` with `time.NewTicker` with proper cleanup
- **Org selector crash on empty organizations** — API failure now shows user-friendly error toast instead of silent empty state
- **Forgot-password raw error leak** — network errors now show sanitized message instead of raw "Failed to fetch"

#### High-Priority Fixes (Sprint 6)
- **RBAC permission checks on flat org-scoped routes** — packages, alerts, settings routes now enforce `RequirePermission` middleware; viewer-role users can no longer create/delete packages or change settings
- **Missing backend routes** — added `PUT /api/auth/me` (profile update) and `POST /api/auth/change-password`
- **DeletePackage missing existence check** — now returns 404 when package doesn't exist instead of false 200
- **UpdateAlert returning stale data** — response now contains current alert state after update
- **N+1 queries in release endpoints** — batch JOINs/Preloads replace per-release queries
- **Unchecked GORM errors** — all database operation results now checked with appropriate HTTP error responses
- **Notification channel cross-org isolation** — `CreateRule` now verifies channel belongs to the same org
- **Notification dispatch UserID=0** — org-wide notifications can now be marked as read by any org member
- **Differ previous release ordering** — uses `published_at` instead of `id` for correct version ordering
- **ListAlerts unqualified column names** — table-qualified column names prevent ambiguous SQL; filter values validated against allowed enums
- **OrgMember API response enriched** — member list now includes email, firstName, lastName from joined users table
- **getStoredOrgId NaN handling** — returns null for non-numeric stored values instead of sending `X-Org-ID: NaN`
- **Auth context token refresh** — now uses `apiRefreshToken()` with proper CSRF token attachment instead of raw `fetch()`
- **Notification optimistic update rollback** — `onError` handlers restore previous data from snapshot on failure
- **Alerts page stale closure** — `handleStatusChange` wrapped in `useCallback` with proper dependency tracking
- **OrgDetailContent setState during render** — state initialization moved to `useEffect`
- **Sidebar Settings active state overlap** — parent link uses exact match instead of `startsWith`
- **Notification bell severity detection** — replaced fragile string-matching with structured data approach
- **markAllNotificationsRead** — uncommented frontend function and wired to notification bell
- **ILIKE wildcard injection** — `%` and `_` escaped in search strings
- **Dynamic route param NaN validation** — all `[id]` pages validate `parseInt` with `isNaN` check
- **setState-in-effect lint errors** — resolved in 4 locations using derived state patterns
- **Package delete confirmation dialog** — destructive action now requires user confirmation
- **Icon-only button accessibility** — all icon-only action buttons have `aria-label` attributes
- **ESLint config for e2e files** — Playwright fixtures no longer flagged as React hooks violations
- **Unused imports and variables** — cleaned up 21 ESLint warnings across test and page files

### Security
- **CSRF protection** with double-submit cookie pattern, constant-time comparison, automatic token rotation
- **Input sanitization** middleware strips potentially dangerous content from all request bodies
- **Security headers** — Content-Security-Policy, X-Frame-Options (DENY), X-Content-Type-Options (nosniff), Strict-Transport-Security, Referrer-Policy, Permissions-Policy
- **Rate limiting** per IP with category-specific thresholds
- **JWT secret rotation** support for zero-downtime secret changes
- **API key prefix lookup** prevents timing attacks on key validation
- **Hashed invitation tokens** — raw tokens never stored in database
- **HMAC-signed webhooks** — notification webhook payloads signed for verification
- **Secrets management** — no hardcoded secrets in codebase; all secrets via environment variables
- **Multi-tenant data isolation** verified across all handlers, poller, and notification subsystems
- **Vulnerability scanning** integrated in CI pipeline (`govulncheck`, `npm audit`)

### Migration Notes

#### For New Deployments
1. Copy `backend/.env.example` to `backend/.env` and configure all required variables
2. Set `JWT_SECRET` to a secure random value: `openssl rand -hex 32`
3. Run `docker compose up -d` — migrations run automatically on first start
4. Register the first user via the UI or API — they will need to create an organization

#### Breaking Changes from Pre-Release
- **Clean Architecture refactor** (Sprint 1): Internal package structure changed completely. Custom extensions must target the new domain/repository/service layers
- **CSRF protection** (Sprint 4): All state-changing API calls from browsers must include the `X-CSRF-Token` header matching the `_csrf_token` cookie. API key authenticated requests are exempt
- **RBAC enforcement** (Sprint 6): Flat org-scoped routes now enforce permission checks. Viewer-role users can no longer modify packages, alerts, or settings
- **Handler decomposition** (Sprint 1): The single `Handlers` struct was split into 11 focused handler groups. Any middleware or test code referencing the old structure must be updated
- **Invitation token hashing** (Migration 000004): Existing invitation tokens are hashed in-place. Old raw tokens in emails will no longer work after migration

---

*Veilence-MX is an enterprise SaaS supply chain compromise monitor for PyPI and npm packages.*

[1.0.0]: https://github.com/veilence/veilence-mx/releases/tag/v1.0.0
