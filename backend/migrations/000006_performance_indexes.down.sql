-- Rollback migration 000006: Remove performance indexes.

DROP INDEX IF EXISTS idx_releases_package_published;
DROP INDEX IF EXISTS idx_alerts_org_status;
DROP INDEX IF EXISTS idx_alerts_org_severity;
DROP INDEX IF EXISTS idx_notifications_org_user_read;
DROP INDEX IF EXISTS idx_packages_org_registry;
DROP INDEX IF EXISTS idx_releases_status;
DROP INDEX IF EXISTS idx_audit_logs_org_created;
