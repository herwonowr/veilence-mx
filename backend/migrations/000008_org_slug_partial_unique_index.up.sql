-- V102-15: Fix org slug soft-delete uniqueness conflict
-- Problem: Full unique index on slug covers soft-deleted rows, preventing
-- recreation of organizations with the same slug after deletion.
-- Fix: Use partial unique index that only enforces uniqueness among active rows.

-- Drop the existing full unique index
DROP INDEX IF EXISTS idx_organizations_slug;

-- Create partial unique index (PostgreSQL) - only non-deleted rows
CREATE UNIQUE INDEX idx_organizations_slug ON organizations(slug) WHERE deleted_at IS NULL;
