-- 014_stream_perf_indexes.up.sql
-- Add performance indexes for high-frequency queries used by streaming and market endpoints.

-- mt_accounts ownership lookup
CREATE INDEX IF NOT EXISTS idx_mt_accounts_user_id ON mt_accounts(user_id);

-- trade_records: common query patterns
CREATE INDEX IF NOT EXISTS idx_trade_records_account_close_time ON trade_records(account_id, close_time DESC);
CREATE INDEX IF NOT EXISTS idx_trade_records_account_symbol_close_time ON trade_records(account_id, symbol, close_time DESC);

-- kline_data: time-series lookups
CREATE INDEX IF NOT EXISTS idx_kline_symbol_tf_open_time_desc ON kline_data(symbol, timeframe, open_time DESC);

-- logs (if exists): often filtered by user and time
-- CREATE INDEX IF NOT EXISTS idx_logs_user_created_at_desc ON logs(user_id, created_at DESC);
