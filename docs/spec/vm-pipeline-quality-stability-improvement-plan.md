# VM 管线质量与稳定性提升设计方案（v2 定稿）

- **设计方 / 决策方**：Devin CLI（项目第一负责人 / 唯一技术决策者 / 独立复审验收方，D-006）
- **日期**：2026-09-16（v1 草案）→ 2026-09-16（v2 源码逐条核验后定稿）
- **类型**：实施性设计方案（非 ADR）；配套决策记录 `docs/handoff/decisions.md` D-009
- **依据**：`docs/audits/vm-pipeline-deep-audit-2026-09-16.md` §11 + `docs/adr/0023-reevaluation-2026-09-16.md` §5 + v2 源码实拍（本文件 §2）
- **状态**：✅定稿（Devin CLI 决策，无需业主逐项确认）。QS 条目已入 `docs/audits/tech-debt-registry.md`，按 §6 顺序一次派一单。
- **最终决策**：Devin CLI（[角色:决策终] 激活）

---

## 0. 设计目标与非目标

| 目标 | 说明 |
|------|------|
| **质量** | 修补已实拍确认的语义边界缺陷，保证 MQL/Python 策略行为与语言预期一致 |
| **稳定性** | 消除 barrier 状态机绕行、panic 与状态不一致、用 goleak 证明无泄漏 |
| **可观测性** | 先拿到 VM 生产耗时数据，再谈性能 |

**非目标**：不推翻架构、不换语言、不重写 VM 核心、**不逆转已验收的 fail-closed 不变量**（VM-RUNTIME-FAILCLOSED-1）、本轮不做性能优化。

## 1. 设计原则

1. 语义边界优先：正确性 > 性能。
2. fail-closed 贯穿：无 authority / 无 provenance / outcomeUnknown / critical builtin 缺失 → 不继续。
3. 同码不变量：回测与实盘共用 VMRunner，改动不得破坏。
4. **复用优先（AGENTS §7.5）**：动手前 `bash scripts/cap.sh`；已存在的作用域栈、opcode、状态机迁移方法必须复用，不另造。
5. 对抗证明：每条修复 mutation RED→restore→GREEN；race×3；check-lines 零警告。
6. **有证据才立项**：没有实拍缺陷或测量数据的"加固/优化"不进施工表。

---

## 2. v1 草案核验结果（源码实拍，2026-09-16）

