# Builder Handoff — VM 返工批第三批：VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD（编译器正确性）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 设计 SSOT：`docs/spec/vm-revert-redo-spec.md`（D1-D4 不变）
> registry spec：`:146`（VM-COMPILER-SEMANTICS-1）、`:147`（BT-FUNC-ENTRYPC-FWD）
> 基线 HEAD：`b73cc98e`（2026-08-26，第二批 VM-TRADE-CONTEXT-1/2 验收通过后）

## 0. 立项背景

**触发**：D-REVERT-SCOPE-DRIFT-001 第三批——revert `830b2c79` 导致 VM-COMPILER-SEMANTICS-1 和 BT-FUNC-ENTRYPC-FWD 修复代码丢失，registry 状态从"施工完成待复审"降级回"🟦open（待施工）"。

**证据链**（基于 HEAD `b73cc98e` 实测）：

VM-COMPILER-SEMANTICS-1：
- `grep -rn "ClassTypes" tools/mql2go/` → 0 匹配（应存在于 `bytecode.go` Bytecode struct + `interp/ir.go` IR struct）
- `grep -rn "methodBuiltinName" tools/mql2go/` → 0 匹配（第二批已加 `registerMethodBuiltinWithObj` 部分覆盖，但 registry spec 要求的独立 `methodBuiltinName` 函数不存在）
- `compile_interp.go:447-469` `compileDeclaration` 只处理第一个 `init_declarator`，遇到就 `return`，不遍历多变量
- `compile_expr.go:487-489` `binaryOp` default 分支 `return OP_ADD`（静默 fallback，不报 error）
- `compile_expr.go:507-509` `compoundAssignOp` default 分支 `return OP_ADD`（同上）
- `compile_expr.go:78` `ExprUnary` default 分支只 `AddBlindSpot`，不设 `c.err`
- `compile_loops.go:105-167` `compileSwitch` 每个 case body 后无条件 `emitJump(OP_JMP, 0)` → 无 fallthrough 支持
- `compile_interp.go:365-402` `compileFor` 只处理 `compound_statement` body，无 single-statement body
- `compile_interp.go:404-415` `compileWhile` 同上
- `compile_interp.go:556-567` `compileDoWhile` 同上
- `vm.go:220-237` `initGlobals` 只处理 `IsArray` 全局变量，不处理 class/struct 类型全局变量

BT-FUNC-ENTRYPC-FWD：
- `grep -rn "patchUserCalls" tools/mql2go/` → 0 匹配
- `grep -rn "userCallPatch" tools/mql2go/` → 0 匹配
- `compile_expr.go:361` `compileCall` 直接写 `fn.EntryPC`（可能为 stale marker PC），无 -1 占位符
- `compile.go:133-143` `CompileAST` 末尾只有 `patchJumps()`，无 `patchUserCalls()` 调用
- `compile.go:56-75` `userFuncNames` 直接 append 未排序（map 迭代非确定性）

**已存在不需重做的修复**：
- ✅ `compile_interp_expr.go:269-286` field/subscript lvalue IsAssign 处理（子修复点①）
- ⚠️ `builtins.go:489-502` `registerMethodBuiltinWithObj`（第二批新增，部分覆盖子修复点④的 CTrade 命名空间解析）

**设计 SSOT 声明**：`docs/spec/vm-revert-redo-spec.md`（D1-D4 设计决策不变）

**约束与目标**：
- 基于 registry 中的原始修复 spec 重新施工，不做架构变更（D3）
- 每批独立验收（D4）
- revert 不可逆，不尝试恢复被 revert 的代码（D1）

**边界/不做**：
- 不改写历史审计事实
- 不改 proto/gen 文件
- 不改第二批已验收的 `registerMethodBuiltinWithObj`（子修复点④由它覆盖，不再加 `methodBuiltinName`）

---

## 1. VM-COMPILER-SEMANTICS-1（S1-S7）

### S1：compileDeclaration 多变量声明 + 无初始化值 declarator

**目标**：`compileDeclaration` 遍历所有 declarator（`init_declarator`/`declarator`/`array_declarator`），每个生成 `ExprDecl`；无初始化值用 `zeroValueForType` 零值；多变量时返回 `ExprSeq`。

