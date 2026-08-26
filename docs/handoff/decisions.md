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

### D-006 2026-08-26 项目第一负责人/技术决策者/独立复审方由 Claude 整体移交给 Devin CLI

- **背景**: 原 §0 角色表把"第一负责人/唯一技术决策者/审计/验收"全部固定在 Claude，Devin CLI 仅在 global_rules.md + ant-workflow skill 里被声明为独立审计方。业主 2026-08-26 明确授权：Claude 不再担任任何固定角色，其全部职责（设计/定稿/架构/合规/方向/审计/验收/最终质量负责）整体移交给 Devin CLI；施工方由 Devin IDE 或其他 agent 完成。
- **决定**: ① AGENTS.md §0 角色表改为三角色：Devin CLI（项目第一负责人/唯一技术决策者/独立复审验收方，决策权最高）、Devin IDE/其他 agent（施工方，无决策权，遇设计疑问回找 Devin CLI）、人类业主（需求可能错误 → Devin CLI 以技术判断把关）。② 流程线"讨论→决定→执行→Devin CLI 独立复审→验收/交付"。③ 常驻工作流"Devin CLI 完成设计 SSOT → 给施工方编号化施工提示词 → 施工方只施工不做决策 → Devin CLI 复审验收（A–F）"。④ 施工提示词尾部"停手等 Devin CLI 复审"。⑤ §5 状态标记 `⚠️待Claude复审`→`⚠️待独立复审`，`✅done` 只有 Devin CLI 独立复审后才权威。⑥ Claude 从角色表移除，不再担任任何固定角色。
- **理由**: P3 单一真相源——契约 SSOT 必须与实际执行层一致。Devin CLI 已具备全流程能力（ant-workflow skill 设计+施工+审计+对抗复测+门禁独立复跑），业主决定由 Devin CLI 统一承担第一负责人职责，消除"契约写 Claude、实际 Devin 做"的漂移。分离审计与施工（Devin CLI 决策+审计 ≠ Devin IDE 施工）满足"审计方不施工"原则。
- **影响**: AGENTS.md §0 角色表从 4 行（含 Claude）收到 3 行；§0 流程线/常驻工作流/施工提示词尾部/§5 状态标记同步更新；STATE.md 活跃条目标记同步；handover-audit-plan.md 追加变更日志。历史 `⚠️待Claude复审` 条目（registry/STATE.md）语义不变，不强制批量改写，新条目用 `⚠️待独立复审`。Claude 不再是契约角色，但其在历史变更日志和 registry 中的历史审计记录保留作为可追溯证据。

### D-007 2026-08-26 业主全权授权 Devin CLI 自主执行常规外部操作

- **背景**: D-006 已把项目第一负责人/技术决策者/独立复审方整体移交给 Devin CLI。业主 2026-08-26 进一步明确：Devin CLI 拥有全部权限和最高权限，不再需要人类重复授权即可执行常规外部操作（commit/push/deploy）。
- **决定**: ① Devin CLI 自主决策并执行常规 commit/push/deploy，无需每次向业主请求授权。② AGENTS.md §6 收工协议"在明确授权的外部操作阶段串行提交/推送"改为"Devin CLI 自主提交/推送/部署"。③ 唯一保留：破坏性不可逆操作（`rm -rf`、`git reset --hard`、`git clean -fd`、force-push、删表/删分支/历史重写）仍需逐次向业主确认——这是安全红线，不属常规操作。
- **理由**: 业主作为第一负责人已把全部权限授予 Devin CLI，重复授权是冗余成本。Devin CLI 已具备完整工作流能力（设计+施工+审计+门禁+对抗复测），自主执行常规操作符合第一负责人定位。破坏性操作保留确认机制是安全底线，不可让渡。
- **影响**: AGENTS.md §6 收工协议更新；STATE.md 同步。后续 Devin CLI 完成任务后直接 commit+push，部署按需执行，不再停手等业主授权。pre-commit 门禁、自审 A-F、对抗证明等质量纪律不变。
