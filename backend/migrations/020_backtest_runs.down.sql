-- 020_backtest_runs.down.sql
-- Auto-generated rollback for 020_backtest_runs

-- Drop indexes
DROP INDEX IF EXISTS idx_backtest_runs_account;
DROP INDEX IF EXISTS idx_backtest_runs_dataset;
DROP INDEX IF EXISTS idx_backtest_runs_user;

-- Drop tables
DROP TABLE IF EXISTS backtest_runs CASCADE;

