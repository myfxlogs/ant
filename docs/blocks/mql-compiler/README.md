# mql-compiler — MQL/Python 编译器

> MQL4/MQL5/Python → tree-sitter → IR → Bytecode → VM。

## 代码位置

```
backend/tools/mql2go/           ← 55+ Go 文件
  interp/                       ← IR 定义、分析器
  compile.go / compile_py.go   ← 编译入口
  vm.go / vm_execute.go        ← 字节码 VM（操作数栈、全局变量、调用深度 256、最大 tick 10M）
  builtins.go                   ← 277+ 内置函数注册
  ast_coverage.go               ← 盲区追踪
```

## 关键设计

- 双编译前端：MQL（tree-sitter CGo） + Python（AST 解析），共用一个 IR 和 VM
- 确定性管道——同一份源码两次编译得到相同字节码
- 盲区追踪：标记未支持的 MQL 操作，Agent 用盲区桥接（agent-engine）翻译到 Python 子集

## 已知限制（Python 子集语义偏差）

> registry `PY-SCOPE-KNOWN-1`（P3 文档化行为，QS-1.3 遗留）。三条均为裁剪子集语义偏差，当前阶段文档化而非修复——触发真实策略问题后按 registry 立项。

### ① `self.x` 与裸 `x` 同槽混同（双向两面，字段无独立命名空间）

`self.x` 与裸 `x` 编译后同名，经同一套 `resolveVar`/`resolveAssignTarget` 作用域机制解析。

- **写侧覆写（①a）**：函数内裸 `x = 1`（含带注解形态）命中 `self.x` 声明的全局槽并覆写之。Python 语义 `self.x` 应保持 10。
  - 机制：`selfFieldName` 剥前缀映射同名（`compile_py_assign.go:271-288`）→ `selfVars`→`ir.Globals`→`isDeclaredGlobal` 命中 → 裸赋值写全局槽（`compile.go:316-333`）。
  - pin：`TestPyScopeKnown_SelfBareWriteClobbersField`。
- **读侧遮蔽（①b）**：同名局部存在时（如 `for x in range(3)` 的循环变量），`self.x` 读到局部值，字段槽本身完好（遮蔽非覆写）。Python 语义 `self.x` 读字段应得 10。
  - 机制：`for` 变量经 `compileDecl` 入 `localScopes[0]`（`compile_expr.go:220-231`）；`resolveVar` localScopes 优先于 GlobalSlots（`compile.go:256-267`）。
  - pin：`TestPyScopeKnown_SelfReadShadowedByLocal`。

### ② 未声明读 → ValNone（非 NameError）

函数内读取未声明变量：编译运行全过，得 `ValNone`（Python 应 NameError）。非全静默——经 `bc.Coverage.BlindSpots` 以 `"implicit variable: <name>"` 上报。
机制：`resolveVar` python 隐式注册分支（`compile.go:284-288`）。
pin：`TestPyScopeKnown_ImplicitReadYieldsNone`。

### ③ `for i in range(N)` 退出值 `i=N`

脱糖为 C 式 `i=0; i<N; i++`（`compile_py_stmt.go:171-211`），循环退出后 `i == N`；Python 应为 `N-1`。
pin：`TestQS13_ForLoopVarSurvivesLoop`（`compile_py_locals_test.go:320`）。

## 依赖

```
MQL/Python 源码 → mql-compiler → Bytecode
```

## 被依赖

```
mql-compiler → strategy-runtime(VM 执行策略)
mql-compiler → agent-engine(编译验证 + 盲区桥接)
mql-compiler → backtest-engine(回测执行)
```
