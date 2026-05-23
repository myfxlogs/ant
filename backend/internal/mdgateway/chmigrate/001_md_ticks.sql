-- 001_md_ticks.sql
-- Market data tick storage, partitioned by month.
-- Ant-adapted: removed tenant_id, added user_id.

CREATE TABLE IF NOT EXISTS md_ticks (
    user_id          LowCardinality(String),
    broker           LowCardinality(String),
    symbol_raw       LowCardinality(String),
    canonical        LowCardinality(String),
    ts_unix_ms       UInt64,
    arrived_unix_ms  UInt64,
    bid              Decimal(18, 6),
    ask              Decimal(18, 6),
    bid_volume       Float64,
    ask_volume       Float64
) ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(toDate(ts_unix_ms / 1000))
ORDER BY (broker, canonical, ts_unix_ms);
