-- Revert token_hash column back to token.
ALTER TABLE invitations RENAME COLUMN token_hash TO token;

DROP INDEX IF EXISTS idx_invitations_token_hash;
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_token ON invitations(token);
