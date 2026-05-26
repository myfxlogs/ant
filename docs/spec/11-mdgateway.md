# 11 · mdgateway 规范（行情网关核心）

> 路径：`backend/internal/mdgateway/`
> 目标 LOC：≤ 1500 行（含测试）；非测试文件 ≤ 800 行（不含 backfiller 子包，见 spec/18）
> 上游：`adapter/mt[45]/`（L2）；下游：`factorsvc/`（L4）+ ClickHouse（存储）+ NATS（fan-out）
>
> **M10 强化叠加**（2026-05-24）：
> - ADR-0008：md_ticks/md_bars ORDER BY 与应用层 dedup hash 对齐；TTL/分桶强制使用 `arrived_unix_ms`
> - ADR-0009：spill_replay 双写 NATS；bar finality 不可变；historical backfill（spec/18）
> - ADR-0010：DLQ + e2e latency histogram + OTel trace + 6 条新 alert
> - ADR-0011：CHWriter 容量调优（QueueSize 50000）+ Buffer engine + envelope encryption + normalizer cache PG NOTIFY 失效

## 1. 文件清单（必须全部存在）

```
backend/internal/mdgateway/
├── manager.go              ≤200 lines  Gateway 注册 + 生命周期 + 熔断注入
├── runner.go               ≤200 lines  装配入口（PG → 启动 gateways）
├── normalizer.go           ≤120 lines  canonical 解析
├── quality.go              ≤180 lines  bid>ask / 5σ / gap / skew
├── tick_dedup.go           ≤80  lines  滑动窗口去重
├── bar_aggregator.go       ≤180 lines  tick → 6 周期 bar
├── publisher.go            ≤180 lines  NATS JetStream
├── clickhouse_writer.go    ≤180 lines  异步 batch 写 CH
├── spill_writer.go         ≤200 lines  本地 jsonl 旋转
├── spill_replay.go         ≤120 lines  启动回放
├── circuit_breaker.go      ≤140 lines  滑动窗口熔断
├── metrics.go              ≤100 lines  Prometheus 指标定义
├── chmigrate/
│   ├── migrate.go
│   ├── 001_md_ticks.sql
│   ├── 002_md_bars.sql
│   ├── 003_factor_values.sql
│   ├── 004_signals.sql
│   ├── 005_schema_version.sql
│   ├── 006_md_ticks_v2.sql       # M10 ADR-0008 ORDER BY + TTL 切到 arrived_unix_ms
│   ├── 007_md_bars_v2.sql        # M10 ADR-0008
│   ├── 008_md_ticks_dlq.sql      # M10 ADR-0010 DLQ 表
│   └── 009_md_buffer_tables.sql  # M10 ADR-0011 Buffer engine
├── backfiller/                   # M10 ADR-0009，详见 spec/18
├── normalizer_invalidator.go ≤80 lines   M10 ADR-0011 PG LISTEN
├── dlq_writer.go             ≤80 lines   M10 ADR-0010 采样写 DLQ
└── *_test.go                每个非测试文件必须有对应 _test.go，覆盖率 ≥ 60%
```

## 2. manager.go

### 2.1 公开接口

```go
package mdgateway

type Manager struct {
    log         *zap.Logger
    normalizer  *Normalizer
    quality     *Quality
    dedup       *TickDedup
    aggregator  *BarAggregator
    publisher   *Publisher
    chWriter    *CHWriter
    spillWriter *SpillWriter
    breakers    map[string]*CircuitBreaker  // brokerKey ("broker|host:port") → breaker（per-broker，非 per-account；见 §11）

    mu       sync.RWMutex
    gateways map[string]Gateway  // accountID → Gateway
}

// NewManager 构造（不启动）。所有依赖通过参数注入，便于测试。
func NewManager(deps ManagerDeps) *Manager

// AddGateway 注册一个新 gateway（启动 + 订阅）。
// 幂等：同 accountID 重复 add 会先 disconnect 旧的。
func (m *Manager) AddGateway(ctx context.Context, gw Gateway, syms []string) error

// RemoveGateway 关闭并移除。idempotent。
func (m *Manager) RemoveGateway(ctx context.Context, accountID string) error

// HandleTick 是 adapter 调用的入口（TickHandler 类型）。
// 串接 normalizer → quality → dedup → aggregator → publisher → chWriter。
func (m *Manager) HandleTick(t *mdtick.Tick)

// Health 返回所有账户健康摘要。
func (m *Manager) Health() []AccountHealth
```

