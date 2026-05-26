# spec/31 · 多用户基础设施补强

> **状态**：M10-BASE 地基阶段引入
> **关联**：`docs/金融架构改造-M11路线图-2026-05-25.md` §十一
> **设计动机**：M10-BASE 35 张卡片覆盖"金融语义正确性"，但漏掉"多用户运行时正确性"。后者不是性能优化，而是 ant 作为商业 SaaS 平台的存活底线。
> **架构基线**（2026-05-25 验收，**已落地的事实**）：
> - **身份模型**：`user_id`（业务）+ `account_id`（MT 账户，1 user N account）+ `broker`（数据源）。**无 tenant 概念**。
> - **Context 传播**：`backend/internal/interceptor/auth.go` 的 `UserIDKey` + `GetUserID(ctx) string` **已建**——本 spec 不重复造轮子。
> - **CH 表 user 列**：`md_ticks/factor_values/signals` 已有 `user_id LowCardinality(String)`；`md_bars` 故意无 user_id（市场数据共享）；PARTITION 仅按时间，无需 user 维度分区。
> - **User 配置表**：`user_risk_profiles`（migration 093）已建，含 12 字段（max_positions / daily_loss_limit / killswitch_enabled / symbol_whitelist 等）。
> - **本 spec 范围**：仅补**已建身份体系之上**的运行时基础设施（限速、容量、生命周期、背压、能力扩展）。

---

## 一、多用户运行时（Multi-user Runtime）

### 1.1 维度选择：user_id vs account_id vs broker

ant 的运行时数据流有 3 个隔离维度，**对不同关注点应用不同维度**：

| 数据 / 操作 | 隔离维度 | 理由 |
|---|---|---|
| 计费 / 配额 / 风控限额 | **user_id** | 商业身份单位 |
| broker 连接 / spill 文件 | **account_id** | 一个 user 可有多个 MT 账户在不同 broker |
| Symbol params / 标准化映射 | **broker** | broker 级共享 |
| **行情存储（md_ticks/md_bars）** | **broker（事实上）** | 见下文"行情存储维度澄清" |
| 平台总量限额 | **global** | 跨 user 聚合 |
| Audit log / tracing | **user_id + account_id 双标** | 完整溯源 |

**反例**（已避免）：~~Per-broker 配额~~——broker 是数据源不是商业实体；~~Per-account 风控~~——同一 user 多账户应合并限额（M10-BASE-C4 跨账户净敞口卡片处理这件事）。

#### 行情存储维度澄清（防止误读"按账户隔离"）

`md_ticks` schema 包含 `user_id` 和 `account_id` 列，但**这两列不在 ORDER BY 也不在 PARTITION BY**：

```sql
-- 已落地（chmigrate/006_md_ticks_v2.sql）：
ENGINE = ReplacingMergeTree(arrived_unix_ms)
PARTITION BY toYYYYMM(toDateTime(arrived_unix_ms / 1000))
ORDER BY (broker, canonical, ts_unix_ms, bid, ask, bid_volume, ask_volume)
-- user_id, account_id 仅作业务标签存在
```

加上应用层双层 dedup：

```go
// 已落地（mdgateway/tick_dedup.go）：
// dedup key = (broker, canonical) → ringHash(ts, bid, ask, volumes)
// 与 user_id / account_id 完全无关
// → 同一 broker 同一 symbol 同一 tick 在不同 account 重复送入时
//   应用层会让首个通过，后续全部当 duplicate 拦截
```

**对内（业务视角）**：mdgateway 接收来自 N 个 account 的 broker 连接，每条 tick 携带 `(user_id, account_id)` 标签用于审计。

**对外（存储视角）**：双层 dedup 后**事实上 per-broker 维度共享存储**：

