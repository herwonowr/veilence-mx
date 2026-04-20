-- Add declined_at column to invitations table for tracking declined invitations.
ALTER TABLE invitations ADD COLUMN declined_at TIMESTAMPTZ;

-- Index for efficiently querying pending invitations by email (used by ListMyInvitations).
CREATE INDEX idx_invitations_email_pending ON invitations(email) WHERE accepted_at IS NULL AND declined_at IS NULL;
