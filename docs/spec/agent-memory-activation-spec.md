# Agent Memory 激活 spec（评估 + 修复"像没 memory"）

> **背景**：用户问"agent 跟 Claude Code 学的，要一样聪明，但好像没 memory"。审计方评估：**agent 有 memory（ADR-0025 三层，DB 持久化，照 Claude Code MEMORY.md 建的）——不是缺架构，是空的+检索弱+没接 KB**。
> **角色**：审计方出 spec；施工方实现+回填，不自行宣告 ✅，等审计方实测。

---

## 0. 评估结论（审计方实测 2026-08-08）

**有的（架构到位）**：`agent/memory.go` ADR-0025 三层——L1 `domain_knowledge`(平台共享/scope 过滤)、L2 `user_strategy_templates`(用户偏好 CRUD)、L3 `agent_experience`(自动积累+fingerprint 去重，注释明说"MEMORY.md materialized view")。DB 持久化、LoadSessionMemory 启动注入 prompt、retrospect enabled、StoreExperience handler+capability 全在。

**空/弱的根因（实测）**：
| 层 | 实测 | 根因 |
|---|---|---|
| L1 domain_knowledge | **0 行** | **无自动闭环**——需手动/admin 播种，从没播过 |
| L3 agent_experience | **0 行** | **闭环接了但休眠**——`generator_agent.go:282` 生成+回测后存 experience，但**没策略经 agent 生成过**（6 条测试策略是 seed 的）→ 路径没跑 → 0 行。**不是 bug，是 dormant** |
| 检索 | ILIKE 文本 | 无 pgvector 语义检索（注释"can be added later"）|
| KB 联动 | 无 | agent 未经 RAG 用 KB（C3 P3）|

**一句话**：memory 不缺架构，缺**被填满 + 语义检索 + 接 KB**。L3 一旦有 agent 生成（AI 供给播种/真实用户）就自动攒；L1 必须手动播。

## §0.5 🔴 硬约束：MQL 学习的 IP 边界（2026-08-08 用户拍板）

> **平台"越用越聪明"必须建，但不靠挖用户代码。** 本约束约束 T1/T4 及未来一切"学习"能力。

**核心红线**：❌ **不从用户上传的 MQL 源码蒸馏策略逻辑**（哪怕只作平台内部 KB 资产、不重放给用户）。
- **动供给侧命根**：作者选 AlphaForge 的核心理由="最好的策略不敢放别处怕被抄；这里代码不出平台"。作者怕的不止源码被下载，更是 **edge 被平台吸收**。一旦"上传=被学走"被知，最好的策略不再来 → 市场失最有价值供给。
- **trust 是基本盘**（本会话定调）：拿 trust 换 KB 资产 = 拿基本盘换次要物。
- **Anthropic 也不用客户数据训练**（其信任卖点 = 我们的"代码不出平台"）；行业趋势往"不学客户数据"走。

**允许的学习路径（平台变聪明的正道）**：
- ✅ **表现/使用数据**：战绩、衰减(FEAT-5)、需求信号(K3 哪些功能被命中)——群体规律，不碰代码逻辑。**已在做**。
- ✅ **agent 自生成闭环**：agent 自生成→回测→结果→KB(K4)→agent-RAG(C3)→生成更好。**复利走 agent 自家产出，零用户 IP**——平台变聪明的主路径（= T4）。
- ✅ **纯聚合统计**（可选，须透明）：仅群体规律（如"趋势跟踪类占比 60%"），**明告作者"只学群体规律、不复制策略"**。

**禁止**：❌ 隐式/强制从用户 MQL 蒸馏策略逻辑进 KB/agent（含内部资产、含初创期）。❌ 用作者 A 的逻辑经 agent 生成给 B。

**未来若碰用户代码逻辑**：须 **opt-in + 补偿**（作者自愿贡献换分成/曝光，主权在作者），非平台偷学；且有战绩+规模后再谈（初创期 trust 最脆，不赌）。

