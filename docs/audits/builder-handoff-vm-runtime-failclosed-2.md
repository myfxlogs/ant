# 施工派工单：VM-RUNTIME-FAILCLOSED-2

> **角色纪律**：你是施工方，不是验收方。完成后保持 `🟦open（施工完成，待独立复审）`，由 Devin CLI 独立复审后才可变更为 `✅done`。禁止扩 scope、提交、push、部署。commit 用 `ANT_ROLE=builder` 前缀。

## 立项背景

**触发**：VM-RUNTIME-FAILCLOSED-1 验收后，独立复审发现 VM 仍有静默算术/栈/槽位失败路径——`arith`/`floorDiv` 除零返回 0、`OP_DUP`/`OP_SWAP` underflow no-op、越界 local/global slot 推 `NoneVal` 或静默丢弃。现有 `StackUnderflow` 只覆盖 `OP_POP`，未覆盖这些分支。

**证据链**（HEAD 实拍，2026-09-16 核实）：
- `vm_helpers.go:86-88` decimal 除零 → `return interp.DecimalVal(decimal.Zero)`（silent）
- `vm_helpers.go:105-107` 整数除零 → `return interp.IntVal(0)`（silent）
- `vm_helpers.go:110-112` 整数取模零 → `return interp.IntVal(0)`（silent）
- `vm_helpers.go:132-134` floorDiv decimal 除零 → `return interp.DecimalVal(decimal.Zero)`（silent）
- `vm_helpers.go:140-142` floorDiv 整数除零 → `return interp.IntVal(0)`（silent）
- `vm_execute.go:192-197` `OP_PUSH_VAR` 越界 → `vm.push(interp.NoneVal())`（silent 假值）
- `vm_execute.go:198-203` `OP_PUSH_GLOBAL` 越界 → `vm.push(interp.NoneVal())`（silent 假值）
- `vm_execute.go:204-207` `OP_STORE_VAR` 越界 → **既不报错也不 pop**（栈泄漏，比 spec 描述更严重）
- `vm_execute.go:208-213` `OP_STORE_GLOBAL` 越界 → `vm.pop()`（silent 丢弃，但至少栈平衡）
- `vm_execute.go:216-219` `OP_DUP` 空栈 → no-op（silent）
- `vm_execute.go:220-224` `OP_SWAP` <2 元素 → no-op（silent）

**不变量**：所有 `setStackError` 经 `runLoop` 顶部 `fatalError` 检查（`vm_execute.go:24` 既有）传播到 `VMRunner.OnBar`→`Engine.Run` fail-closed。本任务**不新增**传播路径，只在新分支调用既有 `setStackError`。

## 设计 SSOT

### S1：`arith` 除零/取模零 fail-closed

`vm_helpers.go:68-116` `arith` 函数：
- 整数除零（`:105-107`）：`if bi == 0 { vm.setStackError("integer division by zero"); return interp.IntVal(0) }`
- 整数取模零（`:110-112`）：`if bi == 0 { vm.setStackError("integer modulo by zero"); return interp.IntVal(0) }`
- decimal 除零（`:86-88`）：`if bd.IsZero() { vm.setStackError("decimal division by zero"); return interp.DecimalVal(decimal.Zero) }`
- decimal 取模零（`:91` 附近，`ad.Mod(bd)` 对零的行为）：`if bd.IsZero() { vm.setStackError("decimal modulo by zero"); return interp.DecimalVal(decimal.Zero) }`（在 `case "%":` 内加 guard，与除零同位置）
- **保留**：返回值仍为 0/Zero（不破坏栈形状，`runLoop` 顶部检查会终止执行，返回值不会被使用）

### S2：`floorDiv` 除零 fail-closed

`vm_helpers.go:127-148` `floorDiv` 函数：
- decimal 除零（`:132-134`）：`if bd.IsZero() { vm.setStackError("decimal floor division by zero"); return interp.DecimalVal(decimal.Zero) }`
- 整数除零（`:140-142`）：`if bi == 0 { vm.setStackError("integer floor division by zero"); return interp.IntVal(0) }`

### S3：`OP_DUP`/`OP_SWAP` underflow fail-closed

`vm_execute.go:216-224`：
- `OP_DUP`（`:216-219`）：`if len(vm.stack) > 0 { ... } else { vm.setStackError("OP_DUP underflow") }`
- `OP_SWAP`（`:220-224`）：`if len(vm.stack) >= 2 { ... } else { vm.setStackError("OP_SWAP underflow") }`

### S4：越界 slot 读写 fail-closed

`vm_execute.go:192-213`：
- `OP_PUSH_VAR`（`:192-197`）：越界 `else` 分支改 `vm.setStackError(fmt.Sprintf("OP_PUSH_VAR slot %d out of range (locals=%d)", ins.A, len(vm.locals)))`，**不再 push NoneVal**（栈不增长，`runLoop` 顶部终止）
- `OP_PUSH_GLOBAL`（`:198-203`）：同上，`vm.setStackError(fmt.Sprintf("OP_PUSH_GLOBAL slot %d out of range (globals=%d)", ins.A, len(vm.globals)))`
- `OP_STORE_VAR`（`:204-207`）：越界 `else` 分支改 `vm.setStackError(fmt.Sprintf("OP_STORE_VAR slot %d out of range (locals=%d)", ins.A, len(vm.locals)))`，**必须 `vm.pop()` 保持栈平衡**（修复 spec 漏列的栈泄漏）
- `OP_STORE_GLOBAL`（`:208-213`）：越界 `else` 分支加 `vm.setStackError(fmt.Sprintf("OP_STORE_GLOBAL slot %d out of range (globals=%d)", ins.A, len(vm.globals)))`，保留既有 `vm.pop()`

