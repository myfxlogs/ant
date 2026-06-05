-- 140: Add execution config columns + proto binary snapshot to backtest_runs.
-- Individual columns for query/display, config_snapshot BYTEA for full reproducibility.

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS commission DOUBLE PRECISION NOT NULL DEFAULT 0.0;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS slippage DOUBLE PRECISION NOT NULL DEFAULT 0.0;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS leverage DOUBLE PRECISION NOT NULL DEFAULT 1.0;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS trade_direction TEXT NOT NULL DEFAULT 'both';

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS strict_mode BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE backtest_runs
  ADD COLUMN IF NOT EXISTS config_snapshot BYTEA;

COMMENT ON COLUMN backtest_runs.config_snapshot IS
  'Serialized ant.v1.BacktestExecutionConfig proto binary';
