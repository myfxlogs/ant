# ADR-0014: ClickHouse 时序存储 — PG 业务库 vs CH 时序库职责边界

- **Status**: Accepted
- **Date**: 2026-05-23
- **Deciders**: ant 架构组
- **Replaces**: 无
- **Affected**: M7.4 mdgateway, M7.8 runner, chmigrate, factorsvc

## Context

ant 项目引入 ClickHouse 作为时序市场数据存储。需要明确 PG (业务库) 和 CH (时序库) 的职责边界，避免双写不一致和查询路径混乱。

## Decision

### 职责划分

| 数据类别 | 存储 | 说明 |
|---------|------|------|
| 用户/账户/策略/订单 | PG | 业务数据，需要 ACID 事务 |
| Tick 行情 | CH `md_ticks` | 时序数据，按月分区，90 天 TTL |
| OHLCV Bar | CH `md_bars` | ReplacingMergeTree，按月分区 |
| 因子值 | CH `factor_values` | MergeTree，2 年 TTL |
| 策略信号 | CH `signals` | MergeTree，2 年 TTL |
| mt_accounts | PG | 账户配置，通过 Runner 加载到内存 |

### TTL 策略

- `md_ticks`: 90 天（高频数据，存储成本高）
- `md_bars`: 无限（聚合数据，查询热点）
- `factor_values`: 2 年（回测需要历史因子）
- `signals`: 2 年（审计需要）

### 写入路径

```
MT Quote → mdgateway Manager → CHWriter (batch) → md_ticks/md_bars
                              ↘ SpillWriter (CH 故障时)
                              ↘ NATS Publisher (跨服务)
```

### 查询策略

- K 线查询：先查 CH `md_bars`，miss 回退 PG `kline_data`（过渡期）
- 因子查询：直接读 CH `factor_values`
- 审计查询：PG 为主（业务语义完整）

## Consequences

- **正**：PG 不再承受时序写入压力，CH 擅长列存+压缩
- **正**：CH TTL 自动清理，无需 PG vacuum 时序表
- **负**：增加运维复杂度（CH 集群、备份）
- **负**：过渡期查询需双路径
