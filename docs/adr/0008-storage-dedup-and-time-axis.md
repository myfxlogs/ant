# ADR-0008 · 存储层去重键对齐 + 时间轴纪律

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/11-mdgateway.md` §6 §7 / `docs/spec/13-clickhouse-schema.md` §2.1 §2.2
- **关联 ADR**：ADR-0004（tick 去重与质量分级）

## 1. 背景

M7 已落地两条独立的去重链路：
1. **应用层**（`mdgateway/tick_dedup.go`）：100-window xxhash，**包含** `(ts_unix_ms, bid, ask, bid_volume, ask_volume)`（D-2 决策修复）
2. **存储层**（CH `md_ticks`）：`ENGINE = ReplacingMergeTree(arrived_unix_ms)` `ORDER BY (broker, canonical, ts_unix_ms)`

两者**去重键不一致**：应用层认为同毫秒 + 不同量是合法两笔 tick；存储层 ORDER BY 元组不含量字段，后台 `OPTIMIZE FINAL` 合并时仍会把它们折成一笔。

副作用：
- `count(md_ticks)` 与 metric `md_tick_total` 长期对不上，无法做端到端对账（spec/19 md-doctor 失效）
- 活跃 broker 欧盘 EURUSD / 黄金美盘 1s 内同价位多笔成交丢失，bar 的 `Volume` 字段失真
- 回测复盘成交量分布出现偏差

并行问题：**TTL 用 broker 时钟（ts_unix_ms）**：
- broker 时钟回拨 1h（节假日时区误配） → 数据被 CH TTL 提前删除
- broker 时钟超前 → spill replay 灌入的"历史 tick" 当下立即被 TTL 判为已过期（取决于 partition 选择）

## 2. 决策

### 2.1 ORDER BY 与 dedup hash 字段对齐

`md_ticks` 与 `md_bars` ORDER BY 元组扩展到与应用层 dedup hash **完全一致**的字段集，由新迁移 `006_md_ticks_v2.sql` `007_md_bars_v2.sql` 通过 `RENAME TABLE → CREATE 新表 → INSERT SELECT → DROP 旧表`（atomic with `EXCHANGE TABLES`）落地，不阻塞写入。

```sql
-- 新 md_ticks
ORDER BY (broker, canonical, ts_unix_ms, bid, ask, bid_volume, ask_volume)
PARTITION BY toYYYYMM(toDateTime64(arrived_unix_ms / 1000.0, 3))
TTL toDateTime64(arrived_unix_ms / 1000.0, 3) + INTERVAL 90 DAY
```

```sql
-- 新 md_bars
ORDER BY (broker, canonical, period, close_ts_unix_ms)
PARTITION BY toYYYYMM(toDateTime64(close_ts_unix_ms / 1000.0, 3))
-- close_ts_unix_ms 由 bar_aggregator 在 emit 时取 ArrivedUnixMs（local 时钟），见 ADR-0009 §2.2
```

ReplacingMergeTree 仍保留（双保险：应用层 dedup ring 满漏出时存储层兜底）。

### 2.2 时间轴纪律（强制规则）

| 字段 | 时钟来源 | 用途 |
|---|---|---|
| `ts_unix_ms` | broker 时钟 | 业务可读、跨 broker **不可比** |
| `arrived_unix_ms` | mdgateway 本地时钟 | 分桶、去重、TTL、分区、ORDER BY、bar 边界 |

**所有面向系统的时间字段（partition、TTL、index、bar_aggregator bucket）必须使用 `arrived_unix_ms`**。`ts_unix_ms` 只用于：
- 业务展示（K 线 X 轴 broker 时间）
- broker 时钟漂移检测（与 arrived 差值进 `md_clock_skew_seconds` gauge）

### 2.3 迁移脚本设计

`backend/internal/mdgateway/chmigrate/006_md_ticks_v2.sql`：
```sql
-- 1. 新表（带 _v2 后缀，schema 同 §2.1）
CREATE TABLE IF NOT EXISTS md_ticks_v2 (...) ENGINE = ReplacingMergeTree(arrived_unix_ms)
PARTITION BY toYYYYMM(toDateTime64(arrived_unix_ms / 1000.0, 3))
ORDER BY (broker, canonical, ts_unix_ms, bid, ask, bid_volume, ask_volume)
TTL toDateTime64(arrived_unix_ms / 1000.0, 3) + INTERVAL 90 DAY;

