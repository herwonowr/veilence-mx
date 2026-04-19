-- Revert workspace → organization rename.

-- Rename tables back
ALTER TABLE workspaces RENAME TO organizations;
ALTER TABLE workspace_members RENAME TO org_members;

-- Rename columns back
ALTER TABLE org_members RENAME COLUMN workspace_id TO org_id;
ALTER TABLE roles RENAME COLUMN workspace_id TO org_id;
ALTER TABLE invitations RENAME COLUMN workspace_id TO org_id;
ALTER TABLE packages RENAME COLUMN workspace_id TO org_id;
ALTER TABLE alerts RENAME COLUMN workspace_id TO org_id;
ALTER TABLE alert_notes RENAME COLUMN workspace_id TO org_id;
ALTER TABLE settings RENAME COLUMN workspace_id TO org_id;
ALTER TABLE audit_logs RENAME COLUMN workspace_id TO org_id;
ALTER TABLE notification_channels RENAME COLUMN workspace_id TO org_id;
ALTER TABLE notification_rules RENAME COLUMN workspace_id TO org_id;
ALTER TABLE notifications RENAME COLUMN workspace_id TO org_id;

-- Revert permission records
UPDATE permissions SET resource = 'org' WHERE resource = 'workspace';

-- Note: index renames are not reverted as PostgreSQL handles them automatically.
-- For a full revert, recreate the original indexes manually.
