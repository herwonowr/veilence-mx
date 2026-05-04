ALTER TABLE sessions DROP COLUMN IF EXISTS auth_provider;
ALTER TABLE audit_logs DROP COLUMN IF EXISTS user_email;