| v1 条目 | 实拍结论 | v2 处置 |
|---|---|---|
| QS-1.1 isFatalUnimplemented | `Object*/Chart*/File*` 在 `unsupportedSymbols`，**编译期拒绝**（`compile_expr.go:386-393`），运行时分支是死路径；只剩 `vm_helpers.go:250` 注释过时 | 降为一行注释修正，并入 QS-1.4 |
| QS-1.2 GetLastError | 校验失败 fatal 是 VM-RUNTIME-FAILCLOSED-1 刻意设计（`vm_builtin_trade.go:29-36`）；v1 未声明冲突。"每事件清零"不是 MQL4 语义（`_LastError` 只在 `GetLastError/ResetLastError` 清零） | 拆为 QS-1.2a（实现 lastError）/ QS-1.2b（不做，见 D-009） |
| QS-1.3 Python 隐式全局 | 缺陷成立（`compile.go:282-289`）。但 `astCompiler` 已有 `localScopes/pushScope/popScope/nextLocalSlot`（`compile.go:244-263`，槽分配 `compile_expr.go:200`）；v1 要在 pyCompiler 再造一套 = 重复。"并发污染"不准确，VM 单线程，是跨事件污染 | 保留，改修法（§3.3） |
| QS-1.4 bool(None) | 缺陷成立（`compile_py_expr.go:165-172` + `value.go:153-168` 混合 None/Int Equal=false → `None!=0` 为 true）。v1 新增 `__bool__` builtin 多余，`OP_NOT` 已走 `IsTrue()` | 保留，改修法（§3.1） |
| QS-1.5 语言检测 | 反例 `from decimal import Decimal\nx=1` 不是可运行策略（无 `on_*`/`StrategyBase`），对真实策略无影响 | 不立债；`LangUnknown` 隐式当 MQL 路由记入 pitfalls |
| QS-1.6 强制 confirmed | 缺陷成立（`mutation_coordinator.go:330-334`）。**v1 修法无效**：`Reconcile` 仅在 `outcomeUnknown` 迁移（`trade_barrier.go:333`），此处状态是 `acceptedUnconfirmed` → no-op | 保留，改修法（§3.2） |
| QS-1.7 open recovery | open 排除是 ④-② 刻意决策（`mutation_coordinator.go:260-266`）；magic+symbol+side+30s 模糊匹配在同策略同向连续开仓下必误匹配。open 已带唯一 `ClientID`（`live_dispatch.go:366`） | 模糊匹配否决；改为前置调研 QS-1.7-INV（§3.5） |
| QS-2.1 watcher 泄漏 | **不是泄漏**：`stopWatcher`+`defer close` 绑定 `WaitConfirmed` 生命周期（`trade_barrier.go:298-308`）。v1 "复用 watcher" 引入共享状态反而危险 | 否决；由 QS-2.2 goleak 测试一条证明 |
| QS-2.2 goleak + resourceTracker | `go.mod` 无 goleak；`resourceTracker` 无缺陷证据 | 仅保留 goleak 集成 |
| QS-2.3 nil 安全 | 101 处 `vm.ctx == nil`；方向可接受但无缺陷证据、churn 大 | 保留，P2 最后做 |
| QS-2.4 race 审计 | 审计性质 | 保留 |
| QS-2.5 panic recovery | v1 "recover 后 return outcomeUnknown" 不够：broker 调用后 panic 必须同时 `barrier.NotifyOutcomeUnknown()` + `SetCircuitOpen(true)`，否则状态与返回值不一致 | 保留，补齐（§4.3） |
| QS-3.x 性能 | `interp.Value` 是值结构体（`value.go:12-21`），`sync.Pool` 池化值类型不成立；decimal 换库触及全部价格计算；Go 稠密 switch 已生成跳转表；全阶段零 baseline、零生产超时证据 | 整体否决；改为 QS-3-BASELINE 测量任务（§5） |

---

## 3. 阶段 1：语义边界修补（P0，按序单发）

### 3.1 [QS-1.4] `bool(x)` 语义 + 注释修正

- **改动**：`compile_py_expr.go:165-172` `bool(x)` 由 `x != 0` 改为双重逻辑非 `ExprUnary{"!", ExprUnary{"!", x}}`（编译为两条 `OP_NOT`，复用 `IsTrue()`：None→false、0→false、""→false、非零→true）。零新 builtin、零新 opcode。
- **顺手**：`vm_helpers.go:250` 注释改为"builtins 表内有名无 handler 且注册表未标 critical → 非致命盲区（注册表 `StatusUnsupported` 符号在编译期已拒绝，不会到此）"。
- **验证**：`bool(None)`→false、`bool(0)`→false、`bool(Decimal("0"))`→false、`bool("")`→false、`bool(1)`/`bool("a")`→true；mutation：恢复 `!= 0` → `bool(None)` 用例 RED。
- **REUSE**：`OP_NOT`@`vm_execute.go:284`、`Value.IsTrue`@`value.go:72`。

### 3.2 [QS-1.6] read-after-write 确认走状态机

- **根因先答**：施工方须先写明 `mutation_coordinator.go:325` `NotifyConfirmationEvent` 在 verify 成功后为何可能未迁移（候选：`action==open` 时 `ticket==0` 被 `:219` 早退；`isUpdateTypeCompatible` 不兼容；barrier 仍在 `submitting`）。答案随自报提交，由决策方回填 registry（施工者不写交接层）。
- **改动**：`TradeBarrier` 新增 `ConfirmByAuthoritativeRead()`：持 `b.mu`，仅当 `state == barrierAcceptedUnconfirmed`（或 `barrierSubmitting`，视根因而定）→ `barrierConfirmed` + `Broadcast`；其余状态 no-op 并返回 bool。`mutation_coordinator.go:333-334` 改为调用该方法后再 `return barrierConfirmed`。**不用** `Reconcile`。
- **验证**：新测试构造 verify 成功 + 事件不匹配场景，断言 `barrier.State()==barrierConfirmed` 在 Release 前成立；mutation：删 `ConfirmByAuthoritativeRead` 调用 → 状态仍 `acceptedUnconfirmed` → RED。
- **REUSE**：`Reconcile`/`NotifyBrokerAccepted` 的锁+Broadcast 模式@`trade_barrier.go:185-206,330-342`。

