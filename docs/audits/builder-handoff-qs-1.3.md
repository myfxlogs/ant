# 施工提示词：QS-1.3 Python 函数局部作用域（复用 astCompiler.localScopes）

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S4 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后先过"施工完成自审"（D-012）修至全绿再交回；自报末行带 `[施工完成:QS-1.3] @<commit-hash>`（D-014）。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015）；**不更新任何交接层文件**。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**
>
> **修正 v2（2026-09-16）**：施工方 S1 复核发现两处提示词事实错误并已 `[转交决策]`，决策方独立核实后全部采信——①`global`/`nonlocal` 在 CST 黑名单而非白名单（S3b 死代码移除）；②批准 Option B：判定谓词 `GlobalSlots` → `GlobalDecls`（关闭编译顺序导致的残余泄漏洞）。行内 ~~删除线~~ 为修正痕迹。
>
> **修正 v3（2026-09-16，决策方复审退回）**：独立复审发现 v2 处方引入新回退——`compileFor`（`compile_loops.go:8,53`）为 MQL `for(int i;;)` 词法域 `pushScope/popScope`，Python `for` 体内新局部按"内层 scope"分配会随循环域消亡，循环后读出 ValNone（Python 无块作用域，应为函数域持久）。裁定：**Python 新局部一律分配到函数基域 `localScopes[0]`**（不变量：函数/事件编译首动作是 pushScope，故 python 代码在函数内时 `localScopes[0]` 即函数域，其上只有循环/块域）。`resolveAssignTarget` 与 `compileDecl` 同改——后者顺带修复既有"for 循环变量出循环即死"偏差（`for i in range` 的 `i` 经 `ExprDecl`→`compileDecl` 绑定）。

## 立项背景（触发 + 证据链）

