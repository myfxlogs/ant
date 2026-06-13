-- 158_market_data_pg.up.sql
-- Create PostgreSQL native partitioned tables for market data (ticks + bars).
-- Replaces ClickHouse md_ticks and md_bars, enabling removal of the ClickHouse container.
--
-- Partitioning: monthly RANGE on timestamp columns
--   - md_ticks:  partitioned on arrived_unix_ms  (90-day TTL)
--   - md_bars:   partitioned on close_ts_unix_ms  (2-year TTL)
-- Indexes: B-tree for point queries + BRIN for range scans

-- ── md_ticks ──────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS md_ticks (
    user_id          TEXT NOT NULL DEFAULT '',
    account_id       TEXT NOT NULL DEFAULT '',
    broker           TEXT NOT NULL,
    symbol_raw       TEXT NOT NULL DEFAULT '',
    canonical        TEXT NOT NULL,
    ts_unix_ms       BIGINT NOT NULL,
    arrived_unix_ms  BIGINT NOT NULL,
    bid              NUMERIC(18,6) NOT NULL,
    ask              NUMERIC(18,6) NOT NULL,
    bid_volume       DOUBLE PRECISION NOT NULL DEFAULT 0,
    ask_volume       DOUBLE PRECISION NOT NULL DEFAULT 0,
    is_replay        SMALLINT NOT NULL DEFAULT 0
) PARTITION BY RANGE (arrived_unix_ms);

-- Composite index for GetLatestTick: canonical + arrived_unix_ms DESC
CREATE INDEX IF NOT EXISTS idx_md_ticks_canonical_arrived ON md_ticks (canonical, arrived_unix_ms DESC);
-- Index for broker-filtered lookups
CREATE INDEX IF NOT EXISTS idx_md_ticks_canonical_broker_arrived ON md_ticks (canonical, broker, arrived_unix_ms DESC);

-- ── md_bars ───────────────────────────────────────────────────────────────

CREATE TABLE md_bars (
    broker           TEXT NOT NULL,
    symbol_raw       TEXT NOT NULL DEFAULT '',
    canonical        TEXT NOT NULL,
    period           TEXT NOT NULL,
    open_ts_unix_ms  BIGINT NOT NULL,
    close_ts_unix_ms BIGINT NOT NULL,
    open             NUMERIC(18,6) NOT NULL,
    high             NUMERIC(18,6) NOT NULL,
    low              NUMERIC(18,6) NOT NULL,
    close            NUMERIC(18,6) NOT NULL,
    volume           DOUBLE PRECISION NOT NULL DEFAULT 0,
    tick_count       INTEGER NOT NULL DEFAULT 0,
    is_replay        SMALLINT NOT NULL DEFAULT 0,
    user_id          TEXT NOT NULL DEFAULT '',
    account_id       TEXT NOT NULL DEFAULT ''
) PARTITION BY RANGE (close_ts_unix_ms);

-- Composite index for GetKlines: canonical + period + close_ts DESC (the hot query)
CREATE INDEX IF NOT EXISTS idx_md_bars_canonical_period_close ON md_bars (canonical, period, close_ts_unix_ms DESC);
-- Index for LoadFinalizedBars and MaxCloseTs
CREATE INDEX IF NOT EXISTS idx_md_bars_broker_canonical_period ON md_bars (broker, canonical, period, close_ts_unix_ms);

-- ── Pre-create current and next month partitions ───────────────────────────
-- Partitions for future months are created by the EnsurePartitions() startup
-- function. This ensures the table is immediately usable after migration.

DO $$
DECLARE
    cur_date   DATE;
    start_ms   BIGINT;
    end_ms     BIGINT;
    part_name  TEXT;
    i          INT;
BEGIN
    FOR i IN 0..2 LOOP
        cur_date := date_trunc('month', now()) + (i || ' months')::INTERVAL;
        start_ms := (EXTRACT(EPOCH FROM cur_date) * 1000)::BIGINT;
        end_ms   := (EXTRACT(EPOCH FROM cur_date + INTERVAL '1 month') * 1000)::BIGINT;
        part_name := 'md_ticks_' || to_char(cur_date, 'YYYYMM');
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF md_ticks FOR VALUES FROM (%L) TO (%L)',
                       part_name, start_ms, end_ms);
    END LOOP;

    FOR i IN 0..5 LOOP
        cur_date := date_trunc('month', now()) + (i || ' months')::INTERVAL;
        start_ms := (EXTRACT(EPOCH FROM cur_date) * 1000)::BIGINT;
        end_ms   := (EXTRACT(EPOCH FROM cur_date + INTERVAL '1 month') * 1000)::BIGINT;
        part_name := 'md_bars_' || to_char(cur_date, 'YYYYMM');
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF md_bars FOR VALUES FROM (%L) TO (%L)',
                       part_name, start_ms, end_ms);
    END LOOP;
END $$;
