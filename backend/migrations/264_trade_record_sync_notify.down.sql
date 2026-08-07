-- 264_trade_record_sync_notify.down.sql
-- Remove trade_record_sync notification trigger.

DROP TRIGGER IF EXISTS trg_trade_record_sync ON trade_records;
DROP FUNCTION IF EXISTS notify_trade_record_sync();
