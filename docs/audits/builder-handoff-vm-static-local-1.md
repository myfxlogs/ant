# 施工提示词：VM-STATIC-LOCAL-1（函数内 `static` 局部变量持久化）

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S6 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后自报证据（机检五件套 + 对抗证明），停手等最终决策者复审。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015 豁免 STATE.md 必更门禁）。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

全功能实盘探针实证：`OnTick(){ static int done; if(done) return; done=1; Print(...); }` **每 tick 都打印**——`done` 不持久。根因已闭合：tree-sitter CST 实证 `static int done = 0;` → `declaration→[storage_class_specifier("static"), primitive_type, init_declarator]`，而 `compileDeclaration`（`compile_interp.go:619`）的 named-child switch 只认 `init_declarator`/`declarator`/`identifier`/`array_declarator`——`storage_class_specifier` 落 default 被**静默丢弃**，static 编译成普通 frame 局部、每调用重初始化。MQL 官方语义已核（MQL5 reference《Declaration/definition statements》）：static 元素"在声明语句首次执行时**创建一次**"——懒初始化。生产库 7 个真实用户策略 `\mstatic\M` 零命中，零兼容风险。registry 行 47。

## 设计 SSOT 声明

- 设计文档：`docs/audits/design-vm-static-local-1.md`（唯一真相源，先全文读完再动手；本提示词仅做执行索引）
- 相关契约：`AGENTS.md` §4-§7 / ADR-0023（MQL→Bytecode VM）
- **方案锁定=编译层脱糖**（mangled global + init-guard）。VM 层持久 frame 方案已否决（不动 opcode/bytecode cache 格式）。偏离设计 = 停下回报，不得自行换方案。

## 约束与目标

- 目标：函数/块/for-init 内 `static` 局部跨调用持久 + 初始化器首达执行一次；局部非法 storage class（`extern`/`input`/`sinput`）fail-closed 编译错（原静默丢弃）。
- Go 文件 ≤450 行红线（`compile_interp.go`/`compile_expr.go`/`compile.go` 改前自查行数，逼近则向决策方报拆分方案）。
- 禁 TODO/HACK/静默兜底；收工只显式 add 本任务文件。

## 边界 / 不做

- `const` 限定符强制不做（另债候选，禁止顺手修）。
- 顶层 `static int g` 不动（单 TU 语义已正确）。
- Python 编译路径（`compile_py_*`）不动。
- `interp.Expr` 不进序列化面已核实——只许加字段，禁动 bytecode cache 格式。

## 施工指令

### S1 — IR 字段 + CST 检测

- **目标**：`static` 在 CST→IR 层被捕获并打标。
- **坐标**：`tools/mql2go/interp/ir.go:7-18`（`Expr` struct）；`tools/mql2go/compile_interp.go:619`（`compileDeclaration`）、`:961`（`isExternDeclaration` 先例）。
- **落点**：
  1. `Expr` 加 `Static bool`（紧跟 `IsAssign` 字段后，注释标明 ExprDecl 用）。
  2. 新 helper（紧邻 `isExternDeclaration`）：`staticClassSpecifier(n, c) string`——首 named child 为 `storage_class_specifier` 则返其 text，否则 `""`。
  3. `compileDeclaration` 函数头：`classSpec := staticClassSpecifier(n, c)`；`classSpec=="static"` → 后续所有 declarator 产出的 `ExprDecl` 置 `Static: true`；`classSpec` 非空且非 `"static"` → `c.err = fmt.Errorf("unsupported storage class %q in local declaration ...")` 返回 nil；首 named child 为 `type_identifier` 且 text∈`{input,sinput}` → 同法 fail-closed。
  4. 函数定义上的 `static`：施工第一步先写临时 dump 测试实证 `function_definition` 的 specifier 节点形态（先例 dump 已证 local 位形态），形态确认后在函数编译入口同样拒（坐标待实证回报后按实际节点位置落点，禁止猜）。