### 2.2 HandleTick 内部流程

```go
func (m *Manager) HandleTick(t *mdtick.Tick) {
    // 1. canonical
    t.Canonical = m.normalizer.Resolve(t.Broker, t.SymbolRaw)

    // 2. quality
    qr := m.quality.Check(t)
    if qr.Dropped {
        return
    }

    // 3. dedup
    if m.dedup.Seen(t) {
        return
    }

    // 4. aggregate (may emit Bar)
    var bars []*mdtick.Bar
    m.aggregator.AddTick(t, func(b *mdtick.Bar) { bars = append(bars, b) })

    // 5. publish tick
    m.publisher.PublishTick(t)

    // 6. publish bars
    for _, b := range bars {
        m.publisher.PublishBar(b)
    }

    // 7. ch write (async; chan-based)
    m.chWriter.EnqueueTick(t)
    for _, b := range bars {
        m.chWriter.EnqueueBar(b)
    }
}
```

## 3. runner.go

### 3.1 公开接口

```go
package mdgateway

// Run 装配并启动 mdgateway。阻塞直到 ctx.Done。
//
// 流程：
//   1. SpillReplay 回放未提交 jsonl
//   2. 从 PG 加载 active 账户（mt_accounts.is_active=true）
//   3. 为每个账户构造 adapter Gateway 并 AddGateway
//   4. 启动 CHWriter 异步 loop
//   5. 启动 health monitor goroutine（每 30s 检查所有 gateway）
func Run(ctx context.Context, deps RunnerDeps) error

type RunnerDeps struct {
    Log       *zap.Logger
    PG        *pgxpool.Pool
    CH        clickhouse.Conn
    NATSConn  *nats.Conn
    SpillDir  string  // 默认 /var/lib/ant/spill
    Vault     vault.Client  // 解密 mt_account.password
}
```

### 3.2 加载账户 SQL

实际查询走 `mt_accounts_v2` 视图（见 `docs/spec/13-clickhouse-schema.md` §4.1），把 v1 列名映射到 v2 命名：

```sql
SELECT
  id, user_id,
  platform,                    -- v1 列名 mt_type
  broker,                      -- v1 列名 broker_company
  mtapi_host,                  -- v1 列名 broker_host
  mtapi_port,                  -- v2 新增
  login,
  password_encrypted,          -- v2 新增（M7.1-2 ETL 后填充）
  mtapi_token_encrypted,       -- v2 新增（同上）
  server,                      -- v1 列名 broker_server
  canonical_subscribed_symbols -- v2 新增
FROM mt_accounts_v2
WHERE is_active = true;
```

**M7.1-2 实施前**：`mt_accounts_v2` 视图未建。runner.go 必须先检查视图存在，否则 fatal。

**ETL 路径**（M7.1-2 卡片同步执行）：
1. `chmigrate` 创建 `mt_accounts_v2` 视图
2. 写一次性 ETL：把 v1 `password` `mt_token` 通过 vault 加密 → 填入 `password_encrypted` `mtapi_token_encrypted`
3. 验证 100% 行已 ETL → 标记 v1 列为 deprecated（注释，不删除）

## 4. normalizer.go

```go
type CanonicalResolver interface {
    Resolve(broker, raw string) string
}

type Normalizer struct {
    pg     *pgxpool.Pool        // broker_symbols 表
    cache  *lru.Cache[string, string]  // key: broker:raw
}

// Resolve 顺序：
//   1. cache 命中 → 返回
//   2. PG.broker_symbols 查询 → 写 cache → 返回
//   3. 算法 fallback：去除常见后缀（.m, .pro, .x, .c, _i 等）
//   4. 缓存 fallback 结果（带 1h TTL，便于 PG 后续补全后刷新）
func (n *Normalizer) Resolve(broker, raw string) string
```

