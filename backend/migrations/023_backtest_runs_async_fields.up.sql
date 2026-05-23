-- 023_backtest_runs_async_fields.up.sql
-- Extend backtest_runs to support async execution lifecycle.

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'SUCCEEDED';

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS error TEXT NOT NULL DEFAULT '';

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS started_at TIMESTAMP;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS finished_at TIMESTAMP;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS strategy_code TEXT;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS initial_capital DOUBLE PRECISION;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_user_created_at ON backtest_runs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_account_created_at ON backtest_runs(account_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_status ON backtest_runs(status);