- registry `QS-1.3`；spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` v2 §3.3（设计 SSOT）。
- `backend/tools/mql2go/compile.go:282-289`：`Version=="python"` 的未解析变量赋值登记 `GlobalSlots` → 函数内 `x=1` 跨事件/跨函数污染（VM 单线程，非并发问题）。
- **决策方预核实的关键事实**（施工时复核，若不符 `[转交决策]`）：
  a. `self.x` 与裸 `x` 在 pyCompiler 阶段都剥前缀映射同名 `ExprAssignment{Name:"x"}`（`compile_py_assign.go:25-38,92-119,124-139`）；self 字段经 `selfVars` 收集进 `ir.Globals`（`compile_py.go:95-101`）→ GlobalSlots。
  b. ~~`global_statement`/`nonlocal_statement` 在子集白名单~~ **修正（施工方 S1 复核纠正，决策方确认）**：二者在 `forbiddenNodeTypes` **黑名单**（`compile_py_subset.go:202-203`，表起于 :185），`checkForbiddenNodes`（:67-72）在 CST 校验层即报 `not allowed in Python subset`——编译期已 fail-closed，`compileStmt` 永不可达该节点。
  c. `resolveVar`（`compile.go:257-297`）查找顺序 localScopes→GlobalSlots→python 隐式全局登记；`compileDecl`（`compile_expr.go:196-208`）已有"localScopes 非空→内层 scope 分配 nextLocalSlot"的既有路径。
  d. `nextLocalSlot`/`EventLocals`/`NumLocals` 由 `compile.go:312-316,342,351` 统一收尾——局部槽分配复用 `nextLocalSlot` 即自动正确。

## 设计 SSOT 声明

- 设计文档：spec v2 §3.3。契约：AGENTS.md §1 fail-closed；决策 D-009/D-012/D-014/D-015。

## 约束与目标（决策方已定的语义裁定）

- **函数内裸名赋值且全槽未命中 → 分配局部槽**（**函数基域 `localScopes[0]`** + `nextLocalSlot`，修正 v3），不落 GlobalSlots。仅 `ExprAssignment`（`compile_expr.go:105-112`）与 `ExprCompoundAssign`（`:180-194`）入口需要处理。
- **Python 局部一律函数域（修正 v3）**：`compileFor` 的 `pushScope/popScope`（`compile_loops.go:8,53`）是 MQL `for(int i;;)` 词法域设施，Python 无块作用域——`resolveAssignTarget` 新局部与 `compileDecl` 的 python 新局部都分配到 `localScopes[0]`，不分配到内层域。MQL 路径完全不变。
- **已声明全局（`GlobalDecls` = `ir.Globals`：self 字段 + 顶层赋值）的名字保持写全局**。判定谓词是 `GlobalDecls` **而非 `GlobalSlots`**（修正 v2，Option B 批准）：`GlobalSlots` 会被"未声明读"的隐式注册污染（`resolveVar` :284-289），若按它判定，先编译函数中的未声明读会让后续函数内同名赋值仍泄漏（编译顺序：函数体先于事件体，`compile.go:77-80` vs :94-111）。`GlobalDecls` 在编译期固定、不受编译顺序影响，且"隐式读注册 ≠ 声明"更贴近 Python 语义。
- **连带语义（已裁定）**：策略参数名（`ir.Params`）在 `GlobalSlots` 但不在 `GlobalDecls` → 函数内给参数名赋值变局部（Pythonic）；`self.x`/裸 `x` 同槽的既有混同不修（超出边界，自报记一笔即可）。
- **`x += 1`/`x = f(x)` 等读到未声明名**：读取路径 `resolveVar` 本轮**不改**（仍走隐式全局 + blind spot）；仅赋值落点改局部。
- **`x += 1` 未声明（无 local 且不在 `GlobalDecls`）→ 编译期 `c.err` 报错**（fail-closed，Python 是 NameError；不得隐式建全局再 +=）。注意：仅被隐式读注册进 `GlobalSlots` 但未声明的名字同样算未声明——与 `resolveAssignTarget` 用同一谓词。
- **`global`/`nonlocal` 已由 CST 黑名单 fail-closed 拦截（修正 v2）**——`compileStmt` 不加 case（死代码）；已加的删除。S4e 保留为回归守卫。
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
  // isDeclaredGlobal reports whether name was declared at module level
  // (self fields + top-level assignments collected into ir.Globals).
  // Names merely registered into GlobalSlots by implicit reads do NOT count —
  // they must not poison later function-scope assignments (QS-1.3).
  func (c *astCompiler) isDeclaredGlobal(name string) bool {
      for _, g := range c.bc.GlobalDecls {
          if g.Name == name {
              return true
          }
      }
      return false
  }

  // resolveAssignTarget resolves the store slot for an assignment target.
  // QS-1.3: inside a Python function/event, assignment to an undeclared name
  // declares a function-local slot instead of leaking into GlobalSlots.
  // Declared globals (self fields, module-level assignments) stay global.
  func (c *astCompiler) resolveAssignTarget(name string) (VarID, bool) {
      if c.bc.Version == "python" && len(c.localScopes) > 0 {
          for i := len(c.localScopes) - 1; i >= 0; i-- {
              if id, ok := c.localScopes[i][name]; ok {
                  return id, false
              }
          }
          if c.isDeclaredGlobal(name) {
              return c.bc.GlobalSlots[name], true
          }
          // Python has function scope, not block scope: allocate in the
          // function base scope (index 0 — the scope pushed by
          // compileUserFuncBody/compileEventBody), so names assigned inside
          // for-loop bodies survive the loop scope's popScope.
          scope := c.localScopes[0]
          scope[name] = VarID(c.nextLocalSlot)
          c.nextLocalSlot++
          return scope[name], false
      }
      return c.resolveVar(name)
  }
  ```
  `case ExprAssignment` 中 `c.resolveVar(e.Name)` 改为 `c.resolveAssignTarget(e.Name)`，emit 逻辑不变。（`GlobalDecls` 是 `[]interp.GlobalVar`，线性扫描；编译期一次性开销，不预建 set。）
- **验证**：S4 测试。

### S3 — `ExprCompoundAssign` 未声明名 fail-closed；`compileStmt` 死代码移除（修正 v2）

