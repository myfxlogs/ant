-- 264: Backtest failure signatures for ADR-0028 §5.2 root cause report.
-- Stores extracted blind_spot signatures from DEGRADED/FAILED backtest runs
-- for clustering and recurrence tracking in the Admin Platform Health Center.

CREATE TABLE IF NOT EXISTS backtest_failure_signatures (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backtest_run_id UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    strategy_id     UUID REFERENCES imported_strategies(id) ON DELETE SET NULL,
    user_id         UUID NOT NULL,
    signature       TEXT NOT NULL,        -- blind_spot_id (e.g. "zero_volume_trade")
    severity        TEXT NOT NULL,        -- "致命" or "提示"
    category        TEXT NOT NULL,        -- "invariant" or "statistical"
    description     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_failure_sig_signature ON backtest_failure_signatures(signature);
CREATE INDEX IF NOT EXISTS idx_failure_sig_created_at ON backtest_failure_signatures(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_failure_sig_strategy ON backtest_failure_signatures(strategy_id);
CREATE INDEX IF NOT EXISTS idx_failure_sig_run ON backtest_failure_signatures(backtest_run_id);
