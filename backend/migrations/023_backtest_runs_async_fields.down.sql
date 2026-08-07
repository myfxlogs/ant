-- 023_backtest_runs_async_fields.down.sql
-- Auto-generated rollback for 023_backtest_runs_async_fields

-- Drop indexes
DROP INDEX IF EXISTS idx_backtest_runs_account_created_at;
DROP INDEX IF EXISTS idx_backtest_runs_status;
DROP INDEX IF EXISTS idx_backtest_runs_user_created_at;