- **坐标**：`compile_expr.go` `func compileCompoundAssign`；`compile_py_stmt.go` `compileStmt`。
- **落点**：
  a. `ExprCompoundAssign`：`Version=="python"` 且 `localScopes` 非空时，先查 localScopes ∪ `isDeclaredGlobal`；完全未命中 → `c.err`（若 nil）= `cannot use augmented assignment on undeclared name %q (Python: NameError)`，return。**已声明者照常 resolveVar**（self 字段回归不破）。判定谓词与 S2 共用 `isDeclaredGlobal`，不得查 `GlobalSlots`。
  b. ~~`compileStmt` 加 `global_statement`/`nonlocal_statement` case~~ **修正 v2：删除已加的死代码 case**——CST 黑名单已拦截，该 case 永不可达（§7.2 禁死代码）。
  c. **`compileDecl` Python 函数域分配（修正 v3）**：`compile_expr.go:211-224`，`len(c.localScopes) > 0` 分支内——`Version=="python"` 时分配到 `c.localScopes[0]` 而非内层域（MQL 仍内层，块作用域保持）。效果：python `for i in range` 的 `i`（ExprDecl 路径）与 `for pos in ctx.positions` 脱糖的 `__i`/`__ticket` 落函数域，出循环仍可读——Pythonic 且消除 for-body 赋值消亡回退。
- **验证**：S4 测试。

### S4 — 测试（先红后绿 + mutation）

- **坐标**：新建 `backend/tools/mql2go/compile_py_locals_test.go`（复用 `compile_py_bool_test.go` 的 CompilePython+RunOnBar+GetGlobal 模式）。
- **落点**（最小集，修正 v2）：
  a. **字面形式转正**：`def on_bar(self): x = 1` + `def helper(self): return x` → helper 读 x 必须 **r=ValNone**（Option B 后隐式读注册不再污染赋值落点）。现有 `TestQS13_LiteralReaderPoisonsSlot` 从"钉洞"改写为断言隔离成立；另保留反向形式 `helper: x=1` / `on_bar: return x` → r=ValNone。
  b. 跨事件隔离：on_bar#1 `x=1`；on_bar#2 未赋值读 x → 不为 1（None）。
  c. 回归守卫：`self.count += 1` 跨两次 on_bar 累加为 2（GlobalDecls 命中 → GlobalSlots 路径不变）。
  d. 参数遮蔽：`def f(self, x): x = x + 1` 编译通过且参数槽语义不变。
  e. `global x` / `nonlocal x` 出现在函数内 → CompilePython 返回明确错误（CST 黑名单报错，回归守卫；基线即绿）。
  f. `x += 1` 完全未声明 → CompilePython 返回明确错误；同一函数先 `x=1` 后 `x+=1` 正常。
  g. NumLocals/EventLocals 正确性：断言局部槽数含新分配。
  h. **参数名赋值变局部（Option B 连带语义 pin）**：`def f(self): <param名> = 99` 后全局 param 值不变；自报注明该语义裁定。
  i. **for 体赋值跨循环存活（修正 v3）**：`for i in range(3): x = i` 后 `self.r = x` → r=2（局部槽，非全局）；断言 `x` 不在 globals。
  j. **循环变量跨循环存活（修正 v3）**：`for i in range(3): pass` 后 `self.r = i` → r=2。
  k. **while 体赋值跨循环存活**：`while` 体内 `x = 1` 后循环外读 → 1（既有行为 pin，防不对称回归）。
- **验证**：a 反向形式 / f 先红后绿；a 字面形式在 Option B 下先红（修复前 r=1）后绿（r=None）；i/j 在"内层域分配"实现下 RED、函数域分配下 GREEN。

## 对抗证明（缺一即未完成）

- mutation：S2 的 `resolveAssignTarget` 退回 `resolveVar`（或删 `Version=="python"` 分支）→ S4a/S4b RED；restore → GREEN。
- mutation：`isDeclaredGlobal` 改回查 `GlobalSlots` → S4a 字面形式 RED（r=1 泄漏复现）；restore → GREEN。
- mutation（修正 v3）：`localScopes[0]` 改回内层域（`localScopes[len-1]`）→ S4i/S4j RED（循环后读出 ValNone 复现）；restore → GREEN。
- mutation（替换原不可行项）：从 `forbiddenNodeTypes` 删 `"global_statement"` 条目 → S4e RED（静默丢弃恢复）；restore → GREEN。

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
