# 第二轮设计复查 · 残余缺陷（已全部澄清）

> **复查日期**：2026-05-23  
> **复查人**：实施 Agent（DeepSeek v4）  
> **澄清人**：Claude Opus 4.7（2026-05-23）  
> **状态**：✅ 全部 3 项已在 spec/ADR/architecture 修正

## 解决摘要

| ID | 决策 | 落地位置 |
|---|---|---|
| **D-1** | canonical 在 **L3 入口**完成（不是 L2 出口）| `02-overview.md` §8 #1 重写 |
| **D-2** | dedup hash 加入 `bid_volume + ask_volume` | `spec/11` §6 改公式 + 注释决策 |
| **D-3** | SpillWriter 连续失败 ≥3 → 触发 per-broker CircuitBreaker | `spec/11` §9 行为契约 + ADR-0005 §5.4/§5.5 |

---

## D-1 · canonical 规范化位置：三处文档互相矛盾 🔴

### 症状

同一个事实——canonical 在哪个环节完成——在三篇文档中有三种不同说法：

| 文档 | 位置 | 说法 |
|---|---|---|
| `architecture/02-overview.md` | §8 不变量 #1 | "**canonical 在 L2 出口完成**：进入 L3 的 Tick 一定有非空 canonical 字段" |
| `spec/10-mt-adapter.md` | §3 Tick 结构体 | `Canonical string  // 规范化后（如 "BTCUSD"）；**adapter 留空，由 mdgateway 填**` |
| `spec/11-mdgateway.md` | §2.2 HandleTick | `t.Canonical = m.normalizer.Resolve(t.Broker, t.SymbolRaw)` — **在 L3（mdgateway.Manager）执行** |

### 影响

实施者无法确定应该把 `Normalizer` 放在 L2（adapter/mt4 和 adapter/mt5 内）还是 L3（mdgateway/manager.go）。当前代码（`adapter/mt4/gateway.go`）在 adapter 层持有 normalizer 引用，但 spec/10 又说 "adapter 留空"。不统一会导致：

- 实现人按 spec/11 走（L3 做），但不变量检查会报违反
- 实现人按不变量走（L2 做），但 normalizer 需要 PG 查询 + LRU cache —— 把数据库依赖注入 adapter 破坏了 adapter 的"纯翻译"职责

### 建议

**统一为：canonical 在 L3 入口（mdgateway.HandleTick 第一步）完成。** 理由：

1. normalizer 依赖 PG `broker_symbols` 表 + LRU cache，不是 adapter 层应承担的依赖
2. spec/11 HandleTick 的调用顺序（normalizer → quality → dedup → ...）已经是最优编排
3. 只需修改 `02-overview.md` 的不变量措辞：将 "L2 出口完成" 改为 "**L3 入口完成**"

需同步修改的文档：
- `architecture/02-overview.md` §8 不变量 #1
- （可选）`01-vision.md` 准则 4 中 "进入 ant 系统的第一行代码" 的描述（当前已模糊，可不改）

---

## D-2 · Tick 去重哈希键遗漏 volume 字段 🟠

### 症状

`spec/11-mdgateway.md` §6 `Seen()` 的去重哈希：

```
hash = xxhash(ts_unix_ms || bid || ask)
```

但 `mdtick.Tick` 包含 `BidVolume` 和 `AskVolume` 两个字段未参与哈希。

### 场景

同一毫秒内 broker 推送两笔同价位的成交——bid 相同、ask 相同、时间戳相同，但 **volume 不同**（例如第一笔 0.5 lot，第二笔 1.0 lot）。这在活跃品种（EURUSD 欧盘、XAUUSD 美盘）的 1 秒内是真实场景。

按当前哈希，第二条会被 `Seen()` 返回 true 并丢弃。丢失的这条 tick 会轻微影响 bar 聚合的 Volume 字段和 bar 的 OHLC 精度（如果第二笔是同一分钟的最后一笔）。

### BACKLOG 现有立场

`BACKLOG.md` RV-C3 判定 "保留 100-window dedup：成本 ~50ns/tick，false-positive ≈ 0"。该判定成立的**前提**是所有 tick 的 `(ts, bid, ask)` 三元组唯一——但 volume 改变了这一前提。

### 建议

将哈希修改为：

```
hash = xxhash(ts_unix_ms || bid || ask || bid_volume || ask_volume)
```

成本变化：从 3 个字段串接到 5 个字段串接，增量可忽略。但消除了同毫秒同价不同量的误判可能。

需同步修改的文档：
- `spec/11-mdgateway.md` §6 `Seen()` 的 hash 公式
- `BACKLOG.md` RV-C3 的 "false-positive ≈ 0" 表述可保留（加了 volume 后确实 ≈ 0）

---

## D-3 · SpillWriter 失败后无背压传播 🟡

### 症状

`spec/11-mdgateway.md` §9 的行为契约：

> `EnqueueTick` chan 满 → **立即** 写 SpillWriter，不阻塞调用方

但没有规定 SpillWriter **也失败**时的行为。可能的失败场景：

- 磁盘满（`/var/lib/ant/spill` 所在卷空间耗尽）
- 目录权限错误（容器重启后 UID 漂移）
- I/O 错误（云盘短暂只读）

当前设计下，SpillWriter 失败后 tick **静默丢失**——CH writer 通道已满，spill 写不进去，调用方不受阻。这违反了 `01-vision.md` 准则 3 "行情链路必须默认能扛住" 的原则。

### 建议

在 spec/11 中增加 SpillWriter 故障的升级路径：

1. SpillWriter.WriteTick / WriteBar 连续失败 **N 次**（建议 N=3）
2. 通过 error callback 通知 `Manager`
3. Manager 对该 broker 触发 CircuitBreaker 打开（记录 reason="spill_full"）
4. metric `md_tick_dropped_total{reason="spill_failed"}` 递增

这与现有的 CircuitBreaker 机制一致——将 "磁盘不可写" 视作与 "broker 不可达" 同等的故障，触发熔断 + 告警。

需同步修改的文档：
- `spec/11-mdgateway.md` §9 CHWriter 行为契约 + §11 CircuitBreaker 集成点
- （可选）`runbook/mt-incidents.md` 新增一条 "spill 目录满" 的故障条目

---

## 总结

| ID | 严重度 | 需修改文档 |
|---|---|---|
| D-1 canonical 矛盾 | 🔴 阻塞实施 | `02-overview.md` §8 #1（改一行） |
| D-2 dedup 哈希缺 volume | 🟠 低概率误判 | `spec/11-mdgateway.md` §6（改 hash 公式） |
| D-3 spill 无背压 | 🟡 极端故障静默丢数据 | `spec/11-mdgateway.md` §9 + §11（增加升级路径） |
