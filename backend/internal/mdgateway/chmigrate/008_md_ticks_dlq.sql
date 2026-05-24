-- 008_md_ticks_dlq.sql
-- M10 ADR-0010: Dead Letter Queue for dropped ticks.
-- quality.go drops are sampled here: parse_error 100%, bid_gt_ask/non_positive 1%.
-- TTL 7 days (short-term debugging only).

CREATE TABLE IF NOT EXISTS md_ticks_dlq (
    user_id          LowCardinality(String),
    account_id       LowCardinality(String),
    broker           LowCardinality(String),
    symbol_raw       LowCardinality(String),
    canonical        LowCardinality(String),
    ts_unix_ms       UInt64,
    arrived_unix_ms  UInt64,
    bid_str          String,
    ask_str          String,
    bid_volume       Float64,
    ask_volume       Float64,
    reason           LowCardinality(String),
    sampled_pct      Float32,
    raw_payload      String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(toDateTime(arrived_unix_ms / 1000))
ORDER BY (reason, broker, arrived_unix_ms)
TTL toDateTime(arrived_unix_ms / 1000) + INTERVAL 7 DAY
SETTINGS index_granularity = 8192;
