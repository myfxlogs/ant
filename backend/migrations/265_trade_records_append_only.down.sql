-- 265_trade_records_append_only.down.sql
-- Reverse: remove append-only DELETE trigger

DROP TRIGGER IF EXISTS prevent_trade_delete ON trade_records;
DROP FUNCTION IF EXISTS prevent_trade_record_delete();
