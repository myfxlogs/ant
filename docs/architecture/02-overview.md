# 02 · 架构总览

> **本文档是 ant 后端 v2 的权威架构参考。所有 spec 文档必须与本图保持一致。**

## 1. 七层架构（自下而上）

```
┌─────────────────────────────────────────────────────────────┐
│  L7  RPC 边界（ConnectRPC + SSE）                           │
│      proto/ant/v1/  +  internal/connect/                    │
└──────────────┬──────────────────────────────────────────────┘
               │
┌──────────────┴──────────────────────────────────────────────┐
│  L6  应用编排（business orchestration）                      │
│      internal/{ai,marketplace,risk,oms,strategysvc}         │
└──────────┬───────────────────────────────────────┬──────────┘
           │                                       │
┌──────────┴────────────┐               ┌──────────┴──────────┐
│  L5  量化引擎          │               │  L5  会话与下单     │
│  internal/quantengine  │               │  internal/mthub     │
│  (DSL eval + ONNX)     │               │  (session+events)   │
└──────────┬────────────┘               └──────────┬──────────┘
           │                                       │
┌──────────┴───────────────────────────────────────┴──────────┐
│  L4  因子计算                                                │
│  internal/factorsvc  (NATS sub → DSL eval → CH write)       │
└──────────────┬───────────────────────────────────────┬──────┘
               │                                       │
┌──────────────┴──────────┐                ┌───────────┴──────┐
│  L3  行情网关            │                │  L3  市场撮合     │
│  internal/mdgateway      │                │  internal/oms     │
│  (normalizer/quality/    │                │  (broker/risk hook│
│   aggregator/publisher/  │                │   /executor)      │
│   chwriter/spill/circuit)│                │                   │
└──────────────┬──────────┘                └───────────────────┘
               │
┌──────────────┴──────────────────────────────────────────────┐
│  L2  MT 适配                                                 │
│  internal/mdgateway/adapter/{mt4,mt5}/                      │
│  (mtapi gRPC → Tick/Bar DTO)                                │
└──────────────┬──────────────────────────────────────────────┘
               │
┌──────────────┴──────────────────────────────────────────────┐
│  L1  外部接口                                                │
│  mtapi.io gRPC (mt4/mt5) | 用户浏览器 | broker 服务器        │
└─────────────────────────────────────────────────────────────┘

存储层（横切）：
  PostgreSQL 18  → 业务数据（user/account_binding/order/risk/ai/market）
  ClickHouse 24  → 时序数据（md_ticks/md_bars/factor_values/signals）
  Redis 8        → 缓存/分布式锁/SSE pub-sub
  NATS JetStream → 行情/因子/信号 fan-out
```

## 2. 各层职责（精确 + 不重叠）

### L1 · 外部接口
- **mtapi.io gRPC**：MT4/MT5 协议封装，第三方维护
- ant **不直接**写 mtapi 协议适配，依赖 mtapi.io 提供的 proto

### L2 · MT 适配（`adapter/`）
- **唯一**翻译层：mtapi proto → ant 内部 DTO（`Tick` / `Bar` / `OrderInfo`）
- mt4 与 mt5 各 ~80 行
- **不做**：质检、规范化、聚合、存储（这些是 L3 的事）
- 详见 `docs/spec/10-mt-adapter.md`

### L3 · 行情网关（`mdgateway/`）
- 职责子组件（每个 ≤ 250 行）：
  | 子组件 | 职责 |
  |---|---|
  | `manager.go` | Gateway 注册 + 生命周期 + CircuitBreaker 注入 |
  | `normalizer.go` | (broker, symbol_raw) → canonical |
  | `quality.go` | bid>ask 丢弃 / 5σ 离群 / gap / clock skew |
  | `bar_aggregator.go` | tick → 6 周期 OHLCV |
  | `publisher.go` | NATS JetStream 发布 md.tick.* / md.bar.* |
  | `clickhouse_writer.go` | 异步 batch 写 CH md_ticks/md_bars |
  | `spill_writer.go` | CH 不通时落本地 jsonl，带旋转 |
  | `spill_replay.go` | 启动时回放 spill |
  | `circuit_breaker.go` | broker 故障熔断（滑动窗口） |
  | `tick_dedup.go` | 100 条窗口去重（broker 重发保护） |
  | `runner.go` | 装配入口（从 PG 加载账户 → 启动全链路） |
