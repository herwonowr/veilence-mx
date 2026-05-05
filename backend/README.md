# Veilence-MX Backend

REST API backend for the Veilence-MX supply chain monitoring platform. Monitors PyPI and NPM registries for new package releases, analyzes code diffs with LLMs to detect suspicious changes, and surfaces alerts for security team triage.

## Tech Stack

- **Go 1.23+** - application language
- **Chi v5** - HTTP router
- **GORM** - ORM for PostgreSQL
- **PostgreSQL 16** - primary database
- **Redis 7** - async job queue (polling, analysis pipeline)
- **LLM providers** - OpenAI, Anthropic, GitHub Copilot, or Ollama for diff analysis

## Architecture

Clean Architecture with strict dependency inversion. Dependencies flow inward only: controller -> usecase -> entity. Infrastructure (repo, pkg) implements interfaces defined by the usecase layer.

See [BACKEND_ARCHITECTURE.md](./BACKEND_ARCHITECTURE.md) for the full rules, layer templates, and code style requirements.

## Project Structure

```
backend/
├── cmd/server/           # Entry point - config loading + app.Run()
├── internal/
│   ├── app/              # Composition root - wires all dependencies
│   ├── config/           # Struct-based config from env vars
│   ├── entity/           # Domain structs (user, package, release, alert, etc.)
│   ├── usecase/          # Business logic per domain:
│   │   ├── contracts.go  #   All repository/external interfaces
│   │   ├── auth/         #   Authentication + sessions
│   │   ├── rbac/         #   Role-based access control
│   │   ├── pkguc/        #   Package management
│   │   ├── releaseuc/    #   Release tracking
│   │   ├── alertuc/      #   Alert triage workflow
│   │   ├── alertnote/    #   Alert notes/comments
│   │   ├── analyzer/     #   LLM-powered diff analysis
│   │   ├── differ/       #   Release diff generation
│   │   ├── poller/       #   Registry polling for new releases
│   │   ├── dashboarduc/  #   Dashboard stats + charts
│   │   ├── notifications/#   Channels, rules, user notifications
│   │   ├── settinguc/    #   Workspace settings + discovery config
│   │   ├── setup/        #   First-run initialization
│   │   ├── audit/        #   Audit logging
│   │   ├── digest/       #   Digest reports
│   │   └── healthuc/     #   Health/readiness checks
│   ├── controller/
│   │   └── restapi/      # HTTP handlers, middleware, request/response DTOs
│   │       ├── router.go #   All route definitions
│   │       ├── middleware/#   Auth, CORS, CSRF, rate limiting, etc.
│   │       └── v1/       #   Handler implementations
│   ├── repo/             # Infrastructure implementations (Postgres, external APIs)
│   └── integration/      # Integration tests
├── pkg/                  # Shared reusable libraries:
│   ├── postgres/         #   DB connection helper
│   ├── queue/            #   Redis job queue
│   ├── llm/              #   LLM client abstraction
│   ├── openai/           #   OpenAI provider
│   ├── anthropic/        #   Anthropic provider
│   ├── copilotapi/       #   GitHub Copilot provider
│   ├── ollama/           #   Ollama provider
│   ├── mailer/           #   SMTP email
│   ├── token/            #   JWT management
│   ├── hasher/           #   Password hashing
│   ├── id/               #   ID generation
│   ├── metrics/          #   Prometheus metrics
│   └── sender/           #   Notification dispatch
└── migrations/           # SQL migration files
```

## Prerequisites

- Go 1.23+
- PostgreSQL 16
- Redis 7
- An LLM provider (one of: OpenAI API key, Anthropic API key, GitHub Copilot via copilot-api, or local Ollama)

## Getting Started

1. **Copy the env file:**

   ```bash
   cp .env.example .env
   ```

2. **Edit `.env`** - set `DATABASE_URL`, `LLM_PROVIDER`, `LLM_API_URL`, `LLM_MODEL`, and `LLM_API_KEY` (if using OpenAI/Anthropic).

3. **Start PostgreSQL and Redis.** Either use the root docker-compose:

   ```bash
   # From the repository root
   docker compose up -d postgres redis
   ```

   Or run them standalone.

4. **Run the backend:**

   ```bash
   # From the backend/ directory
   go run ./cmd/server serve

   # Or from the repository root
   make dev-backend
   ```

   Available subcommands: `serve` (start HTTP server), `migrate` (run migrations), `seed` (seed data).

   The server starts on `http://localhost:8080` by default. Database tables are created automatically on first run.

5. **Initialize the platform:** On first run, call the setup endpoint to create the first admin user and workspace:

   ```bash
   curl -X POST http://localhost:8080/api/setup/initialize \
     -H "Content-Type: application/json" \
     -d '{"email":"admin@example.com","password":"yourpassword","name":"Admin","workspace_name":"My Workspace"}'
   ```

   Or use the frontend setup wizard, which calls the same endpoint.

## Environment Variables

All configuration is loaded from environment variables (12-factor). See [.env.example](./.env.example) for the full list with descriptions.

Key categories:

| Category | Variables | Notes |
|---|---|---|
| Database | `DATABASE_URL` | PostgreSQL connection string (required) |
| Redis | `REDIS_URL`, `QUEUE_*` | Job queue config |
| LLM | `LLM_PROVIDER`, `LLM_API_URL`, `LLM_MODEL`, `LLM_API_KEY` | Diff analysis provider (required) |
| Server | `SERVER_PORT`, `FRONTEND_URL`, `BACKEND_URL`, `APP_ENV` | HTTP server + CORS + SSO callbacks |
| Auth | `JWT_SECRET`, `JWT_SECRET_PREVIOUS` | JWT signing + rotation |
| SSO | `SSO_ENABLED`, `SSO_ENCRYPTION_KEY`, `SSO_SAML_CLOCK_SKEW`, `SSO_STATE_TTL` | Enterprise SSO (SAML, Google, GitHub) |
| SMTP | `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, etc. | Email (optional - tokens logged to console if unset) |
| Registration | `REGISTRATION_ENABLED`, `ALLOWED_EMAIL_DOMAINS` | User registration controls |
| Monitoring | `MONITORING_INTERVAL`, `DISCOVERY_INTERVAL`, `POLLER_CONCURRENCY` | Polling pipeline |
| Analysis | `DIFF_SIZE_LIMIT`, `LLM_MAX_DIFF_LEN`, `LLM_RATE_INTERVAL` | Diff + LLM limits |

## Initial Setup Flow

1. On a fresh install, `REGISTRATION_ENABLED` defaults to `false` (no public sign-ups).
2. Call `POST /api/setup/initialize` (or use the frontend wizard) to create the first admin user and workspace.
3. The admin can then add users directly to workspaces via the member management endpoints or invite them by email.
4. To allow public registration, set `REGISTRATION_ENABLED=true` in your `.env`.

## Build

```bash
go build ./...
```

## Lint

```bash
go vet ./...
```
