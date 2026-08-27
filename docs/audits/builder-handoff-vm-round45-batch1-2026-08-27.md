# Builder Handoff: VM-COMPILER-SEMANTICS-4 + VM-CACHE-INTEGRITY-5（Batch 1）

> **设计/验收方**：Devin CLI
> **施工方**：Devin IDE / Windsurf
> **基线 HEAD**：`260460a4`（工作树干净）
> **边界**：只施工这 2 个 ID 的修复，禁改写历史审计事实，禁扩 scope，禁 commit/push/deploy。
> **施工后状态**：`🟦open（施工完成，待独立复审）`，不得自标 ✅done。

---

## 立项背景

D-REVERT-SCOPE-DRIFT-001 回滚了 commit `830b2c79`，把 VM round 4-5 的几乎所有修复代码删除。当前代码库中 VM-COMPILER-SEMANTICS-4 和 VM-CACHE-INTEGRITY-5 的修复均不存在，需从零重做。

### VM-COMPILER-SEMANTICS-4（P1 编译器正确性）

**根因**：
1. `compile_interp_expr.go:107-116` `comma_expression` 只返回 last expression（`var last *interp.Expr`），丢弃前面子表达式的副作用。MQL `for(int i=0,j=10; ...)` 中 `i=0,j=10` 的 `i=0` 被丢弃。
2. `compile_interp.go:101,113` 用 `strings.Contains(txt, "input ")` 检测 input/extern 声明——字符串匹配误放行 `int x = input ;` 等非法用法。
3. 无 `HasError()` guard——`CompileMQL("int x = ;")` 静默返回 nil 而非报错。

### VM-CACHE-INTEGRITY-5（P1 缓存完整性）

**根因**：
1. `CompilePythonCached`（`interp_runner.go:102-120`）cache hit 时不恢复 `CoverageResult`——coverage 分析丢失。
2. 无 `Version == "python"` 语言校验——MQL bytecode 可被用于 Python source。
3. 无 `maxBytecodePayload` 上限——损坏的超大缓存可造成资源耗尽。

---

## 🔴 绝对边界（违反 = 直接判失败）

1. **只改** `tools/mql2go/` 下的文件 + 新建测试文件 + 文档。**禁止改** `internal/connect/strategy/` 生产代码（本批不涉及 live 路径）。
2. 禁止改 proto / DB schema / 部署 / 其他功能块。
3. 禁止 commit / push / deploy。禁 `--no-verify`。
4. 收工只显式 `git add` 本任务涉及的文件，禁止 `git add -A`。

---

## 施工步骤

### VM-COMPILER-SEMANTICS-4

- **S1** `compile_interp_expr.go:107-116`：`comma_expression` case 改为生成 `ExprSeq`（收集所有子表达式的 `*interp.Expr` 为 `[]*interp.Expr`，返回 `&interp.ExprSeq{Exprs: exprs}`），不再只返回 last。保留 left-to-right 求值顺序。

- **S2** `compile_interp.go`：新增 `HasError(n tree.Node) bool` 函数——递归检查 node 的 named children 是否含 `type == "ERROR"` 节点。在 `CompileToIR` 中对每个 top-level named child 调 `HasError`，命中则 `return nil, fmt.Errorf("syntax error in %s node at line %d", n.Type(), n.StartPoint().Row)`。

- **S3** `compile_interp.go`：替换 `strings.Contains(txt, "input ")` / `strings.Contains(txt, "extern ")` 为结构化检测：
  - `isInputDeclaration(n)` — 检查第一个 named child 是 `type_identifier` 且文本为 `"input"`
  - `isExternDeclaration(n)` — 检查第一个 named child 是 `storage_class_specifier` 且文本为 `"extern"`
  - `isValidInputDeclaration(n)` — 检查 `init_declarator` 最后一个 named child 非空（区分 `input int X = 5;` 和 `input int X = ;`）
  - `checkReservedKeywordUsage(n)` — 拒绝 `input`/`extern` 作为 identifier（catches `int x = input ;`）
  - `collectGlobal` 也改用结构化检测

- **S4** file-lines 拆分：如果 `compile_interp.go` 超 450 行，按语义拆分为 `compile_interp_decls.go`（声明处理）+ `compile_interp_stmts.go`（语句处理）。拆分前先 `bash scripts/cap.sh` 查重。

### VM-CACHE-INTEGRITY-5

- **S5** `interp_runner.go` `CompilePythonCached`：cache hit 时恢复 `CoverageResult`——调 `CompilePythonWithCoverage(source)` 重编译获取 coverage，`InjectCoverageResult` 注入。coverage 恢复失败（`covErr != nil`）返回 error（不静默降级）。新增 `cov == nil` 检查也返回 error。

- **S6** `interp_runner.go` `CompilePythonCached`：新增 `Version == "python"` 语言校验——cache hit 时检查 `r.Bytecode().Version == "python"`，不匹配返回 error。`CompileMQLCached` 同步新增 `isMQLVersion` 校验（Version 是 "mql4" 或 "mql5"）。

- **S7** `bytecode_cache.go` `UnmarshalBytecode`：新增 `maxBytecodePayload`（64MiB）总 payload 上限——`len(data) > maxBytecodePayload` 时返回 error 含 "exceeds max" + "payload size"。

- **S8** 删除 `Bytecode.Language` 死字段（如存在）——`Version` 已作为语言判别器。用 `reflect.TypeOf(Bytecode{}).FieldByName("Language")` 验证字段不存在。

