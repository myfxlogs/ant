# MQL 诚实性审计 spec（地基验证：用户 MQL 绝不静默错）

> **背景**：用户一针见血——"用户提交 MQL 源码，能真正、诚实跑起来，才是整个项目最根本，其他都是辅助。" 所有市场/信任/战绩都建在此之上。
> **核心命题**：用户提交任意 MQL → **要么忠实执行，要么大声报错（编译失败/coverage 盲区/IsReliable=false），绝不静默出 0 或错单。**
> **依据**：`docs/plan/mql-ea-compatibility-proposal.md`（v10 审计）§10.4「诚实的评估」+ §1（VM 重实现 ~428 常量/~200 函数/~40 指标，静默错是核心危险）+ CLAUDE.md「MQL2GO VM Pitfalls」。
> **角色**：施工方执行（建语料+跑+挖裂缝，**重 token 消耗交出去**）；审计方仅收裂缝清单复审。

---

## 1. 审计目标

证明（或证伪）：**对任意用户风格 MQL，我们的 compile→IR→VM→SimBroker 管线不会静默产生错误结果。** 任何"编译过+回测出结果但结果静默错（0/错线/漏单）且未标盲区/未降级"= **🔴 裂缝**，必须挖出来。

**这是地基验证——裂缝清单决定项目可信度上限。**

## 2. 语料（3 层，覆盖真实 EA 形态）

| 层 | 形态 | 预期（诚实系统应如此） |
|---|---|---|
| **T1 简单EA** | 单指标(iMA/iRSI/MACD) + 条件 + OrderSend | 忠实执行，回测合理 |
| **T2 中等EA** | OrderSelect 循环 + 持仓管理 + 多指标组合 + trailing stop | 忠实执行 或 标盲区降级 |
| **T3 故意不支持（诚实探针）** | DLL 调用 / 复杂类继承 / Chart 操作 / 未实现 builtin / 未知常量 / 未知指标 / 前向引用 / OrderType 边界 | **必须大声失败**（编译错 / IRBlindSpot / IsReliable=false / DEGRADED），绝不静默跑 |

每层 ≥5 个 EA（T3 重点：每类不支持特性一个探针）。语料放 `backend/tools/mql2go/testdata/honesty/`。

## 3. 跑法（每条 EA）

1. 提交 → `CompilePythonWithCoverage`/`CompileMQLWithCoverage`（接现有 coverage 分析）。
2. 记录：编译结果（pass/fail + 错误）、`CoverageResult`（blind spots / defense A violations / lookahead）、回测结果（trades/equity/`IsReliable`/DEGRADED）。
3. **判定**（每条三选一）：
   - ✅ **忠实**：编译过 + 无致命盲区 + 回测非退化 + 行为符合预期
   - 🟢 **诚实失败**：编译失败 / 标致命盲区 / IsReliable=false / DEGRADED——**大声告诉用户不支持**
   - 🔴 **裂缝（静默错）**：编译过 + 未标盲区/未降级 + 但回测结果错误（volume=0 / 永不开单 / 错线 / 漏单）——**这就是地基裂缝**

## 4. 输出

- `docs/audits/mql-honesty-audit-report.md`：每条 EA 的判定表 + 所有 🔴 裂缝的**根因**（哪个常量/函数/特性静默错）。
- 每条 🔴 裂缝 → `tech-debt-registry.md` 新建 `MQL-HONESTY-N` 条目 🟦open，附最小复现 EA + 根因 + 修复方向（参照已修的 3 类 pitfall 模式：补常量/两遍编译/OrderType 映射）。
- **统计**：T1/T2 忠实率、T3 诚实失败率（应 100%）、裂缝总数。

## 5. 对抗证明（审计本身即对抗）

- **T3 探针是对抗核心**：故意喂不支持特性，任何一条"静默跑出错结果"= 裂缝被抓。
- 对每条 🟢诚实失败，验证"移除该盲区检测→它就会静默跑错"（证明检测有效，非空跑）——参照 ADR-0028 对抗证明契约。
- 已修 3 pitfall（常量/map/OrderType）作回归项，确认不复现。

## 6. 已知线索（从 pitfall 史 + §1，优先探这些）

- **未知常量→0**：`interp/constants.go` 是否所有路径都报错而非 push 0？（CLAUDE.md 称已补全，验证覆盖率）
- **未知函数→0**：两遍编译后，是否所有前向引用都解析、无残留"unknown function→0"？
- **OrderType/下单类型映射**：所有 OP_* 映射正确？
- **未实现 builtin**：`api_registry.go unsupportedSymbols` 之外的、既未实现又未登记的 builtin，跑到时是 panic/报错 还是 push 0？（**最可能的裂缝带**）
- **未知指标**（iCustom / 冷门指标）：未实现时回退值是否静默错？
- **数值/精度**：NormalizeDouble/Lot 计算/除零 等——静默 NaN/0？

## 7. REUSE（施工方 `bash scripts/cap.sh`）
- `AnalyzeCoverage`/`CoverageResult`（blind spots）、`ValidateDefenseA`、`DetectLookahead`、`CompileMQLWithCoverage`、ADR-0028 防线 B（invariants，回测退化兜底）。
- 已有 EA 语料：LAUNCH-1 的 20 策略（trend/mean_reversion/breakout/grid/multi_tf/oscillator）作 T1/T2 基底。

## 8. 完工回填纪律（施工方）
1. 审计报告 `docs/audits/mql-honesty-audit-report.md` + 每条 🔴 进 `tech-debt-registry.md MQL-HONESTY-N`。
2. `handover-audit-plan.md` 变更日志。
3. **不自行宣告"地基OK"**——裂缝清单交审计方复审；若 T3 诚实失败率<100% 或有 🔴，**项目可信度未达，需逐条修**。

---

> **审计方注**：这是项目最根本的验证。**T3 诚实失败率必须 100%**（不支持的就大声报），🔴 裂缝数必须趋近 0（或全部已知+可修）。结果决定：人类开发者供给能否解锁、实盘战绩可信度上限。