**当前代码**（`compile_interp.go:447-469`）：
```go
func (c *compiler) compileDeclaration(n *sitter.Node) *interp.Statement {
    for i := 0; i < int(n.NamedChildCount()); i++ {
        child := n.NamedChild(i)
        if child.Type() == "init_declarator" {
            name := c.findIdent(child)
            if name == "" { continue }
            valExpr := c.findInitValue(child, name)
            if valExpr != nil {
                return &interp.Statement{  // ❌ 只处理第一个就 return
                    Kind: interp.StmtExpr,
                    Expr: &interp.Expr{
                        Kind: interp.ExprDecl,
                        Name: name,
                        Args: []interp.Expr{*c.compileExpr(valExpr)},
                    },
                }
            }
        }
    }
    return nil
}
```

**施工要求**：
1. 遍历所有 `init_declarator` 和 `declarator` 子节点
2. 每个 declarator 生成一个 `ExprDecl`：
   - 有初始化值：`ExprDecl{Name: name, Args: []Expr{*compileExpr(valExpr)}}`
   - 无初始化值：`ExprDecl{Name: name, Args: []Expr{zeroValueExpr(type)}}`（需从 declarator 推断类型）
3. 单个 declarator：返回单个 `StmtExpr`（保持兼容）
4. 多个 declarator：返回 `StmtExpr` 包裹 `ExprSeq`（`Expr.Kind = ExprSeq`，`Expr.Args = []Expr{decl1, decl2, ...}`）
5. local array 声明（`array_declarator`）：显式编译失败 `c.err = fmt.Errorf("local arrays not supported: %s", name)`

**落点**：`compile_interp.go:447-469` 重写

**注意**：需要检查 `interp.ExprSeq` 是否存在于 `interp/ir.go`。如果不存在，需要新增 `ExprSeq` ExprKind。`zeroValueForType` 已存在于 `vm.go:240`，但它是 VM 层返回 `interp.Value`；编译层需要返回 `interp.Expr`——可以用 `ExprInt(0)` / `ExprDecimal(0)` 等，或新增 `ExprZero(typeName)` helper。

### S2：binaryOp/compoundAssignOp/compileUnary 不支持运算符显式 error

**目标**：不支持的运算符不再静默 fallback 为 `OP_ADD`，改为 `c.err = fmt.Errorf(...)` 编译失败。

**施工坐标**：
- `compile_expr.go:487-489` `binaryOp` default 分支：
  ```go
  default:
      c.bc.Coverage.AddBlindSpot("binary op: " + op)
      return OP_ADD  // ❌ 改为 error
  ```
  改为：
  ```go
  default:
      if c.err == nil {
          c.err = fmt.Errorf("unsupported binary operator: %s", op)
      }
      return OP_ADD // 仍返回值避免 nil panic，但 c.err 会导致 CompileAST 返回 error
  ```

- `compile_expr.go:507-509` `compoundAssignOp` default 分支：
  ```go
  default:
      return OP_ADD  // ❌ 改为 error
  ```
  改为：
  ```go
  default:
      if c.err == nil {
          c.err = fmt.Errorf("unsupported compound assign operator: %s", op)
      }
      return OP_ADD
  ```

- `compile_expr.go:77-79` `ExprUnary` default 分支：
  ```go
  default:
      c.bc.Coverage.AddBlindSpot("unary op: " + e.Op)  // ❌ 改为 error
  ```
  改为：
  ```go
  default:
      if c.err == nil {
          c.err = fmt.Errorf("unsupported unary operator: %s", e.Op)
      }
  ```

### S3：compileSwitch fallthrough 支持

**目标**：`compileSwitch` 支持 C 风格 fallthrough——case body 不以 `break` 结尾时，控制流落入下一个 case 的 body（跳过下一个 case 的比较）。

**当前代码**（`compile_loops.go:105-167`）：每个 case body 后无条件 `endJumps = append(endJumps, c.emitJump(OP_JMP, 0))`，阻止 fallthrough。

