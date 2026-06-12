-- 019_tick_ttl.sql
-- md_ticks had no TTL, growing unbounded (~478K rows/day at 3-account scale).
-- 90-day retention: raw ticks beyond 90 days are auto-dropped by ClickHouse background merge.
-- 90 days of ticks is sufficient for walk-forward backtesting at this scale.

ALTER TABLE md_ticks MODIFY TTL toDateTime(arrived_unix_ms / 1000) + INTERVAL 90 DAY;
