# Veilence-MX Frontend

Frontend dashboard for the Veilence-MX supply chain monitoring platform. Provides real-time visibility into package registry changes, LLM-powered diff analysis, and alert triage for security teams.

## Tech Stack

- **Framework** - Next.js 16 (App Router), React 19, TypeScript (strict)
- **Styling** - TailwindCSS v4, shadcn/ui, @base-ui/react
- **Data** - TanStack Query, TanStack Table, Zod
- **Charts** - Recharts
- **Icons** - Lucide React

## Architecture

Clean Architecture with 5 layers - dependencies flow inward only:

```
src/
├── app/          # Next.js App Router - routing, layouts, page composition only
├── features/     # UI logic per feature - hooks, components, providers
├── domains/      # Pure data - API calls, types, DTOs, mappers (zero React)
├── core/         # Infrastructure - HTTP client, config, providers
└── ui/           # Shared presentational components (zero business logic)
```

See [FRONTEND_ARCHITECTURE.md](./FRONTEND_ARCHITECTURE.md) for the full rules and dependency constraints.

## Project Structure

```
src/
├── app/
│   ├── (auth)/              # Auth routes - login, register, setup, invite, password flows
│   ├── alerts/              # Alert triage
│   ├── notifications/       # Notification center
│   ├── packages/            # Package monitoring
│   ├── releases/            # Release details
│   ├── settings/            # Workspace settings
│   ├── workspaces/          # Workspace management
│   ├── account/             # User account
│   ├── layout.tsx           # Root layout
│   └── page.tsx             # Dashboard
├── features/
│   ├── auth/                # Authentication logic
│   ├── setup/               # Initial setup wizard
│   ├── dashboard/           # Dashboard widgets
│   ├── packages/            # Package monitoring UI
│   ├── alerts/              # Alert triage UI
│   ├── releases/            # Release diff viewer
│   ├── settings/            # Settings panels
│   ├── notifications/       # Notification UI
│   ├── shell/               # App shell (sidebar, header)
│   ├── onboarding/          # Onboarding flows
│   ├── account/             # Account management
│   ├── admin/               # Admin features
│   └── change-password/     # Password change flow
├── domains/
│   ├── auth/                # Auth API, types, mappers
│   ├── packages/            # Package domain
│   ├── alerts/              # Alert domain
│   ├── releases/            # Release domain
│   ├── dashboard/           # Dashboard domain
│   ├── settings/            # Settings domain
│   ├── notifications/       # Notification domain
│   ├── setup/               # Setup domain
│   ├── account/             # Account domain
│   ├── admin/               # Admin domain
│   ├── config/              # Config domain
│   ├── queue/               # Queue domain
│   └── common/              # Shared types and utils
├── core/
│   ├── http.ts              # HTTP client with auth
│   ├── config.ts            # Environment variables (single source of truth)
│   ├── providers/           # React Query, theme providers
│   ├── hooks/               # Shared infrastructure hooks
│   └── types.ts             # Core type definitions
└── ui/
    ├── components/          # shadcn/ui primitives (button, input, dialog, etc.)
    ├── data/                # Data table, pagination, filters
    ├── feedback/            # Empty state, error state, loading
    ├── form/                # Form fields, search inputs
    └── layout/              # App shell, sidebar, header
```

## Prerequisites

- Node.js 22+
- Veilence-MX backend running (default: `http://localhost:8080`)

## Getting Started

```bash
# Install dependencies
npm install

# Set up environment
cp .env.example .env.local

# Start the backend first, then:
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

On first run, if the backend has no users configured, you will be redirected to the `/setup` wizard to create the initial admin account and workspace.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Backend API base URL. All API requests from the frontend are sent to this address. |

Configure via `.env.local` (not committed to git).

## Key Pages

| Route | Description |
|---|---|
| `/setup` | Initial setup wizard (first run only) |
| `/login` | Sign in |
| `/register` | Create account |
| `/change-password` | Forced password change flow |
| `/` | Dashboard - overview of monitored packages and alerts |
| `/packages` | Package monitoring - tracked packages and their status |
| `/alerts` | Alert triage - review and manage security alerts |
| `/releases` | Release details and diff analysis |
| `/settings` | Workspace settings (members, roles, notifications, API keys, audit, discovery) |

## Build

```bash
npm run build
```

## Lint

```bash
npm run lint
npx tsc --noEmit
```
