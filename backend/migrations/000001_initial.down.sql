-- Drop ALL tables in reverse dependency order.

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS notification_rules;
DROP TABLE IF EXISTS notification_channels;
DROP TABLE IF EXISTS alert_notes;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS analyses;
DROP TABLE IF EXISTS diffs;
DROP TABLE IF EXISTS releases;
DROP TABLE IF EXISTS packages;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS users;
