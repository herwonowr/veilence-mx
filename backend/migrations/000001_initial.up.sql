-- Veilence-MX complete schema (merged from migrations 000001-000005).
-- For fresh installs only. Existing deployments that have already run the
-- original migrations should reset migration state:
--   DELETE FROM schema_migrations;
--   INSERT INTO schema_migrations (version, dirty) VALUES (1, false);

-- =========================================================================
-- Users
-- =========================================================================
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT true,
    email_verified BOOLEAN NOT NULL DEFAULT false,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- =========================================================================
-- Workspaces (originally "organizations", renamed in 000004)
-- =========================================================================
CREATE TABLE workspaces (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    owner_id BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_workspaces_slug ON workspaces(slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_workspaces_deleted_at ON workspaces(deleted_at);

-- =========================================================================
-- Roles
-- =========================================================================
CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_roles_workspace_id ON roles(workspace_id);

-- =========================================================================
-- Permissions
-- =========================================================================
CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    resource VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL
);

-- Role-Permission join table
CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- =========================================================================
-- Workspace members (originally "org_members", renamed in 000004)
-- =========================================================================
CREATE TABLE workspace_members (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_workspace_user ON workspace_members(workspace_id, user_id);

-- =========================================================================
-- Invitations
-- =========================================================================
CREATE TABLE invitations (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    email VARCHAR(255) NOT NULL,
    role_id BIGINT NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    invited_by BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_invitations_token_hash ON invitations(token_hash);
CREATE INDEX idx_invitations_workspace_id ON invitations(workspace_id);

-- =========================================================================
-- Refresh tokens
-- =========================================================================
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- =========================================================================
-- API keys (scope renamed to role in 000005, workspace_id added in 000005)
-- =========================================================================
CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    workspace_id BIGINT NOT NULL DEFAULT 0,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(10) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_api_keys_role CHECK (role IN ('viewer', 'member', 'admin'))
);
CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_deleted_at ON api_keys(deleted_at);
CREATE INDEX idx_api_keys_key_prefix ON api_keys(key_prefix) WHERE deleted_at IS NULL;
CREATE INDEX idx_api_keys_workspace_id ON api_keys(workspace_id);

-- =========================================================================
-- Sessions
-- =========================================================================
CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    user_agent VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE UNIQUE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- =========================================================================
-- Password reset tokens
-- =========================================================================
CREATE TABLE password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_password_reset_tokens_token_hash ON password_reset_tokens(token_hash);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);

-- =========================================================================
-- Email verification tokens
-- =========================================================================
CREATE TABLE email_verification_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_email_verification_tokens_token_hash ON email_verification_tokens(token_hash);
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);