- 详见 `docs/spec/11-mdgateway.md`

### L6 · OMS（`oms/`）— 注意：本节订正

> **订正**：早期草图 §1 把 oms 画在 L3。**实际归属 L6**。理由：oms 调用 `mthub.OrderExecutor`（L5），按 §4 "禁止反向依赖" 不变量，依赖上层服务的模块必须在更高层。
> §1 ASCII 图保留历史画法（不易改图），以本节为准。

- **订阅者角色**：订阅 NATS `md.factor.*` / `oms.events.*` 事件，提单 / 跟单
- 调 `mthub.OrderExecutor`（L5）下单、写 PG.orders、调 `risk.PreCheck`（同 L6 同级）
- **不依赖**直接 MT 调用（统一经 mthub）

### L4 · 因子计算（`factorsvc/`）
- 订阅 NATS `md.bar.*` → DSL 引擎求值 → 写 CH `factor_values`
- 滚动窗口缓冲（per symbol+period+factor）
- 详见 `docs/spec/11-mdgateway.md` §"factorsvc 集成"

### L5 · 量化引擎（`quantengine/`）
- 加载 ONNX 模型 + DSL 表达式
- 订阅 CH `factor_values` 流（或 NATS `md.factor.*`）→ 推理 → 输出 signal
- **生产路径零 Python 执行**

### L5 · 会话/下单中心（`mthub/`）
- MT 会话注册中心（accountID → session）
- OrderEventBroker（fan-in MT 事件 → 业务订阅）
- OrderExecutor（统一下单接口）
- 详见 `docs/spec/12-mthub.md`

### L6 · 业务编排（`ai/` `marketplace/` `risk/` `oms/`）
- handler 只编排；业务规则在 service
- 与 L5 通过 interface 解耦

### L7 · RPC 边界（`connect/` + `proto/ant/v1/`）
- ConnectRPC + SSE，前端唯一交互层
- handler 只做：参数校验 → 调 service → 转 proto → 返回

## 3. 数据流向（核心 4 条）

### 3.1 行情数据流（最关键）

```
mtapi.OnQuote
  → adapter/mt[45]/gateway.go (proto → mdtick.Tick)
  → mdgateway.Manager.handleTick(tick)
      ├─ Normalizer.Resolve     (broker, raw) → canonical
      ├─ Quality.Check          drop or pass
      ├─ TickDedup.Seen         skip if duplicate within 100-window
      ├─ BarAggregator.AddTick  emit Bar on period boundary
      ├─ Publisher.PublishTick  → NATS md.tick.<broker>.<canonical>
      ├─ Publisher.PublishBar   → NATS md.bar.<broker>.<canonical>.<period>
      └─ CHWriter.Enqueue       → batch flush 1s/1000 → CH md_ticks
                                                      → CH md_bars
```

### 3.2 因子计算流

```
NATS md.bar.<broker>.<canonical>.<period>
  → factorsvc.Subscriber.OnBar
  → WindowBuffer.Append (per canonical+period)
  → DSL.Eval  (使用 buffer 窗口)
  → factorCHWriter → CH factor_values
  → Publisher → NATS md.factor.<canonical>.<factor_name>
```

### 3.3 信号 / 下单流

```
NATS md.factor.* (or CH factor_values poll)
  → quantengine.OnFactor
      ├─ ONNX 推理 (if model bound)
      └─ DSL 信号规则评估
  → Signal{symbol, side, target_qty}
  → CH signals (审计)
  → oms.SignalRouter
      ├─ risk.PreCheck
      └─ mthub.OrderExecutor.Place
          → adapter/mt[45]/executor.go
          → mtapi.OrderSend
  → MT broker 接受
  → mtapi.OrderEvent (callback)
  → mthub.OrderEventBroker fan-in
  → oms.OrderTracker → PG orders
```

### 3.4 用户查询流（只读）

```
前端 ConnectRPC call
  → connect/market_service.go
  → service/kline_query.go (新)
  → CH md_bars (主路径)
     fallback → PG kline_data (兼容期；M9 删除)
  → return proto
```

## 4. 包依赖图（强制无环）

