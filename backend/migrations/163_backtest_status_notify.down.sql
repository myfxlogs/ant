-- 163_backtest_status_notify.down.sql
DROP TRIGGER IF EXISTS backtest_status_notify ON backtest_runs;
DROP FUNCTION IF EXISTS notify_backtest_status_change();
