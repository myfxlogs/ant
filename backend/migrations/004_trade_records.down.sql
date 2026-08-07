-- 004_trade_records.down.sql
-- Auto-generated rollback for 004_trade_records

-- Drop triggers
DROP TRIGGER IF EXISTS update_trade_records_updated_at ON trade_records;

-- Drop indexes
DROP INDEX IF EXISTS idx_trade_records_account;
DROP INDEX IF EXISTS idx_trade_records_close_time;
DROP INDEX IF EXISTS idx_trade_records_platform;
DROP INDEX IF EXISTS idx_trade_records_symbol;

-- Drop tables
DROP TABLE IF EXISTS trade_records CASCADE;

