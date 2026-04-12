-- Migration 000009: Rename registry → ecosystem across the schema.
-- Column rename, value migration, index updates, and settings key migration.

-- 1. Rename the column
ALTER TABLE packages RENAME COLUMN registry TO ecosystem;

-- 2. Update stored values: 'pypi' → 'python'
UPDATE packages SET ecosystem = 'python' WHERE ecosystem = 'pypi';

-- 3. Recreate affected indexes with the new column name.
--    Drop old indexes first, then create replacements.

-- Unique index from 000001 (org_id, name, registry) → (org_id, name, ecosystem)
DROP INDEX IF EXISTS idx_packages_org_name_registry;
CREATE UNIQUE INDEX idx_packages_org_name_ecosystem ON packages(org_id, name, ecosystem);

-- Composite index from 000006 (org_id, registry) → (org_id, ecosystem)
DROP INDEX IF EXISTS idx_packages_org_registry;
CREATE INDEX idx_packages_org_ecosystem ON packages(org_id, ecosystem);

-- 4. Migrate settings keys
UPDATE settings SET key = 'python_poll_interval' WHERE key = 'pypi_poll_interval';
UPDATE settings SET key = 'python_top_n' WHERE key = 'pypi_top_n';
