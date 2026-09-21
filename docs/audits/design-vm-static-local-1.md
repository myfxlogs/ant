# 设计 SSOT — VM-STATIC-LOCAL-1（函数内 `static` 局部变量不持久）

> 状态：v1 立项（Devin CLI，2026-09-21）
> 触发：全功能实盘探针——`static int done` 在 `OnTick` 内每 tick 重新初始化，`done=1` 永不持久（打印每 tick 重放）
> registry 行 47：`🟦open`

---

## 0. 实查定论（全部已读码/CST 实证坐实）

1. **根因=静默丢弃**：tree-sitter CST 实证 `static int done = 0;` 解析为
   `declaration → [storage_class_specifier("static"), primitive_type, init_declarator]`。
   `compileDeclaration`（`compile_interp.go:619`）的 named-child switch 只认
   `init_declarator`/`declarator`/`identifier`/`array_declarator`——**`storage_class_specifier`
   落入 default 被静默丢弃**，`static` 语义整词蒸发，变量编译成普通局部（每调用重初始化）。
   与 VM-IMPLICIT-VAR-READ-1 同属"静默捏造语义"缺陷族。
2. **局部槽分配机制**：`compileDecl`（`compile_expr.go:223`）把 ExprDecl 注册进
   `localScopes[len-1]`（frame 槽 `nextLocalSlot++`）+ `OP_STORE_VAR`——frame 槽随每次调用
   重建，无任何持久面。
3. **解析层双栈结构**：`resolveVar`/`resolveVarWrite`（`compile.go:262/303`）按
   `localScopes`（内→外）→ `GlobalSlots` 解析；scope 命中恒返 `isGlobal=false`
   ——当前结构里"局部名绑定到全局槽"无表达通道。
4. **MQL 权威语义（官方文档实锤）**：MQL5 reference《Declaration/definition statements》——
   *"If a declaration statement has the static modifier, the corresponding element is
   created only once when the statement is executed for the first time, and remains in
   memory regardless of exit and subsequent entries."* → **懒初始化一次（首次到达声明时），
   而非程序加载时**。`static int x = Bars` 类运行期初始化器取的是**首达时**的真值。
5. **零兼容风险**：生产库 7 个真实用户策略全量扫描 `'\mstatic\M'` **零命中**——
   修复不破坏任何现存策略。
6. **基础设施现成**：`OP_PUSH_GLOBAL`/`OP_STORE_GLOBAL`/`OP_JMP_IF_TRUE` 全在；
   未在 `GlobalDecls` 登记的全局槽零值 = `ValNone` → `IsTrue()=false` → init 旗标天然 falsy。
   `interp.Expr` 不进 bytecode cache 序列化面（cache 只 marshal Bytecode 字段）→
   IR 加字段零格式变更。
7. **同族静默丢弃面**（本立项的边界外，登记备查）：`const`（`type_qualifier` 节点）在局部
   同样被丢——`const int x=5; x=9;` 可编译（const 性不强制）；顶层 `static int g` 语义已正确
   （单 TU 内 internal linkage ≡ 普通全局）；`storage_class_specifier` 挂 function_definition
   时同样被丢（MQL 本不允许 static 函数）。

## 1. 设计决策：编译层脱糖（mangled global + init-guard），零字节码格式变更

**方案 A（选定）**：`static` 局部变量在语义上就是"作用域受限的全局变量"——经典脱糖。
编译期把声明改写为：隐藏全局槽 `__static_<N>` + 初始化旗标 `__sinit_<N>`，
声明点发射**旗标守卫的 init-once 字节码**，作用域内名字经别名表绑定到该全局槽。

```
static int done = 0;            →   if (!__sinit_7) { __static_7 = 0; __sinit_7 = 1; }
// 作用域内 done 的读/写           →   OP_PUSH_GLOBAL/OP_STORE_GLOBAL __static_7
```

**为何不选方案 B（VM 持久 frame 表）**：需要新 opcode 或 Expr 序列化字段 +
bytecode cache 格式 bump + VM 帧外存储表——为一个可纯编译层表达的语义引入
运行时状态面，违反最小真实改动。

**为何不用"程序加载时初始化"**：官方语义是**首达声明时**初始化（见 §0.4）。
加载时初始化对 `static int x = Bars` 这类运行期依赖初始化器**捏造**错误事实
（加载时 Bars=0）。旗标守卫忠实且对常量初始化器观察等价。

**为何 mangled 名注册进 `GlobalSlots` 而非新存储**：globals 数组已是持久存储；
`__static_`/`__sinit_` 前缀不可能撞用户标识符（MQL 标识符不允许双下划线前缀冲突
由 seq 唯一性兜底——seq 单调递增，同名不同位点必不同 N）。

## 2. 组件设计

### S1 — CST 层检测（`compile_interp.go`）

`compileDeclaration` 进入时检测首个 named child：

```go
// 镜像 isExternDeclaration(:961) 的既有先例
func isStaticDeclaration(n *sitter.Node, c *compiler) bool {
    first := n.NamedChild(0)
    return first != nil && first.Type() == "storage_class_specifier" && c.text(first) == "static"
}
```

声明点产出的 `ExprDecl` 置新字段 `Static bool`（`interp/ir.go:17` Expr struct 加字段——
IR 非序列化面，零兼容面）。同 declaration 内多声明符（`static int a=1,b=2;`）逐个置位。

**局部位置的非 static storage_class_specifier**（`extern`/`input`/`sinput` 在函数内——
MQL 本就非法，今天静默丢弃）：`compileDeclaration` 检出后 `c.err` fail-closed
编译错（诚实 > 静默）。顶层位置行为不变（`static` 顶层 ≡ 全局已正确，`extern`/`input`
走既有路径不动）。