后缀剥离规则（**alfq Q-005 + Q-006 实践**）：

```
. → 同名优先（"BTCUSD." → "BTCUSD"）
.c / .pro / .m / .x → 同名 + 后缀小写归一
_i / _r → 同名（institutional / retail tier）
USD$ + suffix in {m,pro} → keep as canonical
```

## 5. quality.go

```go
type QualityConfig struct {
    GapMaxSeconds  float64  // 默认 5
    OutlierSigma   float64  // 默认 5
    SkewMaxSeconds float64  // 默认 30
    HistorySize    int      // 默认 100
}

type CheckResult struct {
    Outlier      bool
    Dropped      bool
    DroppedReason string  // "bid_gt_ask" | "non_positive" | "parse_error" | ""（不再含 outlier；见 §5.1）
}

func (q *Quality) Check(t *mdtick.Tick) CheckResult
```

详细规则：

### 5.1 检查规则（行业标杆：行情数据"少不丢"原则）

> **核心原则**：只在数据**显然不可用**时丢弃；可疑但合法的跳变（新闻、流动性事件）一律放行 + 上报 metric，由下游决定如何应对。
> 参考：LMAX、Bloomberg B-PIPE、Refinitiv Elektron 均采用此哲学。

| 规则 | 触发条件 | 行为 | 理由 |
|---|---|---|---|
| **bid > ask** | `Bid.Cmp(Ask) > 0` | **drop** + `reason="bid_gt_ask"` | 套利不可能；必为 broker bug |
| **非正数价格** | `Bid.Sign()<=0 \|\| Ask.Sign()<=0` | **drop** + `reason="non_positive"` | 价格不可能 ≤ 0 |
| **解析失败** | `decimal.NewFromString` err | **drop** + `reason="parse_error"` | 数据损坏 |
| 价格跳变 | \|bid - median(last 100)\| > N×MAD（默认 N=5）| **不 drop**；置 `outlier=true` + metric | FX 重尾分布；新闻/经济数据公布会真实跳变 |
| Gap | `(cur.ts - last.ts)/1000 > 5s` | **不 drop**；metric `md_gap_total` | 周末/节假日/低流动性正常现象 |
| Clock skew | \|broker.ts - local.now\| > 30s | **不 drop**；gauge `md_clock_skew_seconds` | 仅观测，不参与决策（已用 ArrivedUnixMs） |

**禁止**：用 mean/stdev（FX 重尾不适用）。**必须**用 median + MAD（`alfq quality.go` 同款）。

### 5.2 metric 与 reason 标签

```
md_tick_dropped_total{broker, canonical, reason}
  reason ∈ {bid_gt_ask, non_positive, parse_error, spill_failed}
       前 3 类来自 quality（§5.1）；spill_failed 来自 §9 升级路径（D-3）
md_tick_anomaly_total{broker, canonical, kind}
  kind ∈ {outlier, gap, clock_skew}                     不丢，仅观测
```

**判分依据**（决策记录见 BACKLOG RV-C2）：5σ 跳变在外汇 NFP / FOMC / 黑天鹅事件下是真实价格，drop 会导致策略错过关键信号。代价比留下偶尔的"脏 tick"高得多。

## 6. tick_dedup.go（v2 新增）

```go
type TickDedup struct {
    mu     sync.Mutex
    seen   map[string]*ringHash  // key: broker:canonical
    size   int                   // 默认 100
}

type ringHash struct {
    hashes []uint64  // ring buffer
    cur    int
    set    map[uint64]struct{}  // O(1) 查询
}

// Seen 返回 true 表示该 tick 已在最近 N 条窗口内出现过（应丢弃）。
// hash 计算：xxhash(ts_unix_ms || bid || ask || bid_volume || ask_volume)
//
// 为何含 volume（决策 D-2，2026-05-23）：
// 同一毫秒内 broker 可能推送多笔同价位但量不同的成交（活跃品种 EURUSD 欧盘、
// XAUUSD 美盘 1s 内常见）。仅用 (ts,bid,ask) 三元组会把第二笔误判为重复，
// 影响 bar 的 Volume 字段精度。加上 (bid_volume,ask_volume) 后 false-positive ≈ 0，
// 哈希成本由 3 字段串接增至 5 字段串接，可忽略。
func (d *TickDedup) Seen(t *mdtick.Tick) bool
```

