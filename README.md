# Veilence-MX — Supply Chain Compromise Monitor

Automated monitoring of top PyPI and npm packages for supply chain compromise. Polls both registries for new releases, diffs each release against its predecessor, and uses an LLM (Claude Sonnet 4.6 via copilot-api proxy) to classify diffs as benign, suspicious, or malicious.

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Poller    │────▶│   Differ    │────▶│   Analyzer   │────▶│   Alerts    │
│ (PyPI/npm)  │     │ (tar.gz diff)│    │ (copilot-api)│     │ (Dashboard) │
└─────────────┘     └─────────────┘     └──────┬───────┘     └─────────────┘
       │                    │                   │
       └────────────────────┴───────────────────┘
                            │
                    ┌───────▼───────┐
                    │  PostgreSQL   │
                    │  + Redis      │
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │  REST API     │
                    │  (Chi v5)     │
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │  Next.js 16   │
                    │  Dashboard    │
                    └───────────────┘
```

## Tech Stack

- **Backend**: Go 1.25+, Chi v5, GORM, PostgreSQL 16, Redis 7
- **Frontend**: Next.js 16 (App Router), React 19, TypeScript (strict, no `any`), TailwindCSS v4, shadcn/ui
- **LLM**: copilot-api → GitHub Copilot → Claude Sonnet 4.6
- **Auth**: JWT (access + refresh tokens), RBAC, multi-tenant workspaces
- **Testing**: Go testify, httptest, Playwright E2E

## Prerequisites

- Go 1.25+
- Node.js 22+
- Docker & Docker Compose (for PostgreSQL + Redis)
- GitHub Copilot subscription (for LLM analysis)
- GitHub CLI (`gh`) authenticated (`gh auth login`)

## Quick Start

### 1. Start Infrastructure

```bash
make db-up    # Starts PostgreSQL + Redis
```

### 2. Configure Environment

```bash
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local
# Review both files — defaults work out of the box
```

### 3. Start copilot-api (LLM Proxy)

```bash
make dev-copilot
```

This starts an OpenAI-compatible proxy on port 4141 that routes requests through your GitHub Copilot subscription. Keep this running in a separate terminal.

### 4. Start Backend

```bash
make dev-backend
```

The backend starts on `http://localhost:8080`, auto-migrates the database, and begins polling registries.

### 5. Start Frontend

```bash
cd frontend && npm install
make dev-frontend
```

The frontend starts on `http://localhost:3000`.

### Docker (Full Stack)

```bash
# Development
make docker-up        # Build and start all services
make docker-down      # Stop all services

# Production
make docker-prod-up   # Build and start with production config
make docker-prod-down # Stop production services
```

## Development

### Makefile Shortcuts

```bash
# Infrastructure
make db-up                  # Start PostgreSQL + Redis
make db-down                # Stop all services
make db-test-setup          # Create integration test database

# Docker
make docker-up              # Start full stack (dev)
make docker-down            # Stop full stack
make docker-build           # Build Docker images
make docker-prod-up         # Start full stack (production)

# Development
make dev-copilot            # Start copilot-api proxy
make dev-backend            # Start backend
make dev-frontend           # Start frontend
make dev                    # Start infra + backend + frontend

# Build
make build-backend          # Build Go binary
make build-frontend         # Build Next.js
make build                  # Build both

# Test
make test-backend           # Run Go unit tests
make test-backend-integration # Run Go integration tests
make test-e2e               # Run Playwright E2E tests
make test-e2e-ui            # Playwright interactive UI mode

# Lint
make lint-backend           # Go vet
make lint-frontend          # ESLint
make lint                   # Lint both
```

## API Endpoints

All workspace-scoped routes require `X-Workspace-ID` header and RBAC permissions.

