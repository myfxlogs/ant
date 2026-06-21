-- 163_backtest_status_notify.up.sql
-- PG NOTIFY on backtest_runs status changes so marketplace SSE streams
-- can push updates immediately instead of polling (Push-First architecture).

CREATE OR REPLACE FUNCTION notify_backtest_status_change() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('backtest_status_change', NEW.id::text || ',' || NEW.status);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS backtest_status_notify ON backtest_runs;
CREATE TRIGGER backtest_status_notify
    AFTER INSERT OR UPDATE ON backtest_runs
    FOR EACH ROW
    WHEN (OLD IS NULL OR NEW.status IS DISTINCT FROM OLD.status)
    EXECUTE FUNCTION notify_backtest_status_change();
