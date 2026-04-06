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
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │  REST API     │
                    │  (Chi v5)     │
                    └───────┬───────┘
                            │
                    ┌───────▼───────┐
                    │  Next.js      │
                    │  Dashboard    │
                    └───────────────┘
```

## Tech Stack

- **Backend**: Go 1.23+, Chi v5, GORM, PostgreSQL 16
- **Frontend**: Next.js 16 (App Router), TypeScript (strict, no `any`), TailwindCSS, shadcn/ui
- **LLM**: copilot-api → GitHub Copilot → Claude Sonnet 4.6
- **Testing**: Go testify, httptest, Playwright E2E

## Prerequisites

- Go 1.23+
- Node.js 20+
- Docker (for PostgreSQL)
- GitHub Copilot subscription (for LLM analysis)
- GitHub CLI (`gh`) authenticated (`gh auth login`)

## Quick Start

### 1. Start PostgreSQL

```bash
docker compose up -d postgres
```

### 2. Configure Environment

```bash
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local
# Review both files — defaults work out of the box
```

### 3. Start copilot-api (LLM Proxy)

```bash
npx copilot-api start --github-token "$(cat /Users/herwonowr/.local/share/copilot-api/github_token)" --port 4141 --rate-limit 5 --wait
```

This starts an OpenAI-compatible proxy on port 4141 that routes requests through your GitHub Copilot subscription. Keep this running in a separate terminal.

### 4. Start Backend

```bash
cd backend
go run ./cmd/server/
```

The backend starts on `http://localhost:8080`, auto-migrates the database, and begins polling registries.

### 5. Start Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend starts on `http://localhost:3000`.

## Development

### Build

```bash
# Backend
cd backend && go build ./...

# Frontend
cd frontend && npm run build
```

### Test

```bash
# Backend unit tests
cd backend && go test ./... -count=1

# Frontend E2E tests
cd e2e && npx playwright test

# Playwright UI mode (interactive)
cd e2e && npx playwright test --ui
```

### Makefile Shortcuts

```bash
make db-up              # Start PostgreSQL
make dev-copilot        # Start copilot-api proxy
make dev-backend        # Start backend
make dev-frontend       # Start frontend
make test-backend       # Run Go tests
make test-e2e           # Run Playwright tests
make build-backend      # Build Go binary
make build-frontend     # Build Next.js
make lint-backend       # Go vet
make lint-frontend      # ESLint
```

### VS Code Copilot Agents

This project includes 4 specialized VS Code Copilot agents:

| Agent | Purpose |
|-------|---------|
| `@Backend` | Go backend development |
| `@Frontend` | Next.js frontend development |
| `@Test` | Go backend testing |
| `@UITest` | Playwright E2E testing |

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dashboard/stats` | Dashboard overview statistics |
| GET | `/api/dashboard/recent-releases` | Recent releases with analysis |
| GET | `/api/packages` | List packages (filter: registry, search) |
| POST | `/api/packages` | Add custom package |
| GET | `/api/packages/:id` | Package detail |
| DELETE | `/api/packages/:id` | Remove package |
| GET | `/api/packages/:id/releases` | Package releases |
| GET | `/api/releases/:id` | Release detail with diff + analysis |
| GET | `/api/alerts` | List alerts (filter: severity, status) |
| PATCH | `/api/alerts/:id` | Update alert status |
| GET | `/api/settings` | Get all settings |
| PUT | `/api/settings` | Update settings |
| POST | `/api/sync/top-packages` | Trigger top-N package sync |
| POST | `/api/sync/reanalyze` | Re-queue un-analyzed diffs |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://veilence:veilence_dev@localhost:5432/veilence_mx?sslmode=disable` | PostgreSQL connection string |
| `COPILOT_API_URL` | `http://localhost:4141` | copilot-api proxy URL |
| `COPILOT_MODEL` | `claude-sonnet-4.6` | LLM model to use for analysis |
| `PYPI_POLL_INTERVAL` | `5m` | PyPI polling interval |
| `NPM_POLL_INTERVAL` | `5m` | npm polling interval |
| `PYPI_TOP_N` | `100` | Number of top PyPI packages to monitor |
| `NPM_TOP_N` | `100` | Number of top npm packages to monitor |
| `TOP_N_REFRESH_INTERVAL` | `24h` | How often to re-fetch the top-N lists |
| `POLLER_CONCURRENCY` | `5` | Concurrent package fetches per poll |
| `DIFF_SIZE_LIMIT` | `102400` | Max diff size (bytes) sent to LLM |
| `SERVER_PORT` | `8080` | Backend HTTP port |
| `FRONTEND_URL` | `http://localhost:3000` | Frontend URL for CORS |

## Project Structure

```
veilence-mx/
├── backend/
│   ├── cmd/server/             # Entry point
│   └── internal/
│       ├── api/                # Chi router + handlers
│       ├── analyzer/           # copilot-api HTTP client
│       ├── database/           # GORM connection + migrations
│       ├── differ/             # Tarball diff engine
│       ├── models/             # GORM models
│       ├── poller/             # Registry polling service
│       └── registry/           # PyPI + npm API clients
├── frontend/
│   └── src/
│       ├── app/                # Next.js pages
│       ├── components/         # shadcn/ui components
│       ├── lib/                # API client + utils
│       └── types/              # TypeScript types
├── e2e/                        # Playwright E2E tests
├── .github/
│   ├── agents/                 # VS Code Copilot agent definitions
│   └── skills/                 # Copilot skill instructions
├── docker-compose.yml
├── Makefile
└── copilot-instructions.md     # Project coding conventions
```

## License

MIT
