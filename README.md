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

## License

MIT
