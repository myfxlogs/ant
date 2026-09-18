# Builder Handoff: PY-SCOPE-KNOWN-1 — Python 作用域已知限制文档化 + pin 测试

> **角色**：施工方。**任务 ID**：`PY-SCOPE-KNOWN-1`（P3 文档化行为债，QS-1.3 遗留）。
> **性质**：**零行为变更债**——不改任何编译/运行时语义，只做三件事：① 给两条未 pin 的偏差补 pin 测试；② 把三条已知限制写进模块权威文档与坑库；③ 给 Agent 侧 Python 子集规则补三条防踩坑规则。
> **设计实查**：Devin CLI 已完成，结论与实测坐标见下，均已逐一核实。

## 立项背景与实测结论

registry 行 206 三条 Python 子集语义偏差，设计实查全部**探针实证**：

| # | 偏差 | 实证（探针运行结果） | 机制坐标 |
|---|------|---------------------|----------|
| ①a | `self.x` 与裸 `x` 同槽混同——**写侧** | `self.x=10` 后函数内裸 `x=1`（含 `x: int = 1` 带注解）→ `GetGlobal("x")=1`——字段槽被覆写（Python 语义 `self.x` 应保持 10） | `selfFieldName` 剥前缀映射同名 `Name:"x"`（`compile_py_assign.go:271-288`）→ `selfVars`→`ir.Globals`→`isDeclaredGlobal("x")=true`→裸赋值经 `resolveAssignTarget`（`compile.go:316-333`）写全局槽 |
| ①b | 同根因——**读侧遮蔽** | `self.x=10` 后 `for x in range(3): pass` → `self.x` 读回 **3**（循环局部遮蔽字段），`GetGlobal("x")=10` 完好——遮蔽非覆写 | `for` 变量经 `ExprDecl`→`compileDecl`→`localScopes[0]`（`compile_expr.go:220-231`）；`self.x` 读→`ExprVar{Name:"x"}`→`resolveVar`（`compile.go:256-267`）**localScopes 优先于 GlobalSlots**→读局部 |
| ② | 未声明读 → ValNone 非 NameError | `self.r = never_declared` 编译运行全过，`r=ValNone`（Python 应 NameError）；补偿通道：`bc.Coverage.BlindSpots` 含 `"implicit variable: never_declared"` | `resolveVar` python 分支 `compile.go:284-288`（blind spot + 隐式 GlobalSlots） |
| ③ | `for i in range(N)` 退出值 `i=N` | 脱糖 C 式 `i=0;i<N;i++`（`compile_py_stmt.go:171-211`），Python 应 `N-1` | **已 pin**：`TestQS13_ForLoopVarSurvivesLoop`（`compile_py_locals_test.go:320-344`）断言 `r=3` + 注释注明 Python 应为 2 |

**① 双向性说明**（设计实查新发现，registry 原文只笼统提"同槽混同"）：`self.x` 与裸 `x` 编译后都叫 `"x"`，经同一套 `resolveVar`/`resolveAssignTarget` 作用域机制解析——写侧裸赋值命中 `isDeclaredGlobal` 覆写字段槽（①a）；读侧若同名局部存在（for 变量/`global` 外任何局部来源）`resolveVar` 局部优先使 `self.x` 读到局部（①b）。两副面孔同一根因：字段无独立命名空间。

**裁定**：三条均为裁剪子集语义偏差，当前阶段**文档化而非修复**（触发真实策略问题再立项修）。文档化落点按 P3 单一真相源分层：细节在模块 README（一处），pitfalls.md 只放发现路径指针，Agent 规则只放防踩坑行为约束。

## S1 — 新增 pin 测试 `backend/tools/mql2go/compile_py_scope_known_test.go`

新文件，包 `mql2go`，三个测试 T1a/T1b/T2（③ 已有 pin，本文件头注释指向 `TestQS13_ForLoopVarSurvivesLoop` 即可，不重复）：

```go
// Package pin tests for documented Python-subset scope deviations
// (registry PY-SCOPE-KNOWN-1). These tests PIN the current behavior as
// documented — they are NOT approvals of the semantics. If a future fix
// corrects a deviation, the corresponding pin test must be updated in the
// same commit that changes the behavior.
// ③ for-range exit value is already pinned by TestQS13_ForLoopVarSurvivesLoop.
```

**T1a `TestPyScopeKnown_SelfBareWriteClobbersField`**（pin 偏差①a 写侧混同）：

- 源码（探针实证形态，全在 `on_bar`，不依赖 `__init__`/`RunOnInit`）：
  ```python
  class S:
      def on_bar(self) -> None:
          self.x: int = 10
          x: int = 1
          self.r = self.x
          return
  ```
- `CompilePython` → `RunOnBar`。
- **判别断言**：`GetGlobal("x")` == `ValInt(1)`——pin 住"裸 `x` 写覆写了 `self.x` 的全局槽"（Python 语义字段应保持 10）。⚠️ `GetGlobal("r")==1` 不可作判别量——局部写形态下 `self.x` 读也经 `resolveVar` 局部优先仍得 1；r 断言可保留作文档但不作 pin 证据。
- 注释注明：Python 语义 `self.x` 应为 10；pin 的是文档化偏差本身。

**T1b `TestPyScopeKnown_SelfReadShadowedByLocal`**（pin 偏差①b 读侧遮蔽）：

- 源码（探针实证形态）：
  ```python
  class S:
      def on_bar(self) -> None:
          self.x: int = 10
          for x in range(3):
              pass
          self.r = self.x
          return
  ```
