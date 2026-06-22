-- 168_marketplace_platform_fee.down.sql

ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS platform_fee_rate;
