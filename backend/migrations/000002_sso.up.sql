-- Platform-scoped SSO configurations
CREATE TABLE sso_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(20) NOT NULL,
    display_name VARCHAR(100) NOT NULL DEFAULT '',
    is_enabled BOOLEAN NOT NULL DEFAULT false,
    auto_create_user BOOLEAN NOT NULL DEFAULT false,
    allowed_domains TEXT,

    -- SAML fields
    saml_entity_id TEXT,
    saml_sso_url TEXT,
    saml_certificate TEXT,
    saml_attr_email VARCHAR(100) DEFAULT 'email',
    saml_attr_first_name VARCHAR(100) DEFAULT 'firstName',
    saml_attr_last_name VARCHAR(100) DEFAULT 'lastName',

    -- OAuth fields
    oauth_client_id VARCHAR(255),
    oauth_client_secret_enc TEXT,
    google_hosted_domain VARCHAR(255),
    github_orgs TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_sso_configs_provider CHECK (provider IN ('saml', 'google', 'github'))
);

CREATE INDEX idx_sso_configs_enabled ON sso_configs(is_enabled) WHERE is_enabled = true;
CREATE UNIQUE INDEX idx_sso_configs_oauth_client_id ON sso_configs(oauth_client_id) WHERE oauth_client_id IS NOT NULL AND oauth_client_id != '';

-- External identity links
CREATE TABLE user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    metadata TEXT DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(provider, provider_user_id),
    UNIQUE(user_id, provider),
    CONSTRAINT chk_user_identities_provider CHECK (provider IN ('saml', 'google', 'github', 'local'))
);

CREATE INDEX idx_user_identities_user ON user_identities(user_id);
CREATE INDEX idx_user_identities_provider_email ON user_identities(provider, email);

-- SSO state for CSRF protection (short-lived)
CREATE TABLE sso_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES sso_configs(id) ON DELETE CASCADE,
    state VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL,
    redirect_url TEXT,
    mode VARCHAR(20) NOT NULL DEFAULT 'login',
    code_verifier TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_sso_states_provider CHECK (provider IN ('saml', 'google', 'github')),
    CONSTRAINT chk_sso_states_mode CHECK (mode IN ('login', 'link'))
);

CREATE INDEX idx_sso_states_expires ON sso_states(expires_at);

-- Add auth_provider to users table
ALTER TABLE users ADD COLUMN auth_provider VARCHAR(20) NOT NULL DEFAULT 'local';
ALTER TABLE users ADD CONSTRAINT chk_users_auth_provider CHECK (auth_provider IN ('local', 'saml', 'google', 'github'));

-- Add platform admin and deactivation columns (is_active already exists from initial migration)
ALTER TABLE users ADD COLUMN is_superadmin BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN deactivated_at TIMESTAMPTZ;

-- Allow nullable workspace_id on settings for platform-level settings
ALTER TABLE settings ALTER COLUMN workspace_id DROP NOT NULL;
ALTER TABLE settings DROP CONSTRAINT IF EXISTS settings_workspace_id_fkey;
ALTER TABLE settings ADD CONSTRAINT settings_workspace_id_fkey
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE;

-- Replace unique index to handle NULL workspace_id (platform settings)
DROP INDEX IF EXISTS idx_settings_workspace_key;
CREATE UNIQUE INDEX idx_settings_workspace_key ON settings(workspace_id, key) WHERE workspace_id IS NOT NULL;
CREATE UNIQUE INDEX idx_settings_platform_key ON settings(key) WHERE workspace_id IS NULL;

-- Platform auth settings
INSERT INTO settings (id, workspace_id, key, value, created_at, updated_at)
VALUES
    (gen_random_uuid(), NULL, 'auth.password_login_enabled', 'true', now(), now());
