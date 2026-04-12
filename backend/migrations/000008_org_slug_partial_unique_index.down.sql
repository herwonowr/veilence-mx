-- Revert V102-15: Restore full unique index on slug
DROP INDEX IF EXISTS idx_organizations_slug;
CREATE UNIQUE INDEX idx_organizations_slug ON organizations(slug);
