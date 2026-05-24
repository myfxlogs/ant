-- 006_md_ticks_v2.sql
-- M10 ADR-0008: ORDER BY 字段与应用层 dedup hash 对齐（含 bid/ask/volume）
-- TTL/PARTITION 切到 arrived_unix_ms（本地时钟，不受 broker 时钟漂移影响）
-- 通过 EXCHANGE TABLES 原子切换，不阻塞写入。

-- 1. 创建新 schema（_v2 后缀）
CREATE TABLE IF NOT EXISTS md_ticks_v2 (
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
) ENGINE = ReplacingMergeTree(arrived_unix_ms)
PARTITION BY toYYYYMM(toDateTime(arrived_unix_ms / 1000))
ORDER BY (broker, canonical, ts_unix_ms, bid, ask, bid_volume, ask_volume)
TTL toDateTime(arrived_unix_ms / 1000) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- 2. 历史数据迁移（由 migration runner 执行；长事务 OK，不阻塞写入）
INSERT INTO md_ticks_v2 SELECT *, 0 AS is_replay FROM md_ticks;

-- 3. 原子切换
EXCHANGE TABLES md_ticks AND md_ticks_v2;

-- 4. 旧 schema 保留 24h 兜底（由运维手动 DROP）
RENAME TABLE md_ticks_v2 TO md_ticks_legacy;
