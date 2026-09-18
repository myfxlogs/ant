# 施工派工单：VM-GLOBAL-ARRAY-DECL-1（全局数组端到端修复）

> **立项背景**：VM-ARRAY-OOB-FAILCLOSED-1 施工方 S4 阻断上报后，Devin CLI 探针实证全局数组**端到端从未工作**。本单修复前端声明收集 + 赋值/更新错标族，把 `int g[2]` 全局数组做成真实功能；不可支持的子形态一律编译期显式拒绝（fail-closed），不得静默。
>
> **设计 SSOT**：本文件为唯一施工依据。条款与原 registry 条目（行 220）一致并取代其"修法方向"段。
>
> **波及面审计（已做，零依赖）**：`backend/tools/mql2go/testdata/`（4 个 mq4）+ `reference/Venus.mq4`（812 行）+ 全量内嵌测试源——**无任何策略/测试使用全局数组声明或下标写**（仅 `Close[1]` 序列访问，不同路径）。改动无兼容面负担。

## 1. 探针实证的 parse 形态表（tree-sitter 当前输出）

| 源码 | parse 结构 | 现状行为 | 处置 |
|---|---|---|---|
| `int g[3];` | `declaration→primitive_type+array_declarator(identifier+number_literal)` | **静默丢弃**（collectGlobalVar 无 array_declarator case） | S1 收集 |
| `double g[3];`/`static int g[3];` | 同上（static 多一个 storage_class_specifier 兄弟） | 静默丢弃 | S1 收集（static 对全局无语义差异） |
| `int g[2]={1,2};` | `declaration→init_declarator→array_declarator+initializer_list` | **静默丢弃**（findIdent 看不到嵌套 identifier → name=="" → continue） | S1：编译期拒 initializer |
| `int g[2][3];` | `array_declarator` **嵌套** array_declarator | 静默丢弃 | S1：编译期拒多维 |
| `g[0]=5;` | `assignment_expression→subscript_expression` | findIdent(lhs) 先于 subscript 判断 → **整槽标量写**（ExprAssignment，下标丢弃） | S2 顺序对调 |
| `g[0]+=5;` | 同上 | → ExprCompoundAssign{Name:"g"} **整槽复合运算**（毁掉 ValArray） | S2 编译期拒 |
| `g[0]++;` | `update_expression→subscript_expression` | → ExprUpdate{Name:"g[0]"} → **创建名为 "g[0]" 的幻影全局** | S3 编译期拒 |
| `void f(int a[])` | `parameter_declaration→array_declarator` | findIdent→"" → **参数静默丢弃**，函数内 a[i] 写幻影全局 | S4 编译期拒 |
| `input int g[3];` | `type_identifier"input"+array_declarator(identifier"int"+ERROR"g"+number_literal)` | 畸形 parse；findIdent ERROR 下探可救回 "g" | best-effort 收集（不专项处理） |

**关键机制事实**：
- `findIdent` 只查**直接**命名子节点（+ERROR 下探），不递归 array_declarator。
- `findArraySize`（compile_interp_expr.go:453）找 `subscript_expression`——声明处节点类型实为 `array_declarator`，该函数对本场景**永不命中**（死分支）。
- 管线后端全就绪：`ir.Globals→GlobalSlots`（compile.go:47-49）→`GlobalDecls`→`vm.go:243-250 initGlobals`（`IsArray&&ArraySize>0`→`ValArray`，`zeroValueForType` 全类型覆盖）——只缺收集端。

## 2. 施工步骤

### S1 `collectGlobalVar`（`compile_interp.go:174-223`）——补收集

- 新增 `array_declarator` 子节点 case：
  - `name := c.findIdent(child)`（identifier 为直接子节点，可命中）。
  - **多维检测**：child 的命名子节点中若再含 `array_declarator` → `c.err = fmt.Errorf("multi-dimensional arrays not supported: %s", name)`（fail-closed；ValArray 扁平）。
  - **尺寸提取**：child 的直接 `number_literal` 子节点 → `fmt.Sscanf`；缺失或 ≤0 → `c.err = fmt.Errorf("array size required: %s", name)`（无法初始化未知尺寸，诚实拒绝）。
  - `ir.Globals = append(ir.Globals, interp.GlobalVar{Name:name, Type:typeName, IsArray:true, ArraySize:size})`。
- `init_declarator` 分支（行 178-195）：`findIdent` 前先检查 child 是否含 `array_declarator` 直接子节点：
  - 含 → 查兄弟 `initializer_list` 是否存在 → `c.err = fmt.Errorf("global array initializer not supported: %s", <name>)`（本批不实现数组字面量；诚实拒绝优于静默丢初始化值）。
  - 含 array_declarator 且无 initializer_list → 按上条同法收集（防御性，parse 未见此形态）。
- `findArraySize` 死分支：可顺手修正为同时检查 `array_declarator`，或保留（本批 S1 新 helper 独立提取，不动它以免越界）。**建议**：新 helper `parseArrayDeclarator(n *sitter.Node) (name string, size int, multiDim, ok bool)` 放 collectGlobalVar 旁，内部做 identifier/number_literal/嵌套检查三合一。

