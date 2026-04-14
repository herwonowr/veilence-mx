-- v1.1.0: Refactor polling system — unified monitoring/discovery, package status model
-- See _workspace/specs/v1.1.0-sprint-plan.md Section 5

-- 1. Add new columns to packages table
ALTER TABLE packages ADD COLUMN source VARCHAR(20) NOT NULL DEFAULT 'manual';
ALTER TABLE packages ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';
ALTER TABLE packages ADD COLUMN blocked_at TIMESTAMPTZ;
ALTER TABLE packages ADD COLUMN blocked_reason TEXT;

-- 2. Migrate IsCustom -> Source
UPDATE packages SET source = 'discovered' WHERE is_custom = false;
UPDATE packages SET source = 'manual' WHERE is_custom = true;

-- 3. Migrate soft-deleted packages to 'removed' status
UPDATE packages SET status = 'removed' WHERE deleted_at IS NOT NULL;

-- 4. Remove old columns
ALTER TABLE packages DROP COLUMN is_custom;
ALTER TABLE packages DROP COLUMN deleted_at;

-- 5. Add indexes on status for efficient filtering
CREATE INDEX idx_packages_status ON packages (status);
CREATE INDEX idx_packages_org_status ON packages (org_id, status);

-- 6. Migrate settings: rename old keys to new keys
-- monitoring_interval: use the python_poll_interval value (or default)
INSERT INTO settings (org_id, key, value, created_at, updated_at)
SELECT org_id, 'monitoring_interval', value, NOW(), NOW()
FROM settings WHERE key = 'python_poll_interval'
ON CONFLICT (org_id, key) DO NOTHING;

-- discovery_scan_depth: use the MAX of the two old Top-N values
-- (since depth now applies to EACH ecosystem)
INSERT INTO settings (org_id, key, value, created_at, updated_at)
SELECT org_id, 'discovery_scan_depth',
       CAST(
           GREATEST(
               COALESCE((SELECT s2.value FROM settings s2 WHERE s2.org_id = s.org_id AND s2.key = 'python_top_n'), '100')::int,
               COALESCE((SELECT s3.value FROM settings s3 WHERE s3.org_id = s.org_id AND s3.key = 'npm_top_n'), '100')::int
           )
       AS TEXT),
       NOW(), NOW()
FROM (SELECT DISTINCT org_id FROM settings WHERE key IN ('python_top_n', 'npm_top_n')) s
ON CONFLICT (org_id, key) DO NOTHING;

-- discovery_interval: use the old top_n_refresh_interval value
INSERT INTO settings (org_id, key, value, created_at, updated_at)
SELECT org_id, 'discovery_interval', value, NOW(), NOW()
FROM settings WHERE key = 'top_n_refresh_interval'
ON CONFLICT (org_id, key) DO NOTHING;

-- 7. Delete deprecated setting keys
DELETE FROM settings WHERE key IN (
    'python_poll_interval', 'npm_poll_interval',
    'python_top_n', 'npm_top_n',
    'top_n_refresh_interval',
    'version_depth_mode', 'version_depth_count'
);