- **验证**：临时 dump 输出贴入自报；`static int x=1;` 编译出的 IR 中 ExprDecl.Static=true。

### S2 — 别名栈（astCompiler）

- **目标**：`name → 全局槽` 的作用域级绑定通道。
- **坐标**：`tools/mql2go/compile.go:172`（字段区）、`:245/:250`（`pushScope`/`popScope`——localScopes 栈唯一改动点，已核全仓 5 调用点全走 helper）、`:262`（`resolveVar`）、`:303`（`resolveVarWrite`）。
- **落点**：
  1. `astCompiler` 加 `staticScopes []map[string]VarID` + `staticSeq int`。
  2. `pushScope`/`popScope` 同步维护 `staticScopes`（长度不变量：与 localScopes 恒等）。
  3. `resolveVar`/`resolveVarWrite` 的 scope 循环每层顺序：先 `localScopes[i]`（命中返 `(id,false)`），再 `staticScopes[i]`（命中返 `(id,true)`）。其余分支（globals/constant/python/unknown-error）不动。
- **验证**：S4 的 T5 遮蔽用例证明解析序。

### S3 — init-guard 字节码发射

- **目标**：`ExprDecl{Static:true}` → mangled 全局 + 旗标守卫的 init-once 序列。
- **坐标**：`tools/mql2go/compile_expr.go:223`（`compileDecl`）。
- **落点**（函数顶分流，非 static 原路径原样）：

```go
if e.Static {
    if len(c.staticScopes) == 0 { c.err = fmt.Errorf("static local outside scope"); return }
    seq := c.staticSeq; c.staticSeq++
    varName := fmt.Sprintf("__static_%d", seq)
    flagName := fmt.Sprintf("__sinit_%d", seq)
    varSlot := VarID(len(c.bc.GlobalSlots)); c.bc.GlobalSlots[varName] = varSlot
    flagSlot := VarID(len(c.bc.GlobalSlots)); c.bc.GlobalSlots[flagName] = flagSlot
    top := len(c.localScopes) - 1
    if _, dup := c.localScopes[top][e.Name]; dup { c.err = fmt.Errorf("redefinition: %s", e.Name); return }
    if _, dup := c.staticScopes[top][e.Name]; dup { c.err = fmt.Errorf("redefinition: %s", e.Name); return }
    c.staticScopes[top][e.Name] = varSlot
    c.emit(OP_PUSH_GLOBAL, int32(flagSlot), 0, 0)
    jmpEnd := c.emitJump(OP_JMP_IF_TRUE, 0)
    c.compileExpr(&e.Args[0])                 // 初始化器在守卫内——首达才求值
    c.emit(OP_STORE_GLOBAL, int32(varSlot), 0, 0)
    c.emit(OP_PUSH_CONST, int32(c.addConst(interp.IntVal(1))), 0, 0)
    c.emit(OP_STORE_GLOBAL, int32(flagSlot), 0, 0)
    c.patchJump(jmpEnd)
    return
}
```

  注意：**不**注册 `localScopes`、**不**占 `nextLocalSlot`（EventLocals 计数不受影响，已核 `compile.go:399`）；`e.Args[0]` 含 `ExprArrayNew`（静态数组 `static double arr[4]`）时同路径——`OP_NEW_ARRAY` 只在守卫内执行一次。
- **验证**：S4 T1/T2/T7 绿；`vm.locals` 尺寸不变（EventLocals 不含 static）。

### S4 — 测试（新文件 `tools/mql2go/compile_static_test.go`，先红后绿）

- **目标**：设计文档 §3 的 T1–T11 全量落地。执行基建参照 `compile_implicit_var_test.go`/`live_mql_order_context_vm_test.go`。
- **落点**：T1 跨 3 次 OnTick `d++`→1,2,3（现状红：1,1,1）；T2 初始化器副作用单次；T3 for 体内 static 保持；T4 if/else 同名 static 独立；T5 内层普通局部遮蔽 static；T6 递归共享；T7 `static double arr[4]`；T8 `static int a=1,b=2` 各自 once；T9 `static int x=Bars` 取首达值；T10 局部 `extern`/`input` 编译错；T11 `s++`/`s+=2`/实参传递读写一致。
- **验证**：修复前 T1/T2/T3 等先红（贴红输出），修复后全绿。