### 3.3 [QS-1.3] Python 函数局部作用域

- **改动点在共享编译器，不在 pyCompiler**：`astCompiler` 编译 `ExprAssignment`（及增量赋值）时，若 `c.bc.Version=="python"` 且 `len(c.localScopes)>0` 且目标名在 localScopes / GlobalSlots 均未命中 → 在最内层 scope 分配局部槽（同 `compile_expr.go:200-201` 路径），**不**落 `compile.go:284` 的隐式全局分支。`self.X`（class field，已在 GlobalSlots）与顶层赋值（`compileTopLevelExpr` → `ir.Globals`）不受影响。
- **边界**：Python `global x` 语句——先 grep pyCompiler 是否解析 `global_statement`；未支持则在编译期 `c.errorf` 明确报错（fail-closed），不得静默当局部。
- **验证**：① `def f(self): x=1` 后 `def g(self): return x` → 编译期 unknown variable 错误或运行时 None，不得读到 1（当前读到 1 → RED）；② `on_bar` 内 `x=1`，下一次 `on_bar` 读 `x` 前先赋值路径不变、未赋值读为 None；③ `self.count += 1` 跨事件仍累加（回归守卫）；④ 参数同名局部（`def f(self, x): x = x + 1`）仍走参数槽。
- **风险**：依赖跨事件全局污染的存量 Python 策略会变——这是 bug 依赖而非 feature；覆盖度报告加 `python_local_scope_fix` 标记；registry 记录。
- **REUSE**：`pushScope/popScope/resolveVar/nextLocalSlot`@`compile.go:244-263,174`。

### 3.4 [QS-1.2a] `GetLastError/ResetLastError/SetUserError` 真实实现

- **改动**：`VM` 加 `lastError int32`；`builtinGetLastError` 返回并清零；`builtinResetLastError` 清零；`builtinSetUserError(c)` 写 `ERR_USER_ERROR_FIRST + c`（MQL4：65536+c）。**不**在 `runEvent` 清零（非 MQL4 语义）。`vm_builtin_impls.go:73-74` 两处 `builtinNoop*` 注册改指向新实现；`vm_builtin_checkup.go:102` stub 替换。
- **写入点（本轮）**：仅 `SetUserError`。OrderSend/OrderClose 等交易失败路径**保持 FAILCLOSED-1 fatal 语义不变**（D-009），不写 `lastError`。
- **错误码常量**：`interp/` 新增 `mql_errors.go`，只放本轮用到的 `ERR_NO_ERROR=0`、`ERR_USER_ERROR_FIRST=65536`；不预埋整张表。
- **验证**：`SetUserError(7); GetLastError()==65543; GetLastError()==0`；`ResetLastError` 清零；mutation：删 `vm.lastError = 0` 读后清零 → 第二次读仍 65543 → RED。

### 3.5 [QS-1.7-INV] open outcomeUnknown 恢复可行性调研（只查不改）

- **问题**：`ClientID`（`live_dispatch.go:366` `strategyOrderClientID`）是否随 `OrderRequest.Comment` 下发到 mtapi 并在 `OpenedOrders` 回显？MT4/MT5 adapter 各自确认（不共享代码）。
- **产出**：registry 回填链路事实 + 结论：(a) 回显可靠 → 立 QS-1.7 按 ClientID **精确单匹配** recovery（多匹配/零匹配保持锁仓）；(b) 不可靠 → 维持 ④-② fail-closed，运维手册补 outcomeUnknown 人工处理流程（QS-4.3）。**模糊匹配（magic+symbol+side+时间窗）永久否决。**

---

