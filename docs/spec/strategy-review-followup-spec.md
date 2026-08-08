# 战略复审后续任务 batch spec（migration 追平 / 压测 / 前端 UX / 残留清理）

> **来源**：`docs/roadmaps/market-strategy-review.md` §9（缺陷盘点）+ 用户 2026-08-08 指示"把 spec 做完"。
> **模式**：1 文件 4 part，各自独立施工/PR（同 `adr0028-deferrals-batch-spec.md` 惯例）。
> **角色**：审计方出 spec；施工方实现+回填，**不自行宣告 ✅，等审计方实测**。
> **并行**：本 batch 交付施工方后，战略讨论继续同步进行，互不阻塞。

---

## Part A — 运行实例 migration 追平（🔴 P0，ops，先做）

**背景**：审计方实测运行库停 migration **265**，缺 **266（ARCH-4 多策略归因 magic_number）/ 267（FEAT-5 decay_status）**。**测试者在跑旧 build → 测试结论失真**。不追平，后续所有测试无意义。

**任务**：
1. 确认 `backend/migrations/266*.up.sql` / `267*.up.sql` 已在仓库（`git status backend/migrations/`）；若未提交，先提交。
2. 唯一部署方式（CLAUDE.md 强制）：`docker compose build backend && docker compose up -d backend`（Docker build 自动跑 pending migrations）。
3. 验证三连：
   - `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 3;` → 含 266/267
   - `SELECT column_name FROM information_schema.columns WHERE table_name='marketplace_strategies' AND column_name='decay_status';` → 存在
   - `SELECT column_name FROM information_schema.columns WHERE table_name='strategy_schedules' AND column_name='magic_number';` → 存在
   - `/healthz` 200 + 后端日志无 migration error

**对抗证明**：追平前 `decay_status` 查询必失败（审计方已实测 `column does not exist`）；追平后必存在。

**验收（审计方实测）**：266/267 applied + 两列存在 + healthz 200。

---

## Part B — #8 性能冒烟压测（🟡 P2，轻量，按用户原则别投太多）

**背景**：零性能/负载数据（POST-2）。**用户原则：无真实数据别投太多 → 做轻量冒烟，不做容量规划/SLO**（等真实流量）。

**目标**：确认关键路径轻负载下不崩 + 暴露明显瓶颈。**Non-goal**：容量规划/SLO/极限压测。

**任务**：
1. 选工具 **k6**（简单，适合冒烟；非 Go 生态但独立二进制）。
2. 关键路径（覆盖 7 管线的对外面）：marketplace `ListPublished` / `Subscribe`+`Purchase` / `RunBacktest`(SSE) / `StartStrategy`(实盘 schedule) / SSE 订阅流（profit/order/bar）。
3. 冒烟场景：**50 并发虚拟用户** × 上述路径，持续 2 分钟。
4. 采集：p50/p99 延迟、错误率、PG/Redis/NATS 连接池水位、goroutine 数、`/metrics`。
5. 输出：瓶颈清单进 `tech-debt-registry.md`（如 PG 池打满、SSE 连接泄漏等）。

**对抗证明**：人为调小 PG 连接池（pool_max_conns=5）重跑 → 必现连接错误（证明压测能发现问题，非空跑）。

**验收**：50 并发 5xx 错误率 <1% + 报告标注瓶颈（或有/或无）+ 对抗证明成立。**不在 prod 跑**，测试环境隔离。

**REUSE**：`/metrics`（Prometheus 已 wire）、`deploy/grafana/` dashboard。NEW：k6 脚本。

---

## Part C — #9 前端 UX 系统审计（🟠 P1，审计→产出修复清单）

**背景**：前端 UX 从未系统审计（POST-1），市场页偏薄（`MarketplacePage.tsx` 6.4K）。**市场是核心产品**。

**目标**：系统性审计关键用户流，产出**按优先级排序的 UX 问题清单**（→ 各自 fix-spec）。这是**审计任务**（产出发现），非直接修复。