### S5：行为测试（7 个，落盘 `vm_audit_test.go`）

固定模式：MQL EA 触发目标分支 → 断言 error 返回 + `g_after` 不执行（fail-closed 证据）。

1. `TestVM_Audit_DivisionByZeroStopsExecution`：`int x = 10 / 0; g_after = 42;` → error 含 "division by zero" + `g_after == 0`
2. `TestVM_Audit_DecimalDivisionByZeroStopsExecution`：`double x = 10.0 / 0.0; g_after = 42;` → error + `g_after == 0`
3. `TestVM_Audit_ModuloByZeroStopsExecution`：`int x = 10 % 0; g_after = 42;` → error 含 "modulo" + `g_after == 0`
4. `TestVM_Audit_OpDupUnderflowStopsExecution`：直接构造 bytecode（空栈 + `OP_DUP`）→ error 含 "OP_DUP underflow"
5. `TestVM_Audit_OpSwapUnderflowStopsExecution`：直接构造 bytecode（1 元素 + `OP_SWAP`）→ error 含 "OP_SWAP underflow"
6. `TestVM_Audit_PushVarOutOfRangeStopsExecution`：直接构造 bytecode（`OP_PUSH_VAR` slot=99, locals=0）→ error 含 "OP_PUSH_VAR slot 99 out of range"
7. `TestVM_Audit_PushGlobalOutOfRangeStopsExecution`：直接构造 bytecode（`OP_PUSH_GLOBAL` slot=99, globals=0）→ error 含 "OP_PUSH_GLOBAL slot 99 out of range"

**测试风格**：MQL 测试用 `auditContext` helper（既有，`vm_audit_test.go:1035`）；直接 bytecode 测试用 `NewVM` + `SetContext` + 手动 `runLoop`（参考既有 `TestQS23_*` 模式）。

### S6：对抗证明（4 项突变，每项 RED→restore→GREEN）

1. **arith 整数除零**：恢复 `if bi == 0 { return interp.IntVal(0) }`（删 `setStackError`）→ `DivisionByZeroStopsExecution` RED（"should cause an error, got nil"）
2. **OP_DUP underflow**：恢复 `if len(vm.stack) > 0 { ... }`（删 `else` 分支）→ `OpDupUnderflowStopsExecution` RED
3. **OP_PUSH_VAR 越界**：恢复 `vm.push(interp.NoneVal())`（删 `setStackError`）→ `PushVarOutOfRangeStopsExecution` RED
4. **OP_SWAP underflow**：恢复 `if len(vm.stack) >= 2 { ... }`（删 `else` 分支）→ `OpSwapUnderflowStopsExecution` RED

## 约束与目标

- **范围**：仅 `vm_helpers.go`（`arith` + `floorDiv`）+ `vm_execute.go`（5 个 opcode 分支）+ `vm_audit_test.go`（7 测试）。零交接层改动。
- **不碰**：`OP_POP`（已有 `setStackError`）、`popN`（已有 `setStackError`）、`callBuiltin` error 传播（VM-RUNTIME-FAILCLOSED-1 已验收）、`runLoop` fatalError 检查（既有）。
- **REUSE**：`setStackError` @ `vm_helpers.go:31`；`runLoop` fatalError 检查 @ `vm_execute.go:24`；`auditContext` @ `vm_audit_test.go:1035`；`NewVM`/`SetContext` @ `vm.go`。
- **NEW**：7 个行为测试 @ `vm_audit_test.go`（追加，不新建文件）。
- **门禁**：build ✓ / `go test ./tools/mql2go/...` ✓ / `-race -count=3` ✓ / vet ✓ / gofmt ✓ / `check-file-lines --strict` 0 errors / `git diff --check` clean。
- **串行**：本单完成后报 `[施工完成:VM-RUNTIME-FAILCLOSED-2] @<hash>`，等 Devin CLI 独立复审。勿部署。

## 边界 / 不做

- 不修 `OP_STORE_VAR` 越界时的栈泄漏以外的任何"栈平衡"问题——本任务只覆盖 spec 列出的 4 类分支。
- 不新增 `OP_STORE_VAR`/`OP_STORE_GLOBAL` 越界的独立行为测试（spec 未列；`OP_PUSH_*` 测试已覆盖 slot 越界语义，`OP_STORE_*` 的栈平衡修复由 `OP_DUP`/`OP_SWAP` underflow 测试间接覆盖 underflow 传播）。
- 不改 `decimal.Mod` 对零的实际行为（Go decimal 库可能 panic 或返回零——本任务只加 guard 不改库行为）。
- 不碰 Python `//` 语义（QS-1.4 已验收 `floorDiv` 的 Python 路径）。
