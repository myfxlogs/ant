-- 163_backtest_status_notify.up.sql
-- PG NOTIFY on backtest_runs status changes so marketplace SSE streams
-- can push updates immediately instead of polling (Push-First architecture).

CREATE OR REPLACE FUNCTION notify_backtest_status_change() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('backtest_status_change', NEW.id::text || ',' || NEW.status);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- INSERT: notify on every new run (initial PENDING status).
DROP TRIGGER IF EXISTS backtest_status_notify ON backtest_runs;
CREATE TRIGGER backtest_status_notify
    AFTER INSERT ON backtest_runs
    FOR EACH ROW
    EXECUTE FUNCTION notify_backtest_status_change();

-- UPDATE: notify only when status actually changes (avoids noise from lease updates).
DROP TRIGGER IF EXISTS backtest_status_update_notify ON backtest_runs;
CREATE TRIGGER backtest_status_update_notify
    AFTER UPDATE ON backtest_runs
    FOR EACH ROW
    WHEN (NEW.status IS DISTINCT FROM OLD.status)
    EXECUTE FUNCTION notify_backtest_status_change();
