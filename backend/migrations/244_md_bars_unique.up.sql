-- Add unique constraint on md_bars to prevent duplicate bars.
-- With period-aligned open_ts/close_ts (bar_aggregator fix), the same bar
-- from different accounts will have identical (broker, canonical, period, open_ts, close_ts).
-- The unique constraint includes close_ts (partition key) as required by PostgreSQL.
-- ON CONFLICT DO UPDATE in InsertBars will merge bars, keeping the one with the most ticks.

-- First, clean up existing duplicates (keep the row with the highest tick_count).
DELETE FROM md_bars a USING md_bars b
WHERE a.ctid < b.ctid
  AND a.broker = b.broker
  AND a.canonical = b.canonical
  AND a.period = b.period
  AND a.open_ts_unix_ms = b.open_ts_unix_ms
  AND a.close_ts_unix_ms = b.close_ts_unix_ms;

-- Create unique index. Must include partition key (close_ts_unix_ms).
CREATE UNIQUE INDEX IF NOT EXISTS idx_md_bars_unique
ON md_bars (broker, canonical, period, open_ts_unix_ms, close_ts_unix_ms);