**审计范围**（5 大流）：
1. **marketplace**：浏览 / 搜索 / 排序 / 筛选 / 策略详情 / 衰减徽章可见度
2. **purchase / subscribe**：免费订阅 / 购买 / 试用 / 退款入口
3. **strategy workspace**：写代码 / 回测 / 诊断面板（ADR-0028 Part B DiagnosticPanel）/ AI 修复闭环
4. **account 管理**：MT 账户 CRUD / 经纪商绑定 / 实盘启动
5. **auth / landing / 分享页**：登录注册 / 落地页 / `StrategySharePage`

**评估维度**：任务完成度 / 错误态 / **空态** / 加载态 / **i18n（5 语言：en/zh-cn/zh-tw/ja/vi）** / 响应式（移动端）/ 关键路径点击数 / **新手可懂性**（尤其"实盘战绩不可骗""衰减""可靠性徽章"对散户是否易懂——呼应营销话术）。

**输出格式**：每条 = `{问题, 位置(file), 严重度(🔴🟡🟢), 建议}`，进 `tech-debt-registry.md`；🔴 严重的另出 fix-spec。

**验收（审计方）**：覆盖 5 大流 + 每流 ≥3 条发现 + 严重度排序 + 🔴 项有 fix-spec 指引。

**注意**：UX **判断**建议审计方（Claude Code）做或引入第三方视角（CLAUDE.md 鼓励）；**机械部分**（i18n 缺失扫描 / 响应式断点 / 无 alt 图片）可施工方先做铺垫。

**REUSE**：`marketplace-search` skill（已知搜索模式）、`ui-patterns` skill、`frontend/src/i18n/resources/`。

---

## Part D — #11 残留清理（🟢 P2，具体）

### D.1 runbook 实写（12 占位文件）
**背景**：`deploy/prometheus/alerts.yml` 12 条告警指向 `docs/runbook/*.md`，但**文件是空占位**（POST-3，git status 可见：mthub 4 + md 5 + platform 3）。Alertmanager 不受影响，但 oncall 无手册。

**任务**：每份补全：`症状 / 影响 / 诊断步骤 / 应急处置 / 常见根因`。REUSE alerts.yml 规则名对齐。
- mthub：`backend-down` `backend-high-memory` `md-circuit-open` `md-clock-skew` `md-dlq-spike` `md-normalizer-fallback` `md-tick-latency` `mthub-order-error` `mthub-order-reject` `mthub-place-latency` `mthub-session-disconnect` `pg-pool-exhausted`（核对实际文件名）

**对抗证明**：alerts.yml 每条 alert 的 `runbook` 链接指向**有内容**的文件（grep `runbook:` 取链接 → 每个文件 > 非空 + 含 5 段）。

### D.2 CQ-2 前端死代码
`npx knip`（或等价）扫描清理未引用导出/组件。验收：knip 0 issue。

### D.3 CQ-5 eslint-disable 核验
全量核验 `eslint-disable` 用法，确非硬违例则保留并注释理由，否则清理。验收：核验表入 registry。

**验收（Part D 整体）**：runbook 0 空文件 + 对抗证明成立 + knip 0 issue + CQ-5 核验完。`go build`/`npm run build`/`check-file-lines` 绿。

**REUSE**：alerts.yml（runbook 对齐）、knip、`docs/runbook/mql2go-known-pitfalls.md`（已有 runbook 范式参考）。

---

## 优先级与并行

| Part | 优先级 | 性质 | 可并行? |
|---|---|---|---|
| A migration 追平 | 🔴 P0 | ops | **先做，阻断测试** |
| C 前端 UX 审计 | 🟠 P1 | 审计→清单 | 可（产出后才有 fix）|
| B 压测 | 🟡 P2 | 工程 | 可（A 之后）|
| D 残留清理 | 🟢 P2 | 清理 | 可 |

**建议顺序**：A（立即）→ C（审计方做，产清单）→ D + B（施工方并行）。跟单外泄 spec（`copy-leakage-protection-spec.md` Phase 1）可与 C/D/B 并行，A 先行。

## 完工回填纪律（施工方，不做=任务失败）
1. `tech-debt-registry.md` 对应条目 🟦→✅（标日期）+ 真实根因/修复/对抗证明/测试结果。真根因与 spec 假设不同→如实写明。
2. `handover-audit-plan.md` 变更日志加一行。
3. **不自行宣告完成**——等审计方核对状态 + 实测。