**触发场景**：broker 重连后会重发最近 N 条历史 quote，若不去重会污染 `md_ticks`。

## 7. bar_aggregator.go

```go
var Periods = []struct {
    Name string
    Ms   int64
}{
    {"1m", 60_000},
    {"5m", 300_000},
    {"15m", 900_000},
    {"1h", 3_600_000},
    {"4h", 14_400_000},
    {"1d", 86_400_000},
}

type BarAggregator struct {
    mu     sync.Mutex
    inProgress map[string]*openBar  // key: broker:canonical:period
}

// AddTick 处理 tick，可能 emit 0~6 个完成的 bar。
//
// QUIRK Q-001: 用 ArrivedUnixMs 而不是 TsUnixMs 分桶。
//              MT4 OnQuote.Time 在部分 broker 不实时推进。
func (a *BarAggregator) AddTick(t *mdtick.Tick, onBar func(*mdtick.Bar))
```

**bar 完成的判定**：

```go
bucket := t.ArrivedUnixMs / period.Ms
if open.bucket != bucket {
    // 当前 tick 跨过周期边界
    onBar(open.toBar())
    open.reset(t, bucket)
}
open.merge(t)
```

OHLCV 计算：
- Open = 第一个 tick 的 (Bid+Ask)/2
- High = max((Bid+Ask)/2)
- Low = min((Bid+Ask)/2)
- Close = 最后一个 tick 的 (Bid+Ask)/2
- Volume = Σ (BidVolume + AskVolume)
- TickCount = N

## 8. publisher.go

```go
const (
    StreamName = "MD_EVENTS"
    SubjectTickPrefix = "md.tick."
    SubjectBarPrefix  = "md.bar."
)

type Publisher struct {
    js  nats.JetStreamContext
    log *zap.Logger
}

func (p *Publisher) PublishTick(t *mdtick.Tick) error
func (p *Publisher) PublishBar(b *mdtick.Bar) error
```

Subject 格式：
- Tick: `md.tick.{broker}.{canonical}`
- Bar: `md.bar.{broker}.{canonical}.{period}`

JetStream 配置（启动时确保存在）：

```go
StreamConfig{
    Name: "MD_EVENTS",
    Subjects: []string{"md.tick.>", "md.bar.>"},
    Storage: nats.FileStorage,
    Retention: nats.LimitsPolicy,
    MaxAge: 24 * time.Hour,
    MaxBytes: 10 * 1024 * 1024 * 1024,  // 10GB
}
```

## 9. clickhouse_writer.go

```go
type CHWriterConfig struct {
    // M10 默认值（ADR-0011 §2.1）；env 可覆盖
    FlushInterval time.Duration  // 默认 500ms（M7 旧值 1s）
    MaxBatchSize  int            // 默认 10000（M7 旧值 1000）
    QueueSize     int            // 默认 50000（M7 旧值 5000）；满则走 spill
}

type CHWriter struct {
    cfg    CHWriterConfig
    conn   clickhouse.Conn
    log    *zap.Logger
    spill  *SpillWriter

    tickQ  chan *mdtick.Tick
    barQ   chan *mdtick.Bar
}

func (w *CHWriter) EnqueueTick(t *mdtick.Tick)
func (w *CHWriter) EnqueueBar(b *mdtick.Bar)
func (w *CHWriter) Start(ctx context.Context)
func (w *CHWriter) Flush(ctx context.Context) error  // shutdown 时
```

行为契约：
- `EnqueueTick` chan 满 → **立即** 写 SpillWriter，不阻塞调用方
- 后台 goroutine 每 `FlushInterval` 或 `MaxBatchSize` 触发批量 INSERT
- INSERT 失败 → 整批写 SpillWriter；不重试（spill_replay 启动时回放）
- **SpillWriter 也失败时的升级路径（决策 D-3，2026-05-23）**：
  1. `spill.WriteTick/WriteBar` 连续失败 ≥ 3 次（计数器 `spillFailStreak`）
  2. CHWriter 通过注入的 `OnSpillFail func(brokerKey string, err error)` 回调上报 Manager
  3. Manager 对该 brokerKey 的 CircuitBreaker 调 `OnFailure()`（与 broker 不可达同等故障）
  4. metric `md_tick_dropped_total{reason="spill_failed"}` 递增；alert `SpillUnwritable` 触发
  5. 一次成功写入即重置 streak