-- 2. 历史数据迁移（背景任务，长事务 OK）
INSERT INTO md_ticks_v2 SELECT * FROM md_ticks;

-- 3. 原子切换
EXCHANGE TABLES md_ticks AND md_ticks_v2;

-- 4. 异步删旧表（保留 24h 兜底）
RENAME TABLE md_ticks_v2 TO md_ticks_legacy;
-- 由运维 24h 后手动 DROP TABLE md_ticks_legacy
```

迁移期间 CHWriter 写入路径不变（写 `md_ticks` 名）。`EXCHANGE TABLES` 是 ClickHouse 原子操作，SELECT 不会失败。

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|---|---|---|---|
| **MergeTree（不去重）** | 简单 | 应用层 dedup ring 满 / 重启时漏的重复 tick 永久存在 | 失去存储层兜底，长期数据脏 |
| **CH `dedup` SETTING** | 配置即可 | 仅 INSERT 级，对后台合并无效 | 不解决问题 |
| **同时保留两份表（去重 + 不去重）** | 灵活 | 容量翻倍、运维复杂、查询路径分裂 | ROI 太低 |
| **本决策：ORDER BY 加量字段** | 与应用层一致；ReplacingMergeTree 兜底；零运行时成本 | ORDER BY 元组变长，索引体积小幅增加 | ✓ 选用 |

## 4. 后果

- **正面**：应用层与存储层去重语义完全一致；端到端对账（md-doctor）可用；TTL 不再受 broker 时钟漂移影响
- **负面**：md_ticks 索引体积约 +5%（量字段 LowCardinality(Float64)，影响有限）；一次 `INSERT SELECT` 全表迁移在 1.5T 数据下耗时数小时（背景执行，不阻塞写）
- **中性**：所有"按 broker 时钟"的查询代码必须显式说明意图（注释 `// uses broker clock`）

## 5. 实施约束

1. 新增迁移 `006_md_ticks_v2.sql` `007_md_bars_v2.sql` `008_time_axis_check.sql`（最后一条仅创建系统视图便于审计）
2. `clickhouse_writer.go` `bar_aggregator.go` 任何引用 `ts_unix_ms` 做边界判断的地方改为 `arrived_unix_ms`
3. `spec/13` §2.1 §2.2 内联 schema 同步更新
4. `spec/15` 加 alert：`abs(md_clock_skew_seconds) > 30 for 2m` → `BrokerClockSkewHigh`

## 6. 验证方式

```bash
# 1. ORDER BY 含全部 dedup 字段
docker exec ant-clickhouse clickhouse-client --query "
  SELECT sorting_key FROM system.tables WHERE database='ant' AND name='md_ticks'
" | grep -E 'ts_unix_ms.*bid.*ask.*bid_volume.*ask_volume'

# 2. PARTITION/TTL 用 arrived_unix_ms
docker exec ant-clickhouse clickhouse-client --query "
  SELECT partition_key, engine_full FROM system.tables WHERE database='ant' AND name='md_ticks'
" | grep -E 'arrived_unix_ms'

# 3. 端到端对账：metric 与 CH 计数差 < 0.01%
TICK_METRIC=$(curl -s localhost:8080/metrics | grep '^md_tick_total' | awk '{s+=$NF} END{print s}')
TICK_CH=$(docker exec ant-clickhouse clickhouse-client --query "SELECT count() FROM ant.md_ticks WHERE arrived_unix_ms > now64()*1000 - 600000")
python3 -c "import sys; m,c=$TICK_METRIC,$TICK_CH; sys.exit(abs(m-c)/max(m,1) > 0.0001)"

# 4. 历史数据完整迁移（_legacy 表与新表行数一致）
OLD=$(docker exec ant-clickhouse clickhouse-client --query "SELECT count() FROM ant.md_ticks_legacy")
NEW=$(docker exec ant-clickhouse clickhouse-client --query "SELECT count() FROM ant.md_ticks")
test "$NEW" -ge "$OLD"
```
