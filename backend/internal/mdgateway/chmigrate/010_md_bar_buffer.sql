-- 010_md_bar_buffer.sql
-- M10 ADR-0011: Buffer engine for md_bars write-path decoupling.
CREATE TABLE IF NOT EXISTS md_bars_buffer (
    broker           LowCardinality(String),
    symbol_raw       LowCardinality(String),
    canonical        LowCardinality(String),
    period           LowCardinality(String),
    open_ts_unix_ms  UInt64,
    close_ts_unix_ms UInt64,
    open             Decimal(18, 6),
    high             Decimal(18, 6),
    low              Decimal(18, 6),
    close            Decimal(18, 6),
    volume           Float64,
    tick_count       UInt32,
    is_replay        UInt8 DEFAULT 0
) ENGINE = Buffer(ant, md_bars, 4, 5, 30, 1000, 100000, 1000000, 10000000)
