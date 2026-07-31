# 03 · 数据流时序

> **本文档展示 12 条核心数据流的时序细节。每个箭头都有对应的代码位置与可观测信号。**

## 1. 行情接入流（Tick → NATS/Redis/PG）

> ADR-0012: Tick 不落盘。最新报价缓存到 Redis，Bar 批量写入 PG md_bars。

### 1.1 时序图

```
mtapi.io  adapter/mt4  Manager  Normalizer  Quality  Dedup  BarAgg  Publisher  Redis  PgWriter  NATS
   │         │           │         │          │       │       │        │         │        │      │
   │ OnQuote │           │         │          │       │       │        │         │        │      │
   ├────────►│           │         │          │       │       │        │         │        │      │
   │         │ proto→Tick│         │          │       │       │        │         │        │      │
   │         ├──────────►│         │          │       │       │        │         │        │      │
   │         │           │ Resolve │          │       │       │        │         │        │      │
   │         │           ├────────►│          │       │       │        │         │        │      │
   │         │           │◄────────│          │       │       │        │         │        │      │
   │         │           │ Check              │       │       │        │         │        │      │
   │         │           ├───────────────────►│       │       │        │         │        │      │
   │         │           │◄───────────────────│       │       │        │         │        │      │
   │         │           │ Seen?                      │       │        │         │        │      │
   │         │           ├───────────────────────────►│       │        │         │        │      │
   │         │           │◄───────────────────────────│       │        │         │        │      │
   │         │           │ AddTick                            │        │         │        │      │
   │         │           ├───────────────────────────────────►│        │         │        │      │
   │         │           │ Bar?(可能)                         │        │         │        │      │
   │         │           │◄───────────────────────────────────│        │         │        │      │
   │         │           │ PublishTick/Bar                             │         │        │      │
   │         │           ├──────────────────────────────────────────────►│        │        │      │
   │         │           │              md.tick/md.bar → NATS            ├──────────────────────►│
   │         │           │ SETEX latest_quote:{canonical} 3600s         │        │        │      │
   │         │           ├──────────────────────────────────────────────►│        │        │      │
   │         │           │ EnqueueBar                                          │        │        │      │
   │         │           ├─────────────────────────────────────────────────────────►│        │      │
   │         │           │                                                    │ batch flush → PG md_bars │
   │         │           │                                                    │ ON CONFLICT upsert      │
```

### 1.2 关键决策点

| 决策点 | 文件:函数 | 失败行为 | 指标 |
|---|---|---|---|
| canonical 解析 | `mdgateway/normalizer.go:Resolve` | 走算法 fallback（去后缀） | `md_canonical_fallback_total` |
| Quality.Check | `mdgateway/quality.go:Check` | drop 并计数 | `md_tick_dropped_total{reason}` |
| TickDedup | `mdgateway/tick_dedup.go:Seen` | drop 但不计入 dropped（重发是已知正常）| `md_tick_dedup_total` |
| BarAggregator | `mdgateway/bar_aggregator.go:AddTick` | 用 ArrivedUnixMs 而非 broker TsUnixMs | `md_bar_flushed_total` |
| Publisher | `mdgateway/publisher.go:PublishTick` | 失败计数；不阻塞 | `md_publish_total{kind,status}` |
| Redis cache | `mdgateway/manager_tick.go:HandleTick` | SETEX 失败记 Debug 日志；不阻塞 | — |
| PgWriter | `mdgateway/pg_writer.go:EnqueueBar` | chan 满 → drop + 计数 | `md_chan_full_total` |
| CircuitBreaker | `mdgateway/circuit_breaker.go:OnFailure` | 半开/全开 → 新连接被拒 | `md_circuit_state{account,broker}` |

### 1.3 错误恢复路径

```
PG bar 写入失败
  → PgWriter.retryInsert 指数退避重试 3 次 (200ms/400ms/800ms)
  → 仍失败 → 记 Error 日志 + 丢弃该批次
  → NATS JetStream 提供 bar 持久化兜底（消费者可重放）

Redis SETEX 失败
  → 记 Debug 日志；不阻塞管线
  → 下次 tick 覆盖写入时自动恢复
  → TTL 3600s 过期后 GetLatestTick 返回 error（调用方降级处理）
```

## 2. 因子计算流（Bar → Factor）

### 2.1 时序图