### Public

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/ready` | Readiness check |
| POST | `/api/auth/register` | Register new user |
| POST | `/api/auth/login` | Login |
| POST | `/api/auth/refresh` | Refresh JWT token |
| POST | `/api/auth/forgot-password` | Request password reset |
| POST | `/api/auth/reset-password` | Reset password |
| POST | `/api/auth/verify-email` | Verify email address |
| GET | `/api/invitations/{token}` | Get invitation info |

### Authenticated (User-Scoped)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/auth/me` | Get current user |
| PUT | `/api/auth/me` | Update profile |
| POST | `/api/auth/logout` | Logout |
| POST | `/api/auth/change-password` | Change password |
| GET | `/api/auth/sessions` | List active sessions |
| DELETE | `/api/auth/sessions/{id}` | Revoke a session |
| GET | `/api/notifications` | List user notifications |
| GET | `/api/notifications/unread-count` | Get unread notification count |
| PUT | `/api/notifications/read-all` | Mark all notifications read |
| GET | `/api/permissions` | List all permissions |

### Workspace-Scoped (RBAC)

| Method | Endpoint | Permission | Description |
|--------|----------|------------|-------------|
| GET | `/api/dashboard/stats` | — | Dashboard overview statistics |
| GET | `/api/dashboard/recent-releases` | — | Recent releases with analysis |
| GET | `/api/dashboard/charts` | — | Chart data |
| GET | `/api/packages` | `packages:read` | List packages |
| POST | `/api/packages` | `packages:write` | Add package |
| POST | `/api/packages/bulk-import` | `packages:write` | Bulk import packages |
| GET | `/api/packages/suggestions` | `packages:read` | List package suggestions |
| GET | `/api/packages/stale` | `packages:read` | List stale packages |
| POST | `/api/packages/bulk-approve` | `packages:write` | Bulk approve suggestions |
| GET | `/api/packages/{id}` | `packages:read` | Package detail |
| DELETE | `/api/packages/{id}` | `packages:delete` | Remove package |
| POST | `/api/packages/{id}/block` | `packages:write` | Block package |
| POST | `/api/packages/{id}/unblock` | `packages:write` | Unblock package |
| GET | `/api/packages/{id}/releases` | `releases:read` | Package releases |
| GET | `/api/packages/{id}/analysis-history` | `packages:read` | Analysis history |
| GET | `/api/releases/{id}` | `releases:read` | Release detail with diff + analysis |
| POST | `/api/releases/{id}/reanalyze` | `settings:write` | Re-analyze a release |
| GET | `/api/alerts` | `alerts:read` | List alerts |
| GET | `/api/alerts/{id}` | `alerts:read` | Alert detail |
| PATCH | `/api/alerts/{id}` | `alerts:write` | Update alert status |
| GET | `/api/alerts/{id}/notes` | `alerts:read` | List alert notes |
| POST | `/api/alerts/{id}/notes` | `alerts:write` | Create alert note |
| GET | `/api/settings` | `settings:read` | Get settings |
| PUT | `/api/settings` | `settings:write` | Update settings |
| POST | `/api/sync/discover` | `settings:write` | Trigger package discovery |
| POST | `/api/sync/reanalyze` | `settings:write` | Re-analyze all pending |
| GET | `/api/auth/api-keys` | `api_keys:read` | List API keys |
| POST | `/api/auth/api-keys` | `api_keys:write` | Create API key |
| DELETE | `/api/auth/api-keys/{id}` | `api_keys:write` | Revoke API key |
| GET | `/api/queue/stats` | `settings:read` | Queue statistics |
| GET | `/api/queue/jobs` | `settings:read` | Queue jobs |
| POST | `/api/queue/retry-dead` | `settings:write` | Retry all dead jobs |

### Workspace Management