**施工要求**：
1. 检查 case body 最后一个 statement 是否是 `StmtBreak`
2. 如果是 `StmtBreak`：保持当前行为（emit JMP to end）
3. 如果不是 `StmtBreak`：不 emit JMP to end，控制流自然落入下一个 case 的 body 起点
4. 需要调整 `jmpFalseIndices` patch 逻辑：fallthrough case 的 `JMP_IF_FALSE` 目标应是下一个 case 的 **body** 起点（而非比较起点），因为 fallthrough 时不需要再比较

**落点**：`compile_loops.go:105-167` 重写 `compileSwitch`

**关键设计**（参考 registry spec）：
- 可新增 `compileSwitchCase` helper 函数处理单个 case
- 或在 `compileSwitch` 内联处理 fallthrough 逻辑
- fallthrough 检测：`len(sc.Body) > 0 && sc.Body[len(sc.Body)-1].Kind == interp.StmtBreak` → 有 break；否则 fallthrough

### S4：for/while/do-while single-statement body

**目标**：循环语句支持非 `compound_statement` 的 single-statement body（如 `for(...) doSomething();`）。

**当前代码**：
- `compile_interp.go:365-402` `compileFor`：`case nodeCompoundStatement: stmt.Body = c.compileBlock(child)` — 无 else 分支
- `compile_interp.go:404-415` `compileWhile`：`else if child.Type() == nodeCompoundStatement: stmt.Body = c.compileBlock(child)` — 无 else 分支
- `compile_interp.go:556-567` `compileDoWhile`：`if child.Type() == nodeCompoundStatement: stmt.Body = c.compileBlock(child)` — 无 else 分支

**施工要求**：参考 `compileIf`（`compile_interp.go:355-360`）的 single-statement 处理：
```go
} else if s := c.compileStmt(child); s != nil {
    if stmt.Body == nil {
        stmt.Body = []interp.Statement{*s}
    }
}
```

**落点**：
- `compile_interp.go:365-402` `compileFor` — 在 `case nodeCompoundStatement` 后加 else 分支
- `compile_interp.go:404-415` `compileWhile` — 在 `else if child.Type() == nodeCompoundStatement` 后加 else 分支
- `compile_interp.go:556-567` `compileDoWhile` — 在 `if child.Type() == nodeCompoundStatement` 后加 else 分支

### S5：initGlobals ValClass 初始化 + IR.ClassTypes + Bytecode.ClassTypes

**目标**：`initGlobals` 将 struct/class 类型的全局变量初始化为 `ValClass{Class: &ClassInstance{...}}`，使 `OP_SET_FIELD` 能正确工作。

**当前代码**：
- `vm.go:220-237` `initGlobals` 只处理 `IsArray` 全局变量
- `bytecode.go:116-165` `Bytecode` struct 无 `ClassTypes` 字段
- `interp/ir.go:85-90` `IR` struct 无 `ClassTypes` 字段

**施工要求**：

① 新增 `IR.ClassTypes` 字段（`interp/ir.go:90` 后）：
```go
ClassTypes map[string]bool // class/struct type names (for ValClass initialization)
```

② 在 `compile_interp.go:47-92` `compile` 函数中填充 `IR.ClassTypes`：
- 从 `knownClasses`（:55）和 `isBuiltinClass`（:65）收集所有 class/struct 类型名
- `ir.ClassTypes = knownClasses`（需合并 builtin class）

③ 新增 `Bytecode.ClassTypes` 字段（`bytecode.go:164` 后，`Coverage` 前）：
```go
// ClassTypes lists class/struct type names for ValClass global initialization.
// VM-COMPILER-SEMANTICS-1: initGlobals uses this to initialize class-typed globals.
ClassTypes map[string]bool
```

④ `CompileAST`（`compile.go:12-36`）将 `ir.ClassTypes` 传入 `bc.ClassTypes`：
```go
ClassTypes: ir.ClassTypes,
```

