# Veilence-MX Admin Runbook

**Version:** 1.0.0
**Last Updated:** 2026-04-11

---

## Table of Contents

1. [Infrastructure Overview](#1-infrastructure-overview)
2. [Health Checks & Monitoring](#2-health-checks--monitoring)
3. [User Management](#3-user-management)
4. [Organization Management](#4-organization-management)
5. [Queue Monitoring & Dead Job Recovery](#5-queue-monitoring--dead-job-recovery)
6. [Alert Triage Workflow](#6-alert-triage-workflow)
7. [Package Management](#7-package-management)
8. [Notification Configuration](#8-notification-configuration)
9. [Settings Management](#9-settings-management)
10. [Database Maintenance](#10-database-maintenance)
11. [Log Analysis](#11-log-analysis)
12. [Common Error Resolutions](#12-common-error-resolutions)
13. [Backup & Recovery](#13-backup--recovery)

---

## 1. Infrastructure Overview

### Architecture

```
Poller -> Differ -> Analyzer (LLM) -> Alerts -> Dashboard
```

### Services

| Service    | Container Name           | Port | Description                              |
|------------|--------------------------|------|------------------------------------------|
| Backend    | veilence-mx-backend      | 8080 | Go API server (Chi v5, GORM)             |
| Frontend   | veilence-mx-frontend     | 3000 | Next.js 16 App Router                    |
| PostgreSQL | veilence-mx-db           | 5432 | Primary database (PostgreSQL 16)         |
| Redis      | veilence-mx-redis        | 6379 | Job queue (diff & analysis jobs)         |

### Starting the Stack

```bash
# Development
docker compose up -d

# Production
docker compose -f docker-compose.prod.yml up -d

# Rebuild after code changes
docker compose up -d --build
```

### Stopping the Stack

```bash
docker compose down          # Stop containers, keep volumes
docker compose down -v       # Stop containers AND delete volumes (data loss!)
```

### Key Environment Variables

| Variable               | Required | Default         | Description                                      |
|------------------------|----------|-----------------|--------------------------------------------------|
| `DATABASE_URL`         | Yes      | -               | PostgreSQL connection string                     |
| `REDIS_URL`            | Yes      | -               | Redis connection URL                             |
| `JWT_SECRET`           | Yes*     | dev default     | JWT signing secret (min 32 chars in production)  |
| `JWT_SECRET_PREVIOUS`  | No       | -               | Previous JWT secrets for rotation (comma-sep)    |
| `FRONTEND_URL`         | Yes      | -               | Frontend URL for CORS/CSP headers                |
| `APP_ENV`              | Yes      | `development`   | `development` or `production`                    |
| `SERVER_PORT`          | No       | `8080`          | HTTP server listen port                          |
| `COPILOT_API_URL`      | No       | -               | LLM API proxy URL for analysis                   |
| `COPILOT_MODEL`        | No       | `claude-sonnet-4.6` | LLM model for analysis                      |
| `PYPI_POLL_INTERVAL`   | No       | `5m`            | PyPI polling frequency                           |
| `NPM_POLL_INTERVAL`    | No       | `5m`            | npm polling frequency                            |
| `POLLER_CONCURRENCY`   | No       | `5`             | Max concurrent package fetches                   |
| `QUEUE_MAX_RETRIES`    | No       | `5`             | Max retries before dead-letter                   |
| `QUEUE_LOCK_TIMEOUT`   | No       | `10m`           | Stuck job detection threshold                    |
| `DIFF_SIZE_LIMIT`      | No       | `102400`        | Max diff size (bytes) sent to LLM                |
| `SMTP_HOST`            | No       | -               | SMTP server for email notifications              |
| `SMTP_PORT`            | No       | `587`           | SMTP port                                        |
| `SMTP_USERNAME`        | No       | -               | SMTP auth username                               |
| `SMTP_PASSWORD`        | No       | -               | SMTP auth password                               |
| `SMTP_FROM`            | No       | -               | Sender email address                             |

> *In production (`APP_ENV=production`), `JWT_SECRET` must be set to a secure random value. Generate with: `openssl rand -hex 32`

### Rate Limits

| Category | Limit (req/min) | Applies To                                   |
|----------|-----------------|----------------------------------------------|
| API      | 100             | All API endpoints (global)                   |
| Auth     | 10              | Login, register, password reset              |
| Sync     | 5               | Top packages sync, reanalyze triggers        |

---

## 2. Health Checks & Monitoring

### Health Check Endpoint

```bash
# Basic health (checks DB + Redis connectivity)
curl -s http://localhost:8080/api/health | jq .
```

**Response (healthy):**
```json
{
  "data": {
    "status": "ok",
    "database": { "status": "ok", "latency": "1.234ms" },
    "redis": { "status": "ok", "latency": "0.567ms" }
  }
}
```

**Response (degraded):** HTTP 503
```json
{
  "data": {
    "status": "degraded",
    "database": { "status": "error", "error": "database ping failed" },
    "redis": { "status": "ok", "latency": "0.567ms" }
  }
}
```

### Readiness Probe

Used by Kubernetes readiness probes and load balancers. Returns 200 only when ALL dependencies are reachable.

```bash
curl -s http://localhost:8080/api/ready | jq .
```

### Prometheus Metrics

```bash
curl -s http://localhost:8080/api/metrics
```

Exposes standard HTTP request metrics for Prometheus scraping.

### Docker Health Checks

The backend container has a built-in healthcheck that polls `/api/health` every 15 seconds:

```bash
# Check container health status
docker inspect --format='{{.State.Health.Status}}' veilence-mx-backend

# View recent health check logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' veilence-mx-backend
```

### Infrastructure Checks

```bash
# PostgreSQL connectivity
docker exec veilence-mx-db pg_isready -U veilence -d veilence_mx

# Redis connectivity
docker exec veilence-mx-redis redis-cli ping

# View container logs
docker logs veilence-mx-backend --tail 100 -f
docker logs veilence-mx-db --tail 50
docker logs veilence-mx-redis --tail 50

# Container resource usage
docker stats veilence-mx-backend veilence-mx-db veilence-mx-redis
```

---

## 3. User Management

### Prerequisites

All authenticated API calls require a JWT bearer token or API key. Obtain a token via login:

```bash
# Login and get tokens
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"YourPassword123!"}' \
  | jq -r '.data.accessToken')

# Or use an API key (admin scope)
API_KEY="vmx_your_api_key_here"
```

### Register a New User

```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newuser@example.com",
    "password": "SecurePass123!",
    "firstName": "Jane",
    "lastName": "Doe"
  }' | jq .
```

The user is automatically logged in after registration. Tokens are returned in the response.

### Get Current User Profile

```bash
curl -s http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### Update User Profile

```bash
curl -s -X PUT http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"firstName": "Jane", "lastName": "Smith"}' | jq .
```

### Change User Password

```bash
curl -s -X POST http://localhost:8080/api/auth/change-password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"currentPassword": "OldPass123!", "newPassword": "NewPass456!"}' | jq .
```

### Initiate Password Reset (for Forgotten Passwords)

```bash
# Always returns 200 to prevent user enumeration
curl -s -X POST http://localhost:8080/api/auth/forgot-password \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}' | jq .
```

> If SMTP is not configured, the reset token is logged to the server console instead of being emailed.

### Email Verification

```bash
# Trigger verification email
curl -s -X POST http://localhost:8080/api/auth/send-verification \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .

# Verify with token (received via email or server log)
curl -s -X POST http://localhost:8080/api/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{"token": "verification_token_here"}' | jq .
```

### Invite a User to an Organization

```bash
curl -s -X POST http://localhost:8080/api/orgs/$ORG_ID/invitations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"email": "invite@example.com", "roleId": 3}' | jq .
```

Role IDs (default roles):
- **1** = Owner (full permissions)
- **2** = Admin (manage members, settings, packages)
- **3** = Member (create packages, manage alerts)
- **4** = Viewer (read-only access)

### Change a Member's Role

```bash
curl -s -X PUT http://localhost:8080/api/orgs/$ORG_ID/members/$USER_ID/role \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"roleId": 2}' | jq .
```

> The organization owner's role cannot be changed.

### Remove a Member from Organization

```bash
curl -s -X DELETE http://localhost:8080/api/orgs/$ORG_ID/members/$USER_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

> The organization owner cannot be removed.

### Deactivate a User (Database Direct)

There is no soft-delete endpoint for users. To deactivate a user, remove them from all organizations or disable at the database level:

```sql
-- Connect to PostgreSQL
docker exec -it veilence-mx-db psql -U veilence -d veilence_mx

-- List all users
SELECT id, email, first_name, last_name, email_verified, created_at FROM users;

-- Remove user from all organizations
DELETE FROM org_members WHERE user_id = <USER_ID>;

-- Invalidate all refresh tokens (forces re-login which will fail without org membership)
DELETE FROM refresh_tokens WHERE user_id = <USER_ID>;
DELETE FROM sessions WHERE user_id = <USER_ID>;
```

### API Key Management

```bash
# Create an API key (scopes: read, write, admin)
curl -s -X POST http://localhost:8080/api/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"name": "CI/CD Pipeline", "scope": "read", "expiresAt": "2027-01-01T00:00:00Z"}' | jq .

# List API keys
curl -s http://localhost:8080/api/auth/api-keys \
  -H "Authorization: Bearer $TOKEN" | jq .

# Revoke an API key
curl -s -X DELETE http://localhost:8080/api/auth/api-keys/$KEY_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

> API key authentication is exempt from CSRF validation. Use the `Authorization: Bearer vmx_...` header.

### Session Management

```bash
# List active sessions for current user
curl -s http://localhost:8080/api/auth/sessions \
  -H "Authorization: Bearer $TOKEN" | jq .

# Revoke a specific session
curl -s -X DELETE http://localhost:8080/api/auth/sessions/$SESSION_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

---

## 4. Organization Management

### Create an Organization

```bash
curl -s -X POST http://localhost:8080/api/orgs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"name": "Acme Corp", "slug": "acme-corp", "description": "Supply chain monitoring"}' | jq .
```

The creating user automatically becomes the Owner.

### List User's Organizations

```bash
curl -s http://localhost:8080/api/orgs \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### Get Organization Details

```bash
curl -s http://localhost:8080/api/orgs/$ORG_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### Update Organization

```bash
curl -s -X PUT http://localhost:8080/api/orgs/$ORG_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"name": "Acme Corp Updated", "description": "Updated description"}' | jq .
```

### List Organization Members

```bash
curl -s http://localhost:8080/api/orgs/$ORG_ID/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### List Pending Invitations

```bash
curl -s http://localhost:8080/api/orgs/$ORG_ID/invitations \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### Revoke a Pending Invitation

```bash
curl -s -X DELETE http://localhost:8080/api/orgs/$ORG_ID/invitations/$INVITATION_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

### List Available Roles

```bash
curl -s http://localhost:8080/api/orgs/$ORG_ID/roles \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### List Permissions

```bash
curl -s http://localhost:8080/api/permissions \
  -H "Authorization: Bearer $TOKEN" | jq .
```

### Delete Organization (Soft Delete)

```bash
curl -s -X DELETE http://localhost:8080/api/orgs/$ORG_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

> Requires `org:delete` permission (Owner only).

### View Audit Logs

```bash
# All audit logs for an org
curl -s "http://localhost:8080/api/orgs/$ORG_ID/audit-logs?page=1&limit=50" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Filter by action and date range
curl -s "http://localhost:8080/api/orgs/$ORG_ID/audit-logs?action=create&resource=package&from_date=2026-04-01T00:00:00Z" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Filter by specific user
curl -s "http://localhost:8080/api/orgs/$ORG_ID/audit-logs?user_id=1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

Audit log actions include: `create`, `update`, `delete`, `invite`, `accept`, `remove`, `revoke`, `login`, `login_failed`, `logout`, `reset`.

---

## 5. Queue Monitoring & Dead Job Recovery

The job queue is Redis-backed and processes two job types:
- **diff** -- Generates diffs between package release versions
- **analyze** -- Sends diffs to the LLM for threat analysis

### View Queue Statistics

```bash
curl -s http://localhost:8080/api/queue/stats \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

**Response:**
```json
{
  "data": {
    "diff": { "pending": 5, "processing": 1, "completed": 150, "failed": 0, "dead": 2 },
    "analyze": { "pending": 3, "processing": 0, "completed": 148, "failed": 0, "dead": 0 }
  }
}
```

**Key indicators to watch:**
- `pending` growing consistently = workers not keeping up
- `processing` stuck at same count = possible stuck jobs (lock timeout: 10 min)
- `dead` increasing = recurring failures, investigate root cause

### View Dead Jobs

```bash
# View dead analysis jobs (default)
curl -s "http://localhost:8080/api/queue/dead?type=analyze" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# View dead diff jobs
curl -s "http://localhost:8080/api/queue/dead?type=diff" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

Each dead job includes:
- `id` -- Job ID
- `type` -- `diff` or `analyze`
- `referenceId` -- The release ID this job processes
- `attempts` -- Number of times the job was tried
- `lastError` -- The error that caused the final failure
- `createdAt` / `updatedAt` -- Timestamps (Unix epoch)

### Retry All Dead Jobs

```bash
# Retry all dead analysis jobs
curl -s -X POST "http://localhost:8080/api/queue/retry-dead?type=analyze" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .

# Retry all dead diff jobs
curl -s -X POST "http://localhost:8080/api/queue/retry-dead?type=diff" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

This resets the attempt counter and re-queues all dead jobs.

### Retry Backoff Strategy

Failed jobs are retried with exponential backoff: `30s * 2^(attempt-1)`

| Attempt | Delay   |
|---------|---------|
| 1       | 30s     |
| 2       | 60s     |
| 3       | 120s    |
| 4       | 240s    |
| 5       | Dead    |

After `QUEUE_MAX_RETRIES` (default 5) failed attempts, the job moves to the dead-letter queue.

### Stuck Job Recovery

The queue worker automatically recovers stuck jobs (processing for longer than `QUEUE_LOCK_TIMEOUT`, default 10 minutes). Stuck jobs are either re-queued (if under max attempts) or moved to dead-letter.

### Direct Redis Inspection

```bash
# Connect to Redis CLI
docker exec -it veilence-mx-redis redis-cli

# View pending job counts
LLEN veilence:queue:pending:diff
LLEN veilence:queue:pending:analyze

# View processing jobs (sorted set by timestamp)
ZRANGE veilence:queue:processing:diff 0 -1 WITHSCORES
ZRANGE veilence:queue:processing:analyze 0 -1 WITHSCORES

# View dead jobs
ZRANGE veilence:queue:dead:diff 0 -1 WITHSCORES
ZRANGE veilence:queue:dead:analyze 0 -1 WITHSCORES

# View a specific job's data
GET veilence:queue:job:<JOB_ID>

# View global stats
HGETALL veilence:queue:stats
```

### Queue Emergency: Clear All Jobs

> **WARNING: Data loss!** Only use in emergencies.

```bash
docker exec -it veilence-mx-redis redis-cli

# Clear all pending diff jobs
DEL veilence:queue:pending:diff
# Clear all pending analyze jobs
DEL veilence:queue:pending:analyze
# Clear dead letter queues
DEL veilence:queue:dead:diff
DEL veilence:queue:dead:analyze
# Reset stats
DEL veilence:queue:stats
```

---

## 6. Alert Triage Workflow

### Understanding Alert Severity

| Severity   | Description                                              |
|------------|----------------------------------------------------------|
| `critical` | Confirmed malicious code, backdoors, credential theft    |
| `high`     | Suspicious patterns, obfuscated code, unusual network    |
| `medium`   | Minor suspicious patterns, dependency changes            |
| `low`      | Informational, cosmetic changes, version bumps           |

### Alert Statuses

| Status         | Description                                    |
|----------------|------------------------------------------------|
| `new`          | Freshly generated, not yet reviewed            |
| `acknowledged` | Reviewed by a team member, investigation active|
| `resolved`     | Investigation complete, no further action      |

### List Alerts

```bash
# All alerts for the org
curl -s "http://localhost:8080/api/alerts?page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Filter by severity
curl -s "http://localhost:8080/api/alerts?severity=critical&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Filter by status (new alerts only)
curl -s "http://localhost:8080/api/alerts?status=new&page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Sort by severity descending
curl -s "http://localhost:8080/api/alerts?sort_by=severity&sort_dir=desc" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### Update Alert Status

```bash
# Acknowledge an alert
curl -s -X PATCH http://localhost:8080/api/alerts/$ALERT_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"status": "acknowledged"}' | jq .

# Resolve an alert
curl -s -X PATCH http://localhost:8080/api/alerts/$ALERT_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"status": "resolved"}' | jq .
```

### Recommended Triage Process

1. **Review critical alerts first**: `?severity=critical&status=new`
2. **Acknowledge** the alert to signal you are investigating
3. **Examine the package release** -- view the diff and analysis in the dashboard
4. **Cross-reference** with the package's registry page (PyPI/npm)
5. **Resolve** if benign, or escalate if confirmed malicious
6. Check the audit log to see who reviewed what: `?resource=alert`

### Reanalyze All Releases

If the LLM model is updated or analysis rules change, trigger a full reanalysis:

```bash
curl -s -X POST http://localhost:8080/api/sync/reanalyze \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

> This creates new analyze jobs for ALL releases. Monitor queue depth after triggering.

---

## 7. Package Management

### List Monitored Packages

```bash
# All packages
curl -s "http://localhost:8080/api/packages?page=1&limit=50" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Filter by registry
curl -s "http://localhost:8080/api/packages?registry=pypi" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Only custom packages (not top-N)
curl -s "http://localhost:8080/api/packages?is_custom=true" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Search by name
curl -s "http://localhost:8080/api/packages?search=requests" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### Add a Custom Package to Monitor

```bash
curl -s -X POST http://localhost:8080/api/packages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"name": "my-internal-package", "registry": "npm"}' | jq .
```

Registry values: `pypi` or `npm`.

### Remove a Package from Monitoring

```bash
curl -s -X DELETE http://localhost:8080/api/packages/$PACKAGE_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

### View Package Releases

```bash
curl -s "http://localhost:8080/api/packages/$PACKAGE_ID/releases?page=1&limit=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### Sync Top Packages

Trigger an immediate sync of the top-N packages from registries:

```bash
# Both registries
curl -s -X POST http://localhost:8080/api/sync/top-packages \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .

# PyPI only
curl -s -X POST "http://localhost:8080/api/sync/top-packages?registry=pypi" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .

# npm only
curl -s -X POST "http://localhost:8080/api/sync/top-packages?registry=npm" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

---

## 8. Notification Configuration

### Notification Channels

Supported channel types: `email`, `slack`, `webhook`.

```bash
# List notification channels for an org
curl -s http://localhost:8080/api/orgs/$ORG_ID/notification-channels \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Create a Slack channel
curl -s -X POST http://localhost:8080/api/orgs/$ORG_ID/notification-channels \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"name": "Security Slack", "type": "slack", "config": "https://hooks.slack.com/services/T.../B.../xxx"}' | jq .

# Create a webhook channel (HMAC-signed)
curl -s -X POST http://localhost:8080/api/orgs/$ORG_ID/notification-channels \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"name": "PagerDuty", "type": "webhook", "config": "https://events.pagerduty.com/v2/enqueue"}' | jq .

# Update a channel (toggle active/inactive)
curl -s -X PUT http://localhost:8080/api/orgs/$ORG_ID/notification-channels/$CHANNEL_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"name": "Security Slack", "config": "https://hooks.slack.com/...", "isActive": false}' | jq .

# Delete a channel
curl -s -X DELETE http://localhost:8080/api/orgs/$ORG_ID/notification-channels/$CHANNEL_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

### Notification Rules

Rules connect channels to severity thresholds. When an alert of the specified severity (or higher) is generated, all matching channels are notified.

```bash
# List rules
curl -s http://localhost:8080/api/orgs/$ORG_ID/notification-rules \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .

# Create a rule: notify Slack for critical alerts
curl -s -X POST http://localhost:8080/api/orgs/$ORG_ID/notification-rules \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"channelId": 1, "severity": "critical"}' | jq .

# Delete a rule
curl -s -X DELETE http://localhost:8080/api/orgs/$ORG_ID/notification-rules/$RULE_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

### User Notifications

```bash
# List notifications for current user (across all orgs)
curl -s http://localhost:8080/api/notifications \
  -H "Authorization: Bearer $TOKEN" | jq .

# Only unread
curl -s "http://localhost:8080/api/notifications?unread=true" \
  -H "Authorization: Bearer $TOKEN" | jq .

# Get unread count
curl -s http://localhost:8080/api/notifications/unread-count \
  -H "Authorization: Bearer $TOKEN" | jq .

# Mark a single notification as read
curl -s -X PUT http://localhost:8080/api/notifications/$NOTIFICATION_ID/read \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .

# Mark all notifications as read
curl -s -X PUT http://localhost:8080/api/notifications/read-all \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN" | jq .
```

---

## 9. Settings Management

### View Current Settings

```bash
curl -s http://localhost:8080/api/settings \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Org-ID: $ORG_ID" | jq .
```

### Update Settings

```bash
curl -s -X PUT http://localhost:8080/api/settings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Org-ID: $ORG_ID" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{
    "monitoring_interval": "30m",
    "discovery_scan_depth": "200",
    "diff_size_limit": "204800"
  }' | jq .
```

### Available Settings Keys

| Key                                | Values                      | Default    | Description                                      |
|------------------------------------|-----------------------------|------------|--------------------------------------------------|
| `monitoring_interval`              | Duration (e.g., `1h`)       | `1h`       | How often to check packages for new releases     |
| `discovery_interval`               | Duration (e.g., `24h`)      | `24h`      | How often to run discovery scans                 |
| `discovery_scan_depth`             | Integer (1-1000)            | `50`       | Top N packages per ecosystem per cycle           |
| `diff_size_limit`                  | Integer (bytes, 1024-10MB)  | `102400`   | Max diff size stored per release                 |
| `discovery_auto_approve`           | `true` or `false`           | `false`    | Auto-approve discovered packages for monitoring  |
| `stale_auto_remove_months`         | Integer (0-36)              | `0`        | Remove packages with no updates after N months   |
| `package_count_warning_threshold`  | Integer (0-100000)          | `0`        | Warn when monitored packages exceed this count   |
| `email_digest_enabled`             | `true` or `false`           | `false`    | Enable periodic email digest                     |
| `email_digest_frequency`           | `daily` or `weekly`         | `daily`    | Digest email frequency                           |
| `email_digest_recipients`          | Comma-separated emails      | -          | Digest recipient addresses                       |

> Settings are workspace-scoped. Each workspace has independent configuration.

---

## 10. Database Maintenance

### Connecting to PostgreSQL

```bash
# Via Docker
docker exec -it veilence-mx-db psql -U veilence -d veilence_mx

# Direct connection
psql "postgres://veilence:veilence_dev@localhost:5432/veilence_mx?sslmode=disable"
```

### Database Migrations

Veilence-MX uses embedded versioned migrations via `golang-migrate`. Migrations run automatically on server startup. The current migration files:

| Migration | Description                                |
|-----------|--------------------------------------------|
| 000001    | Initial schema (all tables)                |
| 000002    | Add API key prefix index                   |
| 000003    | Password reset and email verification      |
| 000004    | Hash invitation tokens                     |
| 000005    | API key scoping and sessions               |

Check current migration version:

```sql
SELECT * FROM schema_migrations;
```

### Useful Diagnostic Queries

```sql
-- Count users
SELECT COUNT(*) FROM users;

-- Count packages per org
SELECT org_id, COUNT(*) as package_count FROM packages GROUP BY org_id;

-- Count releases by status
SELECT status, COUNT(*) FROM releases GROUP BY status;

-- Count alerts by severity and status
SELECT severity, status, COUNT(*) FROM alerts GROUP BY severity, status ORDER BY severity, status;

-- View recent audit log entries
SELECT action, resource, details, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT 20;

-- Find organizations with the most packages
SELECT o.name, o.slug, COUNT(p.id) as packages
FROM organizations o
LEFT JOIN packages p ON p.org_id = o.id
GROUP BY o.id
ORDER BY packages DESC;

-- Check for orphaned releases (package deleted)
SELECT r.id, r.version, r.package_id
FROM releases r
LEFT JOIN packages p ON p.id = r.package_id
WHERE p.id IS NULL;

-- View notification channels and their status
SELECT nc.id, nc.name, nc.type, nc.is_active, nc.org_id
FROM notification_channels nc
ORDER BY nc.org_id, nc.name;
```

### Table Size and Index Health

```sql
-- Table sizes
SELECT
  relname AS table_name,
  pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
  pg_size_pretty(pg_relation_size(relid)) AS data_size,
  pg_size_pretty(pg_indexes_size(relid)) AS index_size
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;

-- Index usage (identify unused indexes)
SELECT
  schemaname, relname, indexrelname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC;

-- Long-running queries
SELECT pid, age(clock_timestamp(), query_start), usename, query
FROM pg_stat_activity
WHERE query != '<IDLE>' AND query NOT ILIKE '%pg_stat_activity%'
ORDER BY query_start ASC;
```

### Database Cleanup

```sql
-- Delete old audit logs (older than 90 days)
DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days';

-- Clean up expired invitation tokens
DELETE FROM invitations WHERE expires_at < NOW() AND accepted_at IS NULL;

-- Delete completed job references older than 7 days (from queue stats perspective)
-- Jobs are stored in Redis with TTLs, but if you need to clean release statuses:
UPDATE releases SET status = 'analyzed' WHERE status = 'pending' AND created_at < NOW() - INTERVAL '7 days';

-- Vacuum and analyze after large deletes
VACUUM ANALYZE;
```

---

## 11. Log Analysis

Veilence-MX uses Go's structured logging (`log/slog`) with key-value pairs.

### Viewing Logs

```bash
# Follow backend logs
docker logs veilence-mx-backend -f

# Last 200 lines
docker logs veilence-mx-backend --tail 200

# Since a specific time
docker logs veilence-mx-backend --since 2026-04-11T10:00:00Z
```

### Key Log Messages and Their Meaning

| Log Message                        | Level | Meaning                                         |
|------------------------------------|-------|--------------------------------------------------|
| `starting poller`                  | INFO  | Poller goroutines starting                       |
| `polling registry`                 | INFO  | Polling cycle beginning for a registry           |
| `polling complete`                 | INFO  | Polling cycle finished, includes package count   |
| `new release detected`             | INFO  | New package version found                        |
| `diff job enqueued`                | INFO  | Diff job added to Redis queue                    |
| `job scheduled for retry`          | INFO  | Failed job will retry after backoff              |
| `job moved to dead-letter queue`   | WARN  | Job exceeded max retries                         |
| `recovered stuck job`              | INFO  | Stuck job re-queued by recovery mechanism        |
| `stuck job moved to dead-letter`   | WARN  | Stuck job that exceeded max attempts             |
| `poller settings cache invalidated`| INFO  | Settings were updated, cache cleared             |
| `failed to check package`          | ERROR | Individual package check failed during polling   |
| `failed to enqueue diff job`       | ERROR | Redis enqueue failed                             |
| `failed to get diff queue stats`   | ERROR | Redis stats query failed                         |
| `failed to generate CSRF token`    | ERROR | Crypto random generation failed (system issue)   |

### Filtering Logs

```bash
# Find errors only
docker logs veilence-mx-backend 2>&1 | grep -i "error"

# Find poller activity
docker logs veilence-mx-backend 2>&1 | grep "polling\|poller"

# Find dead-letter events
docker logs veilence-mx-backend 2>&1 | grep "dead-letter\|dead"

# Find authentication events
docker logs veilence-mx-backend 2>&1 | grep "login\|logout\|register"

# Find specific package activity
docker logs veilence-mx-backend 2>&1 | grep "requests"  # package name
```

---

## 12. Common Error Resolutions

### CSRF Token Errors (403 Forbidden)

**Symptom:** API calls return `{"error": "CSRF token missing"}` or `{"error": "CSRF token mismatch"}`

**Cause:** The double-submit cookie pattern requires:
1. A `_csrf_token` cookie (set automatically on GET requests)
2. An `X-CSRF-Token` header matching the cookie value

**Resolution:**
- Ensure the frontend makes a GET request first (e.g., `/api/health`) to receive the CSRF cookie
- For API clients, extract `_csrf_token` from response cookies and send it as `X-CSRF-Token` header
- Pre-auth routes (login, register, forgot-password, reset-password) are exempt from CSRF
- API key authenticated requests are exempt from CSRF
- If using `curl`, pass `-b` and `-c` flags to manage cookies:

```bash
# Step 1: Get CSRF cookie
curl -s -c cookies.txt http://localhost:8080/api/health > /dev/null

# Step 2: Extract token and use it
CSRF=$(grep _csrf_token cookies.txt | awk '{print $NF}')
curl -s -X POST http://localhost:8080/api/some-endpoint \
  -b cookies.txt \
  -H "X-CSRF-Token: $CSRF" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq .
```

### Authentication Failures (401 Unauthorized)

**Symptom:** `"authentication required"` on protected endpoints

**Resolution:**
- Check that the JWT token is not expired (15-minute access token lifetime)
- Refresh tokens using `POST /api/auth/refresh` with the refresh token
- Verify `JWT_SECRET` has not changed (or add old secret to `JWT_SECRET_PREVIOUS`)
- Check if the user's session was revoked

### JWT Secret Rotation

When rotating the JWT secret:

1. Move the current secret to `JWT_SECRET_PREVIOUS`
2. Set `JWT_SECRET` to the new value
3. Restart the backend
4. Remove `JWT_SECRET_PREVIOUS` after 15 minutes (access token duration)

```bash
# Example: rotate JWT secret
# In .env or docker-compose:
JWT_SECRET_PREVIOUS=old-secret-here
JWT_SECRET=new-secret-generated-with-openssl-rand-hex-32
```

### Organization Context Errors (400 Bad Request)

**Symptom:** `"organization context required"` on org-scoped endpoints

**Resolution:**
- Include the `X-Org-ID` header with a valid organization ID
- Or pass `?org_id=<ID>` as a query parameter
- Verify the user is a member of the specified organization

### Poller Not Picking Up New Packages

**Symptom:** Newly added packages show no releases

**Resolution:**
1. Check poller logs: `docker logs veilence-mx-backend 2>&1 | grep polling`
2. Verify the package exists in the correct registry (PyPI/npm)
3. Check settings for the org (poll interval, version depth)
4. Trigger manual sync: `POST /api/sync/top-packages`
5. Check if the poller is scoping to the correct org

### Queue Backlog (Jobs Not Processing)

**Symptom:** Pending queue count keeps growing, analysis not completing

**Resolution:**
1. Check queue stats: `GET /api/queue/stats`
2. Check Redis connectivity: `docker exec veilence-mx-redis redis-cli ping`
3. Check for stuck jobs in processing set (processing count stuck)
4. Check LLM API connectivity (if analyze jobs failing): verify `COPILOT_API_URL`
5. View dead jobs for error patterns: `GET /api/queue/dead`
6. Retry dead jobs after fixing root cause: `POST /api/queue/retry-dead`
7. If Redis is overloaded, check memory: `docker exec veilence-mx-redis redis-cli info memory`

### Redis Connection Issues

```bash
# Check Redis is running
docker exec veilence-mx-redis redis-cli ping

# Check Redis memory usage
docker exec veilence-mx-redis redis-cli info memory

# Check connected clients
docker exec veilence-mx-redis redis-cli info clients

# Redis is configured with maxmemory 256mb and noeviction policy
# If memory is full, no new jobs can be enqueued
docker exec veilence-mx-redis redis-cli config get maxmemory
```

### Database Connection Issues

```bash
# Check PostgreSQL is running
docker exec veilence-mx-db pg_isready -U veilence -d veilence_mx

# Check active connections
docker exec -it veilence-mx-db psql -U veilence -d veilence_mx \
  -c "SELECT count(*) FROM pg_stat_activity WHERE datname = 'veilence_mx';"

# Check for locks
docker exec -it veilence-mx-db psql -U veilence -d veilence_mx \
  -c "SELECT pid, query, state, wait_event_type FROM pg_stat_activity WHERE state != 'idle' AND datname = 'veilence_mx';"
```

### Rate Limiting (429 Too Many Requests)

**Symptom:** API returns 429 status

**Resolution:**
- API endpoints: 100 requests/minute per IP
- Auth endpoints: 10 requests/minute per IP
- Sync endpoints: 5 requests/minute per IP
- Wait for the rate limit window to reset (1 minute)
- For automated clients, implement exponential backoff

### Production Build Failures

```bash
# Backend
cd backend && go build ./...
go vet ./...

# Frontend
cd frontend && npm run build
npm run lint
```

---

## 13. Backup & Recovery

### PostgreSQL Backup

```bash
# Full database dump
docker exec veilence-mx-db pg_dump -U veilence veilence_mx > backup_$(date +%Y%m%d_%H%M%S).sql

# Compressed backup
docker exec veilence-mx-db pg_dump -U veilence -Fc veilence_mx > backup_$(date +%Y%m%d_%H%M%S).dump

# Backup specific tables
docker exec veilence-mx-db pg_dump -U veilence -t users -t organizations -t org_members veilence_mx > users_backup.sql
```

### PostgreSQL Restore

```bash
# Restore from SQL dump
cat backup.sql | docker exec -i veilence-mx-db psql -U veilence -d veilence_mx

# Restore from compressed dump
docker exec -i veilence-mx-db pg_restore -U veilence -d veilence_mx < backup.dump
```

### Redis Backup

Redis is configured with `appendonly yes` for persistence. The AOF file is stored in the `redisdata` Docker volume.

```bash
# Trigger manual RDB snapshot
docker exec veilence-mx-redis redis-cli BGSAVE

# Copy Redis data directory
docker cp veilence-mx-redis:/data ./redis-backup-$(date +%Y%m%d)
```

### Automated Backup Script

```bash
#!/bin/bash
# backup.sh - Run daily via cron
BACKUP_DIR="/backups/veilence-mx"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# PostgreSQL
docker exec veilence-mx-db pg_dump -U veilence -Fc veilence_mx > "$BACKUP_DIR/db_$DATE.dump"

# Redis
docker exec veilence-mx-redis redis-cli BGSAVE
sleep 2
docker cp veilence-mx-redis:/data/appendonly.aof "$BACKUP_DIR/redis_aof_$DATE"

# Retain last 30 days
find "$BACKUP_DIR" -type f -mtime +30 -delete

echo "Backup completed: $DATE"
```

### Disaster Recovery Checklist

1. Restore PostgreSQL from latest backup
2. Verify migration version matches: `SELECT * FROM schema_migrations;`
3. Start Redis (queue will be empty -- this is acceptable)
4. Start backend -- migrations run automatically
5. Verify health: `curl http://localhost:8080/api/health`
6. Re-trigger top package sync for each org
7. Monitor queue for normal processing
8. Verify frontend connectivity

---

*Veilence-MX v1.0.0 Admin Runbook | Generated 2026-04-11*
