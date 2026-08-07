-- 014_stream_perf_indexes.down.sql
-- Auto-generated rollback for 014_stream_perf_indexes

-- Drop indexes
DROP INDEX IF EXISTS idx_logs_user_created_at_desc;
DROP INDEX IF EXISTS idx_mt_accounts_user_id;
DROP INDEX IF EXISTS idx_trade_records_account_close_time;
DROP INDEX IF EXISTS idx_trade_records_account_symbol_close_time;

