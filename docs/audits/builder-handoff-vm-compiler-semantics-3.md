# 施工派工单：VM-COMPILER-SEMANTICS-3

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：`compile_loops.go:105-203` `compileSwitch` 存在两个 P1 语义缺陷，现有测试 `TestVM_Audit_SwitchFallthrough`（`vm_compiler_semantics_redo_test.go:189`）只查 `g_result` global 值不查栈深度 → 假绿。

**证据链**（HEAD `f379d5ac` 实拍，2026-09-16 核实）：

**缺陷1 — default 顺序破坏**：
- `compile_loops.go:114-122` 将 `default` 从 `s.Cases` 抽出：`defaultBody = sc.Body`，`regularCases` 只含非 default case。
- `:133-155` 只遍历 `regularCases` 编译 comparison + body。
- `:157-163` default body 在所有 regular cases 之后编译（`defaultStart`）。
- `:168-184` JMP_IF_FALSE patch：fallthrough case 跳下一个 case body，normal case 跳下一个 case comparison，last case 跳 default 或 POP。
- **问题**：MQL/C 的 `default` 可出现在 case 中间（如 `case 1 / default / case 2`）。当前实现把 default 强制放最后，破坏原始 fallthrough 顺序——case1 不匹配应落入 default（若 default 在 case1 之后），但当前跳到 case2 comparison，case2 不匹配才跳 default。default 在中间时的 fallthrough 语义错误。

**缺陷2 — break 绕过 OP_POP 留栈**：
- `:193` `c.emit(OP_POP, 0, 0, 0)` — emit OP_POP 消费 switch value。
- `:195` `endPC := int32(len(c.bc.Code))` — endPC 是 OP_POP **之后**的位置。
- `:196-197` `for _, ej := range endJumps { c.patchJump(ej) }` — case body 末尾 break 的 JMP patch 到当前 len(Code) = endPC（OP_POP 之后）。
- `:199-200` `for _, bj := range lc.breakJumps { c.bc.Code[bj].A = endPC }` — case body 内 break statement（`compile.go:450-451` emit OP_JMP + append breakJumps）patch 到 endPC（OP_POP 之后）。
- **问题**：所有 break 路径（case body 末尾 break + case body 内 break）都跳到 OP_POP 之后，绕过 OP_POP → switch value 留在栈上。只有 fallthrough 到 switch 末尾（无 break）才执行 OP_POP。`runEvent`（`vm.go:202-204`）每个 event 开始清空 stack，所以留栈不跨 event 累积，但 event 内 switch 后的语句会被残留栈值污染。

**现有测试假绿**：
- `vm_compiler_semantics_redo_test.go:189` `TestVM_Audit_SwitchFallthrough`：case1 fallthrough case2，查 `g_result=110`。g_result 是 global 赋值，不依赖栈，break 留栈不影响结果 → 测试 GREEN 但缺陷存在。

**不变量**：本任务改 `compile_loops.go`（`compileSwitch`）+ 新增/增强测试。不碰其他编译器路径（for/while/do-while 的 break 已正确 patch 到 loop endPC，loop 无 switch value 需 POP）。

## 设计 SSOT

### S1 — 修复 default 顺序（`compile_loops.go:105-203`）

**目标**：保留 `s.Cases` 原始顺序（含 default 在原位置），default 不参与 comparison 但作为 fallthrough target 参与顺序。

**改法**：
- 删除 `:114-122` 的 default 抽取逻辑（`defaultBody` / `regularCases` 分离）。
- 遍历 `s.Cases`（原始顺序），对每个 case：
  - 若 `sc.Expr == nil`（default）：不 emit OP_DUP/OP_EQ/JMP_IF_FALSE，记录 `defaultBodyStart = int32(len(c.bc.Code))`，编译 default body，default body 末尾若有 break emit endJumps（跳到 switch end）。default 作为 fallthrough target：记录到 `caseBodyStarts`（与 regular case body 同列），供前一个 fallthrough case 的 fallthroughJmp target。
  - 若 `sc.Expr != nil`（regular case）：emit OP_DUP/OP_EQ/JMP_IF_FALSE（同现有），编译 body，break/fallthrough 处理同现有。
