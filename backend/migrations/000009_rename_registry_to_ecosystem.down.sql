-- Migration 000009 (down): Revert ecosystem → registry rename.

-- 1. Revert settings keys
UPDATE settings SET key = 'pypi_poll_interval' WHERE key = 'python_poll_interval';
UPDATE settings SET key = 'pypi_top_n' WHERE key = 'python_top_n';

-- 2. Recreate original indexes before renaming the column back.
DROP INDEX IF EXISTS idx_packages_org_name_ecosystem;
CREATE UNIQUE INDEX idx_packages_org_name_registry ON packages(org_id, name, ecosystem);

DROP INDEX IF EXISTS idx_packages_org_ecosystem;
CREATE INDEX idx_packages_org_registry ON packages(org_id, ecosystem);

-- 3. Revert stored values: 'python' → 'pypi'
UPDATE packages SET ecosystem = 'pypi' WHERE ecosystem = 'python';

-- 4. Rename the column back
ALTER TABLE packages RENAME COLUMN ecosystem TO registry;
