# Veilence-MX - Supply Chain Compromise Monitor

Automated monitoring of PyPI and npm packages for supply chain compromise. Polls registries for new releases, diffs each release against its predecessor, and uses an LLM (Claude Sonnet 4.6 via copilot-api proxy) to classify diffs as benign, suspicious, or malicious.

## Architecture

```
Poller (PyPI/npm) --> Differ (unified LCS diff) --> Analyzer (LLM) --> Alerts --> Dashboard
                              |
                    PostgreSQL + Redis
                              |
                      REST API (Chi v5)
                              |
                    Next.js 16 Dashboard
```

**Pipeline**: Poller discovers new releases -> Differ generates unified diffs with LCS algorithm -> Analyzer sends diffs to LLM for classification -> Alerts surface threats for triage

**Monitoring modes**:
- **Top-N**: Automatically monitors the N most popular packages by downloads
- **Manual**: Users add specific packages they depend on

## Tech Stack

- **Backend**: Go 1.23+, Chi v5, GORM, PostgreSQL 16, Redis 7 - Clean Architecture (entity/usecase/repo/controller)
- **Frontend**: Next.js 16 (App Router), React 19, TypeScript strict, TailwindCSS v4, shadcn/ui - Clean Architecture (app/features/domains/core/ui)
- **LLM**: copilot-api -> GitHub Copilot -> Claude Sonnet 4.6
- **Auth**: JWT (access + refresh tokens), RBAC (Owner/Admin/Member/Viewer), multi-tenant workspaces
- **Notifications**: Email (SMTP), Slack webhooks, custom webhooks with configurable rules

## Features

- **Package Monitoring** - Track PyPI and npm packages with automatic discovery and manual import
- **Release Analysis** - Unified diff generation with LCS algorithm, LLM-powered threat classification
- **Alert Management** - Triage workflow (new -> acknowledged -> resolved), notes, filtering
- **Dashboard** - Stats, charts, recent releases, stale package detection, pending invitations
- **Workspaces** - Multi-tenant with member management, invitation flow, RBAC
- **Notifications** - Email/Slack/webhook channels with configurable alert rules
- **API Keys** - Scoped API access with permission management
- **Audit Logs** - Full audit trail with human-readable details
- **Settings** - Discovery configuration, polling intervals, auto-approve controls

## Prerequisites

- Go 1.23+
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
# Review both files - defaults work out of the box
```

### 3. Start copilot-api (LLM Proxy)

```bash
make dev-copilot
```

Starts an OpenAI-compatible proxy on port 4141 that routes requests through your GitHub Copilot subscription. Keep this running in a separate terminal.

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

### All-in-One

```bash
make dev    # Starts infra + backend + frontend
```

### Docker (Full Stack)

```bash
# Development
make docker-up          # Build and start all services
make docker-down        # Stop all services
make docker-destroy     # Stop and remove all data

# Production
make docker-prod-up     # Build and start with production config
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

## License

MIT