⑤ `vm.go:220-237` `initGlobals` 加 class 类型处理：
```go
func (vm *VM) initGlobals() {
    vm.globals = make([]interp.Value, len(vm.bc.GlobalSlots))
    for _, decl := range vm.bc.GlobalDecls {
        slot, ok := vm.bc.GlobalSlots[decl.Name]
        if !ok { continue }
        if decl.IsArray && decl.ArraySize > 0 {
            // existing array init...
            continue
        }
        // VM-COMPILER-SEMANTICS-1: class/struct type → ValClass
        if vm.bc.ClassTypes[decl.Type] {
            vm.globals[slot] = interp.Value{
                Kind: interp.ValClass,
                Class: &interp.ClassInstance{
                    Fields: make(map[string]interp.Value),
                },
            }
            continue
        }
        // Default: zero value
        vm.globals[slot] = zeroValueForType(decl.Type)
    }
}
```

⑥ 序列化 `MarshalBytecode`/`UnmarshalBytecode` 同步增加 `ClassTypes`：
- `bytecode_cache.go:130-143` `MarshalBytecode`：在 Enums 后写入 ClassTypes（sorted keys 保证确定性）
- `bytecode_cache.go:205-218` `UnmarshalBytecode`：在 Enums 后读取 ClassTypes
- 新增 `unmarshalClassTypes` helper（参考 `unmarshalEnums` :483 模式）
- **注意**：写入位置必须在 Enums 之后、trailing bytes 检查之前

### S6：序列化确定性（sorted keys）

**目标**：`ClassTypes` 序列化时按 sorted keys 写入，保证确定性。

**施工要求**：
- 新增 `sortedClassTypeNames` helper（参考 `unmarshalEnums` 的 sorted write 模式）
- `MarshalBytecode` 写入 ClassTypes 时先 `sort.Strings(keys)` 再遍历

### S7：对抗证明

7 项突变，每项删关键行→目标测试 RED→恢复 GREEN：

1. 删 `initGlobals` ValClass 初始化 → `TestVM_Audit_MQLFieldAssignment_VMBehavior` RED（readback=0 want 42）
2. 删 `compileDeclaration` 多变量 ExprSeq（只返回首个）→ `TestVM_Audit_MultiVariableDeclaration` RED
3. 删 `binaryOp` error fallback（恢复 `return OP_ADD` 无 error）→ `TestVM_Audit_UnsupportedBitwiseOperatorRejected` RED（err=nil）
4. 删 `compileSwitch` fallthrough（恢复无条件 JMP to end）→ `TestVM_Audit_SwitchFallthrough` RED
5. 删 `compileFor` single-statement body → `TestVM_Audit_ForLoopSingleStatementBody` RED
6. 删 `compileWhile` single-statement body → `TestVM_Audit_WhileLoopSingleStatementBody` RED
7. 删 `ClassTypes` 序列化 → `TestCompileMQLCached_ClassTypesRoundTrip` RED（cache hit 后 ClassTypes 为空）

---

## 2. BT-FUNC-ENTRYPC-FWD（S8-S12）

### S8：compileCall 占位符 + userCallPatch 记录

**目标**：`compileCall` 发出 `OP_CALL_USER` 时 operand A 写 -1 占位符，同时记录 `userCallPatch{instruction, callee}`。

**当前代码**（`compile_expr.go:354-363`）：
```go
func (c *astCompiler) compileCall(e *interp.Expr) {
    if fn, ok := c.bc.Funcs[e.Name]; ok {
        for i := range e.Args {
            c.compileExpr(&e.Args[i])
        }
        c.emit(OP_CALL_USER, fn.EntryPC, int32(fn.NumParams), 0)  // ❌ 直接写 fn.EntryPC
        return
    }
    // ...
}
```

**施工要求**：
1. `compile.go:151-159` `astCompiler` struct 新增字段：
   ```go
   userCallPatches []userCallPatch // BT-FUNC-ENTRYPC-FWD: pending user-call relocations
   ```
2. `compile.go:168` 附近新增 struct：
   ```go
   type userCallPatch struct {
       instruction int32  // index of OP_CALL_USER in bc.Code
       callee      string // function name to patch
   }
   ```
3. `compile_expr.go:361` 改为：
   ```go
   // BT-FUNC-ENTRYPC-FWD: emit placeholder -1, patch later in patchUserCalls
   instrIdx := c.emit(OP_CALL_USER, -1, int32(fn.NumParams), 0)
   c.userCallPatches = append(c.userCallPatches, userCallPatch{instruction: instrIdx, callee: e.Name})
   ```

