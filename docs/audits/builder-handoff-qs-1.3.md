# 施工提示词：QS-1.3 Python 函数局部作用域（复用 astCompiler.localScopes）

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S4 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后先过"施工完成自审"（D-012）修至全绿再交回；自报末行带 `[施工完成:QS-1.3] @<commit-hash>`（D-014）。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015）；**不更新任何交接层文件**。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

- registry `QS-1.3`；spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` v2 §3.3（设计 SSOT）。
- `backend/tools/mql2go/compile.go:282-289`：`Version=="python"` 的未解析变量赋值登记 `GlobalSlots` → 函数内 `x=1` 跨事件/跨函数污染（VM 单线程，非并发问题）。
- **决策方预核实的关键事实**（施工时复核，若不符 `[转交决策]`）：
  a. `self.x` 与裸 `x` 在 pyCompiler 阶段都剥前缀映射同名 `ExprAssignment{Name:"x"}`（`compile_py_assign.go:25-38,92-119,124-139`）；self 字段经 `selfVars` 收集进 `ir.Globals`（`compile_py.go:95-101`）→ GlobalSlots。
  b. `global_statement`/`nonlocal_statement` 在子集白名单（`compile_py_subset.go:202-203`）但 `compileStmt`（`compile_py_stmt.go:24-69`）无 case → 静默丢弃。
  c. `resolveVar`（`compile.go:257-297`）查找顺序 localScopes→GlobalSlots→python 隐式全局登记；`compileDecl`（`compile_expr.go:196-208`）已有"localScopes 非空→内层 scope 分配 nextLocalSlot"的既有路径。
  d. `nextLocalSlot`/`EventLocals`/`NumLocals` 由 `compile.go:312-316,342,351` 统一收尾——局部槽分配复用 `nextLocalSlot` 即自动正确。

## 设计 SSOT 声明

- 设计文档：spec v2 §3.3。契约：AGENTS.md §1 fail-closed；决策 D-009/D-012/D-014/D-015。

## 约束与目标（决策方已定的语义裁定）

- **函数内裸名赋值且全槽未命中 → 分配局部槽**（内层 localScopes + `nextLocalSlot`），不落 GlobalSlots。仅 `ExprAssignment`（`compile_expr.go:105-112`）与 `ExprCompoundAssign`（`:180-194`）入口需要处理。
- **已命中 GlobalSlots 的名字保持写全局**——含 self 字段（selfVars 收集）与模块级顶层赋值。`self.x`/裸 `x` 同槽的既有混同不修（超出边界，自报记一笔即可）。
- **`x += 1`/`x = f(x)` 等读到未声明名**：读取路径 `resolveVar` 本轮**不改**（仍走隐式全局 + blind spot）；仅赋值落点改局部。
- **`x += 1` 完全未声明（无 local 无 global）→ 编译期 `c.err` 报错**（fail-closed，Python 是 NameError；不得隐式建全局再 +=）。
- **`global`/`nonlocal` 语句 → `compileStmt` 加 case，`c.errorf` 明确报错**（子集不支持；静默丢弃在新语义下危险）。
- **顺序语义**：declare-on-assign（顺序分配），不做函数级预扫描；`x = x + 1` 且 x 未声明时 RHS 仍走隐式全局读（既有行为不变）。

## 边界 / 不做

- 不动 `resolveVar` 的读取路径与 MQL4/MQL5 行为；不动 `compileDecl`；不动 `selfVars` 收集。
- 不实现 `global`/`nonlocal` 语义，只做 fail-closed 报错。
- 不处理 `self.x` 与裸 `x` 同槽混同；不更新交接层文件。

## 施工指令

### S1 — 事实复核（只读）

- **目标**：逐项复核立项背景 a–d 四条事实，尤其确认 `compileStmt` 对 `global_statement` 确实无 case 返回 nil。
- **验证**：自报中逐条给出"属实/不符 + 代码行证据"；任何不符 → `[转交决策]`。

### S2 — `ExprAssignment` 赋值落点改局部

- **坐标**：`backend/tools/mql2go/compile_expr.go:105-112` `case interp.ExprAssignment:`。
- **落点**：抽 helper（与 `compileDecl` 的分配段同形）：
  ```go
  // resolveAssignTarget resolves the store slot for an assignment target.
  // QS-1.3: inside a Python function/event, assignment to an undeclared name
  // declares a function-local slot instead of leaking into GlobalSlots.
  // Names already in GlobalSlots (self fields, module globals) stay global.
  func (c *astCompiler) resolveAssignTarget(name string) (VarID, bool) {
      if c.bc.Version == "python" && len(c.localScopes) > 0 {
          for i := len(c.localScopes) - 1; i >= 0; i-- {
              if id, ok := c.localScopes[i][name]; ok {
                  return id, false
              }
          }
          if id, ok := c.bc.GlobalSlots[name]; ok {
              return id, true
          }
          scope := c.localScopes[len(c.localScopes)-1]
          scope[name] = VarID(c.nextLocalSlot)
          c.nextLocalSlot++
          return scope[name], false
      }
      return c.resolveVar(name)
  }
  ```
  `case ExprAssignment` 中 `c.resolveVar(e.Name)` 改为 `c.resolveAssignTarget(e.Name)`，emit 逻辑不变。
- **验证**：S4 测试。

### S3 — `ExprCompoundAssign` 未声明名 fail-closed + `global`/`nonlocal` 报错

- **坐标**：`compile_expr.go:180-194` `case interp.ExprCompoundAssign:`；`compile_py_stmt.go:24-69` `compileStmt`。
- **落点**：
  a. `ExprCompoundAssign`：在 `resolveVar` 前先查 localScopes/GlobalSlots（同 helper 思路）；`Version=="python"` 且 `localScopes` 非空且完全未命中 → `c.err`（若 nil）= 明确错误（如 `cannot use augmented assignment on undeclared name %q (Python: NameError)`），return。**已命中者照常 resolveVar**（self 字段回归不破）。
  b. `compileStmt` 加 `case "global_statement", "nonlocal_statement": c.errorf(n, "global/nonlocal is not supported in Python subset")`（复用 :66 的 errorf 模式）。
- **验证**：S4 测试。

### S4 — 测试（先红后绿 + mutation）

- **坐标**：新建 `backend/tools/mql2go/compile_py_locals_test.go`（复用 `compile_py_bool_test.go` 的 CompilePython+RunOnBar+GetGlobal 模式）。
- **落点**（最小集）：
  a. `def on_bar(self): x = 1` + `def helper(self): return x`——helper 读 x 不得读到 1（编译错误或返回 None 均可，自报注明实际行为）；先红：修复前读到 1。
  b. 跨事件隔离：on_bar#1 `x=1`；on_bar#2 未赋值读 x → 不为 1（None 或编译期拒绝，以 S2 实现为准注明）。
  c. 回归守卫：`self.count += 1` 跨两次 on_bar 累加为 2（GlobalSlots 路径不变）。
  d. 参数遮蔽：`def f(self, x): x = x + 1` 编译通过且参数槽语义不变。
  e. `global x` / `nonlocal x` 出现在函数内 → CompilePython 返回明确错误。
  f. `x += 1` 完全未声明 → CompilePython 返回明确错误。
  g. NumLocals/EventLocals 正确性：IR/bytecode 层断言局部槽数含新分配（或行为等价断言）。
- **验证**：a/e/f 先红后绿。

## 对抗证明（缺一即未完成）

- mutation：S2 的 `resolveAssignTarget` 退回 `resolveVar`（或删 `Version=="python"` 分支）→ S4a/S4b RED；restore → GREEN。
- mutation：删 S3b 的 `global_statement` case → S4e RED；restore → GREEN。

## 验收标准

- [ ] `cd backend && go build ./...` 通过
- [ ] `cd backend && go test -count=1 ./tools/mql2go/...` 全过
- [ ] `go test -race -count=3 ./tools/mql2go/...` 通过
- [ ] `go run ./tools/check-file-lines --strict` 本任务文件零新增警告
- [ ] `gofmt -l` / `go vet ./tools/mql2go/...` 零输出
- [ ] S1 四条事实复核 + 对抗证明 RED→restore→GREEN（附命令与输出）
- [ ] diff 无交接层文件、无范围外改动

## 施工完成自审（强制，D-012）

交付自报前必须完成并随报提交：
- [ ] 逐项重跑上方验收标准并贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖或重复基础设施
- [ ] 自审发现的缺陷已修复至全绿（自报列出发现项+修复项）
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报：改动文件清单 + 每条验收项证据（命令+关键输出）+ 施工完成自审记录 + S1 复核结果 + 遗留疑问。
**自报最后一行固定为** `[施工完成:QS-1.3] @<commit-hash>`（D-014：无此行 = 未交付，复审不启动）。
**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。**

[施工完成:QS-1.3] @<本任务最终commit>
