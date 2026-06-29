-- 180_backtest_runs_strategy_id.down.sql

DROP INDEX IF EXISTS idx_backtest_runs_strategy_id;
ALTER TABLE backtest_runs DROP COLUMN IF EXISTS strategy_id;
