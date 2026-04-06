-- Reverse S4-9: Drop sessions table
DROP TABLE IF EXISTS sessions;

-- Reverse S4-8: Remove scope from api_keys
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS chk_api_keys_scope;
ALTER TABLE api_keys DROP COLUMN IF EXISTS scope;
