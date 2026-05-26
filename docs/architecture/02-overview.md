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
│  (DSL eval + ONNX)     │               │  ├─ SessionRegistry │
│  ├─ FactorSource iface │               │  ├─ OrderExecutor   │
│  └─ SignalRouter       │               │  ├─ OrderStateMachine│
└──────────┬─────────────┘               │  ├─ IdempotencyGuard │
           │                             │  └─ Reconciliation  │
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
- OrderStateMachine（ant 侧 10-state 订单状态机，镜像 MT5 OrderState）
- IdempotencyGuard（(account_id, client_id) 幂等去重，Redis 24h TTL）
- ReconciliationLoop（启动对账 + 每 30s 主动 polling 对账）
- 详见 `docs/spec/12-mthub.md`、`docs/spec/22-order-state-machine.md`

### L6 · 业务编排（`ai/` `marketplace/` `risk/` `oms/` `paper/`）
- `ai/` — AI 策略生成（自然语言→策略代码），见 `docs/spec/26-ai-strategy-generation.md`
- `risk/` — 风控引擎：PreCheck（同步 4 项检查）+ Monitor（异步 margin call / stop-out），见 `docs/spec/23-risk-management.md`
- `oms/` — SignalRouter + OrderTracker，见 §"OMS"订正
- `paper/` — 仿真交易：PaperExecutor（滑点/延迟/部分成交模型）+ 数据隔离，见 `docs/spec/24-paper-trading.md`
- `marketplace/` — 策略市场基础设施（M11+ 激活变现），当前仅保留数据模型
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

### 3.5 统一回测/实盘路径（ADR-0012）

```
回测模式:
  CH md_bars (历史)
    → ReplaySource (实现 BarSource/FactorSource interface)
    → factorsvc.Subscriber.OnBar
    → DSL.Eval
    → quantengine.OnFactor
    → Signal{Symbol, Side, TargetQty, Source="replay"}
    → CH signals (审计)
    → oms.SignalRouter
        └─ signal.Source == "replay" → paper.PaperExecutor.Place
            ├─ FillModel.SimulateFill
            └─ PG paper_orders

实盘模式:
  NATS md.bar.* (实时)
    → LiveSource (实现 BarSource/FactorSource interface)
    → factorsvc.Subscriber.OnBar
    → DSL.Eval
    → quantengine.OnFactor
    → Signal{Symbol, Side, TargetQty, Source="live"}
    → CH signals (审计)
    → oms.SignalRouter
        ├─ risk.PreCheck (4 项同步阻断)
        └─ mthub.OrderExecutor.Place
            → adapter/mt[45]/executor.go → mtapi.OrderSend
            → PG orders
```

**关键约束**：factorsvc 和 quantengine 只依赖 `Source` interface，业务逻辑零 `if isBacktest` 分支。回测和实盘走完全相同的 factor → signal → execution 代码路径。

### 3.6 Bar 修订级联（ADR-0016）

```
BarAggregator 检测修订 (md_bars.version > 1)
  → NATS bar.revision.<broker>.<canonical>.<period>
  → factorsvc.OnBarRevision
      ├─ 窗口重置 (丢弃旧 bar 版本，用新 bar 重新计算)
      ├─ DSL.Eval (使用新窗口)
      ├─ factor_value.version = bar.version  (同步版本号)
      └─ CH factor_values (INSERT 新版本，非 UPDATE)
  → quantengine.OnFactorRevision
      ├─ 信号重算
      ├─ 信号未执行 (PG signals.state == "pending")
      │     → 更新信号 (PG signals UPDATE target_qty, trigger_price)
      └─ 信号已执行 (PG signals.state ∈ {submitted, filled, partial})
            → 记录偏差 (CH bar_revision_log)
            → 不修改订单 (订单不可变原则)
```

### 3.7 订单状态机恢复流程（ADR-0013）

