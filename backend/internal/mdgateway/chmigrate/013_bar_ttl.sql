-- 013_bar_ttl.sql
-- md_bars had no TTL, growing forever. Add 2-year retention:
-- 1m/1h/1d bars for backtesting are useful within ~2 years.
-- Older bars are auto-dropped by ClickHouse background merge.

ALTER TABLE md_bars MODIFY TTL toDateTime(close_ts_unix_ms / 1000) + INTERVAL 2 YEAR;
