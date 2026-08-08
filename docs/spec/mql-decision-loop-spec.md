# 用户 MQL 决策闭环 + 分层质量门 spec（检测的"另一半"）

> **背景**：MQL 诚实性审计（`mql-honesty-audit-spec.md`）闭合了 **DETECT**（fatal 盲区→`IsReliable=false`）。但检测是半条腿——`IsReliable=false` 现在是"孤儿"：**发布门控（`quality.go`）只堵 DEGRADED，不看 IsReliable/coverage fatal**（已核实 :119/:155），导致带 fatal 盲区的策略能进市场、作者无感。本 spec 补齐**另半条腿**：TELL（告诉作者怎么办）+ ACT（门控）+ 分层修复。
> **决策来源**：用户 4 点设计（2026-08-08）：① 告诉作者问题；② 有问题不能发布（保质量/买家）；③ 修复路径——(a)手改/(b)AI 帮改(作者 token)/(c)**系统确定性改(0 token,因 VM 标准我们已知)**；④ 开放共改(GitHub 式)——**暂缓**。
> **角色**：审计方出 spec；施工方实现+回填，不自行宣告 ✅，等审计方实测。

---

## 1. 设计：分层质量门（DETECT 已成 → 本 spec 补 TELL+ACT+修复）

| 层 | 做什么 | token | 触发 | 例子 |
|---|---|---|---|---|
| **L0 确定性自动修** | 系统已知 VM 标准→规则化变换，**编译/解析期透明应用** | **0** | 命中 compat-fix 注册表 | 常量别名 `clrGreen→Green`、命名 `MODE_SENKOU_A→MODE_SENKOUA`、弃用→现名 |
| **L1 检测+堵+告知** | 无确定性替代的不支持特性→**堵发布/堵实盘** + 可执行告知 | 0 | fatal 盲区（IsReliable=false） | iCustom/DLL/复杂类→"不兼容：改用 X / 路线图 / 此类暂不支持" |
| **L2 AI 辅助** | 真歧义→agent-engine 重写，**作者审 diff+授权** | 作者 token | 作者选"AI 帮改" | 整段逻辑换写法（扩 useAIFix 到 coverage 盲区）|
| **L3 人工** | 作者自改 | 0 | 作者选"手改" | — |

**关键原则**：L0 优先（0 token、零幻觉、可累积、加厚"VM 兼容知识"护城河）；L2 只是兜底，**不可跳过作者审查**（防语义漂移——见 §3c 边界）。

## 2. Phase 1（P0，闭合"孤儿检测"）— 告知 + 堵发布

**T1 扩展发布门控**：`marketplace/quality.go:checkDegradedStatus`(:104) 旁加 `checkUnreliableCoverage`——查策略最新回测的 `IsReliable` + fatal coverage 盲区；不靠时（IsReliable=false 或有 fatal 盲区）返回**非豁免** QualityViolation，并入 :155 的 hard-block。**与 DEGRADED 同级堵发布**。
- 锚点：`quality.go:101-160`（checkDegradedStatus + hard-block 处）；IsReliable 源 `backtest_worker_vm.go:316`。
- **对抗证明**：策略含 fatal 盲区(iCustom)→发布必拒；移除 checkUnreliableCoverage→能发→测试红。

**T2 可执行告知**：fatal 盲区→**人话 + 怎么办**（非仅 "not fully supported"）。建盲区→建议映射：
- 未支持指标/函数且有已知替代 → "改用 {alt}"
- 无替代但在路线图 → "路线图，订阅通知"
- DLL/复杂类 → "此类暂不兼容"
- 落点：发布被堵的错误信息（T1 返回的 violation 描述）+ 前端 `DiagnosticPanel.tsx`（既有盲区显示）补"建议"列。
- REUSE：`DiagnosticPanel`（ADR-0028 Part B）、盲区 severity（fatal/warning/info，已建）。

## 3. Phase 2（P1，系统确定性改 = 用户的 3c，0-token）

