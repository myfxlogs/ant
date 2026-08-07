-- 161_backtest_runs_notify_trigger.down.sql
-- Auto-generated rollback for 161_backtest_runs_notify_trigger

-- Drop triggers
DROP TRIGGER IF EXISTS backtest_pending_notify ON backtest_runs;