| 维度 | 90 天 TTL 稳态存储 | 与用户数关系 |
|---|---|---|
| 直觉算法（per-account 隔离） | 5 broker × 50 symbol × 1000 user × 2 account ≈ **78 TB** | 线性增长 ❌ |
| **ant 实际**（dedup 后 per-broker 共享） | 5 broker × 50 symbol × 1 tick/s × 90 day × 100B ≈ **200 GB** | 与用户数**无关** ✅ |

数据量级差 400×。即便用户数从 1K 涨到 100K，行情存储量不变（只随 broker × symbol 数线性增长）。

**短期 burst 风险**：ReplacingMergeTree 是最终一致去重，merge 不立即触发。NFP 公布瞬间 100 account 并发写入可能短期 100× 副本，5-10min 后 merge 收敛。**M10-BASE-A4 per-account CH-write limiter（5MB/s）专门挡这件事**。

**user_id/account_id 列的真实代价**：LowCardinality 编码后 ~1 byte/row，可忽略。

**M12 之前不做的优化**：拆 `md_ticks_shared`（无 user 维度）+ `md_subscriptions_5m`（订阅审计）。当前方案在 M10-BASE 阶段足够，等 100K+ 用户 + 真有"按用户分计 tick 量"的计费需求时再重构。

### 1.2 Per-user / Per-account Rate Limiter（**真正的缺口**）

**现状**：
- `backend/internal/mdgateway/backfiller/backfiller.go` 已有 `accountLimiters map[string]*rate.Limiter` + `globalLimiter`（mtapi 历史数据回填限速）
- **主管线（信号生成 / 下单 / CH 写带宽）尚无 per-user/account 限速** ← **本卡片要补**

**契约**：

```go
// backend/internal/usermgr/limiter.go (新增)
package usermgr

type LimiterRegistry struct {
    // user_id → 每分钟信号生成上限（默认 60，付费用户上调）
    signalGenPerUser  *lru.Cache[string, *rate.Limiter]
    // user_id → 每分钟下单数上限（默认 20）
    orderPerUser      *lru.Cache[string, *rate.Limiter]
    // account_id → CH 写带宽（默认 5MB/s，防止跑飞策略撑爆 CH）
    chWritePerAccount *lru.Cache[string, *rate.Limiter]
    // global → 平台总量（保护 CH 集群）
    globalCHWrite     *rate.Limiter
}
```

**关键参数**：
- LRU 容量：50000（覆盖 50K 活跃用户）
- 空闲驱逐：30 天无活动 → 从 LRU 弹出
- 内存预算：50K × 3 limiter × ~200B ≈ 30MB（可接受）

### 1.3 User 配置 + Capability 扩展（**扩展现有表**）

**现状**：`user_risk_profiles`（migration 093）已建，但缺 capability 维度：

```sql
-- 已存在（migration 093）：
-- user_id / max_positions / daily_loss_limit / max_drawdown_pct
-- margin_level_min / session_enabled / slippage_max_points
-- reject_rate_max / heartbeat_timeout_s / killswitch_enabled
-- symbol_whitelist / created_at / updated_at
```

**本 spec 在此表上扩展 5 字段**（migration `105_user_risk_profiles_capabilities.up.sql`）：

```sql
ALTER TABLE user_risk_profiles
    ADD COLUMN capability_tier      SMALLINT NOT NULL DEFAULT 0,        -- 0/1/2/3
    ADD COLUMN order_types_allowed  TEXT[]   NOT NULL DEFAULT ARRAY['market'], -- market/limit/stop_loss/trailing_stop
    ADD COLUMN lot_per_order_max    NUMERIC(8,4) NOT NULL DEFAULT 0.01,
    ADD COLUMN daily_order_max      INT      NOT NULL DEFAULT 5,
    ADD COLUMN leverage_max         INT      NOT NULL DEFAULT 50;

COMMENT ON COLUMN user_risk_profiles.capability_tier IS
    'Tier 0=未验证 1=KYC通过 2=Paper14d通过 3=DSR>1.5+Live30d净值正';
```

