-- 178_strategy_runs.up.sql
-- Strategy run lifecycle records + link signals to runs.
-- Also relax strategy_signals.strategy_id (nullable) and account_id (VARCHAR)
-- to support live runner signals that don't have a strategies row.

CREATE TABLE IF NOT EXISTS strategy_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    account_id VARCHAR(100) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    mode VARCHAR(10) NOT NULL DEFAULT 'live',
    strategy_code TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    error TEXT,
    total_signals INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    stopped_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_strategy_runs_user ON strategy_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_account ON strategy_runs(account_id);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_status ON strategy_runs(status);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_started_at ON strategy_runs(started_at DESC);

-- Add run_id to strategy_signals (nullable for backward compat with old signals).
ALTER TABLE strategy_signals ADD COLUMN IF NOT EXISTS run_id UUID REFERENCES strategy_runs(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_strategy_signals_run_id ON strategy_signals(run_id);

-- Relax strategy_id: make nullable (live runner signals have no strategies row).
ALTER TABLE strategy_signals ALTER COLUMN strategy_id DROP NOT NULL;

-- Relax account_id: change from UUID to VARCHAR (paper accounts are not UUIDs).
ALTER TABLE strategy_signals ALTER COLUMN account_id TYPE VARCHAR(100) USING account_id::text;

-- Widen signal_type from VARCHAR(10) to VARCHAR(20) for pending order types (buy_stop_limit, etc).
ALTER TABLE strategy_signals ALTER COLUMN signal_type TYPE VARCHAR(20) USING signal_type::text;
