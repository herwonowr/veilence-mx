-- Add partial index on api_keys.key_prefix for O(1) prefix-based lookup.
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix) WHERE deleted_at IS NULL;