**等级模型**（M10-BASE-E6 PromoteToLive 时自动晋升）：

| Tier | 触发条件 | symbol_whitelist | order_types | lot_max | daily_order_max | leverage_max |
|---|---|---|---|---|---|---|
| 0 | 注册即默认 | `['EURUSD']` | `['market']` | 0.01 | 5 | 10 |
| 1 | KYC 通过 | major FX + 主流 crypto | `['market','limit']` | 0.1 | 20 | 30 |
| 2 | Paper 14d Net P&L > 0 | 全部白名单 | `+'stop_loss'` | 0.5 | 50 | 50 |
| 3 | DSR > 1.5 + Live 30d 净值正 | 全部白名单 | `+'trailing_stop'` | 2.0 | 100 | 100 |

### 1.4 Logger / Tracing 自动注入 user_id（**Metrics 不带**）

**现状**：`interceptor/auth.go` 已把 user_id 注入 ctx，但下游 logger / tracing 调用是否自动带？需补一个 helper。

**关键约束**：

| 输出通道 | 是否带 user_id | 理由 |
|---|---|---|
| **structured log**（zap fields） | ✅ 必须带 | 字段不限基数；retention 按 PG/loki 策略 |
| **OTel tracing span attribute** | ✅ 必须带 | tracing attr 不影响 metric cardinality |
| **Prometheus metric label** | ❌ **绝不能带** | 50K 用户 × 5 broker × 50 symbol = 1250 万 series → Prometheus OOM |
| **CH 用户级聚合表** | ✅ 用 user_id 行 | 50K 行/分钟，CH 健康范围 |

**实现**：

```go
// backend/internal/usermgr/context.go
package usermgr

func LoggerWithUser(ctx context.Context, base *zap.Logger) *zap.Logger {
    if uid := interceptor.GetUserID(ctx); uid != "" {
        return base.With(zap.String("user_id", uid))
    }
    return base
}

func SpanWithUser(ctx context.Context, span trace.Span) {
    if uid := interceptor.GetUserID(ctx); uid != "" {
        span.SetAttributes(attribute.String("user_id", uid))
    }
}
```

**用户级 metric 走 CH 而非 Prometheus**：

```sql
-- chmigrate/016_user_metrics_5m.sql
CREATE TABLE IF NOT EXISTS user_metrics_5m (
    bucket_5m_ts UInt64,                       -- 5min bucket close ts
    user_id      LowCardinality(String),
    metric_name  LowCardinality(String),       -- signal_gen / order_placed / ch_write_bytes
    value        Float64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(toDateTime(bucket_5m_ts / 1000))
ORDER BY (bucket_5m_ts, user_id, metric_name)
TTL toDateTime(bucket_5m_ts / 1000) + INTERVAL 90 DAY;
```

每 5min 由后台 goroutine 从 NATS event stream 流式聚合写入。Grafana 用户级仪表盘走 CH，平台级仪表盘走 Prometheus。

---

## 二、Clock 抽象（确定性 + 可测试性的前置）

### 2.1 接口

```go
// backend/internal/clock/clock.go
package clock

type Clock interface {
    Now() time.Time
    Sleep(d time.Duration)
    NewTicker(d time.Duration) Ticker
    NewTimer(d time.Duration) Timer
    AfterFunc(d time.Duration, fn func()) Timer
}

type Ticker interface {
    C() <-chan time.Time
    Stop()
}

type Timer interface {
    C() <-chan time.Time
    Stop() bool
    Reset(d time.Duration) bool
}
```

### 2.2 强制规则

> **`backend/internal/` 下禁止直接调用 `time.Now()` / `time.Sleep` / `time.NewTicker` / `time.AfterFunc`**
> 例外白名单：
> - `internal/clock/` 包自身
> - `cmd/*/main.go`（顶层 wiring）
> - `_test.go` 测试文件（推荐也走 SimulatedClock）