```
进程启动
  → mthub.ReconciliationLoop.Start()
      ├─ 遍历所有 active sessions (PG mt_accounts WHERE is_active=true)
      ├─ 对每个 account:
      │     ├─ FetchOpenedOrders(account_id)
      │     │     → 获取 MT broker 当前挂单 + 持仓
      │     ├─ FetchOrderHistory(account_id, since=now-5min)
      │     │     → 获取最近交易历史
      │     └─ 三方对账 (ant PG ↔ MT broker ↔ Redis idempotency keys)
      │           ├─ ant 有、MT 无 → ghost order → 标记 CANCELLED + alert
      │           ├─ MT 有、ant 无 → orphan → 补录入 PG orders
      │           └─ 状态不一致 → MT 为准，ant 更新 + metric
      └─ 启动 30s 定时对账 (ReconciliationLoop.Poll)

每 30s:
  → ReconciliationLoop.Poll()
      ├─ 对比 ant_state vs mt_state (per active account)
      ├─ 偏差 → Prometheus mt_reconciliation_mismatch_total +1
      └─ 自动修复 (MT broker 状态为权威来源)
```

### 3.8 仿真交易执行流（ADR-0015）

```
oms.SignalRouter (signal.Source == "replay")
  → paper.PaperExecutor.Place(order)
      ├─ FillModel.SimulateFill(order)
      │     ├─ 滑点计算 (configurable: fixed_pct / normal / skip)
      │     ├─ 延迟模拟 (configurable: fixed_ms / normal / none)
      │     └─ 部分成交判断 (configurable: full_only / uniform / pareto)
      ├─ PG paper_orders (INSERT, 与 orders 表物理隔离)
      └─ 返回 FillResult{filled_volume, fill_price, slippage_bps}

策略升级到实盘:
  → paper.PromoteToLive(paper_strategy_id, user_id)
      ├─ 验证回测指标 (Sharpe > 0, max_drawdown < risk_limit)
      ├─ 克隆策略: paper_strategies → user_strategies
      ├─ 切换数据源: ReplaySource → LiveSource
      ├─ 切换执行器: PaperExecutor → LiveExecutor (mthub)
      └─ 标记 paper_strategy.promoted_at = NOW()
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
| 数据归属 | tenant_id（B2B 多租户）| **平台共享 + 用户私有两段**（C2C 散户平台）| ADR-0006：marketplace/跟单成立的前提 |
| 熔断 | 无 | CircuitBreaker | broker 故障常态 |
| Spill 旋转 | 单文件 | 按大小/时间旋转 | 长时间故障保护 |
| Tick dedup | 无 | 100 条窗口 | broker 重发问题验证存在 |
| 业务功能 | 仅骨架 | AI / marketplace / admin / worker | ant 是产品 |
| Quality dropped reason | 单 metric | 三类 label | SRE 排障粒度 |

## 8. 关键不变量（invariants）

> 这些不变量是 v2 设计的"宪法"。任何 PR 不得违反。

1. **canonical 在 L3 入口完成**：adapter（L2）产出的 Tick `Canonical` 字段为空字符串；mdgateway.Manager.HandleTick 第一步调用 `Normalizer.Resolve(broker, symbol_raw)` 填充。理由：normalizer 依赖 PG `broker_symbols` + LRU cache，不应注入 adapter（破坏"纯翻译"职责）。**进入 L3 之后**的所有处理（quality/dedup/aggregator/publisher/chwriter）一定看到非空 `Canonical`。
2. **生产路径零 Python**：从 NATS factor 到 OMS 下单，全链路 Go
3. **价格类型**：PG `NUMERIC(20,8)` ↔ Go `decimal.Decimal` ↔ CH `Decimal(18,6)`，禁 float
4. **时间统一**：UTC，毫秒精度（`int64`）
5. **每个 Tick 必经过 Quality.Check**：drop 也要计数
6. **CH 写入不阻塞订阅**：CHWriter chan 满 → SpillWriter
7. **CircuitBreaker 不影响其他账户**：单账户故障不传播
8. **业务代码 0 处直调 mt4client/mt5client**（v2 完成后）
9. **业务代码 0 处直读 PG 行情表**（v2 完成后）
10. **每个错误必有 errs.Code + 中文 user_message**
11. **业务数据二分法**（ADR-0006）：
    - **平台共享**（无 `user_id`、所有用户可读）：`platform_strategies`、`platform_factors`、`platform_ai_agents`、`broker_symbols`、`admins`
    - **用户私有**（必须 `user_id` 外键 + RLS）：`mt_accounts`、`user_strategies`、`user_factor_overrides`、`user_ai_agents`、`orders`、`positions`、`trades`、`user_subscriptions`、`copy_trade_links`
    - **禁止**：在 user 表中复制官方/平台数据（如旧 `seed_default_templates.go` per-user 复制）
12. **admin 鉴权独立**（ADR-0006）：废弃 `users.role='admin'`；平台运营走 `admins` 表 + JWT scope `platform:admin`
13. **PlatformScope 接口预留**（ADR-0006）：所有读 `platform_*` 表的查询必须经 `scope.Current(ctx)`（当前 no-op 返回 `'ant'`），为 M10+ 多 tenant 白标输出预留路径
14. **回测与实盘同一代码路径**（ADR-0012）：factorsvc 和 quantengine 只依赖 `Source` interface（`LiveSource` / `ReplaySource`），不得在业务逻辑中判断 `if isBacktest`。违反此条 → 回测结果不可信。
15. **订单不可变**（ADR-0013）：一旦 `ant_state` 进入 `FILLED`/`PARTIALLY_FILLED`，订单参数（volume/price/sl/tp）不可修改。只能通过新建反向订单来改变持仓。
16. **Bar 修订不修改历史行**（ADR-0016）：`md_bars` 使用 INSERT 新版本（`version` 递增），禁止 UPDATE 历史 bar 行。因子值同步携带 bar 版本号。
17. **仿真与实盘数据隔离**（ADR-0015）：`paper_*` 表簇与实盘表（`orders`/`positions`/`trades`）物理隔离，仿真订单不得触发真实 broker 调用。
18. **风控阻断优先于下单**（ADR-0014）：所有订单（包括 AI 生成的策略订单）必须先通过 `risk.PreCheck` 4 项同步检查，不得绕过。风控拒因必须写入审计日志。
19. **AI 策略代码合规必扫**（ADR-0017）：AI 生成的 DSL 代码必须通过 13 条合规规则扫描，未通过不得进入回测。扫描结果写入 metric `ai_strategy_generation_total{result="compliance_blocked"}`。
20. **信号→执行全链路可追踪**（ADR-0018）：每个信号必须有 OTel trace，从 `factor_ts_unix_ms` 到 `broker_ack_ts`，分阶段归因延迟。端到端 SLO P99 < 235ms。

CI 应有 lint 规则强制 1, 2, 3, 8, 9, 11, 12（其他靠代码 review）。

## 9. M11+ 推迟功能

以下功能已在架构中预留接口或数据模型，但**不在 M10 交付范围**内。推迟策略：先让核心量化系统稳定运行 ≥ 30 天，再启动变现层。

| 功能 | 推迟理由 | 预留方式 |
|------|----------|----------|
| 策略市场变现（支付/订阅/分润/评分）| 核心量化系统先稳定 | `platform_strategies` 表、`user_subscriptions` 表已定义 |
| 跟单交易自动化 | 依赖订单执行完全稳定 | `copy_trade_links` 表、OMS SignalRouter `Subscription` 接口预留 |
| 多租户白标（C 模型）| ant 当前单实例 | `PlatformScope` interface（当前 no-op 返回 `'ant'`），见 ADR-0006 |
| Python 沙箱生产路径 | DSL+ONNX 覆盖 90% 策略需求 | `strategysvc` 仅研究模式运行；生产路径强制 Go |
| 策略回测排行榜 | 依赖大量真实回测数据积累 | `BacktestMetrics` 结构体已定义；`paper_strategies` 表预留 `ranking_score` 列 |
| 社交/社区功能 | 非量化核心 | 未预留 |

**触发条件**：M10 验收通过（7 天稳定性测试）+ 生产环境连续 30 天零 P0 事故 → 启动 M11 变现层规划。
