-- 004_signals.sql
-- Strategy signal storage, partitioned by month.
-- Ant-adapted: removed tenant_id, added user_id.

CREATE TABLE IF NOT EXISTS signals (
    user_id        LowCardinality(String) NOT NULL,
    strategy_id    String,
    deployment_id  String,
    symbol         LowCardinality(String),
    ts             DateTime64(3, 'UTC'),
    side           Int8,
    target_qty     Float64,
    limit_price    Nullable(Float64),
    client_id      String,
    created_at     DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(ts)
ORDER BY (user_id, strategy_id, ts)
TTL toDateTime(ts) + INTERVAL 2 YEAR;
