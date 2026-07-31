# ADR 0012 — 移除 ClickHouse，Tick 不落盘，PG 为唯一市场数据存储

**日期**: 2026-07-31
**状态**: 已执行

## 背景

当前 PG 全量存储 Tick（`md_ticks`），2 个交易账号月增 1470 万行 / 2.5 GB。按此线性增长，100 账号将达 7.35 亿行 / 125 GB / 月，每日全量备份不可持续。

`md_ticks` 表来源：原始架构用 ClickHouse 存储时序数据，迁移 158（`48fe502c`）将 CH 表 1:1 机械搬移至 PG 分区表。迁移时未重新评估"PG 是否需要存 Tick"——CH 里合理，PG 里不合理。

## 分析

**Tick 写入路径**：MT4/MT5 → NATS → `PgWriter.InsertTicks()` → PG `md_ticks`

**Tick 读取路径**（全链路审计）：

| 消费者 | 读 md_ticks？ | 说明 |
|--------|:--:|------|
| 回测引擎 | ❌ | `engine.go:150` 用 Bar 收盘价 + 点差合成 Tick，不查 DB |
| 策略运行器 | ❌ | 实时从 NATS 消费报价 |
| `GetLatestTick` | ✅ | 唯一读者，每个品种取最新 1 行 |

**结论**：1470 万行中 99.99% 无人查询。Tick 持久化是从 ClickHouse 时代机械搬运的遗留负担，不是设计意图。

## 决策

**PG 为唯一市场数据存储。Tick 通过 NATS 消费后即丢弃，不落盘。移除 ClickHouse 全部痕迹。**

### 新架构

```
PG（Bar + 业务数据）  ← 回测 / 图表 / 信号计算
NATS（Tick 瞬态）    ← 实时报价流，消费后丢弃
Redis（最新报价缓存） ← GetLatestTick 替代方案
```

### 不复用的代码（需删除）

| 文件 | 原因 |
|------|------|
| `backend/internal/repository/ch_market_data_store.go` | 纯 CH 实现，CH 容器不启动，永不执行 |
| `backend/internal/repository/multi_market_data_store.go` | CH→PG 路由层，CH 移除后无意义，只剩一层转发 |
| `docker-compose.yml` 中 clickhouse 容器 + volume | 从未启动，定义残留 |

## 执行清单（按顺序）

1. **备份脚本** — `scripts/backup-db.sh`
   - `pg_dump` 加 `--exclude-table='md_ticks_*'`
   - 效果：227 MB → ~7 MB，与账号数无关

2. **MVB（Minimum Viable Bridge）——防御性停写**
   - `backend/internal/mdgateway/pg_writer.go`：`InsertTicks` 改为 no-op + `log.Warn("tick persistence disabled, dropping ticks")` + `return nil`
   - 保留调用点和函数签名不变，先跑通再删
   - 这一步确保 PG `md_ticks` 不再增长

3. **GetLatestTick 改读 Redis**
   - `backend/internal/repository/pg_market_data_store.go`（或 PG 实现文件）：
     - `GetLatestTick` → Redis `GET latest_quote:{canonical}`
   - NATS 消费端（`mdgateway` pipeline）：每次收到报价 → `SETEX latest_quote:{canonical} 3600 {bid,ask}`

4. **DROP md_ticks 分区表**
   - 新建迁移 `backend/migrations/159_drop_md_ticks.up.sql`
   - `DROP TABLE IF EXISTS md_ticks CASCADE;`
   - 对应的 down.sql：无（不可逆操作）

5. **清 ClickHouse 残留**
   - `docker-compose.yml`：删除 `clickhouse` 容器定义、`clickhouse_data` volume
   - `.env.example`：删除 `CH_*` 环境变量
   - `backend/cmd/server/main.go`：删除 `connectClickHouse()` 函数和所有调用
   - `backend/internal/repository/ch_market_data_store.go`：删除文件
   - `backend/internal/repository/multi_market_data_store.go`：删除文件，所有调用点直接使用 `PgMarketDataStore`
   - `backend/go.mod`：`go mod tidy` 移除 `clickhouse-go` 依赖

6. **更新文档**
   - `docs/adr/0003-PG-CH-Redis.md`：标注 CH 已废弃
   - `docs/26-依赖与版本管理规范.md`：移除 CH 版本要求
   - `CLAUDE.md`：更新 Data Precision 行（删除 `CH` 引用）

7. **验证**
   - `cd backend && go build ./...`
   - `docker compose up -d` 正常启动
   - 执行一次备份，确认 < 10 MB
   - Redis `latest_quote:*` key 存在且值正确

## 不做的

- 不删除 `md_bars` 表 — Bar 是回测/图表的唯一数据源，必须保留
- 不删除 MT5 `OnTickHistory` 相关 RPC 封装 — 保留为"按需拉取不落盘"能力
- 不删除 Redis 已有配置 — 已部署，直接复用
- 不修改前端

## 回滚

如出问题（如 Redis 不可用时 `GetLatestTick` 失败），临时回退：重启 PG `md_ticks` 写入（revert step 2），前端报错降级为"等待下次报价推送"。Redis 与 PG 同时故障的概率极低。

## 影响

- 备份大小：227 MB → ~7 MB（永久）
- 100 账号月增：125 GB → 1.3 GB（仅 Bar）
- 运维：减 1 个容器定义、~500 行死代码
- 功能：无变化（Tick 原本无消费者）
