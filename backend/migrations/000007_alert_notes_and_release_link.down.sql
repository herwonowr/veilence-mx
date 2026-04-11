-- Remove alert notes table
DROP TABLE IF EXISTS alert_notes;

-- Remove release_id column from alerts
DROP INDEX IF EXISTS idx_alerts_release_id;
ALTER TABLE alerts DROP COLUMN IF EXISTS release_id;
