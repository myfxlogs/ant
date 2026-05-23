# ADR-0004 · Tick 去重与质量分级

- **状态**：Accepted
- **日期**：2026-05-23
- **关联 spec**：`docs/spec/11-mdgateway.md` §"quality.go" + §"tick_dedup.go"

## 1. 背景

行情数据的"垃圾进 → 垃圾出"原则要求在 ant 入口处过滤异常 tick。同时 broker 在网络抖动后会重发最近 N 条 tick，若不去重会污染下游：
- CH `md_ticks` 出现重复行（ReplacingMergeTree 也增加 merge 压力）
- bar 聚合器把同一 tick 算两次（OHLCV 失真）
- 因子值偏移

alfq 的 `quality.go` 实现了 5σ 离群 + bid>ask drop + gap + clock skew，但**未做 tick 去重**。Q-010 是 ant 在生产中验证存在的问题。

## 2. 决策

### 2.1 Quality 三级分类

```
PASS    → 正常 tick，进入聚合 / 发布 / CH
DROP    → 完全不进系统（计 metric）
TAGGED  → 进入系统但打标记 (outlier)，下游可选择忽略
```

判定规则（按 quality.go:Check 优先级）：

| 检查 | 失败行为 | 计数指标 |
|---|---|---|
| Bid/Ask parse 失败或 ≤ 0 | DROP | `md_tick_dropped_total{reason="parse_error"}` |
| Bid > Ask | DROP | `md_tick_dropped_total{reason="bid_gt_ask"}` |
| 5σ 离群（Bid 与 100-window median）| TAGGED outlier | `md_tick_outlier_total` |
| Gap 检查（>5s 与上一条）| 不影响 tick | `md_gap_total` |
| Clock skew（broker 与 local 偏 >30s）| 不影响 tick | `md_clock_skew_seconds` (Gauge) |
| **Dedup**（100-window hash 命中）| DROP（**不计入 dropped**）| `md_tick_dedup_total`（独立指标）|

### 2.2 离群检测算法

用 **median + MAD** (Median Absolute Deviation)，不用 mean + stdev：

```
median = sort(prices)[N/2]
MAD = median(|x - median|)
sigma_robust = 1.4826 * MAD
outlier ⇔ |bid - median| > 5 * sigma_robust
```

理由：FX 价格分布重尾（fat-tailed），mean+stdev 在事件期间会扩大 sigma 误判。

### 2.3 Dedup 算法

```
hash = xxhash(ts_unix_ms || bid_string || ask_string)
ringBuffer 容量 100, 按 (broker, canonical) 维度
Seen ⇔ hash ∈ ringBuffer
```

**为什么不计入 dropped**：dedup drop 是已知预期行为（broker 重发），与"脏数据"不同维度，混入 dropped 会污染监控。

## 3. 备选方案

| 维度 | 备选 | 否决 |
|---|---|---|
| 离群检测 | mean + stdev | 重尾分布误判 |
| 离群检测 | EWMA + 控制图 | 复杂；MAD 已足够 |
| Dedup | broker tick UUID（如有）| 不是所有 broker 提供 |
| Dedup | tick 自增计数器 | 重连后会重置 |

## 4. 后果

### 正面
- 异常 tick 不污染 md_ticks / bar / factor
- 重发 tick 静默丢弃，不引发 ReplacingMergeTree merge 压力
- 三级分类（pass/drop/tagged）便于下游策略选择性处理

### 负面
- Quality + Dedup 各加一层 sync.Mutex（已实测延迟 < 5µs）
- 内存：每 (broker, canonical) ~3KB（100 prices + 100 hashes）
- 100 broker × 100 symbols ≈ 30 MB（可接受）

### 中性
- outlier tick 仍写入 CH（带 outlier=true 字段？v2 暂不加，靠 metric 定位）
- 暂不做 staleness check（broker 周末持续推 0 价时段）；交给业务层判断

## 5. 实施约束

### 5.1 接口

`mdgateway/quality.go`:

```go
type CheckResult struct {
    Outlier       bool
    Dropped       bool
    DroppedReason string  // "" | "parse_error" | "bid_gt_ask"
}

func (q *Quality) Check(t *mdtick.Tick) CheckResult
```

`mdgateway/tick_dedup.go`:

```go
type TickDedup struct {
    size int  // 默认 100
}

func (d *TickDedup) Seen(t *mdtick.Tick) bool
```

### 5.2 调用顺序

```go
// HandleTick (manager.go)
qr := m.quality.Check(t)
if qr.Dropped {
    return  // 不进 dedup（dropped 不污染 dedup ring buffer）
}
if m.dedup.Seen(t) {
    return  // dropped by dedup
}
// 进入 aggregator → publisher → chWriter
```

### 5.3 Metrics 暴露

见 `docs/spec/15-observability.md` §3.1。

## 6. 验证方式

### 单元测试

```go
// quality_test.go
func TestCheck_BidGtAsk_Dropped(t *testing.T) { ... }
func TestCheck_FiveSigmaOutlier_Tagged(t *testing.T) { ... }
func TestCheck_GapCounted_NotDropped(t *testing.T) { ... }
func TestCheck_ClockSkewMetric(t *testing.T) { ... }

// tick_dedup_test.go
func TestSeen_NewTick_False(t *testing.T) { ... }
func TestSeen_SameTickAgain_True(t *testing.T) { ... }
func TestSeen_BeyondWindow_False(t *testing.T) { ... }
func TestSeen_DifferentSymbol_Independent(t *testing.T) { ... }
```

### 集成测试

模拟 broker 重连：
1. 推 100 条 tick → 全 PASS
2. 重新推同样 100 条 → 全 dedup
3. 验证 `md_tick_dedup_total` += 100，`md_tick_dropped_total` 不变
4. 验证 CH `md_ticks` 行数仍为 100（没多写）

### 生产验证

```bash
# 24h 运行后
docker exec ant-clickhouse clickhouse-client --query "
  SELECT
    sum(if(arrived_unix_ms = max_arrived OVER (PARTITION BY broker, canonical, ts_unix_ms, bid, ask), 0, 1)) AS dups
  FROM md_ticks
  WHERE arrived_unix_ms > now64()*1000 - 86400000
" | awk '$1>10{exit 1}'  # 24h 内重复行不超过 10
```
