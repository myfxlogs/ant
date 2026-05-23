# M7 迁移审计清单 — alfq → ant

> 生成日期: 2026-05-23
> 范围: `backend/internal/{mdgateway,factorsvc,factor}` + `research/`
> 状态: 部分已迁移

---

## 概览

| 区域 | 文件数 | 包名 | 状态 |
|------|--------|------|------|
| `backend/internal/mdgateway/` | 20 | `anttrader/internal/mdgateway` | ✅ 已迁移 |
| `backend/internal/mdgateway/adapter/mt4/` | 1 | `anttrader/internal/mdgateway/adapter/mt4` | ✅ 新建 |
| `backend/internal/mdgateway/adapter/mt5/` | 1 | `anttrader/internal/mdgateway/adapter/mt5` | ✅ 新建 |
| `backend/internal/mdgateway/backfill/` | 2 | `anttrader/internal/mdgateway/backfill` | ✅ 新建 |
| `backend/internal/mdgateway/chmigrate/` | 5 | `anttrader/internal/mdgateway/chmigrate` | ✅ 已迁移 |
| `backend/internal/factorsvc/` | 11 | `anttrader/internal/factorsvc` | ✅ 已迁移 |
| `backend/internal/factor/dsl/` | 14 | `anttrader/internal/factor/dsl` | ✅ 已迁移 |
| `research/` | 14 (源码) | — | ✅ 已迁移 |

---

## 逐文件差异清单

### 1. `backend/internal/mdgateway/`

#### 来源对照
这些文件从 alfq 项目迁移，针对 ant 做了以下适配：

| 文件 | alfq 原版 | ant 迁移版 | 关键变更 |
|------|-----------|-----------|----------|
| `types.go` | `pb.Tick` / `pb.Bar` / `pb.Money` (proto) | 本地 `Tick` / `Bar` / `Money` 结构体 | `tenant_id` → `user_id`；proto getter 保留兼容 |
| `deps.go` | alfq 依赖容器 | ant 依赖容器 | `SetRole()` 改为 no-op（ant 无 RLS） |
| `manager.go` | alfq gateway manager | ant gateway manager | `UserID` 字段；平台调度委托给 adapter 包 |
| `canonical.go` | alfq `symbolsync/canonical.go` | ant mdgateway | `.c` 后缀保留逻辑不变 |
| `normalizer.go` | alfq normalizer | ant normalizer | `CanonicalResolver` 接口 + `mapResolver` 实现 |
| `gateway_mt4.go` | alfq stub (未连线) | ant 委托给 `adapter/mt4` 包 | `mt4Gateway` 包装 `adapter/mt4.Gateway` |
| `gateway_mt5.go` | alfq stub (未连线) | ant 委托给 `adapter/mt5` 包 | `mt5Gateway` 包装 `adapter/mt5.Gateway` |
| `bar_aggregator.go` | alfq bar aggregator | ant bar aggregator | `UserID` 字段；`parseFloat` 增加纯数字校验 |
| `publisher.go` | alfq NATS publisher | ant no-op stub | NATS 暂未接入 |
| `clickhouse_conn.go` | alfq CH 连接 | ant CH 连接 | 无变更 |
| `clickhouse_writer.go` | alfq CH writer | ant CH writer | 无变更 |
| `spill_replay.go` | alfq spill replay | ant spill replay | 无变更 |
| `quality.go` | alfq QC 引擎 | ant QC 引擎 | Prometheus 指标可选 |

#### 新增测试文件（alfq 原无）

| 文件 | 覆盖 |
|------|------|
| `types_test.go` | `Money.GetValue()`, `Tick.GetBid()/GetAsk()`, `Bar` 字段 |
| `manager_test.go` | `NewEmptyManager`, `AddGateway/RemoveGateway`, `SetNormalizer`, config 字段 |
| `normalizer_test.go` | `Canonicalize` 全部 suffix 场景, `MapResolver`, `Normalizer.Tick` |
| `quality_test.go` | QC bid>ask drop, valid tick, outlier, medianSigma, Publisher no-op |
| `bar_aggregator_test.go` | 6 个 period 初始化, `FlushAll`, OHLC 完整性, `parseFloat` |

---

### 2. `backend/internal/mdgateway/adapter/mt4/`

| 文件 | 来源 | 说明 |
|------|------|------|
| `gateway.go` | **新建** | 桥接 `anttrader/internal/mt4client` 实现实时报价流。`Connect` → `mt4client.Connect`, `Subscribe` → `SubscribeQuoteStream` + `GetQuoteChannel`。pb `QuoteEventArgs` → 本地 `Tick`。`Conn()` 返回 nil（mt4client 未公开 grpc.Conn） |

---

### 3. `backend/internal/mdgateway/adapter/mt5/`

| 文件 | 来源 | 说明 |
|------|------|------|
| `gateway.go` | **新建** | 桥接 `anttrader/internal/mt5client`。`pb.Quote` → 本地 `Tick`。API 差异：MT5 使用 `GetID()` 而非 `GetToken()`，`Subscribe` 多一个 `interval` 参数 |

---

### 4. `backend/internal/mdgateway/backfill/`

| 文件 | 来源 | 说明 |
|------|------|------|
| `backfill.go` | **新建** | PG → ClickHouse 历史数据回填。`RunBars` 从 `md_bars_raw` → `md_bars`；`RunTicks` 从 `md_ticks_raw` → `md_ticks`。分批插入，默认 1000 行/批 |
| `backfill_test.go` | **新建** | `DefaultConfig`, 字段验证, `New` |