- `CompilePython` → `RunOnBar`。
- **判别断言**：`GetGlobal("r")` == `ValInt(3)`——pin 住"`self.x` 读被同名循环局部遮蔽"（Python 字段读应为 10）。
- **副断言**：`GetGlobal("x")` == `ValInt(10)`——pin 住遮蔽非覆写（字段值完好，只是读不到）。
- 注释注明：Python 语义 `self.x` 读字段应为 10；遮蔽来自 `resolveVar` localScopes 优先 + 字段无独立命名空间。

**T2 `TestPyScopeKnown_ImplicitReadYieldsNone`**（pin 偏差②）：

- 源码：
  ```python
  class S:
      def on_bar(self) -> None:
          self.r = never_declared
          return
  ```
- `CompilePython` **必须成功**（pin：不拒绝）；`RunOnBar` 成功。
- `GetGlobal("r")` 断言 `Kind==ValNone`（pin：ValNone 非 NameError）。
- 断言 `vmRunner.vm.bc.Coverage.BlindSpots` 含 `"implicit variable: never_declared"`（pin：偏差经 coverage 通道上报，非全静默；断言先例 `vm_array_oob_test.go:317`）。

## S2 — 模块文档 `docs/blocks/mql-compiler/README.md`

在 `## 关键设计` 之后新增 `## 已知限制（Python 子集语义偏差）` 小节，逐条写：偏差描述、Python 正确语义、当前行为、机制坐标（file:line）、pin 测试名。内容即上表 ①a/①b/②/③ 四条（① 按双向两面写；③ 注明已 pin）。结尾一句：触发真实策略问题后按 registry `PY-SCOPE-KNOWN-1` 立项修复。

## S3 — 坑库指针 `docs/pitfalls.md`

在 `### 已确认的静默失败模式`（MQL2GO VM Pitfalls 段）追加一条 bullet：

- **Python 子集作用域偏差（PY-SCOPE-KNOWN-1，文档化行为）** — `self.x` 与裸 `x` 同槽混同双向：函数内裸 `x=1` 覆写 `self.x` 字段槽；同名局部（如 `for x in range(N)`）遮蔽 `self.x` 读。未声明读静默返 `ValNone`（非 NameError，经 `bc.Coverage.BlindSpots` 上报）。`for i in range(N)` 退出 `i=N` 非 `N-1`。策略行为诡异且命中上述形态时先查此条。明细与 pin 测试见 `docs/blocks/mql-compiler/README.md` 已知限制节。

只放一行发现路径+指针，细节不复制（P3 单一真相源）。

## S4 — Agent 规则 `backend/internal/ai/python_rules.go`

在 `### What is FORBIDDEN` 列表末尾追加三条（保持现有 bullet 风格，简洁祈使句）：

- reusing a `self.<name>` field name as a local variable name or loop variable (same name = same storage — bare writes corrupt the field, and a same-named local shadows field reads)
- reading a variable before assigning it (undeclared reads silently return None — no error is raised)
- reading the loop variable after a `for` loop (its value overshoots the range bound, not the last item)

**注意**：`PythonSubsetRules` 是常量字符串、被 5 locale 文件内联引用——只追加 bullets，不动既有任何字符；零测试依赖其内容快照（已核实）。

## S5 — 收尾

- registry 行 206 追加施工完成记录（🟦open 施工完成待复审格式）。
- `docs/handoff/STATE.md` 施工表/下一步指针同步。

## 对抗证明（必做）

| # | 变异 | 预期 RED |
|---|------|----------|
| M1 | `compile.go:323-325` `resolveAssignTarget` python 分支注释掉 `isDeclaredGlobal` 判定块（`if c.isDeclaredGlobal(name) { return c.bc.GlobalSlots[name], true }` 三行）→ 裸 `x` 变局部 | **T1a RED**：`GetGlobal("x")=10 want 1`（判别量；`r` 仍=1 不可判别）→ restore → GREEN |
| M1b | `compile_expr.go:226-228` `compileDecl` python 基域分配还原为内层域（删 `if c.bc.Version == "python" { scope = c.localScopes[0] }`）→ for 变量随循环域消亡 | **T1b RED**：`r=10 want 3`（遮蔽消失，读到字段真值）→ restore → GREEN。**预期连带**：QS-1.3 v3 家族测试（`ForBodyAssignSurvivesLoop`/`ForLoopVarSurvivesLoop`/`WhileBodyAssignSurvives`）同步 RED——如实记录，该连带正是 v3 修复面 |
| M2 | `compile.go:284-288` python 隐式注册分支改为编译报错（仿 :291 mql5 分支 `c.err = fmt.Errorf(...)`） | T2 RED（CompilePython 失败）→ restore → GREEN |

M1 若连带其他存量测试 RED，如实记录受影响清单（混同面本就是 GlobalDecls 命中，连带属预期）。

## 验收门禁

`go build ./...` / `go test -count=1 ./tools/mql2go/` / `-race -count=3 ./tools/mql2go/` / `go vet ./tools/mql2go/` / gofmt 本批文件 / `check-file-lines --strict` / `git diff --check`。

## 边界（不做）

- **不改任何语义**：`resolveVar`/`resolveAssignTarget`/`selfFieldName`/`compileDecl`/for 脱糖全部原样。
- 不新增 builtin/opcode/字段；不动 `TestQS13_*` 既有测试。
- 不更新其他 locale 文件（它们内联引用常量自动生效）。
- 勿部署、勿 push、禁 `--no-verify`。完成报证据停手等 Devin CLI 复审。
