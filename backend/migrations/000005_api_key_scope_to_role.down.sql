-- Revert: rename role back to scope and restore values.
DROP INDEX IF EXISTS idx_api_keys_workspace_id;

ALTER TABLE api_keys RENAME COLUMN role TO scope;

UPDATE api_keys SET scope = 'admin' WHERE scope = 'admin';
UPDATE api_keys SET scope = 'write' WHERE scope = 'member';
UPDATE api_keys SET scope = 'read' WHERE scope = 'viewer';

ALTER TABLE api_keys ALTER COLUMN scope SET DEFAULT 'read';

ALTER TABLE api_keys DROP COLUMN IF EXISTS workspace_id;
