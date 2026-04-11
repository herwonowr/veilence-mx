-- Add release_id column to alerts for deep linking
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS release_id BIGINT DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_alerts_release_id ON alerts(release_id);

-- Backfill release_id from analysis → diff → release chain
UPDATE alerts SET release_id = (
    SELECT d.release_id
    FROM analyses a
    JOIN diffs d ON d.id = a.diff_id
    WHERE a.id = alerts.analysis_id
    LIMIT 1
) WHERE release_id = 0;

-- Alert notes table
CREATE TABLE IF NOT EXISTS alert_notes (
    id BIGSERIAL PRIMARY KEY,
    alert_id BIGINT NOT NULL,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    user_email VARCHAR(255),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_alert_notes_alert_id ON alert_notes(alert_id);
CREATE INDEX IF NOT EXISTS idx_alert_notes_org_id ON alert_notes(org_id);
