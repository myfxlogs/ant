# MQL 诚实管线收尾 spec（5 项优化 · 从"能跑诚实"到"复利+耐久+安全"）

> **依据**：KB-P0 之后审计方识别的 5 项收尾/优化（2026-08-08）。KB-P0 建基质；本 spec 把管线从"能用"推到"长期复利且不悄悄退化 + 安全对称"。
> **关联**：`mql-honesty-audit-spec.md` / `mql-decision-loop-spec.md`(MQL-LOOP) / `kb-p0-consolidate-spec.md` / `knowledge-base-architecture.md`。
> **角色**：审计方出 spec；施工方实现+回填，不自行宣告 ✅，等审计方实测。

---

## 优先级 + 依赖

| # | 项 | 优先级 | 依赖 |
|---|---|---|---|
| 1 | T5 实盘门控（安全对称）| 🔴 P0 | 独立（用 checkUnreliableCoverage 逻辑）|
| 2 | CI 诚实回归集（耐久）| 🟠 P1 | 独立（用 honesty 语料）|
| 3 | K3 需求捕获（LEARN）| 🟠 P1 | kb_demand_signal 表（KB-P0 或先建独立表后并）|
| 4 | 作者覆盖率分数（透明）| 🟡 P2 | 独立（用 CoverageResult）|
| 5 | C1 复利实证回归（防退化）| 🟡 P2 | KB-P0（RecordFix）|

---

## 1. 🔴 T5 实盘门控（堵 fatal 盲区上真账户）

**问题**：发布门控堵 fatal 盲区（MQL-LOOP-1），但**实盘没堵**——用户在 workspace 自写策略经 `StartStrategy` 直接跑，带 fatal 盲区(如 iCustom)的策略能在**真账户上不可靠地跑**。安全不对称。

**任务**：`StartStrategy`/`RunLiveStrategy`（共享 chokepoint）启动前校验 fatal 盲区 → 有则拒（与发布门控一致）。策略须有近期 SUCCEEDED 回测且 `IsReliable=true`（无 fatal 盲区）才能 live；无回测则先要求回测，或启动期 compile+coverage 分析。
- REUSE：`checkUnreliableCoverage` 逻辑（MQL-LOOP-1）、`checkBoundAccount` chokepoint 注入模式（LEAKAGE-1）。
- **对抗证明**：iCustom fatal 盲区策略 `StartStrategy` 必拒；移除校验→能跑→红。

## 2. 🟠 CI 诚实回归集（防未来静默错回潮）

**问题**：HONESTY 审计是一次性的；未来改 VM/加特性可能引入**新的静默错**，今天"绝不静默错"明天悄悄破。

**任务**：把 honesty 审计 T1/T2 语料（`backend/tools/mql2go/testdata/honesty/` 的忠实 EA）固化成 **golden 回归集** → CI 测试：compile+backtest，断言**忠实（无新 fatal 盲区 + 交易行为符合预期基线）**。任何"以前忠实现在静默错"= CI 红。
- REUSE：honesty 审计语料 + 检测机制（coverage/invariants/IsReliable）。
- **对抗证明**：人为引入静默错（删一个常量/破一个 builtin）→ golden EA 行为变 → CI 红。

## 3. 🟠 K3 需求捕获（LEARN = 自我进化"学"半边）

**问题**：决策闭环堵了不支持特性，但**没记录"用户常踩哪些"**→ 系统只"自修"不"自学"（不知该优先支持什么）。

**任务**：`checkUnreliableCoverage` 命中 fatal 盲区时 → `kb_demand_signal(identifier) hit_count++`（+ user_count 去重）→ NOTIFY → admin 路线图视图（按命中频次排"该支持什么"）。
- REUSE：MQL-LOOP-1 堵点；`kb_demand_signal` 表（KB arch K3；若 KB-P0 未建则先独立表后并入）。
- **对抗证明**：同一不支持函数被堵 3 次 → hit_count=3；删计数→0→红。

## 4. 🟡 作者覆盖率分数（透明=信任）

**问题**：作者只看到"iCustom 不支持"，看不到整体兼容度。

**任务**：`CoverageResult` 计算 **coverage% = supported 构造数 / 总构造数**；前端 `DiagnosticPanel` 显示（如"95% 兼容，1 个不支持"）+ 不支持清单。帮作者自评 + 呼应 trust 透明。
- REUSE：`CoverageResult`/`IRBlindSpot`（已有）。
- **对抗证明**：20 构造 1 不支持 → 显示 95%；改回 0 不支持 → 100%。

## 5. 🟡 C1 复利实证回归（防"确定性复利"退化）

**问题**：KB-P0 声称"新 compat_fix → 后续 EA 0-token 受益"，需**永久守护**，别退化成纸面。

**任务**：回归测试——`RecordFix(新别名)` → 一个用该标识符的 EA 编译**忠实（无盲区、0 LLM、0 rebuild）**；删该 fix → EA 落盲区 → 红。永久守护 C1 复利。
- REUSE：KB Service `RecordFix`（KB-P0）+ compile。
- **对抗证明**：RecordFix 后 EA 忠实；无 fix → 盲区 → 红。

---

## 验收（审计方实测）
- **T5**：fatal 盲区策略 StartStrategy 被拒 + 对抗证明。
- **CI 回归**：golden 集 CI job 在；引入静默错→红。
- **K3**：堵点计数进 kb_demand_signal + admin 视图。
- **覆盖率**：DiagnosticPanel 显示 coverage%。
- **C1 实证**：RecordFix 回归测在 + 绿。

## REUSE 核对（施工方 `bash scripts/cap.sh`）
`checkUnreliableCoverage`/`checkBoundAccount`(chokepoint 模式)、honesty 语料、`CoverageResult`/`IRBlindSpot`、KB Service `RecordFix`、`pg_notify`/LISTEN。禁新 DB/禁 JSONB/禁轮询。

## 完工回填纪律（施工方）
1. `tech-debt-registry.md`：MQL-LOOP-4(T5 部分) 推进 + 新增 `MQL-CI`/`MQL-K3`/`MQL-COV`/`MQL-C1VERIFY` 条目 🟦→✅ + 对抗证明。
2. `handover-audit-plan.md` 变更日志。
3. 不自行宣告 ✅——等审计方实测。

---

> **审计方注**：5 条按"安全(T5)→耐久(CI)→自学(K3)→透明(COV)→防退化(C1)"收尾。T5 是安全对称（别暂缓）；CI 是把"绝不静默错"变永久；K3 闭合自我进化的"学"。前 3 条（T5/CI/K3）优先级最高，是把管线从"能跑诚实"推到"长期复利且不退化"的关键。④开放共改仍暂缓（撞 IP 卖点）。
