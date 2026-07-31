# 行情引入管线审计报告

**日期**: 2026-07-18
**范围**: `backend/internal/mdgateway/` 全部代码 + `backend/internal/repository/market_data_*.go`
**基准**: ADR-0012 (去除 ClickHouse 和 tick 持久化)

---

## 1. 审计范围

| 模块 | 文件 | 功能 |
|------|------|------|
| Manager | `manager.go`, `manager_tick.go` | 6 阶段 tick 管线编排 |
| Normalizer | `normalizer.go` | (broker, symbol_raw) → canonical 解析 |
| Quality | `quality.go` | tick 质量检查 (bid>ask, 非正, 时钟偏移, 离群点) |
| Dedup | `tick_dedup.go` | 滑动窗口去重 (防 broker 重连重发) |
| BarAggregator | `bar_aggregator.go` | tick → 多周期 OHLCV bar 聚合 |
| Publisher | `publisher.go` | NATS JetStream tick/bar 发布 |
| PgWriter | `pg_writer.go` | 批量 bar 写入 PostgreSQL |
| Backfiller | `backfiller/target.go`, `wiring.go` | 历史 bar 回填 |
| Repository | `market_data_store.go`, `market_data_pg.go`, `market_data_types.go` | PG 市场数据读写 |
| Adapters | `adapter/mt4/quotes.go`, `adapter/mt5/quotes.go` | MT4/MT5 quote 流订阅 |
| DTOs | `adapter/mdtick/mdtick.go` | Tick/Bar 共享数据结构 |

## 2. 发现的问题

### P0 — 🔴 严重: Normalizer 并发 map 无锁 data race

**文件**: `normalizer.go`
**触发场景**: 多账户 recvLoop 并发调用 `Resolve()`
**影响**: Go race detector 必报 fatal; 生产环境 map 并发读写可致 panic 或数据损坏
**根因**: `Normalizer.cache` 为普通 `map[string]string`，无任何锁保护，但 `Resolve()` 被多个 gateway goroutine 并发调用

**修复**: 添加 `sync.RWMutex`，使用 read-lock 快速路径 + write-lock double-check 模式

### P1 — 🟠 高: PgWriter tick 死路径浪费资源

**文件**: `pg_writer.go`, `manager_tick.go`
**触发场景**: 每个 tick 都经过 `EnqueueTick` → channel → batch → `flushTicks` → drop
**影响**: 每 tick 分配 channel slot + batch slice 增长 + 最终丢弃，纯浪费内存/CPU
**根因**: ADR-0012 禁用 tick 持久化后，`flushTicks` 变为纯 log+drop，但 `EnqueueTick` 调用和 tickQ channel 仍在

**修复**: 
- 删除 `tickQ` channel、`EnqueueTick()`、`flushTicks()`
- `manager_tick.go` 移除 `EnqueueTick` 调用
- `Drain()` 和 `Flush()` 签名简化为仅处理 bars
- 删除 `MarketDataStore.InsertTicks` 接口方法和 `TickRecord` 类型

### P1 — 🟠 高: InsertBars 丢失 is_replay/account_id/symbol_raw

**文件**: `market_data_pg.go:InsertBars`, `market_data_types.go:KlineBar`, `bar_aggregator.go`
**触发场景**: backfiller 写入的 bar 和实时聚合的 bar
**影响**: 
- `is_replay` 硬编码为 0 → 回填 bar 无法与实时 bar 区分，`GetKlines` 的 `is_replay = 0` 过滤器无法排除回填数据
- `account_id` 硬编码为空 → 无法追溯 bar 来源账户
- `symbol_raw` 硬编码为 canonical → 丢失 broker 原始符号

**修复**:
- `KlineBar` 添加 `IsReplay`, `AccountID`, `SymbolRaw` 字段
- `mdtick.Bar` 添加 `SymbolRaw` 字段
- `bar_aggregator.go` 的 `openBar` 添加 `symbolRaw`，在 `AddTick` 中传递 `SymbolRaw` 和 `IsReplay`
- `pg_writer.go` 的 `flushBars` 填充新字段
- `InsertBars` 使用 `b.IsReplay`, `b.AccountID`, `b.SymbolRaw` 替代硬编码值

### P2 — 🟡 中: backfiller/target.go CH 残留

**文件**: `backfiller/target.go`, `wiring.go`
**触发场景**: 维护者阅读代码时被误导
**影响**: `chWriter` 字段和 "CH + PG during migration" 注释暗示双写架构仍在

**修复**: 
- `TargetAdapter` 移除 `chWriter` 字段，`NewTarget` 从 4 参数简化为 3 参数
- 注释更新为 "PG storage" 单一后端
- `wiring.go` 调用更新

### P2 — 🟡 中: 多处 CH/ClickHouse 过时注释

**文件**: `user_metrics_flusher.go`, `metrics_histogram.go`, `otel.go`, `market_data_types.go`, `pg_writer.go`
**影响**: 误导维护者，暗示 ClickHouse 仍在使用

**修复**: 全部注释更新为反映 ADR-0012 后的架构

## 3. 变更文件清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `normalizer.go` | P0 修复 | 添加 RWMutex，double-check locking |
| `pg_writer.go` | P1 修复 | 删除 tick 死路径，简化 Flush/Drain |
| `manager_tick.go` | P1 修复 | 移除 EnqueueTick 调用 |
| `runner.go` | P1 修复 | Drain 调用适配新签名 |
| `market_data_store.go` | P1 修复 | 删除 InsertTicks 接口 + TickRecord 类型 |
| `market_data_pg.go` | P1 修复 | 删除 InsertTicks 实现，InsertBars 使用真实字段 |
| `market_data_types.go` | P1 修复 | KlineBar 添加 3 字段 + isReplayToInt16 helper |
| `adapter/mdtick/mdtick.go` | P1 修复 | Bar 添加 SymbolRaw 字段 |
| `bar_aggregator.go` | P1 修复 | openBar 添加 symbolRaw，传递 IsReplay/SymbolRaw |
| `backfiller/target.go` | P2 修复 | 移除 chWriter，NewTarget 3 参数 |
| `wiring.go` | P2 修复 | NewTarget 调用适配 |
| `backfiller_test.go` | P2 修复 | 测试适配新 NewTarget 签名 |
| `pure_test.go` | P1 修复 | Flush/Drain 测试适配新签名 |
| `user_metrics_flusher.go` | P2 清理 | 移除 CH 注释 |
| `metrics_histogram.go` | P2 清理 | 移除 clickhouse_writer.go 引用 |
| `otel.go` | P2 清理 | chwrite → enqueue |

## 4. 验证结果

- `go build ./...` — ✅ 通过
- `go test ./internal/mdgateway/... -count=1` — ✅ 全部通过 (7 包)
- `go test ./internal/repository/... -count=1` — ✅ 通过
- `go run ./tools/check-file-lines --strict` — ✅ 0 errors

## 5. 架构合规性

| 规则 | 状态 |
|------|------|
| ConnectRPC + SSE only (无 REST) | ✅ 未涉及 |
| proto only (无 JSON) | ✅ 未涉及 |
| decimal.Decimal (无 float64 价格计算) | ✅ bar OHLC 使用 decimal |
| Push-first (无 polling) | ✅ gRPC streaming + NATS |
| ADR-0012: 无 tick 持久化 | ✅ 死路径已清除 |
| ADR-0012: Redis 最新报价缓存 | ✅ manager_tick.go 中实现 |
| ADR-0012: PG sole storage | ✅ CH 残留已清除 |
