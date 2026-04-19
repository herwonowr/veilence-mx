-- Rename scope column to role and migrate values.
-- Add workspace_id column to api_keys.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS workspace_id BIGINT NOT NULL DEFAULT 0;

-- Rename scope column to role
ALTER TABLE api_keys RENAME COLUMN scope TO role;

-- Migrate old scope values to new role values
UPDATE api_keys SET role = 'admin' WHERE role = 'admin';
UPDATE api_keys SET role = 'member' WHERE role = 'write';
UPDATE api_keys SET role = 'viewer' WHERE role = 'read';

-- Update the default
ALTER TABLE api_keys ALTER COLUMN role SET DEFAULT 'viewer';

-- Add index on workspace_id
CREATE INDEX IF NOT EXISTS idx_api_keys_workspace_id ON api_keys (workspace_id);
