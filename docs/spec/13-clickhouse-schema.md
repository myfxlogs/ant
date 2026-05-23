# 13 · ClickHouse 时序 schema 规范

> 路径：`backend/internal/mdgateway/chmigrate/`
> 配套迁移：`make migrate-ch`
> 数据库名：`ant`（与 PG 同名，不冲突）

## 1. 表清单

| 表名 | 用途 | 写入方 | 读取方 | TTL |
|---|---|---|---|---|
| `md_ticks` | 实时 tick | mdgateway.CHWriter | research、调试 | 90 天 |
| `md_bars` | OHLCV bar | mdgateway.CHWriter | factorsvc、用户查询、回测 | 不过期 |
| `factor_values` | 因子值 | factorsvc | quantengine、研究 | 2 年 |
| `signals` | 策略信号 | quantengine | 审计、回测对照 | 2 年 |
| `_schema_migrations` | 迁移记录 | chmigrate | 启动时检查 | 永久 |

## 2. 完整 DDL

### 2.1 `001_md_ticks.sql`

```sql
-- 001_md_ticks.sql
-- 实时 tick 存储；按月分区；90 天 TTL
-- ReplacingMergeTree 配合 mdgateway.tick_dedup 提供幂等保证

CREATE TABLE IF NOT EXISTS md_ticks (
    user_id          LowCardinality(String),
    account_id       LowCardinality(String),
    broker           LowCardinality(String),
    symbol_raw       LowCardinality(String),
    canonical        LowCardinality(String),
    ts_unix_ms       UInt64,
    arrived_unix_ms  UInt64,
    bid              Decimal(18, 6),
    ask              Decimal(18, 6),
    bid_volume       Float64,
    ask_volume       Float64,
    INDEX idx_canonical canonical TYPE bloom_filter GRANULARITY 4
) ENGINE = ReplacingMergeTree(arrived_unix_ms)
PARTITION BY toYYYYMM(toDateTime64(ts_unix_ms / 1000.0, 3))
ORDER BY (broker, canonical, ts_unix_ms)
TTL toDateTime64(ts_unix_ms / 1000.0, 3) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
```

### 2.2 `002_md_bars.sql`

```sql
-- 002_md_bars.sql
-- OHLCV bars，6 周期共表，period 作为字段
-- 长期保留，作为回测主数据源

CREATE TABLE IF NOT EXISTS md_bars (
    user_id           LowCardinality(String),
    account_id        LowCardinality(String),
    broker            LowCardinality(String),
    symbol_raw        LowCardinality(String),
    canonical         LowCardinality(String),
    period            LowCardinality(String),  -- '1m','5m','15m','1h','4h','1d'
    open_ts_unix_ms   UInt64,
    close_ts_unix_ms  UInt64,
    open              Decimal(18, 6),
    high              Decimal(18, 6),
    low               Decimal(18, 6),
    close             Decimal(18, 6),
    volume            Float64,
    tick_count        UInt32
) ENGINE = ReplacingMergeTree(close_ts_unix_ms)
PARTITION BY toYYYYMM(toDateTime64(close_ts_unix_ms / 1000.0, 3))
ORDER BY (broker, canonical, period, close_ts_unix_ms)
SETTINGS index_granularity = 8192;
```

### 2.3 `003_factor_values.sql`

```sql
-- 003_factor_values.sql
-- 因子值，2 年 TTL

CREATE TABLE IF NOT EXISTS factor_values (
    user_id      LowCardinality(String),
    account_id   LowCardinality(String),
    broker       LowCardinality(String),
    canonical    LowCardinality(String),
    period       LowCardinality(String),
    factor_name  LowCardinality(String),
    value        Float64,
    ts_unix_ms   Int64,
    created_at   DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (user_id, factor_name, canonical, period, ts_unix_ms)
TTL created_at + INTERVAL 2 YEAR;
```

### 2.4 `004_signals.sql`

```sql
-- 004_signals.sql
-- 策略信号审计表

CREATE TABLE IF NOT EXISTS signals (
    user_id        LowCardinality(String),
    strategy_id    String,
    deployment_id  String,
    account_id     LowCardinality(String),
    canonical      LowCardinality(String),
    ts             DateTime64(3, 'UTC'),
    side           Int8,           -- +1 buy, -1 sell, 0 hold
    target_qty     Float64,
    limit_price    Nullable(Float64),
    client_id      String,
    factor_snap    Map(String, Float64),  -- 触发时的因子快照
    rejected       UInt8,          -- 0 = sent, 1 = rejected by risk
    reject_reason  String,
    created_at     DateTime64(3, 'UTC') DEFAULT now64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(ts)
ORDER BY (user_id, strategy_id, ts)
TTL ts + INTERVAL 2 YEAR;
```