- JMP_IF_FALSE patch（`:168-184`）：按原始顺序，fallthrough case 跳下一个 case body（可能是 default body 或 regular case body），normal case 跳下一个 case comparison（跳过 default 的 comparison——default 无 comparison，直接到 default body 或下一个 regular case comparison）。last case 跳 default body（若有）或 POP。
- no-match（所有 case 不匹配）：跳 default body（若有）或 POP。当前 last case 的 JMP_IF_FALSE 已处理此逻辑，需调整为跳 defaultBodyStart（若 default 在最后）或按原始顺序的 default 位置。

**关键**：default 在中间时，default 前的 case 不匹配应跳到 default body（而非跳过 default 到下一个 regular case）。default 后的 case 不匹配应跳到 default 之后的下一个 case comparison 或 POP。

### S2 — 修复 break 绕过 OP_POP 留栈（`compile_loops.go:192-201`）

**目标**：所有退出路径（break + fallthrough 到末尾）都消费 switch value。

**改法**：
- `:193` emit OP_POP 记录 `popPC := int32(len(c.bc.Code))`，再 emit OP_POP，`endPC := int32(len(c.bc.Code))`（OP_POP 之后）。
- `:196-197` endJumps（case body 末尾 break）patch 到 `popPC`（OP_POP 的位置），让 break 执行 OP_POP 后到 endPC。
- `:199-200` breakJumps（case body 内 break）patch 到 `popPC`（同上）。
- fallthrough 到 switch 末尾（无 break）自然执行 OP_POP（已在 `:193` emit）。

**验证**：event 结束后 `len(vmRunner.vm.stack)` 应为 0（switch 正确消费 switch value）。break 留栈时 `len(vm.stack) > 0`。

### S3 — 增强现有测试 + 新增对抗测试（`vm_compiler_semantics_redo_test.go`）

**S3a — 增强 `TestVM_Audit_SwitchFallthrough`（:189）**：
- 保留现有 `g_result=110` 断言。
- 新增栈深度断言：OnTick 后 `len(vmRunner.vm.stack) == 0`（case2 有 break，break 应消费 switch value）。

**S3b — 新增 `TestVM_Audit_SwitchDefaultBeforeCase`**：
- 策略：default 在 case1 和 case2 之间，验证 default 在中间时的 fallthrough 顺序。
- 源码：
  ```
  int g_result = -1;
  void OnTick() {
      int x = 1;
      switch (x) {
          case 1: g_result = 10; break;
          default: g_result = 999; break;
          case 2: g_result = 20; break;
      }
  }
  ```
- 断言：x=1 → case1 匹配 → g_result=10（break 退出，不 fallthrough 到 default）。x=2 → case1 不匹配 → 跳 default（default 在 case1 之后）→ g_result=999（**关键**：当前实现跳 case2 comparison，case2 匹配 → g_result=20，错误）。x=3 → case1 不匹配 → default → g_result=999。x=2 时正确语义应是 case2 匹配（default 是"else"，case2 匹配优先）——**修正**：C 语义是逐个 case comparison，default 在不匹配时执行，但 default 的位置不影响 case comparison 顺序。case1 不匹配 → 下一个 case comparison（跳过 default，default 无 comparison）→ case2 comparison → 匹配则执行 case2 body。所以 x=2 → case2 匹配 → g_result=20（正确）。default 只在所有 case 不匹配时执行。
- **重新设计**：default 在中间的 fallthrough 顺序问题体现在 **fallthrough**（无 break）场景。case1 无 break fallthrough 到 default（default 在 case1 之后），而非跳到 case2 comparison。源码：
  ```
  int g_result = -1;
  void OnTick() {
      int x = 1;
      switch (x) {
          case 1: g_result = 10; // no break — fallthrough
          default: g_result = g_result + 1000; break;
          case 2: g_result = 20; break;
      }
  }
  ```