| Method | Endpoint | Permission | Description |
|--------|----------|------------|-------------|
| POST | `/api/workspaces` | — | Create workspace |
| GET | `/api/workspaces` | — | List user's workspaces |
| GET | `/api/workspaces/{id}` | `workspace:read` | Get workspace |
| PUT | `/api/workspaces/{id}` | `workspace:write` | Update workspace |
| DELETE | `/api/workspaces/{id}` | `workspace:delete` | Delete workspace |
| GET | `/api/workspaces/{id}/members` | `members:read` | List members |
| GET | `/api/workspaces/{id}/members/me/role` | — | Get own role |
| POST | `/api/workspaces/{id}/invitations` | `members:invite` | Invite member |
| GET | `/api/workspaces/{id}/invitations` | `members:read` | List pending invitations |
| DELETE | `/api/workspaces/{id}/invitations/{id}` | `members:invite` | Revoke invitation |
| DELETE | `/api/workspaces/{id}/members/{userId}` | `members:remove` | Remove member |
| PUT | `/api/workspaces/{id}/members/{userId}/role` | `members:remove` | Update member role |
| GET | `/api/workspaces/{id}/roles` | `roles:read` | List roles |
| GET | `/api/workspaces/{id}/audit-logs` | `audit:read` | Audit logs |
| GET | `/api/workspaces/{id}/notification-channels` | `notifications:read` | List notification channels |
| POST | `/api/workspaces/{id}/notification-channels` | `notifications:create` | Create notification channel |
| PUT | `/api/workspaces/{id}/notification-channels/{id}` | `notifications:update` | Update channel |
| DELETE | `/api/workspaces/{id}/notification-channels/{id}` | `notifications:delete` | Delete channel |
| POST | `/api/workspaces/{id}/notification-channels/{id}/test` | `notifications:update` | Test channel |
| GET | `/api/workspaces/{id}/notification-rules` | `notifications:read` | List rules |
| POST | `/api/workspaces/{id}/notification-rules` | `notifications:create` | Create rule |
| DELETE | `/api/workspaces/{id}/notification-rules/{id}` | `notifications:delete` | Delete rule |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://veilence:veilence_dev@localhost:5432/veilence_mx?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection string |
| `JWT_SECRET` | — | JWT signing secret (required) |
| `FRONTEND_URL` | `http://localhost:3000` | Frontend URL for CORS |
| `SERVER_PORT` | `8080` | Backend HTTP port |
| `APP_ENV` | `development` | Environment (`development` / `production`) |
| `COPILOT_API_URL` | `http://localhost:4141` | copilot-api proxy URL |
| `COPILOT_MODEL` | `claude-sonnet-4.6` | LLM model for analysis |
| `PYPI_POLL_INTERVAL` | `5m` | PyPI polling interval |
| `NPM_POLL_INTERVAL` | `5m` | npm polling interval |
| `PYPI_TOP_N` | `100` | Top PyPI packages to monitor |
| `NPM_TOP_N` | `100` | Top npm packages to monitor |
| `TOP_N_REFRESH_INTERVAL` | `24h` | Top-N list refresh interval |
| `POLLER_CONCURRENCY` | `5` | Concurrent package fetches per poll |
| `DIFF_SIZE_LIMIT` | `102400` | Max diff size (bytes) sent to LLM |

## Project Structure

```
veilence-mx/
├── backend/
│   ├── cmd/server/                # Entry point
│   ├── config/                    # Environment config
│   ├── migrations/                # SQL migrations
│   ├── pkg/                       # Shared libraries (httpserver, metrics, jwt)
│   └── internal/
│       ├── entity/                # Domain entities (pure Go, zero deps)
│       ├── usecase/               # Business logic + interface contracts
│       │   ├── contracts.go       # All repository interfaces
│       │   ├── auth/              # Authentication service
│       │   ├── rbac/              # Role-based access control
│       │   ├── audit/             # Audit logging
│       │   └── ...
│       ├── repo/                  # Infrastructure (DB, external APIs)
│       │   ├── persistent/        # PostgreSQL implementations
│       │   └── webapi/            # External API clients (PyPI, npm, LLM)
│       └── controller/            # HTTP handlers
│           └── restapi/
│               ├── router.go      # Route registration + middleware
│               ├── middleware/     # Auth, CORS, rate limiting, CSRF
│               └── v1/            # Handler implementations
├── frontend/
│   └── src/
│       ├── app/                   # Next.js App Router (routing + composition only)
│       ├── domains/               # Pure data (API calls, types, mappers — zero React)
│       ├── features/              # UI logic per feature (hooks, components, providers)
│       ├── core/                  # Infrastructure (HTTP client, config, providers)
│       └── ui/                    # Shared presentational components (zero business logic)
├── e2e/                           # Playwright E2E tests
├── docker-compose.yml             # Development stack
├── docker-compose.prod.yml        # Production stack
└── Makefile                       # Development shortcuts
```

## License

MIT
