DROP INDEX IF EXISTS idx_invitations_email_pending;
ALTER TABLE invitations DROP COLUMN IF EXISTS declined_at;
