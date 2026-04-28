# Veilence-MX

Real-time supply chain monitoring for Python and NPM ecosystems with LLM-powered diff analysis.

## Overview

Veilence-MX watches package ecosystems for new releases, generates diffs between versions, and uses LLMs to classify code changes as benign, suspicious, or malicious. Threats are surfaced through an alert management dashboard for security team triage.

Packages are monitored in two modes: Top-N (automatically tracks the most popular packages by downloads) and Manual (user-specified packages).

## Architecture

```
                    +-----------+     +-----------+     +----------------+
  Python / NPM     |           |     |           |     |                |
  Ecosystems ------>  Poller   +----->  Differ   +----->  Analyzer      |
                    |           |     |           |     |  (LLM)        |
                    +-----+-----+     +-----------+     +-------+--------+
                          |                                     |
                    +-----v-----+                         +-----v--------+
                    |           |                         |              |
                    | PostgreSQL|                         |    Alerts    |
                    | Redis     |                         |              |
                    |           |                         +-----+--------+
                    +-----+-----+                               |
                          |                               +-----v--------+
                    +-----v-----+                         |              |
                    |           |                         |  Dashboard   |
                    |  REST API |                         |  (Next.js)   |
                    |  (Chi v5) |                         |              |
                    +-----------+                         +--------------+
```

1. **Poll** - Discovers new releases from Python and NPM ecosystems
2. **Diff** - Generates unified diffs between consecutive releases (LCS algorithm)
3. **Analyze** - Sends diffs to an LLM for threat classification
4. **Alert** - Surfaces threats for security team triage
5. **Dashboard** - Visualizes alerts, stats, and package health

## Features

- Package monitoring - Python and NPM with automatic discovery and manual import
- Release analysis - Unified diff generation, LLM-powered threat classification
- Alert management - Triage workflow (new -> acknowledged -> resolved), notes, filtering
- Dashboard - Stats, charts, recent releases, stale package detection
- Workspaces - Multi-tenant with member management, invitation flow, RBAC
- Notifications - Email/Slack/webhook channels with configurable alert rules
- API keys - Scoped API access with permission management
- Audit logs - Full audit trail with human-readable details
- Settings - Discovery configuration, polling intervals, auto-approve controls
- Multi-provider LLM - copilot-api, OpenAI, Anthropic, Ollama

## Tech Stack

- **Backend:** Go 1.23+, Chi v5, GORM, PostgreSQL 16, Redis 7
- **Frontend:** Next.js 16 (App Router), React 19, TypeScript, TailwindCSS v4, shadcn/ui
- **LLM:** copilot-api (default), OpenAI, Anthropic, Ollama
- **Auth:** JWT (access + refresh), RBAC (Owner/Admin/Member/Viewer)
- **Notifications:** Email (SMTP), Slack webhooks, custom webhooks

## Prerequisites

- Go 1.23+
- Node.js 22+
- Docker and Docker Compose (for PostgreSQL + Redis)
- GitHub Copilot subscription (for copilot provider), or OpenAI/Anthropic API key, or local Ollama instance
- GitHub CLI (`gh`) authenticated - only needed for copilot provider

## Quick Start

### 1. Start Infrastructure

```bash
make db-up
```

### 2. Configure Environment

```bash
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local
```

Defaults work for local development. Key settings: `REGISTRATION_ENABLED` (default: false), `ALLOWED_EMAIL_DOMAINS`.

### 3. Start copilot-api (LLM Proxy)

```bash
make dev-copilot
```

Starts an OpenAI-compatible proxy on port 4141 that routes requests through your GitHub Copilot subscription. Keep this running in a separate terminal.

### 4. Start Backend

```bash
make dev-backend
```

Starts on `http://localhost:8080`, auto-migrates the database, and begins polling ecosystems.

### 5. Initial Setup

On first run, the frontend redirects to `/setup` where you create the first admin user and workspace. Alternatively:

```bash
curl -X POST http://localhost:8080/api/setup/initialize \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"yourpassword","firstName":"Admin","lastName":"User","workspaceName":"My Workspace","workspaceSlug":"my-workspace"}'
```

Public registration is disabled by default. Set `REGISTRATION_ENABLED=true` in `backend/.env` to allow open sign-ups.

### 6. Start Frontend

```bash
cd frontend && npm install
make dev-frontend
```

Starts on `http://localhost:3000`.

### All-in-One

```bash
make dev
```

### Docker (Full Stack)

```bash
make docker-up          # Build and start all services (dev)
make docker-down        # Stop all services
make docker-destroy     # Stop and remove all data

make docker-prod-up     # Build and start (production)
make docker-prod-down   # Stop production services
```

## Makefile Reference

```bash
# Infrastructure
make db-up              # Start PostgreSQL + Redis
make db-down            # Stop infrastructure
make db-destroy         # Stop and remove all data volumes

# Docker (full stack)
make docker-up          # Start full stack (dev)
make docker-down        # Stop full stack
make docker-destroy     # Stop and remove all data volumes
make docker-build       # Build Docker images
make docker-prod-up     # Start full stack (production)
make docker-prod-down   # Stop production services

# Development
make dev-copilot        # Start copilot-api proxy
make dev-backend        # Start backend
make dev-frontend       # Start frontend
make dev                # Start infra + backend + frontend

# Build
make build-backend      # Build Go binary
make build-frontend     # Build Next.js
make build              # Build both

# Lint
make lint-backend       # Go vet
make lint-frontend      # ESLint
make lint               # Lint both
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[MIT](LICENSE)
