# Veilence-MX Production Deployment Guide

This guide covers deploying Veilence-MX v1.0.0 in a production environment.

## Prerequisites

| Component     | Minimum Version | Notes                                      |
|---------------|-----------------|---------------------------------------------|
| Go            | 1.23+           | Backend compilation                         |
| Node.js       | 20+             | Frontend build                              |
| PostgreSQL    | 16              | Primary database                            |
| Redis         | 7               | Job queue and caching                       |
| Docker Engine | 24+             | Container runtime (if using Docker Compose) |
| Docker Compose| 2.20+           | Orchestration                               |

## Architecture Overview

```
┌─────────┐     ┌──────────┐     ┌────────────┐     ┌───────┐
│ Browser  │────▶│ Frontend │────▶│  Backend   │────▶│  LLM  │
│          │     │ (Next.js)│     │ (Go / Chi) │     │ Proxy │
└─────────┘     └──────────┘     └─────┬──┬───┘     └───────┘
                                       │  │
                                 ┌─────▼┐ └──▶┌───────┐
                                 │Postgres│    │ Redis │
                                 └───────┘    └───────┘
```

## Environment Variables

### Backend (`backend/.env`)

| Variable                | Required | Default                  | Description                                                      |
|-------------------------|----------|--------------------------|------------------------------------------------------------------|
| `DATABASE_URL`          | Yes      | —                        | PostgreSQL connection string. Example: `postgres://user:pass@host:5432/veilence_mx?sslmode=require` |
| `REDIS_URL`             | Yes      | —                        | Redis connection URL. Example: `redis://:password@host:6379/0`   |
| `JWT_SECRET`            | Yes      | —                        | HMAC secret for JWT signing. **Must be ≥ 32 random bytes.** Generate: `openssl rand -hex 32` |
| `JWT_SECRET_PREVIOUS`   | No       | (empty)                  | Comma-separated previous JWT secrets for key rotation. Remove after 15 min. |
| `SERVER_PORT`           | No       | `8080`                   | HTTP server port                                                 |
| `FRONTEND_URL`          | Yes      | —                        | Frontend origin for CORS/CSP (e.g., `https://app.example.com`)  |
| `APP_ENV`               | No       | `development`            | Set to `production` for strict security defaults                 |
| `COPILOT_API_URL`       | Yes      | —                        | LLM analysis proxy base URL                                     |
| `COPILOT_MODEL`         | No       | `claude-sonnet-4.6`     | Model name for analysis                                          |
| `PYPI_POLL_INTERVAL`    | No       | `5m`                     | How often to poll PyPI for new releases                          |
| `NPM_POLL_INTERVAL`     | No       | `5m`                     | How often to poll npm for new releases                           |
| `PYPI_TOP_N`            | No       | `100`                    | Number of top PyPI packages to monitor                           |
| `NPM_TOP_N`             | No       | `100`                    | Number of top npm packages to monitor                            |
| `TOP_N_REFRESH_INTERVAL`| No       | `24h`                    | How often to refresh the top-N package list                      |
| `POLLER_CONCURRENCY`    | No       | `5`                      | Maximum concurrent package fetches                               |
| `DIFF_SIZE_LIMIT`       | No       | `102400`                 | Max diff size in bytes to send for LLM analysis                  |
| `VERSION_DEPTH_MODE`    | No       | `latest`                 | `latest` (newest only) or `count` (N most recent)                |
| `VERSION_DEPTH_COUNT`   | No       | `3`                      | Number of versions when `VERSION_DEPTH_MODE=count`               |
| `QUEUE_MAX_RETRIES`     | No       | `5`                      | Max retries before moving job to dead letter queue                |
| `QUEUE_LOCK_TIMEOUT`    | No       | `10m`                    | Max time a job can be locked before being considered stuck        |
| `SMTP_HOST`             | No       | (empty)                  | SMTP server for sending emails. Leave blank to log tokens to console |
| `SMTP_PORT`             | No       | `587`                    | SMTP port                                                        |
| `SMTP_USERNAME`         | No       | (empty)                  | SMTP username                                                    |
| `SMTP_PASSWORD`         | No       | (empty)                  | SMTP password                                                    |
| `SMTP_FROM`             | No       | (empty)                  | Sender email address                                             |
| `SMTP_USE_TLS`          | No       | `true`                   | Use TLS for SMTP connections                                     |