### S2 — IR/字节码层绑定（`compile.go` + `compile_expr.go`）

```go
// astCompiler 新增——与 localScopes 平行的别名栈：
staticScopes []map[string]VarID // scope 层级的 name → 全局槽（isGlobal 语义）
staticSeq    int                // 单调递增 mangling 序号
```

- `pushScope`/`popScope`（`compile.go:245/250`）同步 push/pop `staticScopes`。
- `resolveVar`/`resolveVarWrite`（`compile.go:262/303`）的 scope 循环每层：
  先查 `localScopes[i]`（局部遮蔽优先），再查 `staticScopes[i]` 命中返 `(id, true)`。
  同 scope 同名 static+local 重定义 → `c.err` 编译错（MQL 即重定义错误）。
- `compileDecl`（`compile_expr.go:223`）`e.Static` 分支：
  1. `mangled := fmt.Sprintf("__static_%d", c.staticSeq); flag := "__sinit_"+…`；
     `c.staticSeq++`；两符号注册 `GlobalSlots`（`VarID(len(GlobalSlots))` 追加）。
  2. `staticScopes[top][e.Name] = varSlot`——**不**注册 localScopes、**不**占
     `nextLocalSlot`（不占 frame 槽——EventLocals 计数不受影响）。
  3. 发射 init-guard 字节码（全现成 opcode）：
     ```
     OP_PUSH_GLOBAL flagSlot
     OP_JMP_IF_TRUE  →end
       <compileExpr(e.Args[0])>          // 初始化器在守卫内——首达才求值
       OP_STORE_GLOBAL varSlot
       OP_PUSH_CONST 1 → OP_STORE_GLOBAL flagSlot
     end:
     ```
     数组初始化器（`static double arr[4]` → Args[0]=ExprArrayNew）同路径——
     `OP_NEW_ARRAY` 只在守卫内执行一次，天然正确。
- 事件处理器（`compileEventBody`）与用户函数（`compileUserFuncBody`）共用
  `pushScope`/`compileDecl` 路径——两侧自动覆盖，`staticSeq` 不重置（全编译期唯一）。

### S3 — 边界与不做

- **做**：函数内/块内/for-init 内 `static` 标量 + 数组；多声明符；递归共享；
  局部非 static storage class fail-closed。
- **不做**：`const` 限定符强制（另立项 VM-CONST-QUALIFIER-1 候选——需要写位置
  只读检查，独立范围）；顶层 `static`（语义已正确）；function_definition 上的
  `static`（MQL 非法——检测到则编译错，同 S1 fail-closed 分支）；Python 侧
  （无 static 概念）。

## 3. 测试设计（先红后绿）

| # | 用例 | 红证 |
|---|------|------|
| T1 | `OnTick{static int d; d++;}` 跨 3 次 OnTick → d=1,2,3 | 现状 1,1,1 |
| T2 | `static int d = Init()` 初始化器只跑一次（Init 副作用计数） | 现状每调用跑 |
| T3 | 循环内 `static int k`（`for` 体内）→ 首轮初始化后续迭代保持 | 现状每迭代重置 |
| T4 | 嵌套块同名 static（if/else 两枝各 `static int n`）→ 两独立存储 | 防 mangling 撞车 |
| T5 | 同名局部遮蔽：`{static int n} {int n}` 内层普通局部遮蔽正确 | 解析序正确性 |
| T6 | 递归函数内 static → 跨递归调用共享（全局语义） | 防 per-frame 实现 |
| T7 | `static double arr[4]` → 跨调用保持写入 | 数组路径 |
| T8 | 多声明符 `static int a=1,b=2` → 各自 init-once | 逐声明符旗标 |
| T9 | `static int x = Bars` 类运行期初始化器 → 首达值非加载值 | 懒初始化语义 |
| T10 | 局部 `extern`/`input` → compile error（fail-closed） | 原静默丢弃 |
| T11 | static + 读写混合表达式（`s++`、`s+=2`、`OrderSend(...s...)`） | 读写位绑定一致 |

**Mutation（验收必做）**：
- M1：`compileDecl` 删 `e.Static` 分支回退普通局部 → T1/T2/T3/T6 精确复红
- M2：`staticScopes` 不接入 resolveVar（static 名落空）→ T1 编译错红
- M3：init-guard 删旗标判定（每次重初始化）→ T2/T3 复红
- M4：`isStaticDeclaration` 恒 false（回到静默丢弃）→ 全部复红

**兼容门**：7 个真实用户策略 + 既有 golden/e2e 测试全量 strict-compile 零回归；
`go test ./tools/mql2go -count=1` 全绿 + race。

## 4. 验收门

- `go build ./...`、`go vet`、`check-file-lines --strict` 零 error
- mql2go 包全绿 + `-race -run Static` 专项
- mutation M1–M4 全 RED→恢复 GREEN
- 实盘探针复验：`static int done` 探针 tick1 打印后 tick2 静默（done 持久）
- registry 行 47 翻 ✅done + STATE.md 指针行更新

## 5. 风险

- `staticSeq` mangling 名进 `GlobalSlots`——`OrdersTotal` 类用户不可见面无泄漏
  （Globals 不对 MQL 暴露枚举）；`__` 前缀与用户标识符命名空间隔离。
- EventLocals 计数不受 static 影响（不占 frame 槽）——`vm.locals` 尺寸不变。
- 旗标槽为 ValNone→falsy 依赖 `IsTrue()` 语义——已有不变量，无新假设。
- 再编译确定性：staticSeq 按源码顺序单调分配——同源码同 bytecode，hash 稳定。
