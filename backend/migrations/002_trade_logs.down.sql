-- 002_trade_logs.down.sql
-- Auto-generated rollback for 002_trade_logs

-- Drop indexes
DROP INDEX IF EXISTS idx_trade_logs_account;
DROP INDEX IF EXISTS idx_trade_logs_action;
DROP INDEX IF EXISTS idx_trade_logs_created_at;
DROP INDEX IF EXISTS idx_trade_logs_symbol;
DROP INDEX IF EXISTS idx_trade_logs_user;

-- Drop tables
DROP TABLE IF EXISTS trade_logs CASCADE;

