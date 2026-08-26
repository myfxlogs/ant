# Builder Handoff — VM 返工批重新施工（revert 830b2c79 状态漂移修复）

> 日期：2026-08-26 ｜ 审计方：Devin CLI（项目第一负责人） ｜ 施工方：Devin IDE
> 设计 SSOT：`docs/spec/vm-revert-redo-spec.md`
> 根因与对账证据：`docs/audits/tech-debt-registry.md` 条目 `D-REVERT-SCOPE-DRIFT-001`（:1953）
> 每个 ID 的完整修复 spec：registry 中对应条目（行号见各批）

## 0. 立项背景

**触发**：D-REVERT-CLEANUP-001 修复 build 断裂后，对账 VM 返工批 registry 状态与实际代码。

**证据链**：
- commit `830b2c79` commit message 声称"回滚 round 4-5 的 3 个 ID"，实际改了 91 个实现文件
- `git show 830b2c79 --stat -- '*.go'` 证明几乎所有 VM 返工工作被 revert
- 8 个 ID 的关键符号在 HEAD 不存在（`SourceHash`/`hashSource`/`CompilePythonCached`/`invalidateOrderCaches`/`OppositeTicket`/`ClassTypes`/`patchUserCalls`/`extremeIndex`）
- registry 标记"施工完成待复审"但代码已 revert 回施工前 → 状态漂移

**设计 SSOT 声明**：`docs/spec/vm-revert-redo-spec.md`（D1-D4 设计决策）

**约束与目标**：
- 基于 registry 中的原始修复 spec 重新施工，不做架构变更（D3）
- 分 4 批施工，按安全优先级（D2）
- 每批独立验收（D4）
- revert 不可逆，不尝试恢复被 revert 的代码（D1）

**边界/不做**：
- 不改写历史审计事实
- 不扩大到无关功能块
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
- 不 commit/push/deploy（施工方禁外部操作）
- 禁 `--no-verify`

---

## 第一批：VM-CACHE-INTEGRITY-1/2 — SourceHash 绑定（P1 安全）

> registry spec：`:110`（CACHE-INTEGRITY-1）、`:115`（CACHE-INTEGRITY-2）
> 当前代码状态：`SourceHash`/`hashSource`/`CompilePythonCached` 均不存在

### S1 — Bytecode struct 加 SourceHash 字段

**目标**：Bytecode 持久化时绑定源码 hash，cache hit 时校验。

**精确坐标**：
- 文件：`backend/tools/mql2go/bytecode.go:116`（`type Bytecode struct`）
- 字段位置：在 `Version string`（:153）后加 `SourceHash string`（SHA256 hex of source）
- 序列化：`backend/tools/mql2go/bytecode_cache.go:42`（`MarshalBytecode`）和 `:138`（`UnmarshalBytecode`）同步增加 SourceHash 的写入/读取

**落点**：
```go
// bytecode.go:153 后
Version    string
SourceHash string // SHA256 of source, for cache integrity (VM-CACHE-INTEGRITY-1)
```

### S2 — hashSource 函数

**目标**：计算源码 SHA256 hash。

**精确坐标**：
- 文件：`backend/tools/mql2go/interp_runner.go`（新建函数，放在 `CompileMQLCached` 前）
- 方法签名：`func hashSource(source string) string`

**落点**：
```go
func hashSource(source string) string {
	h := sha256.Sum256([]byte(source))
	return hex.EncodeToString(h[:])
}
```

### S3 — CompileMQL 正常路径填充 SourceHash

**目标**：编译成功后设置 SourceHash。

**精确坐标**：
- 文件：`backend/tools/mql2go/interp_runner.go:127`（`func CompileMQL`）
- 落点：`bc, err := CompileAST(ir)` 后、`return NewVMRunner(bc)` 前加 `bc.SourceHash = hashSource(source)`

### S4 — CompileMQLCached cache hit 校验 SourceHash

**目标**：cache hit 时校验 SourceHash，mismatch 强制重编。

**精确坐标**：
- 文件：`backend/tools/mql2go/interp_runner.go:56`（`func CompileMQLCached`）
- 当前代码：`:57-60` 只校验 `CompileMQLFromBytecode` 成功就返回
- 落点：cache hit 后加 `if r.Bytecode().SourceHash != hashSource(source) { /* fall through to recompile */ }`

**对抗证明**：
- 突变：删 SourceHash 校验（始终接受缓存）→ 用 source A 编译缓存，用 source B 调 `CompileMQLCached` → 缓存被接受 → RED
- 恢复：加回校验 → source B 强制重编 → GREEN

### S5 — CompileMQLCached 的 MarshalBytecode error 不再吞

**目标**：marshal 失败返回 error 而非 `return r, nil, nil`。

**精确坐标**：
- 文件：`backend/tools/mql2go/interp_runner.go:71`（当前 `return r, nil, nil`）
- 落点：改为 `return nil, nil, fmt.Errorf("marshal bytecode: %w", mErr)`

**对抗证明**：
- 突变：恢复 `return r, nil, nil` → 注入 marshal 失败 → 测试期望 error 但得 nil → RED
- 恢复 → GREEN

