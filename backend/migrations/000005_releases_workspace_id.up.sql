-- Add workspace_id to releases for defense-in-depth multi-tenant isolation.
ALTER TABLE releases ADD COLUMN workspace_id UUID;
UPDATE releases SET workspace_id = packages.workspace_id
FROM packages WHERE releases.package_id = packages.id;
ALTER TABLE releases ALTER COLUMN workspace_id SET NOT NULL;
CREATE INDEX idx_releases_workspace_id ON releases(workspace_id);
ALTER TABLE releases ADD CONSTRAINT fk_releases_workspace
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;
