-- 104_strategy_state.up.sql
-- M10-BASE-E0: Strategy state persistence for hot-reload.
-- Stores serialized strategy snapshots for cross-instance state transfer.

CREATE TABLE IF NOT EXISTS strategy_state (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id     TEXT NOT NULL UNIQUE,          -- unique strategy instance identifier
    strategy_name   TEXT NOT NULL,                 -- strategy.Name()
    strategy_version TEXT NOT NULL,                -- strategy.Version() (semver)
    account_id      UUID REFERENCES mt_accounts(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL,
    state_data      JSONB NOT NULL DEFAULT '{}',   -- serialized StrategyState (protobuf-compatible JSON)
    schema_version  INT NOT NULL DEFAULT 1,        -- StrategyState.schema_version for migration
    status          TEXT NOT NULL DEFAULT 'active', -- active, stopped, error
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    stopped_at      TIMESTAMPTZ
);

-- Index for looking up active states by user.
CREATE INDEX IF NOT EXISTS idx_strategy_state_user ON strategy_state (user_id, status);
-- Index for lookup by strategy name + version for compatibility checks.
CREATE INDEX IF NOT EXISTS idx_strategy_state_name_version ON strategy_state (strategy_name, strategy_version);
-- Unique constraint: one active state per instance.
CREATE UNIQUE INDEX IF NOT EXISTS idx_strategy_state_active ON strategy_state (instance_id) WHERE status = 'active';
