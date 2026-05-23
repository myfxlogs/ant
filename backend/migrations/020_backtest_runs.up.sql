-- 020_backtest_runs.up.sql
-- Backtest run records for audit and strict reproducibility

CREATE TABLE IF NOT EXISTS backtest_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    account_id UUID NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    dataset_id UUID,
    strategy_code_hash TEXT NOT NULL,
    python_service_version TEXT,
    cost_model_snapshot JSONB,
    metrics JSONB,
    equity_curve JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_backtest_runs_user ON backtest_runs(user_id);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_account ON backtest_runs(account_id);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_dataset ON backtest_runs(dataset_id);
