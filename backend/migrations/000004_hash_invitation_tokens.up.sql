-- Rename token column to token_hash to reflect that we now store
-- SHA-256 hashes instead of plaintext invitation tokens.
ALTER TABLE invitations RENAME COLUMN token TO token_hash;

-- Recreate the unique index with the new column name.
DROP INDEX IF EXISTS idx_invitations_token;
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations(token_hash);