### S9：patchUserCalls 函数

**目标**：所有用户函数 body 编译完成后，统一将 OP_CALL_USER 的 operand A patch 为最终 EntryPC。

**施工要求**：
1. `compile.go` 新增函数（在 `patchJumps` 附近）：
   ```go
   // patchUserCalls resolves all pending user-call relocations.
   // BT-FUNC-ENTRYPC-FWD: called after all user function bodies are compiled,
   // so all EntryPCs are final.
   func (c *astCompiler) patchUserCalls() error {
       for _, p := range c.userCallPatches {
           fn, ok := c.bc.Funcs[p.callee]
           if !ok {
               return fmt.Errorf("patchUserCalls: unknown function %s", p.callee)
           }
           if fn.EntryPC < 0 {
               return fmt.Errorf("patchUserCalls: callee %s has invalid EntryPC %d", p.callee, fn.EntryPC)
           }
           c.bc.Code[p.instruction].A = fn.EntryPC
       }
       return nil
   }
   ```

### S10：CompileAST 末尾调用 patchUserCalls

**目标**：`CompileAST` 在 `patchJumps()` 之后调用 `patchUserCalls()`。

**当前代码**（`compile.go:132-143`）：
```go
c.emit(OP_HALT, 0, 0, 0)
c.patchJumps()
if c.err != nil {
    return nil, c.err
}
return c.bc, nil
```

**施工要求**：
```go
c.emit(OP_HALT, 0, 0, 0)
c.patchJumps()
// BT-FUNC-ENTRYPC-FWD: patch user-call placeholders after all bodies compiled
if err := c.patchUserCalls(); err != nil {
    if c.err == nil {
        c.err = err
    }
}
if c.err != nil {
    return nil, c.err
}
return c.bc, nil
```

### S11：sort.Strings(userFuncNames) 确定性布局

**目标**：`userFuncNames` 排序后编译，保证确定性布局。

**当前代码**（`compile.go:56-75`）：
```go
userFuncNames := make([]string, 0, len(ir.Funcs))
for name, fn := range ir.Funcs {
    if isEventFunction(name) { continue }
    // ...
    userFuncNames = append(userFuncNames, name)
}
for _, name := range userFuncNames {
    fn := ir.Funcs[name]
    c.compileUserFuncBody(name, fn)
}
```

**施工要求**：
1. `compile.go:3` import 加 `"sort"`
2. 在第一个 for 循环后、第二个 for 循环前加：
   ```go
   sort.Strings(userFuncNames) // BT-FUNC-ENTRYPC-FWD: deterministic layout
   ```

### S12：对抗证明

2 项突变：

1. **Mutation 1**（还原 `compileCall` 为直接写 `fn.EntryPC`）：
   - 改 `compile_expr.go` 的 `c.emit(OP_CALL_USER, -1, ...)` 回 `c.emit(OP_CALL_USER, fn.EntryPC, ...)`
   - 删 `userCallPatches` append
   - → `TestVM_Audit_UserToUserForwardReference` RED（stale marker PC → max call depth or wrong result）
   - → `TestVM_Audit_UserToUserForwardReference_Structure` RED（operand != final EntryPC）

2. **Mutation 2**（注释 `c.patchUserCalls()` 调用，保留 -1 占位符）：
   - 在 `CompileAST` 注释掉 `c.patchUserCalls()` 调用
   - → `TestVM_Audit_UserToUserForwardReference` RED（operand=-1 → invalid jump）
   - → `TestVM_Audit_UserToUserForwardReference_Structure` RED（unresolved placeholder A=-1）

**关键纠偏**（来自 registry spec）：测试必须用 `aaa_caller`/`zzz_callee` 命名（caller 字母序在前故 body 先编译），不能用 `caller`/`callee`（callee 字母序在前会先编译 body，bug 不触发）。

---

## 3. 测试文件

### 新建测试文件

**文件**：`backend/tools/mql2go/vm_compiler_semantics_redo_test.go`

