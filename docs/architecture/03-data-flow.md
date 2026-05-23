# 03 · 数据流时序

> **本文档展示 4 条核心数据流的时序细节。每个箭头都有对应的代码位置与可观测信号。**

## 1. 行情接入流（Tick → CH/NATS）

### 1.1 时序图

```
mtapi.io     adapter/mt4    Manager       Normalizer  Quality   Dedup    BarAgg   Publisher  CHWriter  SpillW   ClickHouse  NATS
   │            │              │              │          │        │        │         │          │         │           │       │
   │ OnQuote    │              │              │          │        │        │         │          │         │           │       │
   ├───────────►│              │              │          │        │        │         │          │         │           │       │
   │            │ proto→Tick   │              │          │        │        │         │          │         │           │       │
   │            ├─────────────►│              │          │        │        │         │          │         │           │       │
   │            │              │ Resolve      │          │        │        │         │          │         │           │       │
   │            │              ├─────────────►│          │        │        │         │          │         │           │       │
   │            │              │ canonical    │          │        │        │         │          │         │           │       │
   │            │              │◄─────────────┤          │        │        │         │          │         │           │       │
   │            │              │ Check        │          │        │        │         │          │         │           │       │
   │            │              ├─────────────────────────►        │        │         │          │         │           │       │
   │            │              │ pass/drop    │          │        │        │         │          │         │           │       │
   │            │              │◄─────────────────────────        │        │         │          │         │           │       │
   │            │              │ Seen?        │          │        │        │         │          │         │           │       │
   │            │              ├──────────────────────────────────►        │         │          │         │           │       │
   │            │              │ false→continue          │        │        │         │          │         │           │       │
   │            │              │◄──────────────────────────────────        │         │          │         │           │       │
   │            │              │ AddTick      │          │        │        │         │          │         │           │       │
   │            │              ├───────────────────────────────────────────►         │          │         │           │       │
   │            │              │ Bar?(可能)    │          │        │        │         │          │         │           │       │
   │            │              │◄───────────────────────────────────────────         │          │         │           │       │
   │            │              │ PublishTick                                          │          │         │           │       │
   │            │              ├──────────────────────────────────────────────────────►          │         │           │       │
   │            │              │                                                      │          │ md.tick.<broker>.<canonical>
   │            │              │                                                      ├─────────────────────────────────►       │
   │            │              │ EnqueueTick                                                     │         │           │       │
   │            │              ├─────────────────────────────────────────────────────────────────►         │           │       │
   │            │              │                                                                            │ batch flush 1s/1000
   │            │              │                                                                            ├──────────►│       │
   │            │              │                                                                            │ INSERT INTO md_ticks
   │            │              │                                                                            │           │       │
   │            │              │                                                                            │ on error → SpillW
   │            │              │                                                                            ├───────────────────►│ jsonl rotate
```

### 1.2 关键决策点

| 决策点 | 文件:函数 | 失败行为 | 指标 |
|---|---|---|---|
| canonical 解析 | `mdgateway/normalizer.go:Resolve` | 走算法 fallback（去后缀） | `md_canonical_fallback_total` |
| Quality.Check | `mdgateway/quality.go:Check` | drop 并计数 | `md_tick_dropped_total{reason}` |
| TickDedup | `mdgateway/tick_dedup.go:Seen` | drop 但不计入 dropped（重发是已知正常）| `md_tick_dedup_total` |
| BarAggregator | `mdgateway/bar_aggregator.go:AddTick` | 用 ArrivedUnixMs 而非 broker TsUnixMs | `md_bar_flushed_total` |
| Publisher | `mdgateway/publisher.go:PublishTick` | 失败计数；不阻塞 | `md_publish_total{kind,status}` |
| CHWriter | `mdgateway/clickhouse_writer.go:Enqueue` | chan 满 → SpillWriter | `md_ch_write_errors_total`, `md_spill_writes_total` |
| CircuitBreaker | `mdgateway/circuit_breaker.go:OnFailure` | 半开/全开 → 新连接被拒 | `md_circuit_state{account,broker}` |

### 1.3 错误恢复路径

```
CH 写入失败
  → CHWriter 计数 md_ch_write_errors_total
  → CircuitBreaker.OnFailure
  → 失败次数 ≥ 5 → 熔断打开 30s
  → 熔断期间 CHWriter.Enqueue 直接走 SpillWriter
  → SpillWriter 按时间/大小旋转 jsonl
  → 30s 后 CircuitBreaker 半开
  → 单笔 CH 写入成功 → CircuitBreaker.OnSuccess (count++)
  → 连续 2 次成功 → 关闭熔断
  → 启动时 SpillReplay 读取 spill 目录所有 jsonl
  → 重放到 CHWriter
  → 成功 → 移到 spill/processed/
```