-- =========================================================================
-- Packages (with download fields from 000002, workspace rename from 000004)
-- =========================================================================
CREATE TABLE packages (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    ecosystem VARCHAR(10) NOT NULL,
    latest_version VARCHAR(100),
    description TEXT,
    rank BIGINT,
    source VARCHAR(20) NOT NULL DEFAULT 'manual',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    blocked_at TIMESTAMPTZ,
    blocked_reason TEXT,
    download_count BIGINT NOT NULL DEFAULT 0,
    popularity_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    download_count_updated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_packages_workspace_id ON packages(workspace_id);
CREATE UNIQUE INDEX idx_packages_workspace_name_ecosystem ON packages(workspace_id, name, ecosystem);
CREATE INDEX idx_packages_workspace_ecosystem ON packages(workspace_id, ecosystem);
CREATE INDEX idx_packages_status ON packages(status);
CREATE INDEX idx_packages_workspace_status ON packages(workspace_id, status);

-- =========================================================================
-- Releases
-- =========================================================================
CREATE TABLE releases (
    id BIGSERIAL PRIMARY KEY,
    package_id BIGINT NOT NULL,
    version VARCHAR(100) NOT NULL,
    published_at TIMESTAMPTZ,
    tarball_url TEXT,
    sha256 VARCHAR(64),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_releases_package_id ON releases(package_id);
CREATE UNIQUE INDEX idx_releases_package_version ON releases(package_id, version);
CREATE INDEX idx_releases_package_published ON releases(package_id, published_at DESC);
CREATE INDEX idx_releases_status ON releases(status);

-- =========================================================================
-- Diffs
-- =========================================================================
CREATE TABLE diffs (
    id BIGSERIAL PRIMARY KEY,
    release_id BIGINT NOT NULL,
    prev_release_id BIGINT NOT NULL,
    diff_content TEXT,
    file_changes_count BIGINT DEFAULT 0,
    lines_added BIGINT DEFAULT 0,
    lines_removed BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_diffs_release_id ON diffs(release_id);

-- =========================================================================
-- Analyses
-- =========================================================================
CREATE TABLE analyses (
    id BIGSERIAL PRIMARY KEY,
    diff_id BIGINT NOT NULL,
    classification VARCHAR(20) NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    reasoning TEXT,
    model_used VARCHAR(100),
    analyzer_type VARCHAR(10) NOT NULL,
    raw_response TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_analyses_diff_id ON analyses(diff_id);

-- =========================================================================
-- Alerts
-- =========================================================================
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    analysis_id BIGINT NOT NULL,
    package_id BIGINT NOT NULL,
    release_id BIGINT DEFAULT 0,
    severity VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'new',
    message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_alerts_workspace_id ON alerts(workspace_id);
CREATE INDEX idx_alerts_analysis_id ON alerts(analysis_id);
CREATE INDEX idx_alerts_package_id ON alerts(package_id);
CREATE INDEX idx_alerts_release_id ON alerts(release_id);
CREATE INDEX idx_alerts_workspace_status ON alerts(workspace_id, status);
CREATE INDEX idx_alerts_workspace_severity ON alerts(workspace_id, severity);

-- =========================================================================
-- Alert notes
-- =========================================================================
CREATE TABLE alert_notes (
    id BIGSERIAL PRIMARY KEY,
    alert_id BIGINT NOT NULL,
    workspace_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    user_email VARCHAR(255),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_alert_notes_alert_id ON alert_notes(alert_id);
CREATE INDEX idx_alert_notes_workspace_id ON alert_notes(workspace_id);

-- =========================================================================
-- Settings
-- =========================================================================
CREATE TABLE settings (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL DEFAULT 0,
    key VARCHAR(100) NOT NULL,
    value TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX idx_settings_workspace_key ON settings(workspace_id, key);

-- =========================================================================
-- Audit logs
-- =========================================================================
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    workspace_id BIGINT,
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    resource_id BIGINT,
    details TEXT,
    ip_address VARCHAR(45),
    user_agent VARCHAR(255),
    correlation_id VARCHAR(36),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_workspace_id ON audit_logs(workspace_id);
CREATE INDEX idx_audit_logs_correlation_id ON audit_logs(correlation_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_workspace_created ON audit_logs(workspace_id, created_at DESC);

-- =========================================================================
-- Notification channels
-- =========================================================================
CREATE TABLE notification_channels (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    config TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notification_channels_workspace_id ON notification_channels(workspace_id);

-- =========================================================================
-- Notification rules
-- =========================================================================
CREATE TABLE notification_rules (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    severity VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notification_rules_workspace_id ON notification_rules(workspace_id);
CREATE INDEX idx_notification_rules_channel_id ON notification_rules(channel_id);

-- =========================================================================
-- Notifications (with event fields from 000003)
-- =========================================================================
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    workspace_id BIGINT NOT NULL,
    user_id BIGINT,
    channel_id BIGINT,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    severity VARCHAR(20) NOT NULL DEFAULT '',
    event_type VARCHAR(100) NOT NULL DEFAULT '',
    reference_id BIGINT NOT NULL DEFAULT 0,
    reference_type VARCHAR(50) NOT NULL DEFAULT '',
    is_read BOOLEAN NOT NULL DEFAULT false,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_notifications_workspace_id ON notifications(workspace_id);
CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_channel_id ON notifications(channel_id);
CREATE INDEX idx_notifications_workspace_user_read ON notifications(workspace_id, user_id, is_read);
CREATE INDEX idx_notifications_event_type ON notifications(event_type);
CREATE INDEX idx_notifications_severity ON notifications(severity);