```
NATS                  factorsvc.Subscriber  WindowBuffer  DSL.Eval      PG         Publisher  NATS
 │ md.bar.<>.<>.<>             │                 │            │           │            │         │
 ├───────────────────────────►│                  │            │           │            │         │
 │                            │ Append           │            │           │            │         │
 │                            ├─────────────────►│            │           │            │         │
 │                            │ buf updated      │            │           │            │         │
 │                            │◄─────────────────┤            │           │            │         │
 │                            │ Eval(bufView)               │            │           │            │
 │                            ├──────────────────────────────►│           │            │         │
 │                            │ value/NaN                    │           │            │         │
 │                            │◄──────────────────────────────┤           │            │         │
 │                            │ if !NaN: write factor_values             │            │         │
 │                            ├──────────────────────────────────────────►            │         │
 │                            │                                          │ INSERT      │         │
 │                            │ Publish md.factor.<canonical>.<name>     │            │         │
 │                            ├──────────────────────────────────────────────────────►│         │
 │                            │                                                       ├────────►│
```

### 2.2 关键设计

- **窗口大小**：每个 (canonical, period, factor) 维护 `max(factor.lookback) + 50` bars
- **FactorDef** 来自 PG `factor_definitions` 表（v2 新建，dynamic）
- **多周期**：同一 factor 可订阅多 period（如 `ma20` 同时算 `1m/5m/1h`）
- **NaN 处理**：DSL eval 返回 NaN 表示窗口未填满；不写 PG 不发 NATS

## 3. 信号 → 下单流（Factor → OMS → MT）

### 3.1 时序图

```
NATS md.factor   quantengine    risk.PreCheck   mthub.Exec   adapter/mt[45]   mtapi   PG.orders   NATS oms.events
 │ ────────────►│                   │              │             │             │          │             │
 │              │ ONNX/DSL infer    │              │             │             │          │             │
 │              │ Signal{...}       │              │             │             │          │             │
 │              ├──── PG signals (audit) ─────────────────────────────────────────────────►            │
 │              ├──────────────────►│              │             │             │          │             │
 │              │                   │ allow/deny   │             │             │          │             │
 │              │◄──────────────────┤              │             │             │          │             │
 │              │ Place                                          │             │          │             │
 │              ├─────────────────────────────────►│             │             │          │             │
 │              │                                  │ session?    │             │          │             │
 │              │                                  ├────────────►│             │          │             │
 │              │                                  │ session     │             │          │             │
 │              │                                  │◄────────────┤             │          │             │
 │              │                                  │ OrderSend   │             │          │             │
 │              │                                  ├──────────────────────────►│          │             │
 │              │                                  │ ticket=N    │             │          │             │
 │              │                                  │◄──────────────────────────┤          │             │
 │              │                                  │ INSERT order(state=submitted, broker_ticket=N)
 │              │                                  ├─────────────────────────────────────►│             │
 │              │                                  │ Publish oms.order.created                          │
 │              │                                  ├─────────────────────────────────────────────────────►
 │              │                                  │             │             │          │             │
 │              │                                  │ subscribe broker callbacks                          │
 │              │                                  │             │ OnOrderEvent│          │             │
 │              │                                  │             │◄────────────┤          │             │
 │              │                                  │             │ fan-in      │          │             │
 │              │                                  │ UPDATE order(state=filled, fill_price=...)         │
 │              │                                  ├─────────────────────────────────────►│             │
 │              │                                  │ Publish oms.order.filled                          │
 │              │                                  ├─────────────────────────────────────────────────────►
```

### 3.2 关键不变量

- **每个 Signal 必写 PG `signals`**（审计），无论是否最终下单
- **risk.PreCheck 失败 → 信号被丢弃 + audit 日志记录拒因**
- **broker_ticket 必须从 mtapi 返回，不许伪造**
- **state 转换** 走显式状态机（详见 `docs/spec/12-mthub.md`）

## 4. 用户查询流（前端 → PG）

### 4.1 时序图

```
前端                ConnectRPC handler  KlineService    PG查询    PG
 │ GetKline(req)        │                   │              │       │             │
 ├────────────────────►│                    │              │       │             │
 │                     │ validate           │              │       │             │
 │                     │ → service          │              │       │             │
 │                     ├───────────────────►│              │       │             │
 │                     │                    │ canonical    │       │             │
 │                     │                    │ resolve      │       │             │
 │                     │                    │ SELECT * FROM md_bars WHERE ...    │
 │                     │                    ├──────────────────────►             │
 │                     │                    │ rows         │       │             │
 │                     │                    │◄──────────────────────             │
 │                     │                    │ if 0 rows AND legacy_fallback:     │
 │                     │                    ├─────────────────────────────────────►
 │                     │                    │                                    │ SELECT FROM kline_data
 │                     │                    │◄─────────────────────────────────────
 │                     │                    │ → proto                            │
 │                     │ KlineList          │                                    │
 │                     │◄───────────────────┤                                    │
 │ KlineList           │                                                         │
 │◄────────────────────┤                                                         │
```

