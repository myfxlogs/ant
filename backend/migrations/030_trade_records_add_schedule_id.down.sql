-- 030_trade_records_add_schedule_id.down.sql
-- Auto-generated rollback for 030_trade_records_add_schedule_id

-- Drop indexes
DROP INDEX IF EXISTS idx_trade_records_schedule_id;

-- Drop added columns
ALTER TABLE trade_records DROP COLUMN IF EXISTS schedule_id;