## 4. 阶段 2：运行时稳定性（P1，阶段 1 验收后）

### 4.1 [QS-2.2] goleak 集成
- `go.uber.org/goleak` 加入 `go.mod`（选发布 ≥7 天版本）；`backend/internal/connect/strategy` 与 `backend/tools/mql2go` 的 `TestMain` 加 `goleak.VerifyTestMain`。
- 附带一条 `TestWaitConfirmed_NoGoroutineLeak`（循环 WaitConfirmed+Release ×1000，goleak 断言）作为 QS-2.1 否决的证据。
- 发现的既有泄漏单独立债，不在本条修。

### 4.2 [QS-2.4] data race 审计
- `go test -race -count=3` 覆盖 `tools/mql2go/...`、`internal/connect/strategy/...`、`strategy/...`；审计 `PositionCache` 全部访问在 mutex 下、`TradeBarrier` cond/Broadcast 配对。发现即立债。

### 4.3 [QS-2.5] panic recovery 加固
- `coordinateMutation` 加 `defer recover`：log + **`barrier.NotifyOutcomeUnknown()` + `activeSess.SetCircuitOpen(true)` + `RecordError`**，返回 `mutationResult{state: barrierOutcomeUnknown}`；broker 调用前 panic 可 `NotifyDeterministicRejected + Release`（提前判定的分支须以是否已发出 RPC 为界）。`dispatchLiveSignal` 同模式。
- **验证**：mock broker panic → 进程不崩 + barrier 状态 `outcomeUnknown` + circuit open；mutation：删 `NotifyOutcomeUnknown` → barrier 停在 `submitting` → RED。

### 4.4 [QS-2.3] `vm.ctx` 非 nil 不变量（P2，最后）
- `NewVM` 注入 `noopContext`（所有方法返回零值/nil Broker），删除 101 处 `vm.ctx == nil` 检查；保留 `vm.ctx.Broker() == nil` 检查。
- **连带记债（不在本条修）**：`builtinOrderSend:15-17` 无 broker 时静默返回 -1 + nil error，本身不 fail-closed。

---

## 5. 阶段 3：性能 → 改为 [QS-3-BASELINE] 测量（贯穿，可与阶段 2 并行）

- `go test -bench` 基线：dispatch 循环、builtin 调用、Decimal 算术、典型策略 1000 tick；结果落 `docs/audits/vm-perf-baseline-2026-09.md`。
- 生产 metric：VM 单事件耗时直方图、每 tick 指令数、fatal 计数（prometheus client 已在 go.mod）。
- **决策规则**：仅当生产 p99 单事件耗时 > tick 间隔 10% 或 benchmark 显示某项占比 > 30% 时，才立具体优化条目；候选方向为缩小 `interp.Value` 结构体（指针 union），而非池化。

## 6. 实施顺序

```
阶段 1（P0）  QS-1.4 → QS-1.6 → QS-1.3 → QS-1.2a → QS-1.7-INV
阶段 2（P1）  QS-2.2 → QS-2.4 → QS-2.5 → QS-2.3(P2)
贯穿         QS-3-BASELINE ；QS-4.3 文档（pitfalls: GetLastError 语义 / Python 局部作用域 / LangUnknown 路由）
```

依赖：QS-1.6 根因回答可能影响 `ConfirmByAuthoritativeRead` 允许的起始状态；QS-1.7 是否立项取决于 QS-1.7-INV 结论；阶段 3 是否存在取决于 QS-3-BASELINE 数据。

## 7. 验收标准（每条 QS）

1. 对抗证明 mutation RED→restore→GREEN（真实行为级，不接受 nil panic / 任意 error 当 RED）。
2. 机检：`go build ./...` + 相关包 `go test` + `go test -race -count=3` + `check-file-lines --strict` 零警告。
3. 覆盖度报告无退化；PR 标 `REUSE:`/`NEW:`。
4. 施工方停在待复审状态；Devin CLI 独立复审后才 `✅done`。

---

> v2 基于 `/opt/ant/backend` 源码 2026-09-16 实拍逐条核验。v1 中被否决/降级的条目及理由保留在 §2，禁删。