- 触发场景：磁盘满、目录权限漂移（容器重启 UID 变化）、云盘短暂只读
- 哲学一致：**任何"行情链路最后一道防线"故障都必须熔断而不是静默丢数据**（01-vision §准则 3）

INSERT 模板（用 ClickHouse client `PrepareBatch`）：

```sql
-- M10 ADR-0011：写入 Buffer engine 表，CH 内部异步 flush 到 md_ticks
-- 详见 spec/13 §2.7
INSERT INTO md_ticks_buffer (
    user_id, account_id, broker, symbol_raw, canonical,
    ts_unix_ms, arrived_unix_ms, bid, ask, bid_volume, ask_volume,
    is_replay
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
```

bar 同理 `INSERT INTO md_bars_buffer (...)`。

## 10. spill_writer.go + spill_replay.go

### 10.1 spill_writer

```go
type SpillWriterConfig struct {
    Dir            string         // 默认 /var/lib/ant/spill
    MaxFileBytes   int64          // 默认 100 MB
    MaxFileAge     time.Duration  // 默认 1h
}

type SpillWriter struct {
    cfg     SpillWriterConfig
    log     *zap.Logger

    mu      sync.Mutex
    cur     *os.File
    curBytes int64
    curStart time.Time
}

// WriteTick 把 tick 序列化为 jsonl 一行写入当前 spill 文件。
// 文件超过 MaxFileBytes 或 MaxFileAge → rotate。
// rotate 命名：spill-{ts}.jsonl
func (s *SpillWriter) WriteTick(t *mdtick.Tick) error
func (s *SpillWriter) WriteBar(b *mdtick.Bar) error
```

jsonl 行格式（一行一对象）：

```json
{"_kind":"tick","user_id":"...","account_id":"...","broker":"...","ts_unix_ms":1234567890123, "bid":"108234.56", ...}
```

### 10.2 spill_replay

```go
// Run 扫描 spill 目录的所有 *.jsonl，逐行重放到 CHWriter。
// 成功后移到 dir/processed/{ts}/{filename}。
// 失败行写入 dir/failed/{ts}/{filename}.errors.jsonl。
func (r *SpillReplay) Run(ctx context.Context) error
```

## 11. circuit_breaker.go

```go
type State int
const (
    StateClosed State = iota  // 正常
    StateOpen                 // 熔断中
    StateHalfOpen             // 试探
)

type CircuitBreaker struct {
    failureThreshold int           // 默认 5
    successThreshold int           // 默认 2
    cooldown         time.Duration // 默认 30s
    windowSize       int           // 默认 10（滑动窗口）

    mu        sync.Mutex
    state     State
    failures  int
    successes int
    openedAt  time.Time
}

func (c *CircuitBreaker) Allow() bool       // 半开/关闭时 true
func (c *CircuitBreaker) OnSuccess()
func (c *CircuitBreaker) OnFailure()
func (c *CircuitBreaker) State() State
```

**作用域：每个 broker endpoint 一个 CircuitBreaker（非 per-account）**。

理由（行业标杆 Hystrix / resilience4j）：
- 故障相关性在网络层 — 同 broker 多账户共享 mtapi 网关，一个不可达时全体不可达，per-account 重复决策无意义且制造噪音
- 认证错误（密码错、token 过期）走单独路径：直接置账户为 `disabled`，不进入 breaker（避免污染网络故障判断）

`brokerKey = fmt.Sprintf("%s|%s:%s", cfg.Broker, cfg.MtapiHost, cfg.MtapiPort)`

Manager 在调用 adapter 前调 `breakers[brokerKey].Allow()`，失败时 `OnFailure()`；认证错误**不**调 `OnFailure`，改写 PG `mt_accounts.account_status='auth_failed'` + alert。

## 12. metrics.go（Prometheus）

