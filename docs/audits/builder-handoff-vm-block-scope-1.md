# 施工提示词：VM-BLOCK-SCOPE-1（if/else/loop/switch 块作用域收紧）

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S5 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后自报证据（机检五件套 + 对抗证明），停手等最终决策者复审。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015 豁免 STATE.md 必更门禁）。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

VM-STATIC-LOCAL-1 独立复审发现：`compileBlock`（`compile_interp.go:366`）返扁平 `[]Statement`，`compileIf`（`compile.go:504`）等对 Body/ElseBody 直接 `compileStmts` 不 pushScope——`if(x){int n=1;}else{int n=2;} n++;` 可编译且 n 绑到 else 枝（真实 MQL 报 undeclared）。裸 `{}` 块已正确（StmtBlock 路径），缺口只在结构体分枝。**兼容扫描已实证**：7 个真实用户策略 + 4 仓内 fixture 零泄漏命中（扫描器在 `tools/mql2go/block_scope_scan_test.go`，决策方已写好——**它是审计脚手架，跑完贴输出后删除，不进最终 diff**）。registry 行 48。

## 设计 SSOT 声明

- 设计文档：`docs/audits/design-vm-block-scope-1.md`（唯一真相源）
- 方案锁定=**IR 层 pushScope 包裹**（方案 A）。CST 层 StmtBlock 包装已否决。偏离 = 停下回报。

## 约束与目标

- 目标：if/else/while/do-while/for 体、switch 体的块作用域收紧到 MQL 语义；裸 `{}` 块行为不变（已正确）。
- ≤450 行红线；禁 TODO/HACK/静默兜底；只显式 add 本任务文件。

## 边界 / 不做

- 不改 `compileBlock`/`compileStmt`/CST 层任何函数；不改 `resolveVar`/`resolveVarWrite`/`staticScopes`。
- 不改 VM 运行时、不动 bytecode 格式。
- 函数体本身不包（参数与函数顶层 decl 同层是既有语义，不在本范围）。
- Python 路径不动。不部署、不 push。

## 施工指令

### S1 — `compileIf` 双枝包裹

- **目标**：then/else 体各自独立作用域。
- **坐标**：`tools/mql2go/compile.go:504`（`func (c *astCompiler) compileIf`）。
- **落点**：`c.compileStmts(s.Body)` → `c.pushScope(); c.compileStmts(s.Body); c.popScope()`；`ElseBody` 同法。`compileStmts` 本身不改。
- **验证**：B1/B2/B7 绿。

### S2 — `compileWhile`/`compileDoWhile` 体包裹

- **目标**：循环体独立域。
- **坐标**：`tools/mql2go/compile_loops.go:56`（compileWhile）、`:82`（compileDoWhile）。
- **落点**：两处 `c.compileStmts(s.Body)` 各包 `pushScope`/`popScope`（loopStack push/pop 序不动）。
- **验证**：B4/B9 绿。

### S3 — `compileFor` 内层体包裹 + `compileSwitch` 单层包裹

- **目标**：for 体独立域（for-init 变量域不变）；switch 整体一层域（cases 共享，case 不建域——C 语义）。
- **坐标**：`tools/mql2go/compile_loops.go:8`（compileFor，既有 pushScope 保留，`compileStmts(s.Body)` 外加一层）、`:105`（compileSwitch，整个 case 编译区外包一层）。
- **落点**：compileFor 内 `c.compileStmts(s.Body)` → 包裹；compileSwitch 在 loopStack push 之后、所有 case 编译之外包一层 pushScope/popScope。
- **验证**：B3/B5/B10 绿。

### S4 — 测试 `compile_block_scope_test.go`（先红后绿）

- **目标**：设计文档 §3 的 B1–B11 全量落地。基建参照 `compile_static_test.go`（compileStaticSrc/runStaticTicks/staticGlobalInt 可复用模式——**复制到新文件**，不回改 static 测试）。
- **落点**：B1 兄弟枝独立；B2/B3/B4/B7/B9 块外引用 compile error；B5 for-init 遮蔽正确；B6 static 兄弟枝不退化；B8 裸块 pin；B10 switch 一层域（含 case 间共享 pin）；B11 三层嵌套遮蔽链。
- **验证**：修复前先跑红（贴输出），修复后全绿。

### S5 — 门禁 + 兼容扫描 + 落档

- **落点**：`go build ./...`；`go test ./tools/mql2go -count=1` 全绿（**存量 golden/e2e 若因收紧变红——停下回报，禁改 fixture 硬过**）；`go test -race -run 'Static|BlockScope' ./tools/mql2go` ×3；`go vet`；`gofmt`；`check-file-lines --strict` 零 error；`git diff --check`；重跑 `TestBlockScopeCompatScan` 贴输出后**删除该脚手架文件**；registry 行 48 填施工完成态（不标 ✅done）；`handover-audit-plan.md` 追加日志。

## 验收标准

- [ ] `go build ./...` 通过
- [ ] `go test ./tools/mql2go -count=1` 全过（零新增失败）
- [ ] `check-file-lines --strict` 零错误
- [ ] `gofmt`/`go vet` 零警告
- [ ] `go test -race` ×3 通过
- [ ] 对抗证明：M1–M3 mutation RED→restore→GREEN（附命令与输出）
- [ ] diff 无死代码/TODO/调试残留/范围外改动/脚手架残留

## 施工完成自审（强制，D-012）

- [ ] 逐项重跑验收标准贴真实输出
- [ ] 红队自审 diff 三问：更简等价方案 / 边界·nil·并发 / 逆向依赖
- [ ] 自审发现项已修至全绿并列出
- [ ] 无自审记录 = 复审直接退回

## 交付格式（D-016 六段）

1. 变更文件清单
2. S1–S5 实现摘要（对码坐标）
3. 对抗证明（M1–M3 RED→GREEN 命令+输出）
4. 机检门禁逐项真实输出
5. 范围确认（仅派工单文件）
6. 结束语：`[施工完成:VM-BLOCK-SCOPE-1] @<commit-hash>`

**停手等 Devin CLI 复审；不达标退回。**

---

## 决策方自审记录（D-013）

- 代码事实回验：compileBlock 扁平返回（:366）、compileIf 无 pushScope（compile.go:504 读码实证）、裸块 StmtBlock 已正确（compile_interp.go:418→compile.go:483）、compileFor 已有 pushScope、5 个插入点逐一核实。
- 兼容门实证：扫描器对 11 源（7 真实策略+4 fixture）零泄漏命中（初版误报 Symbol() 调用名已修正为只取声明符首 identifier，复扫 CLEAN）。
- 复用核对：复用既有 pushScope/popScope + StmtBlock 机制，零新基础设施。
- static 交互推演：修复后兄弟枝同名 static 进独立层，T4 语义不变；块后引用 static 名正确转编译错（B7）。

## 派工指令

```
[角色:施工] 开工：读 docs/audits/builder-handoff-vm-block-scope-1.md @<commit-hash>，按 S1 施工。串行，勿部署勿 push，完成报证据等复审。
```
