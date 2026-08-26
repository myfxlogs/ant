# Builder Handoff — VM 返工批重新施工（revert 830b2c79 状态漂移修复）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 根因与对账证据见 `docs/audits/tech-debt-registry.md` 条目 `D-REVERT-SCOPE-DRIFT-001`（必读）。
> 每个 ID 的完整修复 spec 见 registry 中对应条目（行号见下）。

## 0. 背景一句话

commit `830b2c79` 的 revert 实际范围远超 commit message，把 `acaa86db` 引入的几乎所有 VM 返工工作（round 1-5）都 revert 了。8 个 ID 的修复代码已不存在，registry 状态从"施工完成待复审"降级回"🟦open（待施工）"。需要基于 registry 中的原始修复 spec 重新施工。

## 1. 当前代码状态（revert 后，HEAD = `889ff2ec`）

### 不存在的关键符号（需重新实现）

| 符号 | 所属 ID | 当前状态 | registry spec 位置 |
|---|---|---|---|
| `Bytecode.SourceHash` | CACHE-INTEGRITY-1/2 | 字段不存在 | registry :110/:115 |
| `hashSource()` | CACHE-INTEGRITY-1/2 | 函数不存在 | registry :110 |
| `CompilePythonCached()` | CACHE-INTEGRITY-2 | 函数不存在 | registry :115 |
| `Bytecode.Version` | CACHE-INTEGRITY-5 | 字段不存在 | registry :159 |
| `validateBytecode()` | CACHE-INTEGRITY-1 | 函数不存在 | registry :110 |
| `bytecode_validate.go` | CACHE-INTEGRITY-1 | 文件不存在（已删） | registry :110 |
| `invalidateOrderCaches()` | TRADE-CONTEXT-1 | 函数不存在 | registry :109 |
| `OppositeTicket` (sdk.Signal) | TRADE-CONTEXT-2 | 字段不存在 | registry :116 |
| `ClassTypes` (Bytecode/IR) | COMPILER-SEMANTICS-1 | 字段不存在 | registry :146 |
| `patchUserCalls()` | BT-FUNC-ENTRYPC-FWD | 函数不存在 | registry :147 |
| `extremeIndex`/`validSeriesMode` | TIMESERIES-SEMANTICS-1 | 函数不存在 | registry :108 |
| `callBuiltin` fatalError 检查 | RUNTIME-FAILCLOSED-1 | 增强修复不存在 | registry :111 |

### 存在但需增强的符号

| 符号 | 所属 ID | 当前状态 | 需要的增强 |
|---|---|---|---|
| `fatalError` 字段 | RUNTIME-FAILCLOSED-1 | 基本机制✓（unimplemented builtin 设置） | handler 返回 error 后也检查 fatalError |
| `CompileMQLCached()` | CACHE-INTEGRITY-1 | 存在但不校验 SourceHash | 加 SourceHash 校验 |
| `MarshalBytecode` error | CACHE-INTEGRITY-1/2 | error 被吞（`return r, nil, nil`） | 改为返回 error |
| `backtest_worker_python.go` | CACHE-INTEGRITY-2 | 直接用 `CompileMQLFromBytecode` | 改用 `CompilePythonCached` |

## 2. 施工批次与优先级

### 第一批（P1 安全）：VM-CACHE-INTEGRITY-1/2 — SourceHash 绑定

**registry spec**：`:110`（CACHE-INTEGRITY-1）、`:115`（CACHE-INTEGRITY-2）

**施工范围**：
1. `Bytecode` struct 加 `SourceHash string` 字段（`bytecode.go`）
2. 新建 `hashSource(source string) string` — SHA256 of source（`interp_runner.go` 或 `bytecode_cache.go`）
3. `CompileMQL` 正常路径填充 `SourceHash`（`interp_runner.go`）
4. `CompileMQLCached` cache hit 时校验 `r.Bytecode().SourceHash == hashSource(source)`，mismatch 强制重编（`interp_runner.go:56`）
5. `CompileMQLCached` 的 `MarshalBytecode` error 改为返回 error（`interp_runner.go:50`，从 `return r, nil, nil` 改为 `return nil, nil, fmt.Errorf(...)`）
6. `MarshalBytecode`/`UnmarshalBytecode` 序列化/反序列化 `SourceHash` 字段（`bytecode_cache.go`）
7. 新建 `CompilePythonCached(source, cachedBytecode)` 镜像 `CompileMQLCached`（`interp_runner.go`）
8. `backtest_worker_python.go` 改用 `CompilePythonCached`（`internal/connect/strategy/backtest_worker_python.go:30`）
9. `UnmarshalBytecode` map 序列化确定性（sorted keys）+ trailing bytes 拒绝 + 有界解析（`bytecode_cache.go`）
10. 5 个 unmarshal map 函数加 duplicate key 检测（`bytecode_cache.go`）

**对抗证明要求**：
- 删 `CompileMQLCached` 的 SourceHash 校验 → 缓存 mismatch 测试 RED
- 删 `CompilePythonCached` 的 SourceHash 校验 → Python 缓存 mismatch 测试 RED
- 恢复 `MarshalBytecode` error 吞掉 → marshal error 测试 RED
- 删 duplicate key 检测 → duplicate key 测试 RED

### 第二批（P1 交易安全）：VM-TRADE-CONTEXT-1/2 — 交易上下文失真

**registry spec**：`:109`（TRADE-CONTEXT-1）、`:116`（TRADE-CONTEXT-2）

