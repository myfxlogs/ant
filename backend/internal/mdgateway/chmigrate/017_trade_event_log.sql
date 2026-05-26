-- 017_trade_event_log: ClickHouse projection of NATS Tier-0 trade event stream (M10-BASE-B4).
-- All order state changes are first written to NATS JetStream OMS_EVENTS, then
-- this table provides queryable history for analytics, audit, and replay.
CREATE TABLE IF NOT EXISTS trade_event_log (
    arrived_unix_ms Int64 CODEC(DoubleDelta),
    event_id String,
    event_type LowCardinality(String),
    account_id UUID,
    user_id UUID,
    broker LowCardinality(String),
    ticket Int64,
    client_id String,
    canonical LowCardinality(String),
    side LowCardinality(String),
    order_type LowCardinality(String),
    volume Float64,
    price Float64,
    stop_loss Float64,
    take_profit Float64,
    from_state LowCardinality(String),
    to_state LowCardinality(String),
    timestamp DateTime64(3, 'UTC'),
    version Int64
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(timestamp)
ORDER BY (account_id, ticket, arrived_unix_ms)
SETTINGS index_granularity = 8192;