---

### 5. `backend/internal/mdgateway/chmigrate/`

| 文件 | alfq 原版 | ant 迁移版 | 变更 |
|------|-----------|-----------|------|
| `migrate.go` | alfq chmigrate | ant chmigrate | 无变更 |
| `001_md_ticks.sql` | `tenant_id` | `user_id` | `tenant_id` → `user_id` |
| `002_md_bars.sql` | 含 `tenant_id` | 移除 `tenant_id` | bars 表不需要 tenant 维度 |
| `003_factor_values.sql` | `tenant_id` | `user_id` | `tenant_id` → `user_id` |
| `004_signals.sql` | `tenant_id` | `user_id` | `tenant_id` → `user_id` |

---

### 6. `backend/internal/factorsvc/`

#### 来源对照

| 文件 | alfq 原版字段 | ant 迁移版字段 | 变更 |
|------|-------------|--------------|------|
| `bar.go` | `TenantID string` | `UserID string` | `tenant_id` → `user_id` |
| `engine.go` | `bar.TenantID` | `bar.UserID` | 所有引用更新 |
| `subscriber.go` | `FactorValue.TenantID` | `FactorValue.UserID` | 字段重命名 |
| `factor_ch_writer.go` | `factorRow.TenantID` / `Write(tenantID...)` | `factorRow.UserID` / `Write(userID...)` | 字段+参数重命名 |
| `window_buffer.go` | `Push(tenantID...)` / `Snapshot(tenantID...)` / `BootstrapSpec.TenantID` | `Push(userID...)` / `Snapshot(userID...)` / `BootstrapSpec.UserID` | 所有参数+字段重命名 |
| `metrics.go` | alfq prometheus metrics | ant stub metrics | 降级为计数器存根 |

#### 测试文件变更

| 文件 | 变更 |
|------|------|
| `engine_test.go` | `TenantID: "t1"` → `UserID: "t1"` (3 处) |
| `subscriber_test.go` | `TenantID: "t1"` → `UserID: "t1"` (1 处) |
| `window_buffer_test.go` | `TenantID` → `UserID` (6 处)；`BootstrapSpec.TenantID` → `BootstrapSpec.UserID` |

---

### 7. `backend/internal/factor/dsl/`

| 文件 | 来源 | 变更 |
|------|------|------|
| `ast.go` | alfq dsl AST | 无变更 |
| `lex.go` | alfq dsl lexer | 无变更 |
| `parser.go` | alfq dsl parser | 无变更 |
| `compile.go` | alfq dsl compiler | 无变更 |
| `scalar.go` | alfq scalar ops | 无变更 |
| `moving_average.go` | alfq MA ops | 无变更 |
| `oscillators.go` | alfq RSI/MACD/ATR | 无变更 |
| `ref_delta.go` | alfq Ref/Delta/PctChange/ZScore/Rank | 无变更 |
| `statistics.go` | alfq STD/VAR/Min/Max/Sum | 无变更 |
| `corr_cov.go` | alfq Corr/Cov | 无变更 |
| `bb_cross.go` | alfq BBUpper/BBLower | 无变更 |
| `validate.go` | alfq validation | 无变更 |
| `dsl_test.go` | alfq tests | 无变更 |
| `alignment_test.go` | alfq alignment (Go↔Python) | 输出路径更新为 `research/tests/alignment_data.json` |

---

### 8. `research/`

| 目录 | 说明 |
|------|------|
| `ant_research/factor/dsl/` | Python 端 DSL 引擎（ast, lexer, parser, compile, ops） |
| `ant_research/backtest/` | 回测框架桩 |
| `ant_research/cli/` | CLI 入口 |
| `ant_research/client/` | 客户端 |
| `ant_research/data/` | 数据层 |
| `ant_research/model/` | 模型 |
| `tests/` | 对齐测试数据 `alignment_data.json` + `test_alignment.py` + `test_factor_dsl.py` |

---

## 架构差异对照

| 关注点 | alfq (原) | ant (迁移后) |
|--------|----------|-------------|
| 租户标识 | `tenant_id` (proto) | `user_id` (本地字符串) |
| Tick 类型 | `pb.Tick` (protobuf) | `mdgateway.Tick` (Go struct) |
| MT 适配器 | gRPC proto streaming | `adapter/mt{4,5}/gateway.go` 桥接 `mt{4,5}client` |
| NATS | 全功能 NATS 发布 | no-op stub |
| 数据库 RLS | PostgreSQL RLS + `SetRole()` | 无 RLS，`SetRole()` no-op |
| Prometheus | 全量指标注册 | 可选 no-op 存根 |
| ClickHouse | 原生驱动 | 同，批次写入逻辑不变 |

---

## 待补齐

| 项目 | 优先级 | 说明 |
|------|--------|------|
| `mdgateway/adapter/mt4/*_test.go` | P1 | MT4 adapter 测试 |
| `mdgateway/adapter/mt5/*_test.go` | P1 | MT5 adapter 测试 |
| `mdgateway/chmigrate/*_test.go` | P2 | CH 迁移测试 |
| NATS publisher 真实现 | P2 | 需要 NATS 基础设施 |
| `mt4client` 公开 `Conn()/SessionID()` | P2 | 目前返回 nil/empty |
| `mt5client` 公开 `Conn()/SessionID()` | P2 | 目前返回 nil/empty |
