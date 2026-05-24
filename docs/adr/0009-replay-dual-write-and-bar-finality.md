# ADR-0009 · Spill Replay 双写 NATS + Bar 不可变性 + 历史回填

- **状态**：Accepted
- **日期**：2026-05-24
- **决策者**：架构组
- **关联 spec**：`docs/spec/11-mdgateway.md` §10 / `docs/spec/18-backfiller.md`（新建）
- **关联 ADR**：ADR-0005（CircuitBreaker + Spill）、ADR-0008（存储层去重）

## 1. 背景

M7 现状暴露三个数据完整性缺口：

### 1.1 Spill replay 仅补 CH 不补 NATS（H-2）

`spill_replay.go` 启动时把 `*.jsonl` 重放到 `CHWriter`，于是 `md_ticks/md_bars` 数据完整。但 NATS subject `md.tick.>` `md.bar.>` 在 CH 中断窗口内**没有人 publish**，下游 `factorsvc` 永久缺该窗口因子值。回测/实盘信号出现"幻象空洞"。

### 1.2 Bar 重启重算可能产生不一致 OHLC（S-2）

`bar_aggregator` 重启后 `inProgress` map 清空。如果 spill replay 把"上次跨 bar 边界但还没 emit 的 tick"重新喂进 aggregator，OHLC 可能与首次计算结果不同（dedup ring 状态不同 / tick 顺序受文件读顺序影响）。`md_bars ReplacingMergeTree(close_ts_unix_ms)` 的"后写覆盖前写"会用错误数据覆盖正确数据。

### 1.3 历史回填路径完全缺失（M-1）

任何"账户接入前已发生""单 broker 短暂离线""新订阅 symbol"的历史 K 线，CH 都是空的。但 mtapi 提供 `GetPriceHistory` RPC，是免费的回填来源。无此能力则：
- 用户加新品种第一天回测无数据可用
- chaos 期间（broker 离线 1h）这段窗口永久为空
- 跨 broker 切换时新 broker 无历史

## 2. 决策

### 2.1 Spill Replay 双写

`SpillReplay.Run` 改造：

```go
// 旧：只调 CHWriter
chWriter.EnqueueTick(t)

// 新：双写 + 标记
t.IsReplay = true
publisher.PublishTick(t)   // factorsvc 收到 → 重新算因子
chWriter.EnqueueTick(t)
```

`mdtick.Tick` 加字段：
```go
type Tick struct {
    ...
    IsReplay bool  // true = 来自 spill replay 或 backfiller
}
```

下游约定：
- `factorsvc` 收到 `IsReplay=true` 的 bar **正常执行因子计算**，但写 CH `factor_values` 时不发 `md.factor.>` NATS（避免 quantengine 在 replay 期间发"过期信号"）
- `quantengine` 直接订阅时按 `IsReplay==false` 过滤

### 2.2 Bar 不可变性（finality contract）

新增 invariant：**`md_bars` 中已存在的 (broker, canonical, period, close_ts_unix_ms) 不可被 replay/backfill 覆盖**。

实现：`bar_aggregator.go` 启动时执行：
```sql
SELECT broker, canonical, period, MAX(close_ts_unix_ms)
FROM md_bars
GROUP BY broker, canonical, period
```
缓存为 `finalizedBars map[key]int64`。replay/backfill 写 bar 前检查：若 `bar.close_ts_unix_ms <= finalizedBars[key]` 则**跳过**，不入 CH，metric `md_bar_skipped_finalized_total++`。

实时路径不受影响（实时 emit 的 bar 同时更新 finalizedBars[key]）。

### 2.3 历史回填器（backfiller）

新增 `backend/internal/mdgateway/backfiller/`，详细规范见 `docs/spec/18-backfiller.md`。简述：

- 启动时对每 (account, canonical, period in {1m, 5m, 1h, 1d})：
  1. CH 查 `MAX(close_ts_unix_ms)` 作为 `from`
  2. 调 `mtapi.GetPriceHistory(from, now)` 获取历史 bar
  3. 走 `bar_aggregator.IngestBar(bar, IsReplay=true)` → finality 检查 → CH + NATS
- 新订阅 symbol 时同步触发 30 天回填（一次性）
- 限速：每账户每分钟最多 6 次 `GetPriceHistory`（mtapi 约定）

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决 |
|---|---|---|---|
| Replay 仅补 CH（现状） | 简单 | factorsvc 永久缺数据 | 已识别为 bug |
| factorsvc 自己从 CH 回灌 | 解耦 | 需要写 backfiller 子模块 + 状态机；replay 链路复杂化 | 大于 dual-write 改造 |
| Bar 允许覆盖（不加 finality） | 简单 | 重启后 OHLC 变化，回测不可重现 | 违反"数据基础不可变"原则 |
| Backfill 用 CH ETL（外部脚本） | 不动核心代码 | 不能用 mtapi 历史数据；增量逻辑需另写 | 数据源不全 |
| **本决策：Replay 双写 + Bar finality + Backfiller** | 一次性补齐三个缺口；下游零侵入 | 代码量增加 ~250 LOC | ✓ |

## 4. 后果

- **正面**：factorsvc 数据完整；bar OHLC 永久确定；新订阅/历史窗口可回填
- **负面**：spill_replay 期间 NATS 突发流量（按 spill 文件大小，通常 < 1min 流量峰），`md.bar.>` 订阅者必须有 backpressure（已在 spec/11 §13.6 覆盖）
- **中性**：`Tick`/`Bar` DTO 增加 `IsReplay` 字段，proto / json schema 需 minor bump

## 5. 实施约束

1. `mdtick.Tick` `mdtick.Bar` 加 `IsReplay bool`（默认 false）
2. `spill_writer.go` jsonl 序列化时**不写** `is_replay` 字段（写盘永远是当时的实时状态），replay 时由 `spill_replay.go` 设置 `IsReplay=true`
3. `bar_aggregator.go` 启动构造时接受 `finalizedBars map[string]int64` 注入
4. `publisher.go` 不区分 replay/realtime，统一发；下游按 header 字段判断
5. NATS msg header `X-Ant-Replay: 1` 同步设置（便于运维抓包诊断）
6. `factorsvc/subscriber.go` 收到 replay bar 时正常计算，但写 CH 后**不**调 `publisher.PublishFactor`
7. backfiller 配额：每账户每分钟 6 次 `GetPriceHistory`，由 `golang.org/x/time/rate` 限流

## 6. 验证方式

```bash
# 1. Tick/Bar DTO 含 IsReplay
grep -q 'IsReplay\s*bool' backend/internal/mdgateway/adapter/mdtick/mdtick.go

# 2. spill_replay 发 NATS（注入测试）
go test -tags=integration ./internal/mdgateway/ -run TestSpillReplayDualWrite -v
# 测试断言：replay 1000 tick → NATS 收到 1000 个 md.tick.* + CH 1000 行

# 3. Bar finality：重启后不覆盖已 emit bar
go test -tags=integration ./internal/mdgateway/ -run TestBarFinality -v
# 测试断言：CH 已有 bar (close_ts=T)，replay 同 close_ts 的不同 OHLC → CH 仍是原值，metric md_bar_skipped_finalized_total++

# 4. Backfiller：CH 缺 1h bar 自动补齐
go test -tags=integration ./internal/mdgateway/backfiller/ -run TestBackfillGap -v

# 5. NATS header 标记
nats sub 'md.bar.>' --headers-only | grep -q 'X-Ant-Replay'

# 6. Replay 期间 factorsvc 不发 NATS md.factor.>
go test -tags=integration ./internal/factorsvc/ -run TestReplayNoFactorPublish -v
```