### 4.2 切流策略（M7-rewrite 期间）

```
config.LegacyFallback = false（ADR-0012 后 PG 为唯一存储，无 fallback）
  → PG 查询 0 行 → 返回 0 行
  → PG 查询 N 行 → 直接返回
```

## 5. 健康检查流

### 5.1 端点矩阵

| 端点 | 含义 | 失败条件 |
|---|---|---|
| `/healthz` | 进程活 | 进程僵死 |
| `/readyz` | 准备好接流量 | PG/Redis/NATS 任一不通 |
| `/metrics` | Prometheus 抓取 | — |

**已废弃**：原计划的 `/livez/account/{id}` per-account 端点 — k8s liveness 哲学只关心进程级；账户级健康是观测问题（Prometheus），非健康检查问题。改为 `mt_account_connected{account_id, broker}` Gauge，由 AlertManager 告警（决策见 BACKLOG RV-C4）。

### 5.2 readyz 检查项

```
GET /readyz
├─ check PG: SELECT 1
├─ check Redis: PING
├─ check NATS: connection.IsConnected()
└─ check accounts: 至少 50% accounts state ∈ {connected, degraded}
```

## 6. 启动序列

```
ant-server main()
  1. load config (env + yaml)
  2. setup logger (zap)
  3. connect storage (PG, Redis, NATS) — 任一失败 → fatal
  4. run migrations
     - PG migrate (golang-migrate)
  5. init mdgateway components (normalizer, quality, ...)
  6. mdgateway.RunGateway()  ← 从 PG 加载账户 → 启动 gateways
  7. factorsvc.Start()  ← 订阅 NATS md.bar.*
  8. quantengine.Start()  ← 加载 ONNX + DSL
  9. oms.Start()  ← 订阅 NATS md.factor.*
 10. start HTTP server (ConnectRPC + SSE) on :8080
 11. start /metrics, /healthz, /readyz
 12. graceful shutdown signals (SIGTERM)
     → drain ConnectRPC
     → flush mdgateway PgWriter
     → close gateways
     → close storage
```

## 7. 关闭/降级序列

任何关键依赖失败的降级路径：

| 失败 | 立即降级 | 持续 X 后 | 完全恢复 |
|---|---|---|---|
| PG 不可达 | PgWriter 重试 3 次后丢弃批次；NATS JetStream 兜底 | 持续失败 → 告警 | 自动恢复 |
| NATS 不可达 | Publisher 计数失败；不阻塞 | 30s 重连尝试 | 自动续 |
| 单 broker 不可达 | CircuitBreaker 打开；其他账户照常 | 30s 后半开尝试 | 连续 2 次成功 → 关 |
| Redis 不可达 | 缓存穿透到 PG；性能下降 | 60s 持续 → 告警 | 自动续 |

每条降级路径必须有：测试用例（chaos/）+ Prometheus 指标 + runbook 条目（`docs/runbook/mt-incidents.md`）。

## 8. 统一回测/实盘路径时序（ADR-0012）

```
PG (历史) / NATS (实时)    Source iface    factorsvc    quantengine    OMS          PaperExecutor / mthub
 │                              │               │             │            │                │
 │ bars                         │               │             │            │                │
 ├─────────────►                │               │             │            │                │
 │              OnBar           │               │             │            │                │
 │              ├──────────────►│               │             │            │                │
 │              │               │ DSL.Eval      │             │            │                │
 │              │               ├──────────────►│             │            │                │
 │              │               │ factor_value  │             │            │                │
 │              │               │◄──────────────┤             │            │                │
 │              │               │ Publish       │             │            │                │
 │              │               ├─────────────────────────────►            │                │
 │              │               │               │ ONNX/DSL     │            │                │
 │              │               │               ├────────────►│            │                │
 │              │               │               │ Signal{Source}           │                │
 │              │               │               │◄────────────┤            │                │
 │              │               │               │ PG signals (audit)       │                │
 │              │               │               ├─────────────────────────►│                │
 │              │               │               │             │ Signal     │                │
 │              │               │               ├─────────────────────────►│                │
 │              │               │               │             │            │                │
 │              │               │               │    Source=="live"?       │                │
 │              │               │               │    ──→ risk.PreCheck     │                │
 │              │               │               │         → mthub.Place    │                │
 │              │               │               │         → broker ack     │                │
 │              │               │               │    Source=="replay"?     │                │
 │              │               │               │    ──→ PaperExecutor     │                │
 │              │               │               │         → FillModel      │                │
 │              │               │               │         → PG paper_orders│                │
```