- 断言：x=1 → case1 匹配 → g_result=10 → fallthrough 到 default（default 在 case1 之后）→ g_result=1010 → break。**当前实现**：case1 fallthrough 跳 case2 body（regularCases 顺序），default 在最后 → g_result=10 → fallthrough 到 case2 body → g_result=20（错误，应 fallthrough 到 default）。x=1 正确应为 1010。
- 栈深度断言：`len(vm.stack) == 0`（break 消费 switch value）。

**S3c — 新增 `TestVM_Audit_SwitchBreakStackCleanup`**：
- 策略：switch 有 break，break 后 switch 后跟一个表达式语句，验证栈无残留污染。
- 源码：
  ```
  int g_result = -1;
  void OnTick() {
      int x = 1;
      switch (x) {
          case 1: g_result = 10; break;
          case 2: g_result = 20; break;
      }
      g_result = g_result + 5; // switch 后语句，若栈有残留会污染
  }
  ```
- 断言：x=1 → case1 匹配 → g_result=10 → break → g_result=15。`len(vm.stack) == 0`。
- **对抗**：删除 S2 修复（break patch 回 endPC 绕过 OP_POP）→ 栈深度断言 RED（`len(vm.stack) == 1`）→ 恢复 GREEN。g_result 断言可能仍 GREEN（g_result 赋值不依赖栈），栈深度断言是关键对抗。

## 约束与目标

- **范围**：仅 `backend/tools/mql2go/compile_loops.go`（`compileSwitch` :105-203）+ `backend/tools/mql2go/vm_compiler_semantics_redo_test.go`（增强 S3a + 新增 S3b/S3c）。零其他文件。
- **不碰**：`compile.go`（break statement emit :450-451 不变）、`compile_interp.go:475`（switch AST 解析不变）、for/while/do-while 的 break（loop endPC 无 switch value POP）、其他 switch 测试（golden/e2e 若有，跑全量验证不破坏）。
- **REUSE**：`compileSwitch` @ `compile_loops.go:105`；`loopContext.breakJumps` @ `compile.go:164`；`emitJump`/`patchJump` @ `compile.go:198-204`；`getGlobalInt` @ `live_mql_order_context_vm_test.go:157`；`CompileMQL`/`runner.New`/`r.OnTick` 模式 @ `vm_compiler_semantics_redo_test.go:189-234`；`vmRunner.vm` 访问 @ `vm_audit_test.go:41`。
- **NEW**：无新文件（测试加到现有 `vm_compiler_semantics_redo_test.go`）。
- **先红后绿**：S3b（default 顺序）1 项对抗（删除 S1 修复 → RED）；S3c（break 栈）1 项对抗（删除 S2 修复 → RED）。共 2 项 mutation RED→restore→GREEN。
- **门禁**：`go build ./...` ✓ / `go test ./tools/mql2go/...` ✓（全量，含现有 switch 测试 + golden/e2e 不破坏）/ `go test -race -count=3 ./tools/mql2go/...` ✓ / `go vet ./tools/mql2go/` ✓ / gofmt ✓ / `go run ./tools/check-file-lines --strict` 0 errors / `git diff --check` clean。
- **串行**：本单完成后报 `[施工完成:VM-COMPILER-SEMANTICS-3] @<hash>`，等 Devin CLI 独立复审。勿部署。

## 边界 / 不做

- 不改 for/while/do-while 的 break 语义（loop 无 switch value 需 POP，break patch 到 loop endPC 正确）。
- 不改 break statement 的 emit（`compile.go:450-451` 不变，仍 append breakJumps）。
- 不改 switch AST 解析（`compile_interp.go:475` 不变）。
- 不引入 switch 的 continue（MQL switch 无 continue，break only）。
- 不优化 switch 为 jump-table（当前 OP_DUP/OP_EQ/JMP_IF_FALSE 顺序比较保持，只修 default 顺序 + break POP）。
- 不改 default body 的 break 语义（default body 内 break 同 case body break，跳 popPC）。
- 若 default 在最后（常见 case），行为应与当前实现一致（default body 在末尾）——S1 修复不应破坏 default 在最后的正常场景。
