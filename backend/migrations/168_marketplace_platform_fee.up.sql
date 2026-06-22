-- 168_marketplace_platform_fee.up.sql
-- Add platform_fee_rate column for strategy sales commission.

ALTER TABLE marketplace_strategies ADD COLUMN IF NOT EXISTS platform_fee_rate NUMERIC(5,4) NOT NULL DEFAULT 0;

COMMENT ON COLUMN marketplace_strategies.platform_fee_rate IS 'Platform commission rate (0.0000–1.0000). 0 = no fee, 0.1000 = 10%%.';
