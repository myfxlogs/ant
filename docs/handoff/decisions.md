# 决策记录（D#）

> 日常运营决策记录。架构决策见 `docs/adr/`（ADR 0001+），本文件记录非架构级决策。
> ≤ 450 行。超限先滚出到 `docs/handoff/LOG.md`。

## 编号规则

- `D-<seq>` 递增编号，不回收。
- 架构级决策走 `docs/adr/`（ADR-XXXX），本文件只放日常运营决策。
- 每条决策：背景 → 决定 → 理由 → 影响。

## 决策记录

### D-001 2026-08-26 AGENTS.md 拆分瘦身

- **背景**: AGENTS.md 42KB（291 行），含技术约束+pitfalls+角色定位，远超 20KB T0 预算。agent 每次开工读全文 token 开销大。
- **决定**: 拆分为 ≤20KB 契约（AGENTS.md）+ T1 技术约束（docs/constraints.md）+ T1 坑库（docs/pitfalls.md）+ T1 项目定位（docs/项目定位.md）。
- **理由**: P2 成本不变量——T0 必须轻量，agent 每次开工只读契约+STATE；技术约束/pitfalls 按需读取。
- **影响**: AGENTS.md 从 42KB 降到 ~6KB；技术约束和坑库内容完整保留在 T1 文档中。

### D-002 2026-08-26 引入 STATE.md + decisions.md

- **背景**: 原仓库无独立 STATE.md/decisions.md，状态 SSOT 只有 tech-debt-registry.md + handover-audit-plan.md。registry 1293 行，agent 开工读全量 token 开销大。
- **决定**: 引入 STATE.md（轻量交接负载 ≤20KB，指向 registry）+ decisions.md（日常 D# 决策）。registry 保留做技术债务明细（不限行数）。
- **理由**: 分离"当前在哪"（STATE.md ≤20KB）与"所有债务"（registry 不限）。agent 开工读 STATE.md 快速定位，按需读 registry 明细。
- **影响**: STATE.md 只放当前活跃条目指针；registry 保持原职责不变；decisions.md 补日常决策缺口（ADR 已覆盖架构决策）。

### D-003 2026-08-26 引入经验库三件套

- **背景**: 原坑库内容散落在 AGENTS.md pitfalls 章节 + docs/runbook/，无结构化索引，agent 难以按症状检索。
- **决定**: 引入经验库三件套（索引.md + 条目/ + 待归纳.md）。现有 pitfalls 迁到 docs/pitfalls.md（T1），经验库做跨会话经验沉淀。
- **理由**: P1 通道不变量——经验必须存 git 跟踪纯文本文档，不靠私有记忆。结构化索引让 agent 按症状/报错快速检索。
- **影响**: docs/pitfalls.md 保留已确认的静默失败模式（项目特定）；经验库做更通用的经验沉淀（可跨项目复用）。

### D-004 2026-08-26 pre-commit hook 扩展

- **背景**: 原 pre-commit 只检查文档规则（禁删 open 条目/变更日志）。ai-collab-contract 模板要求加 STATE.md 必更新 + 文档预算门禁。
- **决定**: 扩展 pre-commit hook：① 代码变更时 STATE.md 必更新 ② AGENTS.md/STATE.md ≤20KB ③ T1 文档 ≤450 行 ④ 保留原有文档规则检查。
- **理由**: P4 可检查性——无法检查的规则必被违反。STATE.md 必更新是收工协议的核心，必须机器强制。
- **影响**: pre-commit hook 新增 3 项检查；原有文档规则检查保留。

### D-005 2026-08-26 CLAUDE.md / .windsurfrules 改为入口壳

- **背景**: CLAUDE.md 22KB（265 行）含完整技术约束+pitfalls，与 AGENTS.md 大量重叠。.windsurfrules 5.6KB 也有独立技术约束。
- **决定**: CLAUDE.md / .windsurfrules 改为入口壳（`@AGENTS.md`），技术约束统一在 docs/constraints.md，坑库统一在 docs/pitfalls.md。
- **理由**: P3 单一真相源——一事实两份必有一错。CLAUDE.md 和 AGENTS.md 重叠内容会导致漂移。
- **影响**: CLAUDE.md 从 22KB 降到入口壳；.windsurfrules 从 5.6KB 降到入口壳；技术约束单一真相源在 docs/constraints.md。
