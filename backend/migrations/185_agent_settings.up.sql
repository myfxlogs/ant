-- 185_agent_settings.up.sql
-- ADR-0025 §5: Tiered agent settings tables.
-- Priority: managed (admin) > user > default (built-in).

CREATE TABLE IF NOT EXISTS agent_user_settings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key        VARCHAR(128) NOT NULL,
    value      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, key)
);

CREATE INDEX idx_agent_user_settings_user ON agent_user_settings(user_id);

CREATE TABLE IF NOT EXISTS agent_managed_settings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key        VARCHAR(128) NOT NULL UNIQUE,
    value      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
