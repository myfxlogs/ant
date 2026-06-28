-- 178_strategy_runs.up.sql
-- Strategy run lifecycle records + link signals to runs.

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
