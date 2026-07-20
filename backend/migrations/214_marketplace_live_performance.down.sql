-- 214_marketplace_live_performance.down.sql
ALTER TABLE marketplace_strategies DROP COLUMN IF EXISTS linked_account_id;

DROP TABLE IF EXISTS marketplace_live_performance_summary CASCADE;
DROP TABLE IF EXISTS marketplace_live_performance CASCADE;
