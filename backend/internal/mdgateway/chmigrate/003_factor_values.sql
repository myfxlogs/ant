-- 003_factor_values.sql
-- Factor value time-series storage, partitioned by month.
-- Ant-adapted: removed tenant_id, added user_id.

CREATE TABLE IF NOT EXISTS factor_values (
    user_id      LowCardinality(String) NOT NULL,
    account_id   LowCardinality(String),
    symbol       LowCardinality(String),
    factor_name  LowCardinality(String),
    value        Float64,
    ts_ms        Int64,
    created_at   DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (user_id, factor_name, symbol, ts_ms)
TTL toDateTime(created_at) + INTERVAL 2 YEAR;
