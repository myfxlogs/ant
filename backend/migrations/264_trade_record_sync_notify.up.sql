-- 264_trade_record_sync_notify.up.sql
-- FEAT-4: pg_notify on trade_records INSERT for push-first divergence SSE.
-- Fires 'trade_record_sync' notification so WatchDivergenceReport streams
-- push updates immediately when new live trades are synced, instead of polling.

CREATE OR REPLACE FUNCTION notify_trade_record_sync() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('trade_record_sync', NEW.account_id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_trade_record_sync ON trade_records;
CREATE TRIGGER trg_trade_record_sync
    AFTER INSERT ON trade_records
    FOR EACH ROW EXECUTE FUNCTION notify_trade_record_sync();