### S5 — Mutation 红证

- **目标**：四条 mutation 逐条 RED→restore→GREEN，命令+关键输出贴自报。
- **落点**：M1 删 `compileDecl` 的 `e.Static` 分支 → T1/T2/T3/T6 复红；M2 resolveVar/resolveVarWrite 不查 `staticScopes` → T1 编译错红；M3 删旗标判定 → T2/T3 复红；M4 `staticClassSpecifier` 恒返 `""` → 全部复红。

### S6 — 门禁 + 落档

- **目标**：机检全绿 + 文档同步。
- **落点**：`go build ./...`；`go test ./tools/mql2go -count=1`（含 golden/e2e 零新增失败）；`go test -race -run Static ./tools/mql2go` ×3；`go vet`；`gofmt`；`cd backend && go run ./tools/check-file-lines --strict` 零 error；`git diff --check`；registry 行 47 填施工完成态（不标 ✅done）；`docs/audits/handover-audit-plan.md` 追加变更日志一行。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go -count=1` 全过
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误
- [ ] `gofmt` / `go vet` 零警告
- [ ] `go test -race -run Static ./tools/mql2go` ×3 通过
- [ ] 对抗证明：M1–M4 mutation RED → restore → GREEN（附命令与输出）
- [ ] diff 通读无死代码 / TODO / 调试残留 / 范围外改动

## 施工完成自审（强制，D-012）

交付自报前必须完成并随报提交：
- [ ] 逐项重跑上方验收标准并贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖或重复基础设施
- [ ] 自审发现的缺陷已修复至全绿（自报列出发现项+修复项）
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报必须按以下六段顺序（D-016，缺一 = 复审直接退回）：
1. **变更文件清单**：列出本任务改动的所有文件。
2. **S1–S6 实现摘要**：每步落点对码（改了什么、在哪、是否符合派工单坐标）。
3. **对抗证明**：M1–M4 mutation RED → restore → GREEN 的命令与关键输出。
4. **机检门禁**：build / test / race×3 / vet / gofmt / check-lines / diff --check 逐项真实输出。
5. **范围确认**：仅改派工单列出的文件，无范围外改动。
6. **结束语**：`[施工完成:VM-STATIC-LOCAL-1] @<commit-hash>`（D-014，无此行 = 未交付，复审不启动）。

**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。**

---

## 决策方自审记录（D-013）

- 代码事实回验：`compileDeclaration:619` switch 不含 storage_class_specifier（读码实证）；`resolveVar`/`resolveVarWrite` 双层结构、pushScope/popScope 唯一栈改动点、`compileFor` 已 pushScope 包 init、`EventLocals=nextLocalSlot`、`IsTrue(ValNone)=false`（value.go:82）、`OP_PUSH_GLOBAL/STORE_GLOBAL/JMP_IF_TRUE/patchJump/addConst` 全现成、IR 不进 cache 序列化面——逐条已核。
- CST 实证：local 位 `static`=`storage_class_specifier` named child（dump 输出已验）；`function_definition` 位形态待施工方先 dump 再落点（S1.4 已写明实证义务，非歧义）。
- 兼容面：7 个真实用户策略 `static` 零命中；非 static specifier 局部拒收为该族缺陷的 fail-closed 收口，真实 MQL 本就非法。
- 复用核对：`staticClassSpecifier` 镜像 `isExternDeclaration:961` 既有先例；别名栈复用 pushScope/popScope 不变量。

## 派工指令

```
[角色:施工] 开工：读 docs/audits/builder-handoff-vm-static-local-1.md @<commit-hash>，按 S1 施工。串行，勿部署勿 push，完成报证据等复审。
```
