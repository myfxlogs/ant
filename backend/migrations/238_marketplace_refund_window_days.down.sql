-- 238_marketplace_refund_window_days.down.sql
ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS refund_window_days;