### Frontend (`frontend/.env.local`)

| Variable              | Required | Default                  | Description                                |
|-----------------------|----------|--------------------------|--------------------------------------------|
| `NEXT_PUBLIC_API_URL` | Yes      | —                        | Backend API base URL (e.g., `https://api.example.com`) |

## Docker Compose Production Setup

### 1. Prepare Environment Files

```bash
# Clone and enter the repo
git clone <repo-url> veilence-mx && cd veilence-mx

# Create production environment file
cp backend/.env.example .env

# Edit .env with production values — especially:
#   DATABASE_URL, REDIS_URL, JWT_SECRET, FRONTEND_URL, APP_ENV=production
```

### 2. Deploy with Docker Compose

```bash
# Build and start all services
docker compose -f docker-compose.prod.yml up -d --build

# Check all services are healthy
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f backend
```

### 3. Verify Deployment

```bash
# Health check
curl -s http://localhost:8080/api/health | jq .

# Readiness check (used by load balancers)
curl -s http://localhost:8080/api/ready | jq .
```

## Database Setup

### Migrations

Migrations run automatically on backend startup via golang-migrate. The migration files are embedded in the backend binary.

```bash
# Verify current migration version (from within the backend container)
docker compose -f docker-compose.prod.yml exec backend ./server migrate-version

# Manual rollback (if needed)
docker compose -f docker-compose.prod.yml exec backend ./server migrate-down
```

### Initial Data

On first run, the backend automatically:
1. Runs all pending SQL migrations
2. Seeds system permissions (RBAC)

The first registered user should create an organization to begin monitoring packages.

### Connection Pool Settings

The backend configures PostgreSQL connection pooling:
- `MaxOpenConns`: 25
- `MaxIdleConns`: 5

Adjust these via source code if your workload requires different limits.

## Redis Configuration

The production Docker Compose configures Redis with:

```
redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy noeviction
```

- **`appendonly yes`**: Persistence via AOF for job queue durability
- **`maxmemory 256mb`**: Memory cap (increase for large queue volumes)
- **`noeviction`**: Prevents key eviction — returns errors if memory is full (critical for job queue integrity)

### Redis Password (Production)

Set a Redis password in your `.env`:

```
REDIS_URL=redis://:your-strong-password@redis:6379/0
```

Configure the Redis container accordingly:

```yaml
command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy noeviction --requirepass your-strong-password
```

## TLS/SSL Setup

Veilence-MX does not terminate TLS directly. Use a reverse proxy.

### Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name app.example.com;

    ssl_certificate     /etc/ssl/certs/veilence.crt;
    ssl_certificate_key /etc/ssl/private/veilence.key;
    ssl_protocols       TLSv1.3;

    # Frontend
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Backend API
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Increase timeout for LLM analysis endpoints
        proxy_read_timeout 120s;
    }
}
```

### Caddy Example

```caddyfile
app.example.com {
    handle /api/* {
        reverse_proxy localhost:8080
    }
    handle {
        reverse_proxy localhost:3000
    }
}
```

## Backup Procedures

### PostgreSQL

```bash
# Full backup
docker compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U veilence -d veilence_mx --format=custom \
  > backup_$(date +%Y%m%d_%H%M%S).dump

# Restore
docker compose -f docker-compose.prod.yml exec -i postgres \
  pg_restore -U veilence -d veilence_mx --clean --if-exists \
  < backup_20260411_120000.dump
```

**Recommended schedule**: Daily full backups with 30-day retention.

### Redis

Redis is configured with AOF persistence. The data file is stored in the `redisdata` Docker volume.

```bash
# Trigger a snapshot
docker compose -f docker-compose.prod.yml exec redis redis-cli BGSAVE

# Copy the dump file
docker cp veilence-mx-redis:/data/dump.rdb ./redis_backup_$(date +%Y%m%d).rdb
```

**Note**: Redis data is primarily the job queue. Losing Redis data means in-flight jobs restart, but no permanent data loss occurs (PostgreSQL is the source of truth).

### Automated Backup Script

```bash
#!/bin/bash
# backup.sh — run via cron: 0 2 * * * /opt/veilence-mx/backup.sh
set -euo pipefail

BACKUP_DIR="/opt/backups/veilence-mx"
RETENTION_DAYS=30
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

# PostgreSQL backup
docker compose -f /opt/veilence-mx/docker-compose.prod.yml exec -T postgres \
  pg_dump -U veilence -d veilence_mx --format=custom \
  > "$BACKUP_DIR/pg_$TIMESTAMP.dump"

# Redis snapshot
docker compose -f /opt/veilence-mx/docker-compose.prod.yml exec -T redis redis-cli BGSAVE
sleep 5
docker cp veilence-mx-redis:/data/dump.rdb "$BACKUP_DIR/redis_$TIMESTAMP.rdb"

# Cleanup old backups
find "$BACKUP_DIR" -type f -mtime +$RETENTION_DAYS -delete

echo "Backup completed: $TIMESTAMP"
```

## Monitoring

### Prometheus Metrics Endpoint

The backend exposes Prometheus metrics at `GET /api/metrics`:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'veilence-mx-backend'
    scrape_interval: 15s
    static_configs:
      - targets: ['backend:8080']
    metrics_path: '/api/metrics'
```

### Health Check Endpoints

| Endpoint       | Purpose                                    | Use For                      |
|----------------|--------------------------------------------|------------------------------|
| `GET /api/health` | Full health check (DB + Redis latency)  | Monitoring dashboards        |
| `GET /api/ready`  | Readiness probe (200 or 503)            | K8s readiness, load balancers |

### Key Metrics to Monitor

- **HTTP request latency** (p50, p95, p99) — via Prometheus
- **Queue depth** — `GET /api/queue/stats` (pending + processing counts)
- **Dead letter queue** — `GET /api/queue/dead` (should be ~0 in steady state)
- **Database connections** — PostgreSQL `pg_stat_activity`
- **Redis memory usage** — `redis-cli INFO memory`
- **Alert volume** — dashboard stats (sudden spikes may indicate compromise)

### Alerting Recommendations

| Condition                          | Severity | Action                           |
|------------------------------------|----------|----------------------------------|
| `/api/ready` returns 503           | Critical | Check DB/Redis connectivity      |
| Dead letter queue > 10 jobs        | High     | Check LLM proxy, review errors   |
| Pending analysis queue > 500       | Medium   | Scale analysis concurrency       |
| New malicious classification       | High     | Review alert in dashboard        |
| Disk usage > 80%                   | Medium   | Expand storage, run cleanup      |

## Troubleshooting

### Backend Won't Start

1. **Check logs**: `docker compose -f docker-compose.prod.yml logs backend`
2. **Database connection**: Verify `DATABASE_URL` is correct and PostgreSQL is reachable
3. **Migration failure**: Check for dirty migration state — `migrate-version` shows `dirty=true`
4. **JWT_SECRET missing**: Must be set in production (`APP_ENV=production`)

### Frontend Can't Reach Backend

1. Verify `NEXT_PUBLIC_API_URL` points to the correct backend URL
2. Check CORS: `FRONTEND_URL` in backend must match the frontend origin exactly
3. Check network isolation: frontend must be able to reach backend on the configured port

### Queue Jobs Stuck

1. Check Redis connectivity: `docker compose exec redis redis-cli ping`
2. Check dead letter queue: `curl http://localhost:8080/api/queue/dead`
3. Retry dead jobs: `curl -X POST http://localhost:8080/api/queue/retry-dead`
4. Check `QUEUE_LOCK_TIMEOUT` — stuck jobs exceed this duration

### LLM Analysis Failures

1. Verify `COPILOT_API_URL` is reachable from the backend container
2. Check `DIFF_SIZE_LIMIT` — diffs exceeding this are skipped
3. Review dead letter queue for error messages
4. Check the LLM proxy logs for rate limiting or auth issues

### High Memory Usage

1. Check Redis: `docker compose exec redis redis-cli INFO memory`
2. Check PostgreSQL connections: `SELECT count(*) FROM pg_stat_activity;`
3. Review `MaxOpenConns` (default 25) — may need tuning for high-traffic deployments

### Database Performance

The backend includes performance-optimized composite indexes (migration 000006) for:
- Releases by package + published date
- Alerts by org + status, org + severity
- Notifications by org + user + read status
- Packages by org + registry
- Audit logs by org + created date

Run `EXPLAIN ANALYZE` on slow queries to verify index usage.
