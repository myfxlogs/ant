# Builder Handoff: VM-TEST-EVIDENCE-4（Batch 5）

> **设计/验收方**：Devin CLI
> **施工方**：Devin IDE / Windsurf
> **基线 HEAD**：Batch 1-3 验收后开工（依赖 Batch 1-3 的测试文件已存在）
> **边界**：只重写 `docs/audits/vm-adversarial-proofs.md`，禁改生产代码，禁改测试代码。
> **施工后状态**：`🟦open（施工完成，待独立复审）`，不得自标 ✅done。

---

## 立项背景

D-REVERT-SCOPE-DRIFT-001 回滚后，`docs/audits/vm-adversarial-proofs.md` 被标记 SUPERSEDED。Batch 1-3 重新施工后，对抗证明文档需从零重写，指向新的测试文件和 mutation targets。

文档作用：供 Devin CLI 独立复审时快速定位每项修复的对抗证明——mutation target、预期 RED、restore 指令、测试文件位置。

---

## 🔴 绝对边界

1. **只改** `docs/audits/vm-adversarial-proofs.md`。**禁止改** 任何生产代码或测试代码。
2. 文档内容必须指向 Batch 1-3 验收后**实际存在**的测试文件和函数。
3. 禁止 commit / push / deploy。禁 `--no-verify`。

---

## 施工步骤

- **S1** 重写 `docs/audits/vm-adversarial-proofs.md`，包含以下 proof 条目（每条：mutation target、预期 RED、restore 指令、测试文件:行号）：

### VM-COMPILER-SEMANTICS-4 proofs（Batch 1）

- **Proof 1**：comma_expression ExprSeq — mutation: revert `comma_expression` 为只返回 last → `TestCommaExpression_VMSideEffectsExecution` RED（g_a=0）→ restore → GREEN。
- **Proof 2**：HasError guard — mutation: 删 `HasError` 调用 → `TestCompileMQL_CompletelyInvalidSourceRejected` RED → restore → GREEN。
- **Proof 3**：结构化 input/extern 检测 — mutation: revert 为 `strings.Contains` → `TestCompileMML_InvalidInputMissingInitializer` + `TestCompileMML_ReservedKeywordAsIdentifierRejected` RED → restore → GREEN。

### VM-CACHE-INTEGRITY-5 proofs（Batch 1）

- **Proof 4**：coverage restore — mutation: 删 `CompilePythonCached` 的 coverage restore → `TestCompilePythonCached_RestoresCoverageOnCacheHit` RED（CoverageResult nil）→ restore → GREEN。
- **Proof 5**：Version=="python" check — mutation: 删 Version check → `TestCompilePythonCached_RejectsMQLBytecodeForPythonSource` RED → restore → GREEN。
- **Proof 6**：payload limit — mutation: 删 `maxBytecodePayload` guard → `TestUnmarshalBytecode_PayloadLimitExceeded` RED（error 变 "invalid magic"）→ restore → GREEN。

### VM-TRADE-CONTEXT-6 proofs（Batch 2）

- **Proof 7**：OHLCV array length — mutation: 删长度校验 → `TestVMHandleBar_ArrayLengthMismatch` RED → restore → GREEN。
- **Proof 8**：strict decimal parse — mutation: revert 为 `parseDecimal` → `TestVMHandleBar_InvalidDecimalRejected` RED → restore → GREEN。
- **Proof 9**：nil position reject — mutation: 删 nil 检查 → `TestVMHandleBar_NilPositionRejected` RED → restore → GREEN。
- **Proof 10**：lookup fail-closed — mutation: 删 fail-closed → `TestBuildLiveContext_LiveModeLookupFailClosed` RED → restore → GREEN。
- **Proof 11**：validateFirstBarContext — mutation: 删 validate → `TestVMLiveSession_StartRejectsInvalidFirstBarContext` RED → restore → GREEN。
- **Proof 12**：Login injection — mutation: 删 Login 注入 → `TestVMLiveSession_EndToEndAccountNumberReadback` RED → restore → GREEN。

### VM-API-TRUTH-3 proofs（Batch 3）

- **Proof 13**：IsConnected from context — mutation: revert 为 `return BoolVal(true)` → `TestBuiltinIsConnected_ReadsFromContext` RED → restore → GREEN。
- **Proof 14**：IsTradeAllowed from context — mutation: revert 为 `return BoolVal(true)` → `TestBuiltinIsTradeAllowed_ReadsFromContext` RED → restore → GREEN。
- **Proof 15**：end-to-end IsConnected — mutation: 删 context 传递 → `TestVMLiveSession_IsConnectedEndToEnd` RED → restore → GREEN。

---

## 验收标准

- 每个 proof 条目包含：mutation target（精确 file:line + 改什么）、预期 RED（测试名 + 断言失败消息）、restore 指令、测试文件位置。
- 所有引用的测试文件必须实际存在（`ls` 验证）。
- 所有引用的测试函数必须实际存在（`grep` 验证）。
- 文档不超 450 行（T1 预算）。

---

## 回填与收尾

registry 本条回填 + `handover-audit-plan.md` 追加一行。**状态填 `🟦open（施工完成，待独立复审）`。**

> **勿部署、勿 push、停手等 Devin CLI 复审。禁止 `--no-verify`。**