### 2.5 `005_schema_version.sql`

```sql
-- 005_schema_version.sql
-- 迁移版本追踪（chmigrate 启动时检查）

CREATE TABLE IF NOT EXISTS _schema_migrations (
    version    UInt32,
    name       String,
    applied_at DateTime64(3, 'UTC') DEFAULT now64(3),
    checksum   String              -- SHA256(sql) 用于检测漂移
) ENGINE = ReplacingMergeTree(applied_at)
ORDER BY version;
```

## 3. 查询模式（reference patterns）

### 3.1 用户查询最近 1000 根 1m K 线

```sql
SELECT
    open_ts_unix_ms,
    open, high, low, close,
    volume, tick_count
FROM md_bars
WHERE user_id = {user_id:String}
  AND canonical = {canonical:String}
  AND period = '1m'
  AND close_ts_unix_ms <= {to_ms:UInt64}
ORDER BY close_ts_unix_ms DESC
LIMIT 1000;
```

### 3.2 因子时间序列（用于研究）

```sql
SELECT
    ts_unix_ms,
    canonical,
    value
FROM factor_values
WHERE user_id = {user_id:String}
  AND factor_name = {factor_name:String}
  AND period = '1m'
  AND ts_unix_ms BETWEEN {from_ms:Int64} AND {to_ms:Int64}
ORDER BY canonical, ts_unix_ms;
```

### 3.3 信号审计

```sql
SELECT
    ts, canonical, side, target_qty,
    rejected, reject_reason, factor_snap
FROM signals
WHERE user_id = {user_id:String}
  AND strategy_id = {strategy_id:String}
  AND ts BETWEEN {from:DateTime64} AND {to:DateTime64}
ORDER BY ts DESC
LIMIT 500;
```

### 3.4 实时监控（最近 1 分钟 tick rate）

```sql
SELECT
    broker, canonical,
    count() AS ticks,
    max(ts_unix_ms) AS latest_ms
FROM md_ticks
WHERE arrived_unix_ms > now64(3) * 1000 - 60000
GROUP BY broker, canonical
ORDER BY ticks DESC;
```

## 4. PG 业务表配套（v2 仅新增）

### 4.1 `mt_accounts`（保留 + 修改）

ant 已有 `mt_accounts` 表（来自 v1）；v2 仅新增字段，不破坏：

```sql
-- migrations/0XX_mt_accounts_v2_fields.up.sql
ALTER TABLE mt_accounts
    ADD COLUMN IF NOT EXISTS mtapi_host TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS mtapi_port TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS mtapi_token_encrypted BYTEA,
    ADD COLUMN IF NOT EXISTS canonical_subscribed_symbols TEXT[]
        NOT NULL DEFAULT ARRAY[]::TEXT[];
```

字段说明：
- `mtapi_host/port`: 该账户使用的 mtapi.io 网关地址（多区域支持）
- `mtapi_token_encrypted`: AES-GCM 加密的 mtapi token
- `canonical_subscribed_symbols`: 该账户订阅的 canonical 列表（启动时 SubscribeMany 用）

### 4.2 `broker_symbols`（v2 新建）

```sql
-- migrations/0XX_broker_symbols.up.sql
CREATE TABLE IF NOT EXISTS broker_symbols (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broker          TEXT NOT NULL,
    symbol_raw      TEXT NOT NULL,
    canonical       TEXT NOT NULL,
    digits          INT NOT NULL DEFAULT 5,
    point_value     NUMERIC(20,8) NOT NULL DEFAULT 0.00001,
    lot_size        NUMERIC(20,8) NOT NULL DEFAULT 100000,
    lot_step        NUMERIC(20,8) NOT NULL DEFAULT 0.01,
    lot_min         NUMERIC(20,8) NOT NULL DEFAULT 0.01,
    lot_max         NUMERIC(20,8) NOT NULL DEFAULT 1000,
    trade_mode      INT NOT NULL DEFAULT 4,
    last_updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (broker, symbol_raw)
);

CREATE INDEX idx_broker_symbols_canonical ON broker_symbols(canonical);
```

### 4.3 `factor_definitions`（v2 新建）

