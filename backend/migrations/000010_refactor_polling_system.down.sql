-- v1.1.0: Reverse migration (best-effort)

ALTER TABLE packages ADD COLUMN is_custom BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE packages ADD COLUMN deleted_at TIMESTAMPTZ;

UPDATE packages SET is_custom = true WHERE source = 'manual';
UPDATE packages SET is_custom = false WHERE source = 'discovered';
UPDATE packages SET deleted_at = NOW() WHERE status = 'removed';

ALTER TABLE packages DROP COLUMN source;
ALTER TABLE packages DROP COLUMN status;
ALTER TABLE packages DROP COLUMN blocked_at;
ALTER TABLE packages DROP COLUMN blocked_reason;

DROP INDEX IF EXISTS idx_packages_status;
DROP INDEX IF EXISTS idx_packages_org_status;

-- Settings rollback is lossy -- old keys won't have exact original values
DELETE FROM settings WHERE key IN (
    'discovery_scan_depth', 'discovery_interval',
    'monitoring_interval'
);