**接口契约**：

```go
type BarSource interface {
    Subscribe(ctx context.Context, canonical, period string) (<-chan *Bar, error)
}
type FactorSource interface {
    Subscribe(ctx context.Context, canonical string) (<-chan *FactorValue, error)
}
// LiveSource: NATS JetStream 订阅
// ReplaySource: PG SELECT 历史查询 + 时间推进
```

## 9. 信号→执行延迟预算时序（ADR-0018）

```
factorsvc      quantengine     OMS/risk       mthub         adapter/mt    mtapi/broker
 │                 │               │             │               │             │
 │ factor trigger   │               │             │               │             │
 │ (T0)             │               │             │               │             │
 ├────────────────►│               │             │               │             │
 │ < 5ms           │               │             │               │             │
 │                 │ ONNX/DSL      │             │               │             │
 │                 │ < 20ms P99    │             │               │             │
 │                 ├──────────────►│             │               │             │
 │                 │ Signal        │ PreCheck    │               │             │
 │                 │               │ < 5ms P99   │               │             │
 │                 │               ├────────────►│               │             │
 │                 │               │ Place       │               │             │
 │                 │               │             │ < 10ms P99    │             │
 │                 │               │             ├──────────────►│             │
 │                 │               │             │ OrderSend     │             │
 │                 │               │             │               │ < 200ms P99 │
 │                 │               │             │               ├────────────►│
 │                 │               │             │               │ broker ack  │
 │                 │               │             │               │◄────────────┤
 │                 │               │             │               │             │
 │                 │               │             │ T_end (broker_ack_ts)       │
 │                 │               │             │               │             │
 │ ◄───────────────┬───────────────┬─────────────┬───────────────┬─────────────┤
 │                 OTel trace 贯穿全链路，分阶段归因延迟                          │
 │                 signal_to_execution_latency_seconds.Observe(T_end - T0)      │
```

**分阶段预算**：

| 阶段 | P99 预算 | 测量指标 |
|------|----------|----------|
| factorsvc → NATS publish | < 5ms | — |
| quantengine.OnFactor → signal | < 20ms | `quant_inference_latency_seconds` |
| oms.SignalRouter → risk.PreCheck | < 5ms | `oms_risk_check_latency_seconds` |
| oms → mthub.Place | < 10ms | `mthub_place_latency_seconds` |
| mthub → adapter.OrderSend → broker ack | < 200ms | `mthub_broker_ack_latency_seconds` |
| **总计** | **< 235ms** | `signal_to_execution_latency_seconds` |

## 10. 订单状态机恢复时序（ADR-0013）

```
启动                    mthub.ReconciliationLoop    MT broker        PG orders       Redis
 │                           │                        │                │               │
 │ Start()                   │                        │                │               │
 ├──────────────────────────►│                        │                │               │
 │                           │ FetchOpenedOrders      │                │               │
 │                           ├───────────────────────►│                │               │
 │                           │ open orders            │                │               │
 │                           │◄───────────────────────┤                │               │
 │                           │ FetchOrderHistory(since=now-5min)       │               │
 │                           ├───────────────────────►│                │               │
 │                           │ recent history         │                │               │
 │                           │◄───────────────────────┤                │               │
 │                           │                        │                │               │
 │                           │ SELECT * FROM orders WHERE account_id=? │               │
 │                           ├────────────────────────────────────────►│               │
 │                           │ ant orders                              │               │
 │                           │◄────────────────────────────────────────┤               │
 │                           │                        │                │               │
 │                           │ 三方对账:                                │               │
 │                           │ ─ ant 有, MT 无 → ghost → CANCEL        │               │
 │                           │ ─ MT 有, ant 无 → orphan → INSERT       │               │
 │                           │ ─ 状态不一致 → MT 为准 → UPDATE          │               │
 │                           │                        │                │               │
 │                           │ 写入 idempotency keys                    │               │
 │                           ├────────────────────────────────────────────────────────►│
 │                           │                        │                │   SET (account_id, client_id) 24h TTL
 │                           │                        │                │               │
 │                           │ === 30s timer ===      │                │               │
 │                           │ Poll()                 │                │               │
 │                           ├───────────────────────►│                │               │
 │                           │ 对比 ant_state vs mt_state              │               │
 │                           │ 偏差 → metric + auto-fix                │               │
```