**lint 实现**：`golangci-lint` 加 `forbidigo` 规则：

```yaml
linters-settings:
  forbidigo:
    forbid:
      - p: '^time\.Now$'
        msg: 'use clock.Clock interface instead of time.Now'
      - p: '^time\.Sleep$'
        msg: 'use clock.Clock.Sleep instead'
```

### 2.3 注入路径

```go
// 顶层 main.go
realClock := clock.NewRealClock()
mdgatewayMgr := mdgateway.New(cfg, ..., realClock)
oms := oms.New(cfg, ..., realClock)
risksvc := risksvc.New(cfg, ..., realClock)

// 回测 main.go
simClock := clock.NewSimulatedClock(startTime)
backtestEngine := backtest.New(cfg, ..., simClock)
```

---

## 三、Determinism 契约

### 3.1 策略代码契约

| 规则 | 实现 |
|---|---|
| 禁止 wall clock | Clock interface 强制 |
| 禁止 PID/hostname/env | strategy sandbox 屏蔽 |
| RNG 必须 seed | 注入 `*rand.Rand`，禁用 `math/rand` 全局 |
| map 遍历必须排序 | lint rule + code review |
| Goroutine 并发结果合并必须 deterministic | 按 key 排序后合并 |

### 3.2 事件处理契约

```
同 timestamp 多 tick → 按 (user_id, account_id, symbol) 字典序处理
JetStream 消费 → 严格 stream sequence，禁止 out-of-order ack
回测多 source 合并 → 按 (ts, source_id) 排序
```

### 3.3 验证手段

```bash
# 同一回测跑 2 次，输出 hash 必须完全相同
go test -tags=integration -run TestBacktestDeterminism ./internal/...
# 内部断言：hash(run1.trades) == hash(run2.trades)
```

---

## 四、Strategy Lifecycle 抽象

### 4.1 接口

```go
// backend/internal/strategy/lifecycle.go
type Strategy interface {
    OnStart(ctx context.Context, state []byte) error
    OnBar(ctx context.Context, bar *mdtick.Bar) (*Signal, error)
    OnTick(ctx context.Context, tick *mdtick.Tick) (*Signal, error)  // 可选
    OnOrderEvent(ctx context.Context, event *OrderEvent) error
    OnEndOfDay(ctx context.Context) error
    OnStop(ctx context.Context) ([]byte, error)  // 返回 snapshot

    Snapshot() ([]byte, error)
    Restore(state []byte) error
}
```

### 4.2 状态持久化

```sql
-- backend/migrations/104_strategy_state.up.sql
CREATE TABLE IF NOT EXISTS strategy_state (
    strategy_id     uuid NOT NULL,
    strategy_version int NOT NULL,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state_data      bytea NOT NULL,          -- protobuf 序列化
    state_version   bigint NOT NULL,         -- 每次 snapshot 递增
    snapshot_ts     timestamptz NOT NULL,
    PRIMARY KEY (strategy_id, strategy_version)
);
CREATE INDEX ON strategy_state (user_id, snapshot_ts DESC);
```

### 4.3 Hot-reload 路径

```
old_strategy.OnStop() → snapshot
PG.strategy_state UPDATE
new_strategy = compile(new_code)
new_strategy.OnStart(ctx, snapshot)  // 同一 state 接续
```

**关键不变量**：snapshot 二进制兼容性由 `strategy_version` 保证；不兼容版本拒绝 hot-reload，强制 rollover（旧策略平仓 → 新策略空载启动）。

---

## 五、端到端 Backpressure 规约

每段流水线必须明确"上游压力"的处理策略：