### S2 `compileAssignment`（`compile_interp_expr.go:254-313`）——顺序对调 + 复合拒绝

- `lhs.Type()=="subscript_expression"` 判断**移到 findIdent 之前**：
  - `op=="="`：`subExpr := c.compileSubscript(lhs)`；nil → `c.err`；否则 `subExpr.Args = []interp.Expr{c.mustExpr(rhs)}` 返回（astCompiler compile_expr.go:136 写路径已就绪：全局 → OP_STORE_ARRAY，局部 → 负编码 → 运行时 error）。
  - `op!="="`：`c.err = fmt.Errorf("compound assignment on array element not supported: %s", <lhs text>)`（本批不实现元素级复合写，显式拒优于整槽错标）。
- findIdent 分支及其后逻辑保持不变（subscript 已被前段截获，行 304 块可删——它成为真正的不可达死代码，删掉防混淆）。

### S3 `compileUpdate`（`compile_interp_expr.go:315-323`）——下标 ++/-- 拒绝

- 入参 `n`（update_expression）的命名子节点若含 `subscript_expression` → `c.err = fmt.Errorf("++/-- on array element not supported: %s", c.text(n))`，return nil。
- 不得再产出 `ExprUpdate{Name:"g[0]"}`（现行会注册幻影全局槽 "g[0]"）。

### S4 参数数组拒绝（`compile_interp.go:625-648` collectParams 循环内）

- `pd.Type()=="parameter_declaration"` 时，检查其命名子节点含 `array_declarator` → `c.err = fmt.Errorf("array parameters not supported: %s", pName)`（MQL 引用语义无法支持；现行静默丢参→幻影全局，必须显式拒）。check order：在 append 前。

### S5 验证 vm.go 初始化链路（只读核对，不改）

- `vm.go:243-250`：`decl.IsArray && decl.ArraySize>0` → ValArray 零值数组——S1 落地后自动生效，核对 `zeroValueForType` 覆盖 int/long/datetime/bool→0、double/float→0、string→"" 即可。

### S6 测试（新文件 `backend/tools/mql2go/vm_global_array_test.go`）

行为断言（CompileMQL + RunOnInit/RunOnBar，沿用 vm_array_oob_test.go 的 newSourceVM/accountStatusTestContext 模式）：

- **合法读写（迁移 S4-1）**：`int g[2]; OnInit(){g[0]=5;g[1]=7;} OnBar(){x=g[0]+g[1];}` → x==12。
- **读 OOB（迁移 S4-2）**：`x=g[5]` → OnBar err 含 `index 5 out of range (len=2)`。
- **写 OOB（迁移 S4-6）**：`g[9]=1` → OnBar err 含 `OP_STORE_ARRAY index 9 out of range`。
- **负索引**：`x=g[-1]` → err。
- **类型覆盖**：`double d[3]`/`string s[2]`/`bool b[4]` 各读写一轮 + 未写元素为零值（0/""/false）。
- **声明后使用无关顺序**：先引用后声明的合法 MQL 不受影响（Globals 先收集，两阶段）。
- **static int g[3]** → 正常收集使用。
- **编译期拒×4**：`int g[2][3]`（多维）/`int g[2]={1,2}`（initializer）/`void f(int a[])`（参数数组）/`g[0]+=5`（复合）/`g[0]++`（更新）→ `CompileMQL` err 非 nil 且消息含对应关键词。
- **局部数组仍拒**：函数内 `int a[2]` → 既有 compile error 不破。
- **不误伤**：`int g` 标量、`x=arr[i]` 读、既有 e2e 全绿。

**Mutation 证明（每项 RED→restore→GREEN）**：
- M1：删 S1 array_declarator case → 合法读写测试 RED（g 未收集 → 读 not-an-array）。
- M2：compileAssignment 恢复 findIdent 优先 → `g[0]=5` 变整槽写 → 读回 `g[0]` err/值错 → RED。
- M3：删 S3 守卫 → `g[0]++` 静默编译通过 → 编译期拒测试 RED。
- M4：删 S4 守卫 → `void f(int a[])` 编译通过 → 编译期拒测试 RED。

## 3. 门禁

`go build ./...` / `go test ./tools/mql2go/` / `go test -race -count=3 ./tools/mql2go/` / `go vet` / `gofmt` / `go run ./tools/check-file-lines --strict`（0 errors；**预警**：compile_interp.go 现 846 行，净增 ~40 行可能触阈——超限则把 `collectGlobalVar`+新 helper 拆到 `compile_interp_globals.go` 同 package）/ `git diff --check`。

## 4. 边界/不做

- **不做**局部数组（`compileDeclaration` 行 583 已编译期拒，保持）。
- **不做**多维数组、数组 initializer、数组参数、元素级 `+=`/`++`——全部编译期显式拒（本批落错误消息，不实现语义；若 e2e 需求浮现另立债）。
- **不做** `input` 数组专项（畸形 parse，best-effort）。
- 不改 VM 运行时（vm.go 链路只读核对）、不改 OP_*_ARRAY 行为、不碰 python 前端。
- 勿部署、勿 push 远端，完成报六段式证据等 Devin CLI 复审。
