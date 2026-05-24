# M10 设计缺陷审查 · 需澄清

> **写给 Claude**：以下是在 M10 数据基础 A+ 硬化完成后，从架构角度审查发现的 9 个设计缺陷。请在 M10 runner 联调前处理 🔴 严重项，🟡 中等项可在联调中逐步修正。
>
> **审查日期**：2026-05-24

---

## 🔴 严重（阻塞性）

### S-1 ReplacingMergeTree 无 FINAL 保障 — 读路径返回重复行

**位置**：`md_ticks` / `md_bars` 表的查询代码全仓库仅一处（`cmd/e2e-check/main.go:73`），且不加 `FINAL`。

**问题**：ReplacingMergeTree 只在后台 merge 时去重，查询不保证去重。不加 `FINAL` 的 `SELECT count()` 会返回偏大值，导致 md-doctor reconcile 误报数据不一致、回测 OHLCV 失真。

**修复**：所有对 `md_ticks` / `md_bars` 的分析类查询加 `FINAL` 关键字，并在 `docs/spec/13-clickhouse-schema.md` 中记录此查询规范。

---

### S-2 Buffer engine 静默丢数据 — CH OOM 时无感知

**位置**：`clickhouse_writer.go` INSERT 目标为 `md_ticks_buffer` / `md_bars_buffer`（Buffer engine）。

**问题**：Buffer engine 在 CH 进程 OOM 时内存数据直接丢失，但 INSERT 已返回成功 → 应用层认为数据已落盘。spill 降级路径仅在 INSERT 失败时触发，不覆盖 CH 进程崩溃场景。

**修复**（任选其一）：
- **方案 A**：加 env 开关 `ANT_CH_BUFFER_ENABLED`，默认 true；运维可在发现 CH 不稳定时设为 false 直接写 `md_ticks`
- **方案 B**：缩短 Buffer engine 的 `min_time`/`max_time` 参数（减少窗口内数据量）
- **方案 C**：在 `docs/spec/13-clickhouse-schema.md` §2.7 明确声明丢数据风险及降级方案

---

### S-3 IngestExternalBar 逻辑缺陷 — 历史回填 bar 会被全部拒绝

**位置**：`backend/internal/mdgateway/bar_aggregator.go` `IngestExternalBar` 方法。

**当前代码**：
```go
if finalized, ok := a.finalizedBars[fk]; ok && b.CloseTsUnixMs <= finalized {
    return false // skip
}
```

**问题**：`finalizedBars` 加载的是 `SELECT MAX(close_ts_unix_ms)`（最新 bar 时间戳）。backfiller 回填的是**历史缺口**（例如 3 天前的 1h bar），其 `CloseTsUnixMs` 必然远小于 MAX。按当前逻辑**所有历史回填 bar 都会被拒绝**。这是阻塞性 bug。

**修复**：finality 应检查 bar 是否**已精确存在**，而非时间戳是否小于等于最新值：
```go
// 方案：记录已存在 bar 的 close_ts 集合（非仅 MAX），精确去重
if existing, ok := a.existingCloseTs[fk]; ok && existing[b.CloseTsUnixMs] {
    return false
}
```

---

## 🟡 中等（影响正确性或性能）

### M-1 Backfiller 全局限速器配置错误

**位置**：`backend/internal/mdgateway/backfiller/backfiller.go` `New` 函数。

**当前代码**：
```go
limiter: rate.NewLimiter(rate.Every(10*time.Second), 1), // 6 req/min/account
```

**问题**：这是**全局单一** limiter。100 个活跃账户共享 6 req/min，回填 30 天 1m 缺口需数小时。设计意图是 **per-account** 6 req/min。

**修复**：改为 per-account limiter map + 全局总限速：
```go
type Backfiller struct {
    accountLimiters map[string]*rate.Limiter // key: accountID, 6 req/min each
    globalLimiter   *rate.Limiter            // 60 req/s global ceiling
}
```

---

### M-2 Spill replay 无去重 — NATS 重复发布