| 段 | 满载策略 | metric |
|---|---|---|
| broker → mdgateway tick chan | bounded(50000)，满 → spill | `md_chan_full_total` |
| mdgateway → CH writer | bounded(50000)，满 → spill 到 jsonl | `md_spill_writes_total` |
| mdgateway → NATS publish | JetStream 满 → drop oldest tick + metric | `md_nats_publish_dropped_total` |
| NATS → factorsvc consumer | consumer lag > 10000 → 告警 + 限流入口 tick rate | `md_consumer_lag` |
| factorsvc → quantengine | bounded chan，满 → drop factor + metric | `md_factor_dropped_total` |
| quantengine → risksvc | bounded chan，满 → drop signal + metric | `signal_dropped_total` |
| risksvc 通过率 < 5% 持续 1min | 信号源标记 "degraded"，拒新策略提交 | `risk_pass_rate` |
| oms → mthub 待提交 > 100 | StrategyBreaker 自动触发（M10-BASE-F7） | `oms_pending_orders` |

---

## 六、Capability Model（扩展 user_risk_profiles）

参见 §1.3 的 5 个新增字段 + 4 等级模型。

**M10-BASE-C1 集成点**：

```go
// backend/internal/risksvc/capability.go (新增)
package risksvc

// LoadCapability 从 user_risk_profiles 加载 user 的能力档案
func (e *Engine) LoadCapability(ctx context.Context, userID string) (*Capability, error) { ... }

// PreCheck 在原有 6 条规则之外加 capability 检查：
//   - symbol 必须在 symbol_whitelist
//   - order_type 必须在 order_types_allowed
//   - lot ≤ lot_per_order_max
//   - 当日订单数 < daily_order_max
//   - leverage ≤ leverage_max
func (e *Engine) PreCheck(ctx context.Context, req *CheckRequest) (*Decision, error) {
    cap, _ := e.LoadCapability(ctx, req.UserID)
    if !contains(cap.SymbolWhitelist, req.Symbol) {
        return &Decision{Allowed: false, Reason: "symbol_not_in_whitelist"}, nil
    }
    // ... 其他 capability 检查
}
```

**注意**：本 spec 不与现有 `user_risk_profiles` 字段冲突；现有 `symbol_whitelist` / `killswitch_enabled` 直接被 capability 检查复用。

---

## 七、与 M10-BASE 的卡片对应

| 章节 | 对应 M10-BASE 新增/调整卡片 | 工日 |
|---|---|---|
| §一 多用户运行时 | **M10-BASE-A4**（新增，Per-user/account limiter + log/trace user_id propagation + user_metrics_5m CH 表） | 1.5 |
| §二 Clock + §三 Determinism | **M10-BASE-A5**（新增） | 2 |
| §四 Strategy Lifecycle | **M10-BASE-E0**（新增） | 3 |
| §五 Backpressure | M10-BASE-B6 增重（+1d） | +1 |
| §六 Capability Model | M10-BASE-C1 增重（+0.5d，仅扩展 user_risk_profiles 5 字段 + capability.go） | +0.5 |
| **小计** | | **+8 工日** |

完整路线图见 `docs/plan/ROADMAP.md` M10-BASE。

---

## 附录：与最初草稿（tenant 模型）的差异

| 项 | 最初草稿（已废弃） | 本最终版（与代码现状一致） |
|---|---|---|
| 身份维度 | tenant_id 独立概念 | user_id（业务）+ account_id（MT）+ broker（数据源） |
| ctx propagation | 新建 `internal/tenant/context.go` | 复用 `internal/interceptor/auth.go` 已有 `UserIDKey` |
| User 配置表 | 新建 `tenant_config` JSONB key/value | 扩展现有 `user_risk_profiles`（已有 12 字段，新增 5 字段） |
| CH 分区 | `PARTITION BY (month, tenant_id)` | `PARTITION BY toYYYYMM(...)` 仅按时间，已对 |
| Metric label | 自动 propagate tenant_id | **不打 user label**（cardinality）；改 log/trace + CH `user_metrics_5m` |
| Rate limiter LRU | 1000 tenant | 50000 user + idle 驱逐 |
| 工日 | +10 | **+8**（A4 / C1 已部分落地） |