**测试函数**（VM-COMPILER-SEMANTICS-1，7 个行为测试）：
1. `TestVM_Audit_MQLFieldAssignment_VMBehavior` — CTrade 全局变量 + SetExpertMagicNumber + readback
2. `TestVM_Audit_MultiVariableDeclaration` — `int a = 1, b = 2;` 两个变量都初始化
3. `TestVM_Audit_UninitializedLocalDeclaration` — `int x;` 局部变量零值初始化
4. `TestVM_Audit_UnsupportedBitwiseOperatorRejected` — `a | b` 编译失败
5. `TestVM_Audit_SwitchFallthrough` — case 无 break 时 fallthrough 到下一 case
6. `TestVM_Audit_ForLoopSingleStatementBody` — `for(...) doSomething();` 无花括号
7. `TestVM_Audit_WhileLoopSingleStatementBody` — `while(cond) doSomething();` 无花括号

**测试函数**（BT-FUNC-ENTRYPC-FWD，2 个行为测试 + 1 个结构测试）：
8. `TestVM_Audit_UserToUserForwardReference` — `OnTick→aaa_caller→zzz_callee`，100 次迭代断言 g_result==42
9. `TestVM_Audit_UserToUserForwardReference_Structure` — 断言每个 OP_CALL_USER operand == callee 最终 EntryPC 且目标非 OP_ENTER_FUNC marker

**测试函数**（序列化，1 个）：
10. `TestCompileMQLCached_ClassTypesRoundTrip` — 编译含 CTrade 全局变量的 MQL → marshal → unmarshal → ClassTypes 非空

**REUSE**：
- `CompileMQL` @ `interp_runner.go:133`
- `NewVM` @ `vm.go`
- `VMRunner.Bytecode()` / `VMRunner.GetGlobal` @ `interp_runner.go`
- `zeroValueForType` @ `vm.go:240`
- `isBuiltinClass` @ `preprocess.go:65`
- `unmarshalEnums` 模式 @ `bytecode_cache.go:483`（参考写法，不直接调用）

---

## 4. 验收

### 机检五件套
```bash
cd backend && go build ./...
cd backend && go test ./tools/mql2go/... -count=1
cd backend && go test -race ./tools/mql2go/... -count=1  # ×3 次
cd backend && go vet ./...
cd backend && go run ./tools/check-file-lines --strict
git diff --check
```

### 对抗证明
- VM-COMPILER-SEMANTICS-1：7 项 RED→restore→GREEN
- BT-FUNC-ENTRYPC-FWD：2 项 RED→restore→GREEN
- 总计 9 项

### 文件清单（预期改动）
| 文件 | 改动类型 |
|------|----------|
| `tools/mql2go/compile_interp.go` | S1 (compileDeclaration) + S4 (for/while/do-while single-stmt) + S5 (IR.ClassTypes 填充) |
| `tools/mql2go/compile_expr.go` | S2 (binaryOp/compoundAssignOp/unary error) + S8 (compileCall 占位符) |
| `tools/mql2go/compile_loops.go` | S3 (compileSwitch fallthrough) |
| `tools/mql2go/compile.go` | S8 (userCallPatch struct + astCompiler 字段) + S9 (patchUserCalls) + S10 (CompileAST 调用) + S11 (sort.Strings) |
| `tools/mql2go/vm.go` | S5 (initGlobals ValClass) |
| `tools/mql2go/bytecode.go` | S5 (Bytecode.ClassTypes 字段) |
| `tools/mql2go/interp/ir.go` | S5 (IR.ClassTypes 字段) |
| `tools/mql2go/bytecode_cache.go` | S6 (ClassTypes 序列化) |
| `tools/mql2go/vm_compiler_semantics_redo_test.go` | 新建（10 个测试） |

### REUSE/NEW 标记
- **REUSE**: `CompileMQL`/`NewVM`/`zeroValueForType`/`isBuiltinClass`/`unmarshalEnums` 模式
- **NEW**: `IR.ClassTypes`/`Bytecode.ClassTypes` 字段；`userCallPatch` struct；`patchUserCalls` 函数；`sortedClassTypeNames` helper（如需）；`ExprSeq` ExprKind（如不存在）

---

## 5. 固定尾部

**勿部署，停手等 Devin CLI 复审。**
**禁 `--no-verify`。**
**收工更新 registry + handover + STATE.md。**
