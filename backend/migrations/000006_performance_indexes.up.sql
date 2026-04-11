-- Migration 000006: Add composite indexes for commonly queried columns.
-- These indexes target the query patterns used by the dashboard, alerts list,
-- release browsing, and notification endpoints.

-- releases: commonly queried by package_id + published_at (dashboard charts, recent releases)
CREATE INDEX IF NOT EXISTS idx_releases_package_published
    ON releases(package_id, published_at DESC);

-- releases: classification filtering joins through analyses; index package_id + status for dashboard
-- (The "classification" column lives on analyses, not releases, so we index the join-friendly
-- pattern: diffs and analyses are indexed by diff_id/release_id already.)

-- alerts: filtered by org_id + status (alert list page with status filter)
CREATE INDEX IF NOT EXISTS idx_alerts_org_status
    ON alerts(org_id, status);

-- alerts: filtered by org_id + severity (alert list page with severity filter, charts)
CREATE INDEX IF NOT EXISTS idx_alerts_org_severity
    ON alerts(org_id, severity);

-- notifications: filtered by org_id + user_id + read status (unread count, notification list)
CREATE INDEX IF NOT EXISTS idx_notifications_org_user_read
    ON notifications(org_id, user_id, is_read);

-- packages: filtered by org_id + registry (package list with registry filter)
CREATE INDEX IF NOT EXISTS idx_packages_org_registry
    ON packages(org_id, registry);

-- releases: status-based queries scoped through package (pending analysis count)
CREATE INDEX IF NOT EXISTS idx_releases_status
    ON releases(status);

-- audit_logs: commonly filtered by org_id + created_at (paginated listing)
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_created
    ON audit_logs(org_id, created_at DESC);
