package models

// OrgScoping documents the multi-tenancy model for organization scoping.
//
// Phase 3 Implementation (COMPLETED):
// The following models have been modified to add organization scoping:
//
// 1. Package model (package.go):
//    - Added: OrgID uint `gorm:"not null;index" json:"orgId"`
//    - All package queries are scoped by OrgID via handlers
//    - Unique constraint is now (org_id, name, registry)
//
// 2. Alert model (alert.go):
//    - Added: OrgID uint `gorm:"not null;index" json:"orgId"`
//    - Alerts inherit org scope from their parent package
//    - All alert queries are scoped by OrgID
//
// 3. Setting model (setting.go):
//    - Added: OrgID uint `gorm:"index;not null;default:0" json:"orgId"`
//    - OrgID=0 represents global default settings
//    - Org-specific settings (OrgID > 0) override globals
//    - Unique constraint is now (org_id, key)
//
// Data Migration:
// - The database migration in database.go backfills OrgID for existing records
// - Existing packages and alerts with OrgID=0 are assigned to a default org
// - The migration is idempotent (safe to run multiple times)
//
// Route Scoping:
// - Package, alert, settings, dashboard, and queue routes are behind RequireOrg
// - OrgID is extracted from the X-Org-ID header, org_id query param, or URL param
// - The poller accepts OrgID to scope synced packages to the correct org