**T3 compat-fix 注册表 + 透明应用**：把 HONESTY-1/2（constants.go 别名）**系统化**为可扩展的确定性修复层：
- 新建 `interp/compat_fixes.go`（或扩 `constants.go`/`api_registry.go`）：注册表 `{problematic_id → deterministic_resolution}`（常量别名/命名归一化/弃用→现名映射）。
- 解析/编译期：遇未知标识符→**先查 compat-fix 注册表**→命中则透明应用（=已知 VM 标准的确定性变换）→**不产生盲区、不调 LLM、0 token**。
- 命中失败才落回现有"盲区检测"（L1）。
- REUSE：HONESTY-1/2 已验证的别名模式（clr*/MODE_SENKOU_*）；`classifySeverity`/`SeverityForBuiltin`。
- **对抗证明**：`clrGreen`/`MODE_SENKOU_A` 策略→L0 透明修→0 盲区+0 token；删注册表条目→盲区复现→测试红。

## 4. Phase 3（P2，扩展）

**T4 L2 AI 辅助扩到 coverage 盲区**：`useAIFix`（ADR-0028 Part B，revise→diff→updateCode）当前面向防线盲区；扩到 coverage fatal 盲区——作者点"AI 帮改"→agent 重写避开不支持函数→**重跑诚实管线校验**（compile+coverage+不变量+回测+IsReliable 必过）→**作者审 diff**→授权保存。**不可跳过作者审查**（语义漂移：AI 把 iCustom 换 iMA 可能"合法但行为不同"，诚实机制抓"坏"抓不到"不一样"）。
**T5 实盘门控**：`StartStrategy`/`RunLiveStrategy` 启动前查 fatal 盲区→堵（不在真账户跑不可靠策略），与发布门控一致。REUSE：`checkBoundAccount` 模式（LEAKAGE-1 的 chokepoint 注入）。

## 5. ❌ 暂缓：开放共改（GitHub 式，用户的 ④）

**暂缓理由**：撞核心卖点"代码不出平台+作者 IP 保护"（共改=别人能看/改作者代码→IP 瓦解）；经济模型冲突（GitHub 开源免费 vs 我们市场卖策略）；IP 归属+贡献者分成未解；Y1 超范围。**AI 迭代（FEAT-5）已覆盖大部分"共改"价值且不泄 IP。** 若未来做：须作者主权(接/拒)+平台内(代码不离)+不泄 IP+分成机制，Y2+ 再议。本条仅留档，不施工。

## 6. 验收（审计方实测）
- **P1**：fatal 盲区策略发布被拒 + 错误信息含"怎么办"；对抗证明成立；build/test 绿。
- **P2**：L0 注册表命中→0 盲区 0 token（clrGreen/MODE_SENKOU_A 实测）；对抗证明成立。
- **P3**：AI 帮改 coverage 盲区→重跑校验过+作者审 diff；实盘 fatal 盲区被堵。

## 7. REUSE（施工方 `bash scripts/cap.sh`）
`checkDegradedStatus`/QualityViolation、`IsReliable`(backtest_worker_vm.go)、`DiagnosticPanel`/`useAIFix`、`classifySeverity`、HONESTY-1/2 别名模式、agent-engine(FEAT-5)、`checkBoundAccount` chokepoint 模式。

## 8. 完工回填纪律（施工方）
1. `tech-debt-registry.md` 新增 `MQL-LOOP-*` 条目 🟦→✅ + 对抗证明。
2. `handover-audit-plan.md` 变更日志。
3. 不自行宣告 ✅——等审计方实测。

---

> **审计方注**：本 spec 把检测的"另一半"补齐，并把你（用户）的 3c 洞察（**确定性 0-token 系统改，因 VM 标准已知**）作为 L0 首选——它比 LLM 路径优（免费/零漂移/可累积/加厚护城河）。P0 先闭合"孤儿检测"（堵发布+告知），P1 落 L0，P2 扩展。④ 留档暂缓。
