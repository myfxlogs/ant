-- 007_md_bars_v2.sql
-- M10 ADR-0008/0009: TTL/PARTITION 切到 close_ts_unix_ms（实质 = arrived_unix_ms，见 ADR-0009 §2.2）
-- ORDER BY 不变（bar 不需要 volume 去重）；仅修正时间轴来源。
-- 通过 EXCHANGE TABLES 原子切换，不阻塞写入。

-- 1. 创建新 schema（_v2 后缀）
CREATE TABLE IF NOT EXISTS md_bars_v2 (
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
) ENGINE = ReplacingMergeTree(close_ts_unix_ms)
PARTITION BY toYYYYMM(toDateTime(close_ts_unix_ms / 1000))
ORDER BY (broker, canonical, period, close_ts_unix_ms)
SETTINGS index_granularity = 8192;

-- 2. 历史数据迁移
INSERT INTO md_bars_v2 SELECT *, 0 AS is_replay FROM md_bars;

-- 3. 原子切换
EXCHANGE TABLES md_bars AND md_bars_v2;

-- 4. 旧 schema 保留 24h 兜底（由运维手动 DROP）
RENAME TABLE md_bars_v2 TO md_bars_legacy;
