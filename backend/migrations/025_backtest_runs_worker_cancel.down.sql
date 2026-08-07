-- 025_backtest_runs_worker_cancel.down.sql
-- Auto-generated rollback for 025_backtest_runs_worker_cancel

-- Drop indexes
DROP INDEX IF EXISTS idx_backtest_runs_cancel_requested_at;
DROP INDEX IF EXISTS idx_backtest_runs_lease_until;