### S6 — CompilePythonCached 函数

**目标**：Python 缓存路径镜像 MQL 的 SourceHash 校验。

**精确坐标**：
- 文件：`backend/tools/mql2go/interp_runner.go`（新建函数，放在 `CompileMQLCached` 后）
- 方法签名：`func CompilePythonCached(source string, cachedBytecode []byte) (runner *VMRunner, bytecode []byte, err error)`
- 逻辑：镜像 `CompileMQLCached`，先校验 `SourceHash == hashSource(source)`，不匹配则 `CompilePython` 重编

### S7 — backtest_worker_python.go 改用 CompilePythonCached

**目标**：Python 回测路径使用带 SourceHash 校验的缓存函数。

**精确坐标**：
- 文件：`backend/internal/connect/strategy/backtest_worker_python.go:30`
- 当前代码：`if r, err := mql2go.CompileMQLFromBytecode(cachedBytecode); err == nil {`
- 落点：改为 `if r, bcData, err := mql2go.CompilePythonCached(params.code, cachedBytecode); err == nil {`
- cache hit 时通过 `CompilePythonWithCoverage` + `InjectCoverage` + `InjectDefenseAViolations` 恢复 coverage（镜像 `backtest_worker_vm.go:43-49` 的模式）
- 方法名已确认：`InjectCoverage`（`interp_runner.go:333`）、`InjectDefenseAViolations`（`interp_runner.go:344`）——注意不是 `InjectCoverageResult`

### S8 — MarshalBytecode/UnmarshalBytecode 序列化 SourceHash

**目标**：SourceHash 字段参与序列化/反序列化。

**精确坐标**：
- `MarshalBytecode`（`bytecode_cache.go:42`）：Version 写入在 `:125`（`w.writeString(bc.Version)`）→ 在其后加 `w.writeString(bc.SourceHash)`
- `UnmarshalBytecode`（`bytecode_cache.go:138`）：Version 读取在 `:196`（`if bc.Version, err = r.readString(); err != nil {`）→ 在其后加 SourceHash 读取
- **顺序一致性**：Marshal 先 Version 后 SourceHash，Unmarshal 必须相同顺序

### S9 — unmarshal map 函数加 duplicate key 检测

**目标**：5 个 unmarshal map 函数检测 duplicate key，返回 error 而非静默覆盖。

**精确坐标**（两种返回签名，分别处理）：
- 返回 `(map[K]V, error)` 的函数（插入 `m[key] = v` 处加 duplicate 检测）：
  - `unmarshalGlobalSlots`（`bytecode_cache.go:266`）
  - `unmarshalFuncs`（`bytecode_cache.go:314`）
  - `unmarshalBuiltins`（`bytecode_cache.go:358`）
  - `unmarshalEnums`（`bytecode_cache.go:439`）
- 返回 `error` 的函数（直接写入 `bc.EventLocals[pc] = int(count)` 处加 duplicate 检测）：
  - `unmarshalEventLocals`（`bytecode_cache.go:407`）——签名 `func(r *bytecodeReader, bc *Bytecode) error`，写入 `bc.EventLocals[pc]`
- 每个函数的 map 插入处加 `if _, exists := m[key]; exists { return nil, fmt.Errorf("duplicate key: %s", key) }`（EventLocals 用 `return fmt.Errorf(...)`）

**对抗证明**：
- 构造 little-endian 重复 key 的 enums section → `unmarshalEnums` 返回 error → GREEN
- 删 duplicate key 检测 → 返回 nil error（静默覆盖）→ RED

### S10 — UnmarshalBytecode trailing bytes 拒绝 + 有界解析

**目标**：防止损坏缓存造成非确定输出。

**精确坐标**：
- `UnmarshalBytecode`（`bytecode_cache.go:138`）：末尾加 `if r.pos != len(data) { return nil, fmt.Errorf("trailing bytes: %d", len(data)-r.pos) }`
- `readCount`（如不存在则在 `bytecode_cache.go` 新建）：检查 `count*minBytes` 不超过剩余数据

**对抗证明**：
- 构造 trailing bytes 的缓存 → `UnmarshalBytecode` 返回 error → GREEN
- 删 trailing bytes 检查 → 返回 nil error → RED

### 第一批验收标准

1. **对抗证明 4 项**（S4/S5/S9/S10），每项 RED→restore→GREEN
2. **门禁全绿**：
   - `go build ./...`
   - `go test ./tools/mql2go/... -count=1`
   - `go test -race ./tools/mql2go/... -count=1` ×3
   - `go test ./internal/connect/strategy -count=1`
   - `go vet ./...`
   - `go run ./tools/check-file-lines --strict`（0 errors）
   - `git diff --check`
3. **file-lines**：`check-file-lines --strict` 只检查 `backend/internal`/`backend/cmd`/`frontend/src`/`proto`/`scripts`（不检查 `tools/`）。但 AGENTS.md §4 的 450 行规范仍适用——`bytecode_cache.go` 当前 562 行（pre-existing 超限），加 SourceHash 序列化（~10 行）后如进一步膨胀，施工方应考虑拆分 unmarshal 函数到独立文件。`backtest_worker_python.go` 在 `internal/` 范围内，如新增 coverage 恢复代码导致超限需拆分。
4. **复用核对**：`bash scripts/cap.sh hashSource` / `cap.sh SourceHash` / `cap.sh CompilePythonCached`

