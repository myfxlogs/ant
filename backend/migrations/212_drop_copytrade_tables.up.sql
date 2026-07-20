-- 212_drop_copytrade_tables.up.sql
-- Drop copytrade tables — feature removed per v4 product boundary (no copytrade).

DROP TABLE IF EXISTS copytrade_signals CASCADE;
DROP TABLE IF EXISTS copy_trade_links CASCADE;
