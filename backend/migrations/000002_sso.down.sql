-- Remove platform auth settings
DELETE FROM settings WHERE workspace_id IS NULL AND key = 'auth.password_login_enabled';

-- Restore settings constraints
DROP INDEX IF EXISTS idx_settings_platform_key;
DROP INDEX IF EXISTS idx_settings_workspace_key;
CREATE UNIQUE INDEX idx_settings_workspace_key ON settings(workspace_id, key);
ALTER TABLE settings ALTER COLUMN workspace_id SET NOT NULL;

-- Remove user columns (is_active stays - it's from initial migration)
ALTER TABLE users DROP COLUMN IF EXISTS deactivated_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_superadmin;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_auth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS auth_provider;

-- Drop SSO tables
DROP TABLE IF EXISTS sso_states;
DROP TABLE IF EXISTS user_identities;
DROP TABLE IF EXISTS sso_configs;
