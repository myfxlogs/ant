# M10 文档审核 · 需澄清项

> **写给 Claude/DeepSeek**：以下是在 M10 文档审核中发现的不一致、陈旧引用和需要修正的问题。请在 M10.1-1 开工前处理 P0 三项，P1 可在实施过程中逐项修正。
>
> **生成日期**：2026-05-24

---

## P0 · 必须修正（阻塞实施）

### P0-1 spec/11 §9 CHWriter 配置值与 ADR-0011 不一致

**位置**：`docs/spec/11-mdgateway.md` 第 368–373 行

**问题**：spec/11 文件头（第 7–11 行）已声明 M10 强化叠加（含 ADR-0011 容量调优），但正文 §9 `CHWriterConfig` 结构体注释仍是 M7 旧值：

```go
// 当前（旧值，与 ADR-0011 矛盾）
FlushInterval time.Duration  // 默认 1s
MaxBatchSize  int            // 默认 1000
QueueSize     int            // 默认 5000；满则走 spill
```

**ADR-0011 §2.1 要求的新值**：

| 参数 | 旧 | 新 |
|---|---|---|
| `QueueSize` | 5000 | 50000 |
| `MaxBatchSize` | 1000 | 10000 |
| `FlushInterval` | 1s | 500ms |

**修复**：将 §9 的注释更新为新值。同步更新同段落的 INSERT 模板（第 406 行 `INSERT INTO md_ticks` → `INSERT INTO md_ticks_buffer`，对应 ADR-0011 的 Buffer engine 要求）。

---

### P0-2 spec/11 §9 INSERT 目标表名未更新

**位置**：`docs/spec/11-mdgateway.md` 第 404–411 行

**问题**：INSERT 模板写的是：

```sql
INSERT INTO md_ticks (
    user_id, account_id, broker, symbol_raw, canonical,
    ts_unix_ms, arrived_unix_ms, bid, ask, bid_volume, ask_volume
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
```

**ADR-0011 + spec/13 §2.7 要求**：INSERT 目标应为 `md_ticks_buffer`（Buffer engine 表），CH 内部再异步 flush 到 `md_ticks`。

**修复**：将 `md_ticks` 改为 `md_ticks_buffer`。bar 的 INSERT 同理改为 `md_bars_buffer`。

---

### P0-3 docs/README.md 目录索引严重过时

**位置**：`docs/README.md` "目录结构" 段

**问题**：

| 区域 | README 列出 | 实际存在 | 缺失 |
|---|---|---|---|
| adr/ | 0001–0005 | 0001–0011 | **0006–0011 全部缺失** |
| spec/ | 10–16 | 10–20 | **17, 18, 19, 20 全部缺失** |
| plan/ | ROADMAP.md, BACKLOG.md | + M10-DEEPSEEK-PROMPT.md | 缺少 PROMT 文件 |
| handover/ | 未列出 | M7 verify logs, M7 closure | 缺少说明 |

此外，"阅读路径"表格中"决策审计"行的指引应更新（ADR 已从 5 篇增至 11 篇）。

**修复**：更新目录结构、阅读路径表格、ADR 编号范围。

---

## P1 · 建议修正（不阻塞，但影响实施准确性）

### P1-1 docs/adr/README.md 缺少 0008–0011

**位置**：`docs/adr/README.md`

**问题**：ADR 索引表只列到 0007（M7-M9 回顾），0008–0011 四篇 M10 ADR 未录入。

**修复**：在索引表追加：

```
| 0008 | 存储层去重键对齐 + 时间轴纪律 | Accepted |
| 0009 | Spill Replay 双写 + Bar 不可变性 + 历史回填 | Accepted |
| 0010 | SLO + Alert + DLQ + Trace 框架 | Accepted |
| 0011 | 容量调优 + Vault 轮换 + Normalizer 缓存失效 | Accepted |
```

---

### P1-2 spec/13 §2.1/§2.2 旧版 DDL 缺少"将被替换"标注

**位置**：`docs/spec/13-clickhouse-schema.md` §2.1（`001_md_ticks.sql`）和 §2.2（`002_md_bars.sql`）

**问题**：这两段展示的是 M7 的原始 DDL（ORDER BY 不含量字段，TTL 用 `ts_unix_ms`）。§2.8 给出了 v2 schema，但 §2.1/§2.2 没有任何"此 DDL 已被 006/007 迁移替代"的说明。读者可能误认为当前生产表仍是此 schema。

**修复**：在 §2.1 和 §2.2 的 DDL 代码块上方各加一行注释：

```markdown
> ⚠️ **M10 更新**：此 DDL 已被 `006_md_ticks_v2.sql` / `007_md_bars_v2.sql` 通过 EXCHANGE TABLES 替换，当前生产 schema 见 §2.8。
> 本段保留作为初始创建参考。
```

---

### P1-3 ADR-0010 §2.3 "M8 计划提前到 M10"措辞不准确

**位置**：`docs/adr/0010-slo-alert-dlq-trace.md` §2.3

**问题**：原文写"数据基础引入 OTel（M8 计划提前到 M10）"。但 BACKLOG.md 显示 M8.4-4（trace_id 全链路）状态为"未实施"。OTel 不是"提前"，是 M8 遗留未完成、M10 承接。

**修复**：改为：

> 数据基础引入 OTel（M8 遗留项，M10 承接）

---

### P1-4 PG migration 编号从 064 跳到 102 需要说明

**位置**：ADR-0011 §5 引用 `migrations/102_broker_symbols_notify.up.sql`

