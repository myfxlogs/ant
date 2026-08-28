-- 276_marketplace_live_performance_account_type.down.sql
-- TRUST-1: reverse the account_type column additions.

ALTER TABLE marketplace_live_performance_summary DROP COLUMN IF EXISTS account_type;
ALTER TABLE marketplace_live_performance DROP COLUMN IF EXISTS account_type;
DROP INDEX IF EXISTS idx_marketplace_live_performance_account_type;