## 2. 因子计算流（Bar → Factor）

### 2.1 时序图

```
NATS                  factorsvc.Subscriber  WindowBuffer  DSL.Eval      CH         Publisher  NATS
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
- **NaN 处理**：DSL eval 返回 NaN 表示窗口未填满；不写 CH 不发 NATS

## 3. 信号 → 下单流（Factor → OMS → MT）

### 3.1 时序图

```
NATS md.factor   quantengine    risk.PreCheck   mthub.Exec   adapter/mt[45]   mtapi   PG.orders   NATS oms.events
 │ ────────────►│                   │              │             │             │          │             │
 │              │ ONNX/DSL infer    │              │             │             │          │             │
 │              │ Signal{...}       │              │             │             │          │             │
 │              ├──── CH signals (audit) ─────────────────────────────────────────────────►            │
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

- **每个 Signal 必写 CH `signals`**（审计），无论是否最终下单
- **risk.PreCheck 失败 → 信号被丢弃 + audit 日志记录拒因**
- **broker_ticket 必须从 mtapi 返回，不许伪造**
- **state 转换** 走显式状态机（详见 `docs/spec/12-mthub.md`）

## 4. 用户查询流（前端 → CH）

### 4.1 时序图

```
前端                ConnectRPC handler  KlineService    CH查询    CH         PG (fallback)
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
config.LegacyFallback = true（默认 ON，M9 删除）
  → CH 查询 0 行 + 时间窗在 M7 启动前 → 退到 PG
  → CH 查询 0 行 + 时间窗在 M7 启动后 → 返回 0 行（不退 PG，避免误导）
  → CH 查询 N 行 → 直接返回（覆盖率 ≥ 90% 即 M7.8 验收通过）
```

## 5. 健康检查流

### 5.1 端点矩阵

| 端点 | 含义 | 失败条件 |
|---|---|---|
| `/healthz` | 进程活 | 进程僵死 |
| `/readyz` | 准备好接流量 | PG/CH/Redis/NATS 任一不通 |
| `/livez/account/{id}` | 单账户健康 | gateway disconnected ≥ 60s |
| `/metrics` | Prometheus 抓取 | — |

### 5.2 readyz 检查项

```
GET /readyz
├─ check PG: SELECT 1
├─ check CH: SELECT 1
├─ check Redis: PING
├─ check NATS: connection.IsConnected()
└─ check accounts: 至少 50% accounts state ∈ {connected, degraded}
```

## 6. 启动序列

```
ant-server main()
  1. load config (env + yaml)
  2. setup logger (zap)
  3. connect storage (PG, CH, Redis, NATS) — 任一失败 → fatal
  4. run migrations
     - chmigrate.Run(ch)
     - PG migrate (golang-migrate)
  5. init mdgateway components (normalizer, quality, ...)
  6. SpillReplay.Run()  ← 启动时优先回放未提交的 spill
  7. mdgateway.RunGateway()  ← 从 PG 加载账户 → 启动 gateways
  8. factorsvc.Start()  ← 订阅 NATS md.bar.*
  9. quantengine.Start()  ← 加载 ONNX + DSL
 10. oms.Start()  ← 订阅 NATS md.factor.*
 11. start HTTP server (ConnectRPC + SSE) on :8080
 12. start /metrics, /healthz, /readyz
 13. graceful shutdown signals (SIGTERM)
     → drain ConnectRPC
     → flush mdgateway CHWriter
     → close gateways
     → close storage
```

## 7. 关闭/降级序列

任何关键依赖失败的降级路径：

| 失败 | 立即降级 | 持续 X 后 | 完全恢复 |
|---|---|---|---|
| CH 不可达 | SpillWriter 接管 | jsonl 旋转持续；告警 | 启动 SpillReplay |
| NATS 不可达 | Publisher 计数失败；不阻塞 | 30s 重连尝试 | 自动续 |
| 单 broker 不可达 | CircuitBreaker 打开；其他账户照常 | 30s 后半开尝试 | 连续 2 次成功 → 关 |
| Redis 不可达 | 缓存穿透到 PG；性能下降 | 60s 持续 → 告警 | 自动续 |
| PG 不可达 | API 503；新订单拒绝；行情链路照常 | 即时告警 | restart 后续 |

每条降级路径必须有：测试用例（chaos/）+ Prometheus 指标 + runbook 条目（`docs/runbook/mt-incidents.md`）。
