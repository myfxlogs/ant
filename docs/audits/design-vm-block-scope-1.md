# 设计 SSOT — VM-BLOCK-SCOPE-1（if/else/loop/switch 块不建作用域）

> 状态：**v2 修订**（Devin CLI，2026-09-21）——v1 经决策方独立审计发现 4 处事实错误/缺陷，本文已全部修正（修订点见 §6）
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
3. **后果**（决策方审计实测，纯读位探针）：`if(g>0){int n=1;} g=n;` 可编译且
   `n` 泄漏到外层——真实 MQL 报 `n` undeclared。实测矩阵：if/else/while/
   do-while/switch 体 decl→块外**读**全部泄漏；嵌套 if 内层 decl 对中间层可见
   （泄漏）；**for 体 decl 不泄漏**（compileFor 外层 pushScope 罩住 init+cond+
   body+update——实测 `for(…){int x=7;} g=x;` 已拒）——for 的唯一缺口是
   **体 decl 与 for-init 同层重绑定不遮蔽**（`for(int i;;){int i=9;}`）；裸 `{}`
   块已正确（`{int b=1;} g=b;` 实测拒）；switch case 间共享同层（=C 语义，pin）。
   所有局部变量受影响（非 static 独有；`if(g){static int s=1;} g=s;` 同漏）。
4. **兼容扫描已做**（CST 级精确扫描器，非正则）：生产库 7 个真实用户策略 +
   仓内 4 个 .mq4 fixture 共 11 源，"块内声明→块外引用"泄漏形态**零命中**——
   严格块作用域对已知语料零破坏。扫描器已入库为常驻回归门
   （`block_scope_scan_test.go`，b388b94e，空跑/泄漏均 FAIL）。
5. **边界（v2 新增）**：块外**写位/更新位**引用（`n=5`、`n++`）走
   `resolveVarWrite`——mql4 保留隐式注册 shim，未声明名写位注册成**新全局**
   而非报错。这是预存的有意行为（write-shim），**不在本债范围**；因此所有
   "块外引用→拒"用例必须用**纯读位**（`g=n`），写位形态修复后仍编译过，
   拿它当红测会假绿。

## 1. 设计决策：IR 层 pushScope 包裹（方案 A），非 CST 层 StmtBlock 包装

**方案 A（选定）**：astCompiler 的 5 个函数对体部 `compileStmts` 调用包
`pushScope`/`popScope`：

| 函数 | 位置 | 包裹面 |
|---|---|---|
| `compileIf`（compile.go，`func (c *astCompiler) compileIf`） | `compileStmts(s.Body)` 与 `compileStmts(s.ElseBody)` 各自包一层 | then/else 独立域 |
| `compileWhile`（compile_loops.go） | `compileStmts(s.Body)` 包一层（loopStack 序不动） | 循环体域 |
| `compileDoWhile`（compile_loops.go） | `compileStmts(s.Body)` 包一层 | 循环体域 |
| `compileFor`（compile_loops.go） | **内层**再包 `compileStmts(s.Body)`（保留外层 for 域罩 init/cond/update——顺序：pushScope(for)→init→cond→pushScope(body)→body→popScope(body)→update→jmp→popScope(for)） | 体部独立域 |
| `compileSwitch`（compile_loops.go） | 整个 case 编译区包**一层**（cases 共享 switch 块域——C/MQL 语义：case 标签不建域） | switch 体域 |

> 行号坐标会随前序提交漂移——以函数名+文件为准，不许按行号盲改。

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

| # | 用例（后引用一律**纯读位** `g=x`，禁写位 `x=`/`x++`——§0.5） | 预期 | 现状（审计实测） |
|---|---|---|---|
| B1 | `if(g>0){int n=1;ga=n*10;}else{int n=2;gb=n*10;}` 兄弟枝同名 decl 各占各槽 | 独立 | **已绿**（重绑定在编译期，两枝代码各绑各槽，实测 gb=20）——pin 防退化 |
| B2 | `if(g>0){int n=1;} g=n;` → **compile error** | 拒 | 红：编译过（泄漏） |
| B3 | `for(int i=0;i<2;i++){int x=7;} g=x;` → compile error | 拒 | **已绿**（for 外层域罩住 body）——pin 防退化 |
| B4 | `while(g<0){int y=1;} g=y;` → compile error | 拒 | 红：泄漏 |
| B5 | `for(int i=0;i<2;i++){int i=9;}` 体 i 遮蔽 for-i（循环按 for-i 正常计满 2 圈） | 遮蔽 | 红：同层重绑定 |
| B6 | `if(g>0){static int n=1;}else{static int n=2;}` 兄弟枝 static 独立存储（T4 不退化） | 绿 | 防回归 pin |
| B7 | `if(g>0){static int n=1;} g=n;` → compile error | 拒 | 红：static 名同漏（实测） |
| B8 | 裸 `{int x=1;} g=x;` → compile error（原已正确——回归 pin） | 拒 | **已绿** pin |
| B9 | `do{int z=1;}while(g<0); g=z;` → compile error | 拒 | 红：泄漏 |
| B10 | `switch(g){case 1:int w=0; break;} g=w;` → error；`case1` decl `case2` 内读=可达（同层共享，C 语义 pin） | 一层域 | 红：外漏拒→绿；共享 pin 现已成立 |
| B11 | 嵌套 `if{if{}}` 三层遮蔽链：每层同名 decl 各绑各层、读各见各层 | 绿 | — |
| B12 | `if(g>0){if(g>1){int q=3;} g=q;}` → compile error（内层 decl 对中间层不可见） | 拒 | 红：泄漏（审计实测） |

**Mutation（验收必做，v2 修正映射）**：
- M1：`compileIf` 删两个 pushScope/popScope → B2/B7/B12 复红（B1 运行时本就独立，是 pin 不承重）
- M2：`compileFor` 删**内层**包裹 → **B5 复红**（B3 不红——外层 for 域仍罩住，不算证据）
- M3：`compileWhile`+`compileDoWhile`+`compileSwitch` 包裹全回退 → B4/B9/B10 复红
- 每个构造的包裹各自承重：删任一处→对应 B 用例红，缺一即退回

**兼容门**：`block_scope_scan_test.go` 已入库为常驻回归门（b388b94e——repo
fixtures + `/tmp/strats/*.mq4` 外部语料，空跑或泄漏均 FAIL）。施工方重跑一次
贴输出，**文件保留不许删**（v1 施工单"跑完删除"指令作废）。

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

## 6. v1→v2 修订记录（决策方独立审计产出）

| # | v1 错误/缺陷 | v2 修正 |
|---|---|---|
| 1 | §3-B3「for 体 decl 现状泄漏」 | 实测已拒（for 外层域罩住）——B3 改 pin 用例；for 唯一缺口=B5 遮蔽 |
| 2 | B2/B4/B7/B9 等「后引用」未限定读位 | `n++`/`n=` 写位走 resolveVarWrite 隐式注册全局假绿——全部钉死纯读位 `g=x` |
| 3 | M2 含 B3、M3 含 B3 | B3 永不受 for 内层包裹影响——M2 只 B5 承重；M3 收窄 while/do/switch |
| 4 | 施工单「扫描器跑完删除」 | 扫描器已入库为常驻回归门（b388b94e），指令作废 |
| + | 新增 | §0.5 写位 shim 边界声明；B12 嵌套中间层泄漏用例；坐标改函数名锚定；B1 修正为已绿 pin（兄弟枝 decl 编译期重绑定→运行时本就各占各槽，实测 gb=20，非红用例） |
