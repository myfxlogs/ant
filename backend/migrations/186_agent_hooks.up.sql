-- 186_agent_hooks.up.sql
-- ADR-0025 §8: Lifecycle hook configurations (command/webhook types).

CREATE TABLE IF NOT EXISTS agent_hook_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event           VARCHAR(64) NOT NULL,
    type            VARCHAR(16) NOT NULL,
    command         TEXT NOT NULL DEFAULT '',
    webhook_url     TEXT NOT NULL DEFAULT '',
    timeout_seconds INT NOT NULL DEFAULT 10,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_hook_configs_event ON agent_hook_configs(event) WHERE enabled = true;