**对本 spec 的约束**：
- **T1**（播 domain_knowledge）：种子来自**公开交易域知识 / agent 自生成沉淀 / admin 手编**，**禁来自用户上传 MQL 的蒸馏**。
- **T4**（agent↔KB RAG）：KB 的 K4（AI learnings）只来自 **agent 自家生成+结果**，**禁来自用户 MQL 逻辑**。

## 1. 任务

### T1（P0）播种 L1 domain_knowledge
**问题**：L1 空 + 无自动闭环 → agent 无基线交易域知识。
**任务**：admin 播种 domain_knowledge（交易域基线：指标在什么 regime 有效、常见坑、风格分类、风险原则）。可通过 admin RPC 或 seed 脚本。scope 按需（symbol/timeframe 或全局）。
- **对抗证明**：播种后 LoadSessionMemory 注入的 DomainKnowledge 非空；删一条→少一条。

### T2（P0）验证 + 确保 L3 经验闭环真触发
**问题**：`generator_agent.go:282` StoreExperience 在生成+回测后写 experience，但 0 行（没跑过）。需确认它在**活路径**且真写。
**任务**：① 确认 :282 所在函数在生成主路径被调（非死代码）；② 加测试/实测：跑一次 agent 生成+回测 → `agent_experience` 出现一行；③ 注：AI 供给播种会自然触发它（耦合）。
- **对抗证明**：一次成功生成 → agent_experience +1（fingerprint 去重生效：同源再生成不重复+1）。

### T3（P1）pgvector 语义检索
**问题**：`SearchExperiences`/experience recall 用 ILIKE 文本（:214），非语义。Claude Code 的 memory recall 按相关性。
**任务**：`agent_experience` 加 embedding 列（pgvector）；StoreExperience 时生成 embedding；SearchExperiences 用向量相似度（`<->`）替代 ILIKE，按意义召回。
- REUSE：KB 架构的 pgvector 方案（`knowledge-base-architecture.md`）。
- **对抗证明**：语义近但文本不同的两条 experience（如"趋势跟踪"vs"trend following"）能互相召回；ILIKE 召不回。

### T4（P2）Agent ↔ KB RAG 联动（= C3）
**问题**：agent 只有自己经验(ADR-0025)，没用平台累积知识(KB: compat/战绩/需求)。
**任务**：agent 生成前 RAG 检索 KB（K1 避开不支持 / K2 哪些策略赚 / K4 失败模式）→ inject prompt。详见 `knowledge-base-architecture.md` C3 + `kb-p0-hardening-spec`。
- 注：agent-RAG 花 token、漂移降不零（靠诚实管线重验兜底）；确定性 KB(K1)优先。

## 2. 验收（审计方实测）
- **T1**：domain_knowledge 有数据 + LoadSessionMemory 注入非空。
- **T2**：一次生成 → agent_experience +1（活路径确认）。
- **T3**：pgvector 语义召回（跨语言/近义）。
- **T4**：agent 生成 prompt 含 KB 检索结果。

## 3. REUSE（施工方 `bash scripts/cap.sh`）
`MemoryStore`(agent/memory.go)、`StoreExperience`/`SearchExperiences`、`generator_agent.go:282`、pgvector(KB 架构)、`LoadSessionMemory`/`InjectIntoPrompt`、admin memory handlers。

## 4. 完工回填（施工方）
1. `tech-debt-registry.md` 新增 `AGENT-MEM-1/2/3/4` 🟦→✅ + 对抗证明。
2. `handover-audit-plan.md` 变更日志。
3. 不自行宣告 ✅——等审计方实测。

---

> **审计方注**：memory 架构不重写（照 Claude Code 建的，到位）。T1(播 L1)+T2(验证 L3 触发) 是 P0——让 memory 真有内容；T2 会随 AI 供给播种自然激活。T3(pgvector)+T4(KB 联动) 是 P1/P2，提检索质量+平台知识反哺。**关键认知**：L3 闭环没坏，是 dormant——一跑生成就攒；L1 才是真缺口（无自动闭环，必须播）。
