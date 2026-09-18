# Builder Handoff — MQL-COMPILER-LOCAL-ARRAYS

> MQL 函数内数组声明（`double p[]` / `int a[3]`）编译硬失败 `local arrays not supported`。
> 设计实查另实证两个同族活 bug：`ArrayResize` 对全局数组静默无效 + 局部数组初始化器静默误编译。
> 设计 SSOT：Devin CLI。施工只执行、不决策。完成报证据等独立复审。

## 0. 立项证据链（设计实查已核实）

**症状**（registry `MQL-COMPILER-LOCAL-ARRAYS` 节，2026-09-08 立项）：`compile_interp.go:665` `array_declarator` 分支硬编码 `local arrays not supported: <name>` —— 常见 MQL 写法 `double price[]; ArrayResize(price, N)` 无法导入 VM，Agent 被迫桥接翻译。

**新实证 bug A（`ArrayResize` 全局静默无效）**：`builtinArrayResize`（`vm_builtin_string.go:200-215`）改 `args[0].Array` —— `interp.Value.Array` 是 `[]Value` slice header，builtin 收到的是栈上弹出的 Value **副本**；re-slice/`append` 只改副本 header，**永不回写槽位**。探针实证（2026-09-19）：`double g[2]; ArrayResize(g,5)` 后 `globals[g]` len 仍 2、`ArraySize` 报 2。元素级写入经共享 backing 传播，故 `ArrayCopy`/`ArrayFill` 正常——只有改变长度的操作断。

**新实证 bug B（局部初始化器静默误编译）**：`void f(){ int a[2]={1,2}; }` **今天编译通过**（err=nil，探针实证）—— `init_declarator` 分支 `findIdent` 收 "a"、编译 initializer_list 为标量 init expr → 产出一个标量局部 `a`，数组性与初始化器全部丢弃。比显式拒收更糟的静默失真。

**已就绪基建**（复用面，勿重建）：
- `ExprSubscript{Name,Index}` 按名寻址 → `compile_expr.go:140/157` `resolveVar` 局部槽**负编码** `-int32(slot)-1`（VM-ARRAY-OOB-FAILCLOSED-1 已铺好编码约定，uint16→int32 再取负）
- `vm_execute.go:309/333` 负编码路径已存在但 fail-closed `"local array slot %d not supported"` —— 换成 `vm.locals` 实读实写即可
- `vm.locals []interp.Value` 帧槽模型：`executeCallUser` 每次调用换 `newLocals` 并拷入 args —— 局部数组访问在用户函数内自动正确（槽索引与 `OP_PUSH_VAR` 同一编号空间）
- `compileDecl`（`compile_expr.go:220`）：`Args[0]` 编译 → `localScopes` 登名 → `nextLocalSlot` 分配 → `OP_STORE_VAR` —— 局部数组声明只需产出 `ExprDecl{Name, Args:[数组构造Expr]}`，槽分配与落槽零新增代码
- 全局先例：`parseArrayDeclarator`（`compile_interp.go:180`）+ `collectGlobalVar` 的 multiDim/initializer 拒绝（`:209-216/:246-255`）+ `initGlobals` ValArray 初始化（`vm.go:243-251`）+ `builtinArrayResize` 的 NoneVal 填充（`vm_builtin_string.go:212`）

**tree-sitter 节点形态实证**（ParseMQL dump）：
- `int a[3]` → `array_declarator(identifier, number_literal)`
- `double p[]` → `array_declarator(identifier)`（无维度子节点 → size=0 动态空数组）
- `int a[n]` → `array_declarator(identifier, identifier)`（第二 identifier = 非常量维度 → 必须报错，否则静默当空数组）
- `int a[2]={1,2}` → `init_declarator(array_declarator(identifier,number_literal), initializer_list)` —— **走 init_declarator 分支不走 array_declarator 分支**

## 施工步骤

### S1 — `interp/ir.go`：新 Expr kind
`Expr` kind 枚举追加 `ExprArrayNew`（注释：`Name`=元素类型名、`Val`=IntVal 声明尺寸，0=动态空数组）。

### S2 — `compile_interp.go` `compileDeclaration`（`:620-668`）
- **init_declarator 分支前置**：扫描 `child` 的命名子节点，含 `array_declarator` → `c.err = fmt.Errorf("local array initializer not supported: %s", c.findIdent(sub))` + `return nil`（镜像 global `:209-216`；关 bug B）
- **array_declarator 分支**（替换 `:663-667` 硬错误）：
  1. `parseArrayDeclarator` → name/size/multiDim；`!ok || name==""` → `continue`（多维时 name 空，用 `c.text(child)` 回退——镜像 global `:248-251`）
  2. `multiDim` → `multi-dimensional arrays not supported: <label>` + `return nil`（镜像 `:246-255`）
  3. **非常量维度**：遍历命名子节点，跳过首个 identifier 与嵌套 `array_declarator`，任何非 `number_literal` 子节点 → `array size must be a constant: %s`（`int a[n]` 第二 identifier 在此命中）
  4. 正常：`ExprDecl{Name:name, Args:[interp.Expr{Kind:interp.ExprArrayNew, Name:typeName, Val:interp.IntVal(int32(size))}]}` 入 `decls`（`double p[]` → size=0；多个声明符时与标量 decl 同走 ExprSeq 包装——既有逻辑无需改）

### S3 — `bytecode.go`：两枚新 opcode
枚举尾部追加 `OP_NEW_ARRAY`、`OP_ARRAY_RESIZE`（追加不插队，marshal 为 int32 泛型序列化，bytecode_cache 版本校验兜底兼容）。