**施工范围**：
1. 新建 `invalidateOrderCaches()` — 清空 `cachedPositions`/`cachedOrders`/`cachedHistory`/`positionsLoaded`/`ordersLoaded`/`historyLoaded`/`currentPos`/`currentOrder`（`vm_helpers.go`）
2. 所有 mutation builtin（OrderSend/OrderClose/OrderCloseBy/OrderModify/OrderDelete/CTrade.Buy/Sell/BuyLimit/SellLimit/BuyStop/SellStop/CTrade.PositionClose/ClosePartial/CloseBy/Modify/OrderDelete/CloseAll）成功后统一调用 `invalidateOrderCaches()`
3. `builtinOrderSelect` 顶部 reset `currentPos`/`currentOrder`（`vm_builtin_trade.go`）
4. `runEvent` 顶部清空所有 cache（`vm.go`）
5. `CTrade.SetExpertMagicNumber`/`SetDeviationInPoints` 透传到 `vm.tradeMagic`/`vm.tradeDeviation`（`vm_builtin_trade.go`）
6. `sdk.Signal` 加 `OppositeTicket int64` 字段（`strategy/sdk/strategy.go`）
7. `builtinOrderCloseBy`/`builtinCTradePositionCloseBy` signal mode 设置 `OppositeTicket`（`vm_builtin_trade_signals.go`）
8. `sdk.AccountInfo` 加 `Login int64` + `Company string`（`strategy/sdk/broker.go`）
9. `builtinAccountNumber` 从 `vm.ctx.Account().Login` 读取（`vm_builtin_string.go`）
10. `brokerImpl` 查询 error fail-closed（`strategy/runner/broker.go` + `runner.go`）

### 第三批（P1 编译器正确性）：VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD

**registry spec**：`:146`（COMPILER-SEMANTICS-1）、`:147`（BT-FUNC-ENTRYPC-FWD）

**施工范围**：
1. `compileAssignment` 在 `findIdent` 前检查 field/subscript lhs（`compile_interp_expr.go`）
2. `compileDeclaration` 遍历所有 declarator（`compile_interp.go`）
3. 不支持的运算符显式报错而非 fallback（`compile_interp_expr.go`）
4. `methodBuiltinName` 按 CTrade 命名空间解析（`compile_interp.go`）
5. `IR.ClassTypes` + `Bytecode.ClassTypes` 从 `knownClasses` + `isBuiltinClass` 收集（`ir.go` + `bytecode.go`）
6. `initGlobals` 对 `ClassTypes[decl.Type]` 的全局初始化为 `ValClass`（`compile.go`）
7. `MarshalBytecode`/`UnmarshalBytecode` 同步增加 `ClassTypes`（`bytecode_cache.go`）
8. `compileCall` 发出 `OP_CALL_USER` 时 operand A 写 -1 占位符，记录 `userCallPatch`（`compile_expr.go`）
9. 所有用户函数 body 编译完成后 `patchUserCalls()` 统一 patch（`compile.go`）

### 第四批（P1 语义正确性）：VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1

**registry spec**：`:108`（TIMESERIES-SEMANTICS-1）、`:111`（RUNTIME-FAILCLOSED-1）

**施工范围**：
1. `CopyTime` 把 bar 的 `unix_ms` 转为 MQL `datetime`（unix seconds）（`vm_builtin_mql5_ts.go`）
2. `iHighest`/`iLowest` 按 `ENUM_SERIESMODE`（0-5）选择字段（`vm_builtin_mql5_ts.go`）
3. `validSeriesMode` 校验 mode 0-5，非法返回 -1（`vm_builtin_mql5_ts.go`）
4. 越界 guard：`start<0 || start>=Len()` 返回 -1（`vm_builtin_mql5_ts.go`）
5. `iBarShift` exact 参数支持（`vm_builtin_mql5_ts.go`）
6. `Copy*` 方向语义：count>0 chronological，count<0 reverse（`vm_builtin_mql5_ts.go`）
7. `callBuiltin` 在 handler 返回 nil error 后检查 `fatalError`（`vm_helpers.go:212`）

## 3. 通用施工边界

- **身份与边界**：你是施工方，不是验收方。只处理上述 ID 的修复，不改写历史审计事实，不扩大到无关功能块。
- **复用核对**：动工新 file/function 前 `bash scripts/cap.sh <词>` 查能力，PR 标 `REUSE:`/`NEW:`。
- **file-lines**：`cd backend && go run ./tools/check-file-lines --strict` → 0 errors。超限先拆分。
- **对抗证明**：每个关键修复必须有 mutation RED→restore→GREEN 证据。nil panic、另一条错误、callback-only 或"任意 error"均不算证据。
- **门禁**：`go build ./...` / `go test ./tools/mql2go/... -count=1` / `go test -race ./tools/mql2go/... -count=1` / `go test ./internal/connect/strategy -count=1` / `go test -race ./internal/connect/strategy -count=1` / `go vet ./...` / `check-file-lines --strict`（0 errors）/ `buf lint` / `git diff --check`。
- **禁提交**：完成后 registry 条目保持 `🟦open（施工完成，待独立复审）`，在 handover 追加真实证据并停工。禁 commit/push/deploy，禁 `--no-verify`，禁 `git add -A`（只 add 本任务文件）。
- **收工**：回填 registry 当前 ID 的真实实现 + REUSE/NEW 结论 + 测试和 proof 结果，handover 追加一行；状态填 `🟦open（施工完成，待独立复审）`，不得自标 ✅done。

## 4. 开工前必读

1. `AGENTS.md`（契约 SSOT）
2. `docs/handoff/STATE.md`（当前状态）
3. `docs/audits/tech-debt-registry.md` 中对应 ID 条目（完整修复 spec）
4. `docs/audits/handover-audit-plan.md`（历史复审记录，了解之前为何未通过）
5. 执行 `git status` / `git diff` / 相关 `git log` / `git blame`
6. 新 file/function 前执行 `bash scripts/cap.sh` 多关键词复用核对
