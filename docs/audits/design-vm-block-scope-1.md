# 设计 SSOT — VM-BLOCK-SCOPE-1（if/else/loop/switch 块不建作用域）

> 状态：v1 立项（Devin CLI，2026-09-21）
> 触发：VM-STATIC-LOCAL-1 独立复审顺手发现——施工方偏差申报暴露 if/else 两枝共享 map 层
> registry 行 48：`🟦open`

---

## 0. 实查定论（全部读码坐实）

1. **裸块已正确**：`compileStmt`（`compile_interp.go:418`）对裸 `compound_statement`
   产 `StmtBlock` → astCompiler `compileStmt` case `StmtBlock`（`compile.go:483-486`）
   pushScope/popScope——**独立 `{}` 块作用域已正确**。
2. **缺口只在结构体分枝**：`compileBlock`（`compile_interp.go:366`）返回扁平
   `[]interp.Statement`（不产 StmtBlock）；`compileIf`/`compileWhile`/`compileDoWhile`/
   `compileSwitch`（CST 层）把块体直接铺进 `s.Body`/`ElseBody`；astCompiler 侧
   `compileIf`（`compile.go:504`）、`compileWhile`/`compileDoWhile`/`compileFor`/
   `compileSwitch`（`compile_loops.go`）对 Body 直接 `compileStmts` **不 pushScope**。
3. **后果**：`if(x){int n=1;}else{int n=2;} n++;` 可编译且 `n` 绑到 else 枝——
   真实 MQL 报 `n` undeclared。所有局部变量受影响（非 static 独有）。
   `compileFor` 有 pushScope（init+cond+body+update 共享一层）——for 循环变量
   域正确，但**循环体内声明与 for-init 变量同层**（`for(int i;;){int i;}` 不遮蔽
   而重绑定）。
4. **兼容扫描已做**（CST 级精确扫描器，非正则）：生产库 7 个真实用户策略 +
   仓内 4 个 .mq4 fixture 共 11 源，"块内声明→块外引用"泄漏形态**零命中**——
   严格块作用域对已知语料零破坏。

## 1. 设计决策：IR 层 pushScope 包裹（方案 A），非 CST 层 StmtBlock 包装

**方案 A（选定）**：astCompiler 的 5 个函数对体部 `compileStmts` 调用包
`pushScope`/`popScope`：

| 函数 | 位置 | 包裹面 |
|---|---|---|
| `compileIf`（compile.go:504） | `compileStmts(s.Body)` 与 `compileStmts(s.ElseBody)` 各自包一层 | then/else 独立域 |
| `compileWhile`（compile_loops.go:56） | `compileStmts(s.Body)` 包一层 | 循环体域 |
| `compileDoWhile`（compile_loops.go:82） | `compileStmts(s.Body)` 包一层 | 循环体域 |
| `compileFor`（compile_loops.go:8） | **内层**再包 `compileStmts(s.Body)`（保留外层 for 域罩 init/cond/update） | 体部独立域 |
| `compileSwitch`（compile_loops.go:105） | 整个 case 编译区包**一层**（cases 共享 switch 块域——C/MQL 语义：case 标签不建域） | switch 体域 |

**为何不选 CST 层 StmtBlock 包装（方案 B）**：需要改动 `compileIf`/`compileWhile`/
`compileDoWhile` 的 CST 产形（Body 从 []Statement 变 [StmtBlock{[]Statement}]）——
IR 形状变化面更大且 `compileBlock` 同时服务函数体（那里包 StmtBlock 是冗余层级）。
IR 层包裹零 IR 结构变更、零 CST 变更，5 个插入点语义等价。

**嵌套/花括号语义**：`else if` 链的 ElseBody 含嵌套 StmtIf→递归自包 ✓；
无花括号单语句体（`if(x) int n=1;`）在 s.Body 里同样是裸语句，同一包裹覆盖 ✓。

