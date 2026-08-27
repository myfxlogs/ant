# 施工提示词：VM-AUDIT-2026-08-27 批次 2（P2/P3 VM 鲁棒性加固）

## 立项背景
- 触发：VM 管线审计批次 1 已验收通过（`25930b36`，✅done）；本批为防御性加固，3 个防御性 bug。
- 证据链：spec `docs/spec/vm-audit-2026-08-27-spec.md` §3「-3」「-4」「-5」；registry `VM-AUDIT-2026-08-27-3/4/5`。
- BUG-3：`executeCallUser` 内联循环缺 MaxStackDepth 检查 → 用户函数内栈可增长到 MaxTicks=10M（~80-160MB 尖峰）才被兜底。
- BUG-4：`OP_CALL_BUILTIN` 在 `popN` 栈下溢后仍执行 `callBuiltin` → builtin 用部分参数执行产生副作用。
- BUG-5：`VMLiveSession.dispatch` default 分支在 `bctx != nil` 时误当 bar 处理未知请求类型。

## 设计 SSOT 声明
- 唯一真相源：`docs/spec/vm-audit-2026-08-27-spec.md` §3 对应章节。本提示词与 spec 冲突时以 spec 为准。

## 约束与目标
- BUG-3/4 是 defense-in-depth：正常 MQL 编译器生成的 bytecode 不会触发，测试需直接构造 bytecode（不走编译器）。
- 每步必须有真实 mutation RED→restore→GREEN 对抗证明（nil panic / 另一条错误 / callback-only / "任意 error" 均不算证据）。
- REUSE 优先：`MaxStackDepth` 常量（`vm.go:20`）、fatalError 检查模式（`vm_execute.go:123-125`）、错误响应格式（`vm_live_session.go:169`）。

## 边界 / 不做
- 不重构 VM 执行模型、不部署、不 push、不改 git config、禁 `--no-verify`。
- 不改 `popN` 本身的行为（只加 callBuiltin 前的 early return）。
- 不做批次 3 的任何内容（-6/-7/-8）。
- 不动批次 1 已验收的代码（`vm_live_dispatch.go` Python 路径、`vm_live_session.go` Python 路径、`vm.go:192` fatalError 重置）。

## 正文

### S1：executeCallUser 加 MaxStackDepth 检查（BUG-3）
- 目标：防止用户函数内联循环栈增长绕过 MaxStackDepth=4096 限制。
- 坐标：`backend/tools/mql2go/vm_execute.go:332-358` `executeCallUser` 内联循环。
- 落点：在 `vm.ticks > MaxTicks` 检查（`:348-352`）之后、`vm.execute(ins2)`（`:353`）之前加：
  ```go
  if len(vm.stack) > MaxStackDepth {
      vm.locals = oldLocals
      vm.callDepth--
      return fmt.Errorf("strategy exceeded max stack depth (%d)", len(vm.stack))
  }
  ```
  注意：必须恢复 `vm.locals = oldLocals` + `vm.callDepth--`（与其他错误退出路径一致：`:349-351` MaxTicks、`:354-356` execute error）。

### S2：OP_CALL_BUILTIN 在 popN 栈下溢后 early return（BUG-4）
- 目标：防止栈下溢时 builtin 用部分参数执行产生副作用。
- 坐标：`backend/tools/mql2go/vm_execute.go:116-125` `OP_CALL_BUILTIN` case。
- 落点：在 `args := vm.popN(nArgs)`（`:118`）之后、`result := vm.callBuiltin(ins.A, args)`（`:119`）之前加 fatalError 检查，提前 return 跳过 callBuiltin：
  ```go
  case OP_CALL_BUILTIN:
      nArgs := int(ins.B)
      args := vm.popN(nArgs)
      if vm.fatalError != "" {          // NEW: popN 栈下溢 → 不执行 builtin
          return fmt.Errorf("VM fatal: %s", vm.fatalError)
      }
      result := vm.callBuiltin(ins.A, args)
      vm.push(result)
      if vm.fatalError != "" {          // EXISTING: callBuiltin 内部错误（:123-125 保留）
          return fmt.Errorf("VM fatal: %s", vm.fatalError)
      }
  ```
  注意：`:123-125` 已有的 fatalError 检查（callBuiltin 返回后）保留不变。新增的检查是 popN 后、callBuiltin 前的 early return。

