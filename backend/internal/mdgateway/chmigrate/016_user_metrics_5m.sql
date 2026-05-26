-- 016_user_metrics_5m.sql
-- M10-BASE-A4: Per-user 5-minute aggregated metrics (SummingMergeTree).
-- Used by the background 5min flush goroutine in mdgateway.
-- Key constraint: Prometheus metrics do NOT carry user_id label (cardinality).

CREATE TABLE IF NOT EXISTS user_metrics_5m (
    bucket_5m_ts   DateTime,
    user_id        LowCardinality(String),
    metric_name    LowCardinality(String),
    metric_value   Float64,
    count          UInt64
) ENGINE = SummingMergeTree()
ORDER BY (bucket_5m_ts, user_id, metric_name)
SETTINGS index_granularity = 8192;