**问题**：项目当前 PG migration 最高为 064（`064_agent_job_experiment_assets.up.sql`）。从 064 跳到 102 是合理的技术选择（为 M8/M9 的业务迁移预留 065–101），但没有在任何文档中说明。

**修复**：在 ADR-0011 §5 或 ROADMAP M10 段的注释中加一句：

> PG migration 编号从 102 起（065–101 预留给 M8/M9 业务迁移），与 CH migration（chmigrate/ 001–009）独立编号。

---

### P1-5 spec/20 引用 `rules.yml`，spec/15 引用 `alerts.yml`，关系不清

**位置**：`docs/spec/20-slo.md` §3（recording rules 放 `deploy/prometheus/rules.yml`）vs `docs/spec/15-observability.md` §6（alert rules 放 `deploy/prometheus/alerts.yml`）

**问题**：两个文件名的关系未说明。合理的解释是 recording rules 和 alert rules 分文件管理，但文档没有澄清。

**修复**：在 spec/20 §3 的 recording rule YAML 块前加一句：

> recording rules 存入 `deploy/prometheus/rules.yml`（与 alert rules 文件 `alerts.yml` 分文件管理，见 spec/15 §6）。

---

### P1-6 ROADMAP M10.4-2 负载测试验收命令的参数不完整

**位置**：`docs/plan/ROADMAP.md` M10.4-2 卡片

**问题**：验收命令是：

```bash
go test -tags=loadtest -timeout 15m -run Test100AccountsNoSpill ./tests/loadtest/...
```

但 ADR-0011 §6 的验证命令更完整，包含了"验证 spill writes = 0 且 P99 < 500ms"的断言逻辑。ROADMAP 卡片的验收命令应与 ADR 对齐。

**修复**：在 ROADMAP 卡片验收命令中补充预期的断言检查（或引用 ADR-0011 §6 的完整命令）。

---

## 汇总：修复优先级

| 顺序 | 问题 | 涉及文件 | 工作量 |
|---|---|---|---|
| 1 | P0-1 spec/11 CHWriter 配置旧值 | `docs/spec/11-mdgateway.md` | 改 3 行注释 |
| 2 | P0-2 spec/11 INSERT 目标表名 | `docs/spec/11-mdgateway.md` | 改 2 处 |
| 3 | P0-3 README 索引更新 | `docs/README.md` | 追加约 10 行 |
| 4 | P1-1 ADR 索引更新 | `docs/adr/README.md` | 追加 4 行 |
| 5 | P1-2 spec/13 旧 DDL 标注 | `docs/spec/13-clickhouse-schema.md` | 加 2 行注释 |
| 6 | P1-3 ADR-0010 "提前"措辞 | `docs/adr/0010-slo-alert-dlq-trace.md` | 改 1 个词 |
| 7 | P1-4 migration 编号说明 | `docs/adr/0011-capacity-vault-cache-hardening.md` | 加 1 行注释 |
| 8 | P1-5 rules.yml vs alerts.yml | `docs/spec/20-slo.md` | 加 1 行说明 |
| 9 | P1-6 负载测试验收对齐 | `docs/plan/ROADMAP.md` | 改 1 行命令 |

---

## 修复回执（2026-05-24，由 Cascade 完成）

| 编号 | 状态 | 文件 | 提交 |
|---|---|---|---|
| P0-1 | ✅ | `docs/spec/11-mdgateway.md` §9 CHWriterConfig 三个默认值更新为 ADR-0011 §2.1 新值 | 见后续 commit |
| P0-2 | ✅ | `docs/spec/11-mdgateway.md` §9 INSERT 模板改 `md_ticks_buffer`，加 `is_replay` 列；bar 同理 | 同上 |
| P0-3 | ✅ | `docs/README.md` 阅读路径表 + 目录结构 + ADR 编号差异表 全部刷新到 11 ADR / 20 spec | 同上 |
| P1-1 | ✅ | `docs/adr/README.md` 索引追加 0008–0011 | 同上 |
| P1-2 | ✅ | `docs/spec/13-clickhouse-schema.md` §2.1 §2.2 加 ⚠️ M10 替换标注 | 同上 |
| P1-3 | ✅ | `docs/adr/0010-slo-alert-dlq-trace.md` §2.3 措辞改"M8 遗留项 M8.4-4，M10 承接" | 同上 |
| P1-4 | ✅ | `docs/adr/0011-capacity-vault-cache-hardening.md` §5 加 PG migration 编号说明 | 同上 |
| P1-5 | ✅ | `docs/spec/20-slo.md` §2.1 加 rules.yml vs alerts.yml 关系说明 | 同上 |
| P1-6 | ✅ | `docs/plan/ROADMAP.md` M10.4-2 验收命令引用 ADR-0011 §6 两条断言 | 同上 |

## 前端现状澄清（连带处理）

用户确认：**旧 frontend 已基本全部删除**，与"路线 A（全量重写）"无差别。连带修复：

| 改动 | 文件 |
|---|---|
| ⚠️ 前端现状警告块（M10 = 纯后端，禁止 UI 验收） | `docs/plan/ROADMAP.md` §M10 头部 |
| 当前阶段速览：M10 + 前端待 M11 | `AGENT.md` §1 §14 |
| ADR-0007 §7 补丁段：M7.5-8 / M7.2-8 / M7.2-3 状态修订 | `docs/adr/0007-post-m7-retrospective.md` |
| M7.5-8 `☑ → ⊘` | `docs/plan/ROADMAP.md` |
| M7.2-8 `☑ → ⊘` | `docs/plan/ROADMAP.md` |
| Cascade 长期记忆：禁止假定前端可复用 | memory store |

**待立项**：M11 前端从零重建（待 M10 关闭后单独 ADR 0012+）。
