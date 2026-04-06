-- S4-8: API Key Scoping
-- Default to 'admin' for backward compatibility: existing keys retain full access.
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS scope VARCHAR(20) NOT NULL DEFAULT 'admin';

ALTER TABLE api_keys ADD CONSTRAINT chk_api_keys_scope
    CHECK (scope IN ('read', 'write', 'admin'));

-- S4-9: Session Management
CREATE TABLE IF NOT EXISTS sessions (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    VARCHAR(255) NOT NULL,
    ip_address    VARCHAR(45),
    user_agent    VARCHAR(512),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
