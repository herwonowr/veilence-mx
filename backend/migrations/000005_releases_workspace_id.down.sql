ALTER TABLE releases DROP CONSTRAINT IF EXISTS fk_releases_workspace;
DROP INDEX IF EXISTS idx_releases_workspace_id;
ALTER TABLE releases DROP COLUMN IF EXISTS workspace_id;
