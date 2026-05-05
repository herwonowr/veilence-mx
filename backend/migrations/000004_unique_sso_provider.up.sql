-- Enforce one SSO config per provider type (max one SAML, one Google, one GitHub).
CREATE UNIQUE INDEX idx_sso_configs_unique_provider ON sso_configs(provider);