### 第一批红队自审（施工方完工前必答）

1. SourceHash 校验在 cache corrupted（`CompileMQLFromBytecode` 失败）路径是否正确 fall through？
2. `CompilePythonCached` 的 coverage 恢复路径是否镜像 `backtest_worker_vm.go:43`？
3. 5 个 unmarshal map 函数的 duplicate key 检测是否覆盖所有 map 插入点？
4. trailing bytes 检查在所有 early return 路径之后？
5. SourceHash 序列化顺序：Marshal 先 Version（:125）后 SourceHash，Unmarshal 必须相同顺序（Version 在 :196 读取后紧接 SourceHash）。顺序不一致会导致反序列化错位。

### 第一批回填纪律

1. registry `VM-CACHE-INTEGRITY-1`（:110）和 `VM-CACHE-INTEGRITY-2`（:115）：状态改为 `🟦open（施工完成，待独立复审）` + 真实实现 + 对抗证明结果
2. `handover-audit-plan.md` 变更日志加一行
3. **不自行宣告完成**——停手等 Devin CLI 复审

---

## 第二批：VM-TRADE-CONTEXT-1/2 — 交易上下文失真（P1）

> registry spec：`:109`（TRADE-CONTEXT-1）、`:116`（TRADE-CONTEXT-2）
> 当前代码状态：`invalidateOrderCaches`/`OppositeTicket` 均不存在
> **第二批在第一批验收通过后派工**

### S1 — invalidateOrderCaches 函数

**精确坐标**：
- 文件：`backend/tools/mql2go/vm_helpers.go`（新建函数）
- 方法签名：`func (vm *VM) invalidateOrderCaches()`
- 清空：`cachedPositions`/`cachedOrders`/`cachedHistory`/`positionsLoaded`/`ordersLoaded`/`historyLoaded`/`currentPos`/`currentOrder`

### S2 — mutation builtin 成功后调用 invalidateOrderCaches

**精确坐标**：
- `backend/tools/mql2go/vm_builtin_trade.go` — 所有 mutation builtin（OrderSend/OrderClose/OrderCloseBy/OrderModify/OrderDelete）成功后
- `backend/tools/mql2go/vm_builtin_trade_signals.go` — CTrade.Buy/Sell/BuyLimit/SellLimit/BuyStop/SellStop/PositionClose/ClosePartial/CloseBy/Modify/OrderDelete/CloseAll 成功后

### S3 — runEvent 顶部清空所有 cache

**精确坐标**：
- `backend/tools/mql2go/vm.go` — `runEvent` 函数顶部加 `vm.invalidateOrderCaches()`

### S4 — sdk.Signal 加 OppositeTicket

**精确坐标**：
- `backend/strategy/sdk/strategy.go` — `type Signal struct` 加 `OppositeTicket int64`

### S5 — builtinOrderCloseBy 设置 OppositeTicket

**精确坐标**：
- `backend/tools/mql2go/vm_builtin_trade_signals.go` — `builtinOrderCloseBy`/`builtinCTradePositionCloseBy` signal mode 设置 `OppositeTicket`

### 第二批验收标准

（同第一批，对抗证明要求：删 `invalidateOrderCaches` 调用 → OrderSelect 返回 stale 数据 → RED）

---

## 第三批：VM-COMPILER-SEMANTICS-1 + BT-FUNC-ENTRYPC-FWD（P1）

> registry spec：`:146`（COMPILER-SEMANTICS-1）、`:147`（BT-FUNC-ENTRYPC-FWD）
> **第三批在第二批验收通过后派工**

### S1-S7 详见 registry :146/:147

（精确坐标在第二批验收后补充，避免在代码变化后坐标失效）

---

## 第四批：VM-TIMESERIES-SEMANTICS-1 + VM-RUNTIME-FAILCLOSED-1（P1）

> registry spec：`:108`（TIMESERIES-SEMANTICS-1）、`:111`（RUNTIME-FAILCLOSED-1）
> **第四批在第三批验收通过后派工**

### S1-S7 详见 registry :108/:111

（精确坐标在第三批验收后补充）

---

## 通用范围约束

One task = one scope：只动 VM 返工批 8 个 ID 的修复——`backend/tools/mql2go/`（bytecode.go/bytecode_cache.go/interp_runner.go/vm_helpers.go/vm_builtin_trade.go/vm_builtin_trade_signals.go/vm.go/vm_builtin_mql5_ts.go/compile.go/compile_loops.go/compile_interp.go/compile_interp_expr.go）+ `backend/internal/connect/strategy/backtest_worker_python.go` + `backend/strategy/sdk/`。不顺手重构、不改无关逻辑、不动 broker/handler 业务语义。

## 固定尾部

**勿部署，停手等 Devin CLI 复审。** 禁 `--no-verify`。禁 commit/push/deploy。只 add 本任务文件，禁 `git add -A`。