```sql
-- migrations/0XX_factor_definitions.up.sql
CREATE TABLE IF NOT EXISTS factor_definitions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,            -- 例如 "ma20_ratio"
    expression    TEXT NOT NULL,            -- DSL 字符串
    canonicals    TEXT[] NOT NULL,          -- 适用的 canonical 列表（空=全部订阅）
    periods       TEXT[] NOT NULL,          -- '1m','5m',...
    lookback      INT NOT NULL DEFAULT 100,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, name)
);
```

### 4.4 老 `kline_data` 表（M7-rewrite 期保留 + M9 删除）

- M7 期间：`kline_data` 设为只读（CHECK INSERT 失败）
- M8：业务层 grep 0 处直读
- M9：DROP TABLE 并删除 `migrations/003_kline_data.up.sql`

```sql
-- M9 时执行的清理 migration
-- migrations/0XX_drop_legacy_kline.up.sql
DROP TABLE IF EXISTS kline_data;
DROP TABLE IF EXISTS tick_data;  -- 如有
```

## 5. chmigrate 实现

### 5.1 `chmigrate/migrate.go`

```go
// Package chmigrate runs ClickHouse schema migrations.
package chmigrate

import (
    "context"
    "crypto/sha256"
    "embed"
    "encoding/hex"
    "fmt"
    "regexp"
    "sort"
    "strings"

    "github.com/ClickHouse/clickhouse-go/v2"
    "go.uber.org/zap"
)

//go:embed *.sql
var migrations embed.FS

var migFileRe = regexp.MustCompile(`^(\d{3})_(.+)\.sql$`)

// Run executes pending migrations idempotently.
// 1. 确保 _schema_migrations 表存在
// 2. 列出 embed 中所有 sql 文件，按版本排序
// 3. 对每个文件：
//    - 计算 checksum
//    - 如 _schema_migrations 已有同 version 同 checksum → skip
//    - 如已有但 checksum 不同 → fatal（schema drift）
//    - 否则执行 + 记录
func Run(ctx context.Context, conn clickhouse.Conn, log *zap.Logger) error
```

### 5.2 启动调用

`backend/cmd/ant-server/main.go`：

```go
ch, err := clickhouse.Open(&clickhouse.Options{...})
if err := chmigrate.Run(ctx, ch, log); err != nil {
    log.Fatal("ch migration", zap.Error(err))
}
```

## 6. PG ↔ CH 边界（决策快照）

| 数据类别 | 存储 | 理由 |
|---|---|---|
| 用户、账户绑定、订单、风控配置 | PG | 事务、外键约束、关系查询 |
| AI 对话、策略市场、admin 审计 | PG | 同上 |
| Tick / Bar / Factor / Signal | CH | 时序、压缩、亿行查询 |
| 配置（broker_symbols / factor_definitions） | PG | 业务可编辑 |
| Session / 缓存 | Redis | TTL、低延迟 |

详细决策见 `docs/adr/0002-clickhouse-as-timeseries.md`。

## 7. 容量规划（M7 立项）

| 表 | 平均行宽 | 行/账户/天 | 100 账户/年 | 压缩后 |
|---|---|---|---|---|
| md_ticks | ~80B | 5M | 18 万亿行 | ~150 GB（90d TTL）|
| md_bars | ~120B | 1.5K | 5400 万行 | ~2 GB（终生）|
| factor_values | ~70B | 50K (5 因子) | 18 亿行 | ~30 GB（2y）|
| signals | ~200B | 50 | 180 万行 | ~50 MB |

满足 PG **绝不**能承受、CH 轻松吃下的设计目标。

## 8. 验收命令

```bash
# 启动 CH
docker compose up -d ant-clickhouse
sleep 5

# 跑迁移
docker exec ant-backend /app/ant-server migrate-ch  # 或 make migrate-ch

# 检查表
docker exec ant-clickhouse clickhouse-client --query "
  SELECT name FROM system.tables WHERE database='ant' ORDER BY name
" | sort > /tmp/ch_tables.txt

cat <<EOF | sort | diff - /tmp/ch_tables.txt
_schema_migrations
factor_values
md_bars
md_ticks
signals
EOF

# 检查 TTL
docker exec ant-clickhouse clickhouse-client --query "
  SELECT name, ttl FROM system.tables WHERE database='ant' AND name IN ('md_ticks','factor_values','signals')
" | grep -E "INTERVAL.*(90|2 YEAR)" || exit 1

# 检查 ReplacingMergeTree
docker exec ant-clickhouse clickhouse-client --query "
  SELECT name, engine FROM system.tables WHERE database='ant' AND name IN ('md_ticks','md_bars')
" | grep -c ReplacingMergeTree | grep -q 2 || exit 1
```