---

## 测试与对抗证明（缺一即未完成）

### VM-COMPILER-SEMANTICS-4 测试

- **T1** `TestCommaExpression_VMSideEffectsExecution`：MQL `int g_a,g_b,g_c; void OnTick(){ (g_a=1, g_b=2, g_c=3); }` → 编译+执行 → 断言 g_a=1, g_b=2, g_c=3（全部副作用执行）。
- **T2** `TestCommaExpression_VMFunctionCallSideEffects`：MQL 用 comma 调用多个函数 → 断言每个函数都被调用（g_counter 递增）。
- **T3** `TestCommaExpression_VMReturnValueIsLast`：MQL `int r; void OnTick(){ r = (1, 2, 42); }` → 断言 r=42（返回最后一个值）。
- **T4** `TestCompileMQL_InvalidInputMissingInitializer`：`input int X = ;` → 编译返回 error（不再静默接受）。
- **T5** `TestCompileMQL_ValidInputAccepted`：`input int X = 5;` → 编译成功。
- **T6** `TestCompileMQL_CompletelyInvalidSourceRejected`：`int x = ;` → 编译返回 error（HasError guard）。
- **T7** `TestCompileMQL_ReservedKeywordAsIdentifierRejected`：`int x = input ;` → 编译返回 error。
- **T8** `TestCompileToIR_RootNeverErrorForAnyInput`：穷举测试 `}}}(((///`、`!!!@@@###`、`""` 等 → tree-sitter 永不返回 root ERROR（正向证据，证明 HasError guard 不是死代码）。

### VM-CACHE-INTEGRITY-5 测试

- **T9** `TestCompilePythonCached_RestoresCoverageOnCacheHit`：cache hit 时 CoverageResult 非 nil。
- **T10** `TestCompilePythonCached_CoverageRestoreFailureReturnsError`：注入 non-nil coverage + error（sentinel）→ 返回 error（不静默降级）。
- **T11** `TestCompilePythonCached_CoverageRestoreNilCoverageReturnsError`：注入 nil coverage → 返回 error。
- **T12** `TestCompilePythonCached_RejectsMQLBytecodeForPythonSource`：MQL bytecode（Version="mql4"）+ Python source → 返回 error。
- **T13** `TestUnmarshalBytecode_PayloadLimitExceeded`：构造 64MiB+1 payload → 返回 error 含 "exceeds max" + "payload size"。
- **T14** `TestBytecode_NoLanguageField`：`reflect.TypeOf(Bytecode{}).FieldByName("Language")` 返回 false（字段不存在）。
- **T15** `TestCompilePythonCached_CacheHitVsColdCompileCoverageEqual`：cache hit vs cold compile 的 CoverageResult BlindSpot.Builtin/Severity 和 DefenseAViolation.Rule identity 一致。

### 对抗证明（每项 RED→restore→GREEN）

- **P1**：revert `comma_expression` 为只返回 last → T1 RED（g_a=0, g_b=0）→ 恢复 → GREEN。
- **P2**：删 `HasError` guard → T6 RED（`int x = ;` 不报错）→ 恢复 → GREEN。
- **P3**：revert 结构化检测为 `strings.Contains` → T4/T7 RED → 恢复 → GREEN。
- **P4**：删 coverage restore → T9 RED（CoverageResult nil）→ 恢复 → GREEN。
- **P5**：删 `Version == "python"` check → T12 RED（MQL bytecode accepted）→ 恢复 → GREEN。
- **P6**：删 payload guard → T13 RED（error 变为 "invalid magic" 不含 "exceeds max"）→ 恢复 → GREEN。

每项记录 mutation 命令、RED 输出摘要、restore 后 GREEN。**nil panic、另一条错误、"任意 error" 均不算证据。**

---

## 红队自审（施工后切换怀疑者视角，逐条书面回答）

1. `comma_expression` 改为 ExprSeq 后，VM 执行时是否真正按顺序执行所有子表达式？引用 VM 执行 ExprSeq 的代码行。
2. `HasError` guard 是否会误拒合法的 input/extern 声明（tree-sitter false positive）？给出测试证据。
3. 结构化 input/extern 检测是否覆盖所有合法形式（`input int X = 5;` / `extern int Y;` / 多变量 `input int A=1, B=2;`）？
4. coverage restore 重编译是否影响性能？cache hit 的意义是否被削弱？
5. `maxBytecodePayload` 64MiB 是否足够大（不误拒合法缓存）又足够小（防资源耗尽）？

---

## 验收门禁（逐条贴真实输出）

```
gofmt -l <改动文件>  # 输出为空
go build ./...
go vet ./tools/mql2go/...
go test ./tools/mql2go/... -count=1
go test -race ./tools/mql2go/... -count=1  # 连跑 3 次
go test ./internal/connect/strategy/... -count=1  # 确认无回归
go run ./tools/check-file-lines --strict  # 0 errors, info 需披露
git diff --check
```

---

## 回填与收尾

registry 本条回填真实实现 + REUSE/NEW 结论 + 对抗证明输出 + 红队自审 5 问答；`handover-audit-plan.md` 追加一行。**状态填 `🟦open（施工完成，待独立复审）`，不得自标 ✅done。**

> **勿部署、勿 push、停手等 Devin CLI 复审。禁止 `--no-verify`。收工只显式 `git add` 本任务涉及的文件，禁止 `git add -A`。**