### S4 — `compile_expr.go` astCompiler
- `compileExpr` switch 加 `case interp.ExprArrayNew` → `c.emit(OP_NEW_ARRAY, int32(e.Val.Int), 0, 0)`（尺寸为编译期常量，非常量维度已在 S2 拒绝）
- **ArrayResize 拦截**（`:405` builtin 分支前置）：`e.Name=="ArrayResize" && len(e.Args)==2 && e.Args[0].Kind==interp.ExprVar` → `c.compileExpr(&e.Args[1])`（newSize 入栈）→ `slot, isGlobal := c.resolveVar(e.Args[0].Name)` → 局部 `c.emit(OP_ARRAY_RESIZE, -int32(slot)-1, 0, 0)` / 全局 `c.emit(OP_ARRAY_RESIZE, int32(slot), 0, 0)` → `return`（结果由 opcode 自推）。非 `ExprVar` arg0 走原 builtin 路径不变。
- **删两处盲记** `:142` `AddBlindSpot("local array write: "+e.Name)` 与 `:159` `AddBlindSpot("local array read: "+e.Name)` —— 路径已真支持，盲记即失真。

### S5 — `vm_execute.go`
- `case OP_NEW_ARRAY`：`ins.A < 0 || ins.A > 1_000_000` → `setStackError("OP_NEW_ARRAY size %d out of range")` + return（fail-closed，与既有 OOB 家族同形）；`vm.push(interp.Value{Kind:interp.ValArray, Array: make([]interp.Value, ins.A)})` —— 元素零值 `interp.NoneVal()`，与 `builtinArrayResize:212` 填充语义一致（`IsTrue=false`/`ToInt=0`/`ToDecimal=0`/`ToString=""` 全读法等价零）。
- `case OP_ARRAY_RESIZE`：`idx := vm.pop()`（IntVal newSize）；槽解码 `ins.A>=0`→globals / `ins.A<0`→`vm.locals[-ins.A-1]`；bounds 出界 → `setStackError("OP_ARRAY_RESIZE slot %d out of range")`；`Kind!=ValArray` → `"...slot %d is not an array"`；`newSize<0 || >1_000_000` → `"...size %d out of range"`；resize（`arr[:n]` 或 append `NoneVal`）→ **回写槽位** `vm.globals[A]`/`vm.locals[i] = interp.Value{Kind:interp.ValArray, Array:arr}`（这是 builtin 做不到的缺失环节）；`vm.push(interp.IntVal(newSize))`。
- `executePushArray`/`executeStoreArray` 负编码路径（`:305-311`/`:330-335`）：`localIdx := -ins.A-1` → `localIdx >= len(vm.locals)` → `OP_*_ARRAY local slot %d out of range (locals=%d)`；`Kind!=ValArray` → `local slot %d is not an array`；index bounds 同全局路径；读 `arrVal.Array[i]` / 写 `arrVal.Array[i]=val`（slice backing 共享，元素写天然传播）。

### S6 — 测试 `vm_local_array_test.go`（新建）+ 存量 pin 反转
- **T1** 报告症状端到端：`void f(){ double p[]; ArrayResize(p,3); p[1]=2.5; r=p[1]; }` → r=2.5（`CompileMQL`+RunOnInit 经 `newSourceVM` 惯例）
- **T2** 尺寸局部数组：`int a[3]` → 三元素零值 + `a[2]=7` 写读回；且在**用户函数内**（帧 locals 实考）
- **T3** 全局 resize 回归（bug A）：`double g[2]; ArrayResize(g,5)` → `ArraySize(g)=5` + `g[4]` 可写读
- **T4** 缩容：`ArrayResize(g,1)` → len=1
- **T5** 编译拒绝四件：局部 `int a[2]={1,2}`（`local array initializer not supported`，**关 bug B**）、`int a[n]`（`array size must be a constant`）、`int a[2][3]`（`multi-dimensional`）、`double p[]={}`（含 initializer 即拒）
- **T6** 局部 OOB fail-closed：`a[5]` 读/`a[-1]` 写 → VM error 含 `local`+`out of range`
- **T7** `ArraySize(p)` 局部 resize 后返回值正确
- **改写 `vm_global_array_test.go:208` `TestLocalArrayStillRejected`**：旧 pin 反转——`int a[2]`/`double p[]` 局部声明现在**必须编译通过**（改名 `TestLocalArrayDeclAccepted` 之类，断言 `err==nil`）

### 对抗证明规格（验收必演）
- M1 删 `case OP_NEW_ARRAY`（或 push ValNone 代 ValArray）→ T1/T2 RED
- M2 `executePushArray` 负路径恢复 `not supported` → T2/T6 RED
- M3 删 `OP_ARRAY_RESIZE` 槽位回写 → T3 RED（len 停旧值，bug A 复活）
- M4 删 init_declarator 前置扫描 → T5 initializer 子用例 RED（静默误编译复活）
- restore → 全 GREEN，工作区逐字节一致

## 边界（不做）
- **数组参数仍拒**：`void f(int a[])` 保持 `array parameters not supported`（参数引用语义另债）
- multi-dim / initializer / compound assign / `++` `--` on element 全部维持拒绝——与全局一致
- `builtinArrayResize` 保留（非变量 arg0 回退路径；元素级语义不变）
- `vm.globals`/`GlobalDecls`/marshal 格式不动

## 门禁
`go build ./...` / `go vet ./...` / `go test ./tools/mql2go/` 全绿 / `go test -race` ×3 / `go run ./tools/check-file-lines --strict` 0 errors / `git diff --check` / gofmt 触碰文件净。**勿部署，停手等 Devin CLI 复审。**