```
connect/  ────► service/* ────► mthub/ ────► mdgateway/* ────► adapter/mt[45]/
   │                │                │             │                  │
   │                ▼                ▼             ▼                  ▼
   │          quantengine ─► factorsvc           oms/             mtapi gRPC
   │                │           │                │
   │                ▼           ▼                ▼
   ▼              factor/dsl  storage/{ch,pg}  storage/pg
common/{errs,logger,types,decimal,trace}  ◄── 所有包共享
```

**禁止反向依赖**：
- adapter 不许 import mdgateway（adapter 在 L2，mdgateway 在 L3）
- mdgateway 不许 import service（service 在 L6）
- 跨层 import 必须经 interface

## 5. 进程拓扑

v2 仅有 **1 个 Go 二进制**（`backend/cmd/ant-server`），内部以 goroutine 隔离运行：

```
ant-server (单进程)
  ├─ HTTP server (ConnectRPC + SSE)         port 8080
  ├─ mdgateway runner (per account)          (goroutine pool)
  ├─ factorsvc subscriber                    (goroutine pool)
  ├─ quantengine subscriber                  (goroutine pool)
  ├─ oms event consumer                      (goroutine pool)
  └─ background workers (cron)               (goroutine)
```

不拆服务的理由：
- 单机 docker-compose 部署
- 进程内通信延迟 < 1µs vs NATS ~100µs（行情敏感）
- 运维心智成本低
- alfq 拆为 4 个服务（mdgateway / mthub / quantengine / strategysvc）是为多机扩展，ant 当前不需要

外部辅助进程：
- `ant-strategy-service`（Python，仅研究模式沙箱）
- `ant-frontend`（Nginx 静态托管）

## 6. 端口/网络规范

| 容器 | 容器内端口 | 宿主端口 | 用途 |
|---|---|---|---|
| ant-frontend | 8080 | `${ANT_FRONTEND_PORT:-8022}` | Nginx，唯一对外端口 |
| ant-backend | 8080 | — | ConnectRPC + SSE |
| ant-strategy-service | 8081 | — | Python 沙箱 |
| ant-postgres | 5432 | — | PG |
| ant-redis | 6379 | — | Redis |
| ant-clickhouse | 9000 / 8123 | — | CH native + HTTP |
| ant-nats | 4222 | — | NATS |

所有内部通信走 `ant-network` Docker bridge。**只有 frontend 暴露宿主端口**。

## 7. 与 alfq 的差异（必须明白）

ant v2 **不是 alfq 的克隆**。差异：

| 维度 | alfq | ant v2 | 理由 |
|---|---|---|---|
| 进程数 | 4 | 1 | 单机部署、降低心智成本 |
| 多租户 | tenant_id | user_id（单租户） | ant 是用户中心架构（ADR 0005） |
| 熔断 | 无 | CircuitBreaker | broker 故障常态 |
| Spill 旋转 | 单文件 | 按大小/时间旋转 | 长时间故障保护 |
| Tick dedup | 无 | 100 条窗口 | broker 重发问题验证存在 |
| 业务功能 | 仅骨架 | AI / marketplace / admin / worker | ant 是产品 |
| Quality dropped reason | 单 metric | 三类 label | SRE 排障粒度 |

## 8. 关键不变量（invariants）

> 这些不变量是 v2 设计的"宪法"。任何 PR 不得违反。

1. **canonical 在 L2 出口完成**：进入 L3 的 Tick 一定有非空 `canonical` 字段
2. **生产路径零 Python**：从 NATS factor 到 OMS 下单，全链路 Go
3. **价格类型**：PG `NUMERIC(20,8)` ↔ Go `decimal.Decimal` ↔ CH `Decimal(18,6)`，禁 float
4. **时间统一**：UTC，毫秒精度（`int64`）
5. **每个 Tick 必经过 Quality.Check**：drop 也要计数
6. **CH 写入不阻塞订阅**：CHWriter chan 满 → SpillWriter
7. **CircuitBreaker 不影响其他账户**：单账户故障不传播
8. **业务代码 0 处直调 mt4client/mt5client**（v2 完成后）
9. **业务代码 0 处直读 PG 行情表**（v2 完成后）
10. **每个错误必有 errs.Code + 中文 user_message**

CI 应有 lint 规则强制 1, 2, 3, 8, 9（其他靠代码 review）。