### S3：VMLiveSession.dispatch default 分支直接返回 error（BUG-5）
- 目标：防止未知请求类型 + non-nil BarContext 被误当 bar 事件执行。
- 坐标：`backend/internal/connect/strategy/vm_live_session.go:164-171` default 分支。
- 落点：删除 `if bctx != nil` 条件分支，整个 default 直接返回 error：
  ```go
  default:
      return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("unknown request type: %s", req.GetRequestType())}
  ```
  注意：`import "fmt"` 已存在于文件中（`:5`）。

### T1：对抗测试 TestVM_CallUserStackDepthLimit（BUG-3）
- 直接构造 bytecode（不走 MQL 编译器——编译器会平衡 push/pop，正常 MQL 无法触发此场景）——构造一个 user function（有 `Funcs` entry + `OP_CALL_USER` 调用），函数体内是一个循环不断 `OP_PUSH_CONST` 但不 pop，超过 MaxStackDepth=4096 → 调用 → 应返回 "strategy exceeded max stack depth" error。
- 突变：删除 S1 新增的 stack 检查 → 测试 RED（栈增长到 MaxTicks=10M 才停止，错误信息是 "strategy exceeded instruction limit" 而非 "max stack depth"）→ 恢复 → GREEN。
- 注意：这是 defense-in-depth 测试——正常 MQL 编译器生成的 bytecode push/pop 平衡，不会触发此路径。此测试针对编译器 bug 或恶意 bytecode 注入场景。

### T2：对抗测试 TestVM_PopNStackUnderflowStopsBuiltin（BUG-4）
- 直接构造 bytecode（不走 MQL 编译器——编译器会保证参数数量正确）——构造一条 `OP_CALL_BUILTIN` 指令，`ins.B=3`（需要 3 参数），但栈上只有 1 个值 → 调用 → 验证 builtin handler 不被调用（用 mock builtin 注册一个计数器）+ 返回 error 含 "stack error"。
- 突变：删除 S2 新增的 fatalError 检查（popN 后的 early return）→ builtin 被调用（计数器增加）→ RED → 恢复 → GREEN。
- 注意：这是 defense-in-depth 测试——正常 MQL 编译器生成的 bytecode 参数数量与 builtin 签名匹配，不会触发此路径。此测试针对编译器 bug 或恶意 bytecode 注入场景。

### T3：对抗测试 TestVMLiveSession_UnknownRequestType（BUG-5）
- 构造 `ExecuteLiveRequest` with unknown RequestType（如 `REQUEST_TYPE_UNSPECIFIED`）+ non-nil BarContext → 验证返回 `Success: false` + Error 含 "unknown request type"（不是执行 bar 事件返回 success）。
- 突变：恢复旧 default 分支（`if bctx != nil { resp = vmHandleBar(...) }`）→ 测试 RED（返回 `Success: true` 而非 `false`）→ 恢复 → GREEN。

## 验收标准
1. 先红后绿：T1/T2/T3 必须先 RED（突变）再 GREEN（恢复），证据留测试文件内。
2. 机检五件套：`go build ./...` / `gofmt -l .` / `go vet ./...` / `go test ./tools/mql2go/... -count=1` / `go test ./internal/connect/strategy -count=1`。
3. race×3：`go test -race ./tools/mql2go/... -count=1` ×3 + `go test -race ./internal/connect/strategy -count=1` ×3。
4. check-file-lines：`cd backend && go run ./tools/check-file-lines --strict`（0 errors）。
5. `git diff --check` clean。
6. 收工：更新 registry 三条目（-3/-4/-5）状态为 `🟦open（施工完成，待独立复审）` + STATE.md 施工表；不得自标 ✅done。

## 尾部
勿部署，停手等 Devin CLI 复审。禁 `--no-verify`。
