-- Rename organization → workspace across all tables and columns.

-- 1. Rename the organizations table to workspaces
ALTER TABLE organizations RENAME TO workspaces;

-- 2. Rename org_members table to workspace_members, rename org_id column
ALTER TABLE org_members RENAME TO workspace_members;
ALTER TABLE workspace_members RENAME COLUMN org_id TO workspace_id;

-- 3. Rename org_id columns in all other tables
ALTER TABLE roles RENAME COLUMN org_id TO workspace_id;
ALTER TABLE invitations RENAME COLUMN org_id TO workspace_id;
ALTER TABLE packages RENAME COLUMN org_id TO workspace_id;
ALTER TABLE alerts RENAME COLUMN org_id TO workspace_id;
ALTER TABLE alert_notes RENAME COLUMN org_id TO workspace_id;
ALTER TABLE settings RENAME COLUMN org_id TO workspace_id;
ALTER TABLE audit_logs RENAME COLUMN org_id TO workspace_id;
ALTER TABLE notification_channels RENAME COLUMN org_id TO workspace_id;
ALTER TABLE notification_rules RENAME COLUMN org_id TO workspace_id;
ALTER TABLE notifications RENAME COLUMN org_id TO workspace_id;

-- 4. Rename indexes (drop old, create new)
-- Organizations/Workspaces
DROP INDEX IF EXISTS idx_organizations_slug;
CREATE UNIQUE INDEX idx_workspaces_slug ON workspaces(slug) WHERE deleted_at IS NULL;
DROP INDEX IF EXISTS idx_organizations_deleted_at;
CREATE INDEX idx_workspaces_deleted_at ON workspaces(deleted_at);

-- Workspace members
DROP INDEX IF EXISTS idx_org_user;
CREATE UNIQUE INDEX idx_workspace_user ON workspace_members(workspace_id, user_id);

-- Roles
DROP INDEX IF EXISTS idx_roles_org_id;
CREATE INDEX idx_roles_workspace_id ON roles(workspace_id);

-- Invitations
DROP INDEX IF EXISTS idx_invitations_org_id;
CREATE INDEX idx_invitations_workspace_id ON invitations(workspace_id);

-- Packages
DROP INDEX IF EXISTS idx_packages_org_id;
CREATE INDEX idx_packages_workspace_id ON packages(workspace_id);
DROP INDEX IF EXISTS idx_packages_org_name_ecosystem;
CREATE UNIQUE INDEX idx_packages_workspace_name_ecosystem ON packages(workspace_id, name, ecosystem);
DROP INDEX IF EXISTS idx_packages_org_ecosystem;
CREATE INDEX idx_packages_workspace_ecosystem ON packages(workspace_id, ecosystem);
DROP INDEX IF EXISTS idx_packages_org_status;
CREATE INDEX idx_packages_workspace_status ON packages(workspace_id, status);

-- Alerts
DROP INDEX IF EXISTS idx_alerts_org_id;
CREATE INDEX idx_alerts_workspace_id ON alerts(workspace_id);
DROP INDEX IF EXISTS idx_alerts_org_status;
CREATE INDEX idx_alerts_workspace_status ON alerts(workspace_id, status);
DROP INDEX IF EXISTS idx_alerts_org_severity;
CREATE INDEX idx_alerts_workspace_severity ON alerts(workspace_id, severity);

-- Alert notes
DROP INDEX IF EXISTS idx_alert_notes_org_id;
CREATE INDEX idx_alert_notes_workspace_id ON alert_notes(workspace_id);

-- Settings
DROP INDEX IF EXISTS idx_settings_org_key;
CREATE UNIQUE INDEX idx_settings_workspace_key ON settings(workspace_id, key);

-- Audit logs
DROP INDEX IF EXISTS idx_audit_logs_org_id;
CREATE INDEX idx_audit_logs_workspace_id ON audit_logs(workspace_id);
DROP INDEX IF EXISTS idx_audit_logs_org_created;
CREATE INDEX idx_audit_logs_workspace_created ON audit_logs(workspace_id, created_at DESC);

-- Notification channels
DROP INDEX IF EXISTS idx_notification_channels_org_id;
CREATE INDEX idx_notification_channels_workspace_id ON notification_channels(workspace_id);

-- Notification rules
DROP INDEX IF EXISTS idx_notification_rules_org_id;
CREATE INDEX idx_notification_rules_workspace_id ON notification_rules(workspace_id);

-- Notifications
DROP INDEX IF EXISTS idx_notifications_org_id;
CREATE INDEX idx_notifications_workspace_id ON notifications(workspace_id);
DROP INDEX IF EXISTS idx_notifications_org_user_read;
CREATE INDEX idx_notifications_workspace_user_read ON notifications(workspace_id, user_id, is_read);

-- 5. Update permission records: "org" → "workspace"
UPDATE permissions SET resource = 'workspace' WHERE resource = 'org';
