-- 009_md_tick_buffer.sql
-- M10 ADR-0011: Buffer engine for md_ticks write-path decoupling.
CREATE TABLE IF NOT EXISTS md_ticks_buffer (
    user_id          LowCardinality(String),
    broker           LowCardinality(String),
    symbol_raw       LowCardinality(String),
    canonical        LowCardinality(String),
    ts_unix_ms       UInt64,
    arrived_unix_ms  UInt64,
    bid              Decimal(18, 6),
    ask              Decimal(18, 6),
    bid_volume       Float64,
    ask_volume       Float64,
    is_replay        UInt8 DEFAULT 0
) ENGINE = Buffer(ant, md_ticks, 16, 1, 5, 10000, 1000000, 10000000, 100000000)