| 指标名 | 类型 | Labels |
|---|---|---|
| `md_tick_total` | Counter | broker, canonical |
| `md_tick_dropped_total` | Counter | broker, canonical, reason |
| `md_tick_dedup_total` | Counter | broker |
| `md_bar_flushed_total` | Counter | broker, canonical, period |
| `md_publish_total` | Counter | kind={tick,bar}, status={ok,err} |
| `md_publish_latency_seconds` | Histogram | kind |
| `md_ch_write_total` | Counter | kind={tick,bar}, status |
| `md_ch_write_latency_seconds` | Histogram | kind |
| `md_spill_writes_total` | Counter | reason={ch_error,queue_full} |
| `md_spill_replay_total` | Counter | status={ok,err} |
| `md_circuit_state` | Gauge | broker, mtapi_endpoint（per-broker，非 per-account；见 §11）|
| `md_tick_anomaly_total` | Counter | broker, canonical, kind={outlier,gap,clock_skew}（见 §5）|
| `md_gateway_connected` | Gauge | account_id, broker, platform |
| `mt_account_connected` | Gauge | account_id, broker, platform（同 md_gateway_connected 别名；前端/Grafana 用此名查询，决策 RV-C4）|
| `md_clock_skew_seconds` | Gauge | broker |
| `md_canonical_fallback_total` | Counter | broker |

## 13. factorsvc 集成（短摘要）

`factorsvc/subscriber.go` 订阅 NATS subject `md.bar.>`，每条 bar：
1. 找出适用的 FactorDef（按 canonical + period 匹配）
2. WindowBuffer.Append → DSL.Eval
3. 写 CH `factor_values` + 发布 `md.factor.{canonical}.{factor_name}`

详见 `docs/spec/12-mthub.md`（占位）+ 后续 `docs/spec/20-factorsvc.md`（M8 撰写）。

## 13.5 NATS JetStream stream 配置（B1）

`internal/storage/nats/client.go::ensureStream` 在启动时确保下列 stream 存在；不存在则按下表创建，已存在则校验关键字段一致（不一致 fatal）：

| Stream | Subjects | Storage | Retention | MaxAge | MaxBytes | Replicas | Discard |
|---|---|---|---|---|---|---|---|
| `MD_EVENTS` | `md.tick.>` `md.bar.>` `md.factor.>` | File | Limits | 24h | 20 GiB | 1 | OldFirst |
| `OMS_EVENTS` | `oms.order.>` `oms.signal.>` | File | Limits | 7d | 5 GiB | 1 | OldFirst |

理由：md 数据量大 → 短保留；oms 业务数据 → 长保留。Replicas=1 因 v2 单机部署（ADR 0001 §3）。Discard=OldFirst 保护 publisher 不阻塞。

## 13.6 backpressure 策略（B3 + M10-BASE-B6，全链统一）

| 队列 | 满了怎么办 | metric | 上限/默认 |
|---|---|---|---|
| `CHWriter.tickQ` | 直接 SpillWriter（不阻塞） | `md_chan_full_total` | 50000 |
| `CHWriter.barQ` | 同上 | `md_chan_full_total` | 50000 |
| `Publisher` NATS | JetStream Discard=OldFirst + metric increment | `md_nats_publish_dropped_total` | stream MaxBytes |
| `Manager.HandleTick` pipeline | 各阶段 bounded chan 满载 drop | `md_chan_full_total` | per-stage |
| NATS consumer | 消费滞后监控 | `md_consumer_lag` | gauge |
| signal 通路 | 阻塞 100ms 后丢弃 | `signal_dropped_total` | 1000 |
| SSE per-client | 阻塞 50ms 后断开 client | `sse_client_dropped_total` | 256 events |

**核心原则**：上游永远不许因下游拥塞而阻塞 → tick 接收链路 0 阻塞点。

### 13.6.1 M10-BASE-B6 Backpressure Metrics

四条核心 backpressure metric（Prometheus 格式，`/metrics` 端点输出）：

