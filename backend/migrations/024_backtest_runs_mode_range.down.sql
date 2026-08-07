-- 024_backtest_runs_mode_range.down.sql
-- Auto-generated rollback for 024_backtest_runs_mode_range

-- Drop indexes
DROP INDEX IF EXISTS idx_backtest_runs_mode;
DROP INDEX IF EXISTS idx_backtest_runs_range;