**位置**：`backend/internal/mdgateway/spill_replay.go` `replayFile` 方法。

**问题**：如果 replay 被中断重跑，同一 spill 文件被重放两次，NATS 收到两份 replay 数据。下游 factorsvc 重复计算因子。

**修复**：在 `publisher.go` 的 `PublishMsg` 调用中加入 `Nats-Msg-Id` header（基于 broker+canonical+ts+xxhash 生成），依赖 NATS JetStream 消息去重。或在 `spill_replay.go` 记录已处理文件名到本地 marker 文件。

---

### M-3 runner.go 静默降级 — finalizedBars 为空时 finality 失效

**位置**：`backend/internal/mdgateway/runner.go` `loadFinalizedBars` 函数。

**问题**：CH 不可达时返回空 map，runner 继续启动，`finalizedBars` 全空 → `IngestExternalBar` 对所有 bar 放行 → bar finality 保护完全失效。

**修复**：CH 不可达时 `loadFinalizedBars` 应返回 error 并阻塞启动（fatal），或在 spill 目录持久化 `finalized-bars.json` 作为重启快照兜底。

---

## 🟢 轻微（工程卫生）

### L-1 DLQ 写入在 HandleTick 热路径同步执行

**位置**：`backend/internal/mdgateway/quality.go` `Check` 方法 + `dlq_writer.go` `WriteTick`。

**问题**：drop 时调 `dlq.WriteTick(nil, t, "bid_gt_ask", "")`，内部做 `PrepareBatch → Append → Send` 全部同步阻塞。broker 脏数据风暴时 1% 采样也会在热路径产生延迟。

**修复**：DLQ 写入改为异步（go routine + buffered channel），或复用 `CHWriter.tickQ` 的 spill 降级路径。

---

### L-2 EXCHANGE TABLES 迁移无超时保护

**位置**：`backend/internal/mdgateway/chmigrate/006_md_ticks_v2.sql`。

**问题**：`INSERT INTO md_ticks_v2 SELECT * FROM md_ticks` 在 1.5TB 数据下可能运行数小时。期间 CH 重启或磁盘满会导致迁移中断但 `md_ticks_v2` 表已部分填充，后续 `EXCHANGE TABLES` 交换不完整数据。

**修复**：在 spec/13 或迁移脚本注释中加入前置条件："迁移前验证磁盘 ≥ 原表 × 1.5，在维护窗口执行"。

---

### L-3 md_bars 长期容量无规划

**位置**：`docs/spec/13-clickhouse-schema.md`。

**问题**：`md_bars` 无 TTL（"不过期"）。100 账户 × 6 周期 = 315M 行/年，5 年后磁盘压力显著且查询变慢。当前 spec 无冷热分层或归档策略。

**修复**：在 spec/13 §8 容量规划中加入长期预估，建议 3 年 TTL 或年度归档到 S3/CH 冷存储。

---

## 修复优先级

| 顺序 | ID | 问题 | 涉及文件 | 工作量 |
|---|---|---|---|---|
| 1 | S-3 | finality 拒绝历史回填 | `bar_aggregator.go` | 改 10 行 |
| 2 | S-1 | FINAL 缺失 | `e2e-check/main.go` + spec/13 | 加 FINAL + 文档 |
| 3 | S-2 | Buffer engine 静默丢数据 | `clickhouse_writer.go` + spec/13 | 加开关 + 文档 |
| 4 | M-1 | 限速器改为 per-account | `backfiller/backfiller.go` | 改 20 行 |
| 5 | M-2 | replay NATS 去重 | `spill_replay.go` | 加 marker 文件 |
| 6 | M-3 | finalizedBars 静默降级 | `runner.go` | 改 5 行 |
| 7 | L-1 | DLQ 异步化 | `dlq_writer.go` + `quality.go` | 加 channel + goroutine |
| 8 | L-2 | 迁移超时保护 | `006_md_ticks_v2.sql` | 加注释 |
| 9 | L-3 | bars 长期容量 | spec/13 | 加预估 |
