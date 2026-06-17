-- 161_backtest_runs_notify_trigger.up.sql
-- PG NOTIFY on new backtest_runs so the worker can wake up immediately
-- instead of polling. Replaces the 3-second ticker loop.

CREATE OR REPLACE FUNCTION notify_backtest_pending() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('backtest_pending', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS backtest_pending_notify ON backtest_runs;
CREATE TRIGGER backtest_pending_notify
    AFTER INSERT ON backtest_runs
    FOR EACH ROW
    WHEN (NEW.status = 'PENDING')
    EXECUTE FUNCTION notify_backtest_pending();
