-- Remove the partial index on api_keys.key_prefix.
DROP INDEX IF EXISTS idx_api_keys_key_prefix;