**static 交互**：修复后 `if(a){static int n=1;}else{static int n=2;}` 两枝进独立
scope 层——`staticScopes` 平行栈自动隔离，T4（兄弟枝独立存储）语义不变；
`if(x){static int n;} n++;` 后引用 n → 绑定随块消亡 → `unknown variable: n`
编译错（=真实 MQL 行为，正确收紧）。

## 2. 组件设计（唯一改动面：`compile.go` + `compile_loops.go`）

```
compileIf:    pushScope → compileStmts(Body) → popScope
              （else 存在时）pushScope → compileStmts(ElseBody) → popScope
compileWhile: pushScope → compileStmts(Body) → popScope   （loopStack 序不变）
compileDoWhile: 同上
compileFor:   既有 pushScope(for 域) 保留；compileStmts(s.Body) 外加一层 push/pop
compileSwitch: 整个 case 编译循环外包一层 push/pop（cases 共享一层——case 不建域）
```

**禁止改动**：`compileBlock`/`compileStmt` 的 StmtBlock 产形（裸块已正确）；
CST 层编译函数；`resolveVar`/`resolveVarWrite`/`staticScopes`（机制不变）。

## 3. 测试设计（新文件 `compile_block_scope_test.go`，先红后绿）

| # | 用例 | 预期 | 现状 |
|---|---|---|---|
| B1 | `if(g){int n=1;}else{int n=2;}` 各自独立（双枝各写各的 global 回读） | 独立 | 现状重绑定同层 |
| B2 | `if(x){int n=1;} n++;` → **compile error**（undeclared） | 拒 | 现状编译通过 |
| B3 | `for(;;){int x=0;}` 后引用 x → compile error | 拒 | 现状泄漏 |
| B4 | `while(){int y}` 后引用 y → compile error | 拒 | 同上 |
| B5 | `for(int i=0;i<2;i++){int i=9;}` 循环体 i 遮蔽 for-i（体内写不打乱循环计数） | 遮蔽 | 现状同层重绑定 |
| B6 | `if(x){static int n=1;}else{static int n=2;}` 兄弟枝 static 仍独立存储（T4 不退化） | 绿 | 防回归 |
| B7 | `if(x){static int n=1;} n++;` → compile error | 拒 | static 亦收域 |
| B8 | 裸 `{int x=1;}` 块行为不变（原已正确——回归 pin） | 绿 | — |
| B9 | `do{}while` 体 decl 泄漏拒 | 拒 | 同上 |
| B10 | `switch` 体 decl：`switch(x){case 1:int n=0;}` 后引用 n → error；case 间共享域（`case1` 声明 `case2` 引用=同层可达，与 C 一致——pin 住此宽松） | 一层域 | — |
| B11 | 嵌套 `if{if{}}` 三层深度遮蔽链 | 绿 | — |

**Mutation（验收必做）**：
- M1：`compileIf` 删两个 pushScope/popScope → B1/B2/B7 复红
- M2：`compileFor` 删内层包裹 → B3/B5 复红
- M3：全部包裹回退 → B2/B3/B4/B7/B9 复红

**兼容门**：施工单附带扫描器（`block_scope_scan_test.go` 临时测试，审计侧已写好可用——
**注意它是审计脚手架，不进最终 diff**：跑完记录结果后删除）。11 源零命中已实证；
施工方重跑一次贴输出。

## 4. 验收门

- mql2go 全量绿（含 golden/e2e——若有存量测试依赖泄漏形态会先红，**停下回报**不许改 fixture 硬过）
- race×3、check-lines 零 error、build/vet/gofmt/diff-check
- 落档：registry 行 48 翻施工完成态、STATE.md、handover-audit-plan 追加

## 5. 风险

- **语义收窄风险**：此前可编译的泄漏代码变编译错——兼容扫描已证真实语料零命中；
  若 CI/存量 fixture 命中则回报决策（选项：修 fixture 或降级为 warning 模式）。
- `for` 体新增内层 scope → 体部 decl 与 for-init 同名时行为从"重绑定"变"遮蔽"——
  属正确化，MQL 一致。
- `staticScopes` 无需改动：它是 localScopes 的平行栈，pushScope/popScope 一变俱变。