## 11. Bar 修订级联时序（ADR-0016）

```
BarAggregator     NATS              factorsvc       quantengine      OMS            PG signals
 │                 │                    │                │              │                │
 │ 检测 version>1  │                    │                │              │                │
 ├────────────────►│                    │                │              │                │
 │ Publish          │                    │                │              │                │
 │ bar.revision     │                    │                │              │                │
 │                  │ OnBarRevision      │                │              │                │
 │                  ├───────────────────►│                │              │                │
 │                  │                    │ 窗口重置       │              │                │
 │                  │                    │ DSL.Eval       │              │                │
 │                  │                    │ (新 bar 窗口)  │              │                │
 │                  │                    │ PG INSERT       │              │                │
 │                  │                    │ factor_values   │              │                │
 │                  │                    │ (version=bar.version)          │                │
 │                  │                    │                │              │                │
 │                  │                    │ Publish        │              │                │
 │                  │                    │ md.factor.*    │              │                │
 │                  │                    ├───────────────►│              │                │
 │                  │                    │                │ 信号重算     │                │
 │                  │                    │                ├─────────────►│                │
 │                  │                    │                │              │ SELECT state  │
 │                  │                    │                │              │◄───────────────┤
 │                  │                    │                │              │                │
 │                  │                    │                │ state==pending?               │
 │                  │                    │                │ ──→ UPDATE signal             │
 │                  │                    │                │ state∈{submitted,filled}?      │
 │                  │                    │                │ ──→ PG bar_revision_log        │
 │                  │                    │                │     (记录偏差，不修改订单)      │
```

## 12. 仿真交易时序（ADR-0015）

```
oms.SignalRouter    PaperExecutor    FillModel      PG paper_orders    PG paper_positions
 │ (Source=="replay")    │               │                │                   │
 ├──────────────────────►│               │                │                   │
 │ Place(order)          │               │                │                   │
 │                       │ SimulateFill  │                │                   │
 │                       ├──────────────►│                │                   │
 │                       │               │ slippage calc  │                   │
 │                       │               │ latency sim    │                   │
 │                       │               │ partial fill?  │                   │
 │                       │ FillResult    │                │                   │
 │                       │◄──────────────┤                │                   │
 │                       │               │                │                   │
 │                       │ INSERT (state=filled, fill_price, slippage_bps)    │
 │                       ├──────────────────────────────►│                   │
 │                       │                               │                   │
 │                       │ UPSERT paper_positions         │                   │
 │                       ├───────────────────────────────────────────────────►│
 │                       │                               │                   │
 │ FillResult            │                               │                   │
 │◄──────────────────────┤                               │                   │
```

**升级到实盘**：

```
用户触发            PaperExecutor      risk.PreCheck    mthub           PG
 │                      │                   │              │              │
 │ PromoteToLive(id)    │                   │              │              │
 ├─────────────────────►│                   │              │              │
 │                      │ Validate:         │              │              │
 │                      │ Sharpe > 0?       │              │              │
 │                      │ drawdown < limit? │              │              │
 │                      │                   │              │              │
 │                      │ Clone strategy:   │              │              │
 │                      │ paper → user_strategies         │              │
 │                      ├──────────────────────────────────────────────►│
 │                      │                   │              │              │
 │                      │ Switch Source:    │              │              │
 │                      │ Replay → Live     │              │              │
 │                      │ Switch Executor:  │              │              │
 │                      │ Paper → Live(mthub)              │              │
 │                      │                   │              │              │
 │                      │ Mark promoted_at=NOW()           │              │
 │                      ├──────────────────────────────────────────────►│
 │                      │                   │              │              │
 │ OK                   │                   │              │              │
 │◄─────────────────────┤                   │              │              │
```

## 13. 信号 SLO 告警流

```
Prometheus                    Alertmanager         SRE
 │                               │                  │
 │ 每 5min 评估:                  │                  │
 │ signal_to_execution_latency   │                  │
 │ P99 > 500ms?                  │                  │
 ├──────────────────────────────►│                  │
 │                               │ SignalExecution  │
 │                               │ LatencyHigh      │
 │                               ├─────────────────►│
 │                               │ (page)           │
 │                               │                  │
 │ Error budget burn rate:       │                  │
 │ (1 - P99_within_SLO)          │                  │
 │  > 14.4 × (1 - 0.99)         │                  │
 │  for 1h                       │                  │
 ├──────────────────────────────►│                  │
 │                               │ ErrorBudgetBurn  │
 │                               │ Signal           │
 │                               ├─────────────────►│
 │                               │ (page)           │
```