| Metric | 类型 | 含义 |
|---|---|---|
| `md_chan_full_total` | counter | bounded channel 满载 drop 累计次数 |
| `md_nats_publish_dropped_total` | counter | NATS JetStream publish 失败累计次数 |
| `md_consumer_lag` | gauge | NATS consumer 滞后消息数 |
| `signal_dropped_total` | counter | quantengine signal 被丢弃累计次数 |

验收：`curl -s localhost:8080/metrics | grep -cE '^(md_chan_full_total|md_nats_publish_dropped_total|md_consumer_lag|signal_dropped_total)' | awk '$1>=4{exit 0}{exit 1}'`

## 13.7 graceful shutdown 顺序（B2）

收到 SIGTERM 后 `runner.Stop(ctx)` 严格按以下顺序：

```
1. HTTP server.Shutdown(ctx, 5s)           // 停接新请求；正在跑的 RPC 自然结束
2. for each gateway: g.Disconnect(ctx)     // 不再产生新 tick
3. wait 100ms                              // 让 in-flight tick 跑完 HandleTick
4. CHWriter.Flush(ctx, 10s)                // drain tickQ/barQ → CH（CH 不通则 spill）
5. Publisher.Drain(ctx, 5s)                // NATS publish ack 等待
6. SpillWriter.Close()                     // fsync 当前 jsonl
7. close storage (NATS, CH, Redis, PG)
8. logger.Sync()
```

每步带超时；总预算 ≤ 30s（k8s/systemd 默认 stopGracePeriod 内）。任一步超时 → `log.Warn` 并继续；不阻断后续步骤。

## 15. Bar 修订检测与处理

> **详细规范**：`docs/spec/25-bar-revision-cascade.md`
> **关联 ADR**：ADR-0016

### 15.1 修订检测（`bar_aggregator.go` 追加）

```go
// BarAggregator.flushBar 在 bar 完成时检测是否为修订版：
//   SELECT version FROM md_bars
//   WHERE broker=? AND canonical=? AND period=? AND bar_start_unix_ms=?
//   ORDER BY version DESC LIMIT 1
// 若已有 version ≥ 1 → 本次 bar 的 version = prev_version + 1
// 首次写入 → version = 1
```

### 15.2 修订通知

```go
// Publisher 新增方法：
func (p *Publisher) PublishBarRevision(b *mdtick.Bar) error
// Subject: bar.revision.<broker>.<canonical>.<period>
```

### 15.3 不变性保证

- `md_bars` 仅 INSERT，永不做 UPDATE（即使同一 bar_start 有修订版）
- 修订版与初版通过 `(broker, canonical, period, bar_start_unix_ms, version)` 唯一标识
- 查询历史 bar 时始终取 `MAX(version)` per bar_start

### 15.4 指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `md_bar_revision_total` | Counter | broker, canonical, period |
| `md_bar_skipped_finalized_total` | Counter | broker, canonical, period（bar 已固化后拒绝修订） |

## 16. 验收命令

```bash
# 编译 + 测试
( cd backend && go build ./internal/mdgateway/... \
                && go test -race -cover ./internal/mdgateway/... )

# LOC 上限
LOC=$(find backend/internal/mdgateway -name "*.go" -not -name "*_test.go" \
       | xargs wc -l | tail -1 | awk '{print $1}')
test "$LOC" -le 800 || { echo "LOC=$LOC > 800"; exit 1; }

# 必有文件检查
for f in manager.go runner.go normalizer.go quality.go tick_dedup.go \
         bar_aggregator.go publisher.go clickhouse_writer.go \
         spill_writer.go spill_replay.go circuit_breaker.go metrics.go; do
  test -f "backend/internal/mdgateway/$f" || { echo "MISSING $f"; exit 1; }
done

# 不许 import mt4client/mt5client
! grep -rE "anttrader/internal/mt[45]client" backend/internal/mdgateway/ \
  || { echo "FAIL: mdgateway imports legacy mt[45]client"; exit 1; }

# 启动 docker 后跑数据流端到端
docker exec ant-clickhouse clickhouse-client --query \
  "SELECT count() FROM md_ticks WHERE arrived_unix_ms > $(date -d '5 minutes ago' +%s%3N)" \
  | awk '$1<10{exit 1}'  # 5 分钟内至少 10 条 tick
```
