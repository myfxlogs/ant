# 施工提示词：QS-1.4 Python `bool(x)` 语义修复 + `vm_helpers.go:250` 注释修正

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 严格按 S1–S3 施工，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后自报证据（机检五件套 + 对抗证明），停手等最终决策者复审。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

- registry `QS-1.4`；spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` v2 §3.1（设计 SSOT）。
- 2026-09-16 VM 深度审计 P1-4：`backend/tools/mql2go/compile_py_expr.go` `case "bool"` 把 `bool(x)` 编译为 `x != 0`；`interp/value.go` `Value.Equal` 对 None/Int 混合返回 false → `bool(None)` 得到 true，与 Python 语义相反。
- 顺手项（原 QS-1.1 降级而来）：`backend/tools/mql2go/vm_helpers.go:250` 注释"Non-fatal: Object/Chart/File operations — silent blind spot"过时——这些符号在 `unsupportedSymbols`（`interp/api_registry.go`）已在编译期被 `compile_expr.go` 拒绝，运行时不会到达此分支。

## 设计 SSOT 声明

- 设计文档：`docs/spec/vm-pipeline-quality-stability-improvement-plan.md` §3.1（唯一真相源，本提示词仅做执行索引）。
- 相关契约：`AGENTS.md` §1 fail-closed、§7.3 自审 A–F；决策 `docs/handoff/decisions.md` D-009。

## 约束与目标

- `bool(x)` 必须等价于 Python 语义：`bool(None)`/`bool(0)`/`bool(0.0)`/`bool("")`/`bool(False)` → false；`bool(1)`/`bool("a")`/`bool(非空)` → true。
- 不新增 builtin、不新增 opcode；复用 `OP_NOT`（执行见 `vm_execute.go` `case OP_NOT: vm.push(interp.BoolVal(!a.IsTrue()))`）与 `Value.IsTrue()`（`interp/value.go`）。
- 单 commit，只显式 add 本任务文件。

## 边界 / 不做

- 不改 `Value.Equal`、不改 `IsTrue`、不改 OP_NOT 实现。
- 不动 `case "int"`/`case "str"`/`case "Decimal"` 等其他转换分支。
- 不动 `vm_helpers.go` 除 :250 注释以外的任何代码。
- 不处理 QS-1.3（Python 局部作用域）等其他 QS 条目。

## 施工指令

### S1 — 修正 `bool(x)` 编译形态

- **目标**：`bool(x)` 生成双重逻辑非而非 `x != 0`。
- **坐标**：`backend/tools/mql2go/compile_py_expr.go`，`case "bool":` 分支（约 :165-172）。
- **落点**：把返回的 `ExprBinary{Op:"!=", Args:[x, IntVal(0)]}` 改为：
  `&interp.Expr{Kind: interp.ExprUnary, Op: "!", Args: []interp.Expr{{Kind: interp.ExprUnary, Op: "!", Args: []interp.Expr{args[0]}}}}`
  零参数分支保持不变（返回 `BoolVal(false)`）。形态先例见同文件 :57-64 与 `compile_py_compare.go` "not in" 分支。
- **验证**：编译 `class S: def on_bar(self): if bool(None): ctx.log("x")` 不再产出 `!=`；运行行为由 S2 测试断言。

### S2 — 修正过时注释

- **目标**：`vm_helpers.go:250` 注释与注册表真实行为一致。
- **坐标**：`backend/tools/mql2go/vm_helpers.go:250`。
- **落点**：改为说明"到达此处 = builtins 表有名但无 handler 且注册表未标 fatal；`StatusUnsupported`（Object/Chart/File 等）在编译期已由 compile_expr.go 拒绝，不会到此"。只改注释文字，不删不改任何代码行。
- **验证**：`git diff` 该行仅为注释文本变化。

### S3 — 行为级测试（先红后绿）

- **目标**：新测试断言 `bool()` 各边界。
- **坐标**：`backend/tools/mql2go/` 测试包（参考既有 Python 编译/运行测试如 `compile_py_test.go`、`vm_round45_batch1_test.go` 的 CompilePython + Execute 调用方式）。
- **落点**：测试用例最小集：`bool(None)`→false、`bool(0)`→false、`bool("")`→false、`bool(False)`→false、`bool(1)`→true、`bool("a")`→true。断言方式为运行编译后字节码观测结果（行为级，非仅检查 IR 形态）。
- **验证**：先在 S1 实施**之前**跑一次新测试确认 `bool(None)` 断言失败（先红）；S1 实施后转绿（后绿）。

## 对抗证明（缺一即未完成）

- mutation：把 S1 的表达式改回 `ExprBinary{Op:"!=", ...}`（或删内层一层 `!`）→ `bool(None)` 用例必须 RED；restore → GREEN。附真实命令与输出。

## 验收标准

- [ ] `cd backend && go build ./...` 通过
- [ ] `cd backend && go test ./tools/mql2go/...` 全过
- [ ] `cd backend && go run ./tools/check-file-lines --strict` 零错误零警告
- [ ] `gofmt -l` / `go vet ./tools/mql2go/...` 零输出
- [ ] `go test -race -count=3 ./tools/mql2go/...` 通过
- [ ] 对抗证明 RED→restore→GREEN 附命令与输出
- [ ] diff 通读无死代码 / TODO / 调试残留 / 范围外改动

## 交付格式

自报：改动文件清单 + 每条验收项证据（命令+关键输出）+ 遗留疑问。
**停手等 Devin CLI 复审；不达标将由决策方开缺陷清单退回。**
