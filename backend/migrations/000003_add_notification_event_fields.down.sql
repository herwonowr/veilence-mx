DROP INDEX IF EXISTS idx_notifications_severity;
DROP INDEX IF EXISTS idx_notifications_event_type;

ALTER TABLE notifications DROP COLUMN IF EXISTS reference_type;
ALTER TABLE notifications DROP COLUMN IF EXISTS reference_id;
ALTER TABLE notifications DROP COLUMN IF EXISTS event_type;
ALTER TABLE notifications DROP COLUMN IF EXISTS severity;
