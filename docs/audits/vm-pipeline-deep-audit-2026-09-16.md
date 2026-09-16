# VM 管线深度审计报告（整条管线）

- **审计方**：Devin CLI（项目第一负责人 / 独立审计验收方，D-006）
- **日期**：2026-09-16
- **范围**：MQL/Python 源码 → CST → IR(=AST) → Bytecode → VM → SDK → Runner → 回测/实盘派发 整条管线
- **模式**：只读审计，未修改任何生产代码
- **依据**：ADR-0023（AST 分析 + Bytecode VM 执行 + MQL 源码为唯一真相源）、AGENTS.md §0/§7、STATE.md 活跃条目
- **方法**：4 个并行只读 subagent 分子系统深探（Builtin / Mutation Coordinator / 指标缓存 / Python 子集编译器）+ 主审计方通读核心文件交叉验证

---

## 0. 执行摘要

VM 管线是 ADR-0023 落地的产物，替代了旧的 `IR 提取 + WASM 沙箱 + Go 代码生成` 三件套。实际架构为四段式：**MQL/Python 源码 → tree-sitter CST → IR(=AST) → Bytecode → VM**，其中 IR 与 AST 在代码里是同一个结构体（`interp.IR`），ADR 文档中的 "AST/IR" 混用指同一物。

管线整体健康，多轮深度审计（VM-CACHE-INTEGRITY / VM-COMPILER-SEMANTICS / VM-TRADE-CONTEXT / VM-RUNTIME-FAILCLOSED / LIVE-INDICATOR-1 / LIVE-ORDER-REENTRY-1 / VM-AUDIT-2026-08-27-1..8）已覆盖关键硬伤并 ✅done。本次审计未发现 P0 级新缺陷，但识别出若干 P1/P2 级潜在风险与盲区，详见 §11。

**关键结论**：
1. 回测/实盘共用同一 VMRunner（`signalMode` 切换），同码不变量已落地。
2. 实盘经同步 mutation barrier 派发，恢复 MT4 EA `OrderSend` 单线程语义（LIVE-ORDER-REENTRY-1）。
3. 指标缓存用 `RevisionedBarSource` + `SeriesCache` 双层策略解决 500-bar 滚动窗口冻结问题（LIVE-INDICATOR-1）。
4. Python 子集编译器与 MQL 共用 `CompileAST` + `VM`，通过 `Version` 字段分支区分。
5. fail-closed 贯穿全管线：无 authority / 无 provenance / outcomeUnknown / critical builtin 缺失 / 栈下溢 / 指令超限 均选择"不继续"而非"乐观继续"。

---

## 1. 设计契约与实际架构对齐

### 1.1 ADR-0023 决策点 vs 实际落地

| ADR 决策 | 实际落地 | 对齐 |
|----------|----------|------|
| D1: MQL 源码唯一真相源 | `imported_strategies.source_code` 存 MQL；`strategy_templates.code` 存 MQL；`SourceHash` 绑定缓存 | ✅ |
| D2: AST 分析 + Bytecode VM 执行 | `CompileToIR`(CST→IR) + `CompileAST`(IR→Bytecode) + `VM`；IR 即 AST | ✅ |
| D3: 去掉 WASM 沙箱 | 无 `wasm_executor.go`；进程内执行 | ✅ |
| D4: 去掉运行时 Go 代码生成 | `gen.go` 仅 CLI；运行时无 Go 生成 | ✅ |
| D5: Agent 架构（未来） | `agent-engine` skill 覆盖；MQL 层 Agent 工具 | ✅（独立子系统） |
| D6: 盲区三层处理 | 静态 AST 覆盖度 + 运行时分级（Fatal/Warning/Info）+ 运行后上报 | ✅ |

### 1.2 命名澄清

ADR 文档把中间层叫 "AST/IR"，**实际代码里 AST 和 IR 是同一个东西**——`interp.IR` 结构体（`tools/mql2go/interp/ir.go`）就是语义 AST。`CompileToIR` 实际是 CST→IR，`CompileAST` 实际是 IR→Bytecode。命名上 "AST" 和 "IR" 混用，但指同一个结构体。本报告统一称 IR。

### 1.3 实际管线全景

```
MQL/Python 源码
  │
  ├─ PreprocessMQL (#define/#property/#include 预处理)        compile_interp.go:29
  ├─ ParseMQL / ParsePython (tree-sitter cgo → CST)           mql_lang.go / py_lang.go
  ├─ detectMQLVersion (mql4/mql5) / 固定 "python"
  ├─ compiler.compile / pyCompiler.compile (CST → IR)         compile_interp.go:52 / compile_py.go:73
  │     ├─ 第一遍：收集 class/struct/enum 类型
  │     ├─ 第二遍：declaration/function_definition/event → IR
  │     └─ Defense A 校验（checkReservedKeywordUsage 等）
  │
  ├─ CompileAST(IR → Bytecode)  一次性 ~300ms                  compile.go:13
  │     ├─ registerBuiltins（注册 ~250 个 builtin ID）
  │     ├─ 分配 global slots（params + globals）
  │     ├─ 两遍编译用户函数（先登记 EntryPC，再编 body，修前向引用）
  │     ├─ 编译事件入口（OnInit/OnBar/OnTick/OnTrade/OnTimer/OnDeinit/OnTradeTransaction/OnBookEvent）
  │     ├─ 全局初始化器拼到 OnInit 前面
  │     ├─ patchJumps + patchUserCalls（前向跳转/调用回填）
  │     └─ SourceHash 写入 bytecode（缓存完整性）
  │
  └─ VMRunner = NewVM(Bytecode)                                 interp_runner.go:34
        │  实现 sdk.Strategy 接口
        │  SetSignalMode(true) → 实盘；false → 回测
        │
        ├─ 回测路径：backtest.Engine.Run → 每 bar 调 OnBar
        └─ 实盘路径：VMLiveSession → 每事件调 SendEvent → dispatch
```

---

## 2. Bytecode 设计

**Stack-based 指令集**（ADR §5.4 选 stack 而非 register，理由是实现更简单、I/O 密集场景性能差异可忽略）：

| 类别 | 指令 |
|------|------|
| 栈 | `PUSH_CONST/PUSH_VAR/PUSH_GLOBAL/STORE_VAR/STORE_GLOBAL/POP/DUP/SWAP` |
| 算术 | `ADD/SUB/MUL/DIV/MOD/FLOOR_DIV/NEG` |
| 比较 | `EQ/NE/LT/LE/GT/GE` |
| 逻辑 | `AND/OR/NOT` |
| 控制流 | `JMP/JMP_IF_FALSE/JMP_IF_TRUE` |
| 调用 | `CALL_BUILTIN/CALL_USER/ENTER_FUNC/LEAVE_FUNC` |
| 事件入口 | `ENTER_ONINIT/ONBAR/ONTICK/ONTRADE/ONTIMER/ONDEINIT/ONTRADETRANSACTION/ONBOOKEVENT`（标记位） |
| 返回 | `RETURN/HALT` |
| 序列 | `PUSH_SERIES`（Close[i]/Open[i] 等） |
| 数组 | `PUSH_ARRAY/STORE_ARRAY` |
| 字段 | `GET_FIELD/SET_FIELD`（obj.field / CTrade.method） |

`Instruction{Op, A, B, Line}`——A/B 是通用操作数（const ID/var ID/func ID/跳转目标/参数个数），Line 保留源码行号供诊断。

`Bytecode` 结构还携带：Consts 池、GlobalSlots 映射、Funcs 表、Builtins 映射、8 个事件入口 PC、EventLocals（每事件局部槽位数）、Params、Version、SourceHash、Enums、ClassTypes、CoverageReport。

---

## 3. VM 执行核心

### 3.1 VM 状态（`vm.go:23-64`）

```
stack[]interp.Value       显式数据栈（非 Go 调用栈 → 无栈溢出）
globals[]interp.Value     全局变量槽
locals[]interp.Value      扁平局部变量空间（帧连续存放）
pc int32                  程序计数器
ticks int64               指令计数器（每事件重置）
callDepth int             用户函数调用深度
fatalError string         致命错误（critical builtin 缺失/栈下溢/builtin error）
signalMode bool           实盘=true（Order* 造 Signal）/ 回测=false（直调 broker）
funcByEntryPC map         EntryPC→FuncEntry 查找（避免 O(n) 扫描）
runtimeBlindSpots map     运行时盲区命中计数
lastIndicators map        L2 诊断（指标返回点捕获）
diagKeyCache map          诊断 key 缓存（避免热路径 fmt.Sprintf 分配）
cachedPositions/Orders/History  Order* builtin 懒加载缓存
currentPos/currentOrder        当前选中订单
tradeMagic/tradeDeviation      CTrade setter 状态
```

### 3.2 主循环 `runLoop`（`vm_execute.go:11`）

扁平的 `for pc < len(Code)` 循环，每轮：
1. 顶部检查 `fatalError`（ADR §5.4 critical builtin 缺失立即停）。
2. 每 10000 ticks 检查一次 `ctx.Done()`（超时/取消）。
3. `ticks++`，超 `MaxTicks=10_000_000` 报指令上限（每事件预算，几乎必是死循环）。
4. 栈深 > `MaxStackDepth=4096` 报错。
5. 取指、`pc++`、`execute(ins)` 分发。

`execute` 是一个大 switch，按 opcode 分发到 `executeStack/executeArith/executeCompare/executeLogical`，跳转指令直接改 `pc`，`CALL_USER` 走 `executeCallUser`。

### 3.3 用户函数调用 `executeCallUser`（`vm_execute.go:317`）

- `callDepth >= MaxCallDepth=256` 报错（防递归爆 Go 栈）。
- 保存 `oldLocals`，新建 `newLocals[NumLocals]`，`copy(args)` 进参数槽。
- **内嵌子循环**（不回外层 runLoop）——所以子循环里也独立检查 ctx/ticks/stackDepth，并在错误路径恢复 `locals/callDepth`（VM-AUDIT-2026-08-27-3 的防御纵深）。
- 遇 `RETURN/HALT` 跳出，恢复 `locals/pc/callDepth`。

### 3.4 事件入口 `runEvent`（`vm.go:187`）

每事件重置：`stack[:0]`、`invalidateOrderCaches`、`callDepth=0`、`signal=nil`、`fatalError=""`（VM-AUDIT-2026-08-27-2：上一事件的 fatal 不阻塞后续事件）、`lastIndicators` 清空、按 `EventLocals[entryPC]` 分配局部空间、`ticks=0`。然后 `runLoop`。

### 3.5 安全约束落地

| 约束 | 值 | 位置 |
|------|-----|------|
| 源码大小上限 | 500 KB | `interp_runner.go:44` MaxSourceSize |
| 指令计数器 | 10M/事件 | `vm.go:14` MaxTicks |
| 调用深度 | 256 | `vm.go:17` MaxCallDepth |
| 栈深度 | 4096 | `vm.go:20` MaxStackDepth |
| panic 恢复 | safeRun defer recover | `interp_runner.go:276` |
| ctx 超时 | 调用方设（回测 5min，实盘按事件） | ADR §5.4 |

---

## 4. Builtin 实现子系统

### 4.1 注册机制

- `builtinRegistry` 是**有序 slice** `[]builtinEntry`，每个元素含 `name` 与 `fn` 两个字段（`builtins.go:19`）。
- 名字注册集中在 `builtins.go`；handler 挂载分散在 `vm_builtin_impls.go` 与 `vm_builtin_wiring.go` 的多个 `init()` 中（`vm_builtin_wiring.go:7-20`）。
- 分离目的：① 编译器解析阶段即可拿到完整名字→BuiltinID 映射，不依赖实现文件是否已加载；② 实现代码可按 domain 拆分到多个文件，遵守 450 行文件上限。
- `id(name)` 函数线性扫描 registry 返回索引；未找到时 `panic`（编译期配置错误，`vm_builtin_impls.go:245`）。
- `registerMethodBuiltin` 直接按方法名查；`registerMethodBuiltinWithObj` 遍历 IR globals 找对象类型拼 `Type.method`（如 `CTrade.Buy`）再查（`builtins.go:489-502`）。CTrade 方法注册为 `CTrade.Buy` 全名，正确解析路径是后者。

### 4.2 分发与分级（`vm_helpers.go:220 callBuiltin`）

- handler 返回 error → 设 `fatalError`（fail-closed，VM-RUNTIME-FAILCLOSED-1）。
- handler 设过 `fatalError`（如 iADX:MODE_PLUSDI）→ 不 push 结果，顶部检查捕获。
- 无 handler → `isFatalUnimplemented` 查 API registry 分级：
  - `StatusImplemented` → fatal（开发漏挂 handler）
  - `StatusUnsupported` → fatal（已知不支持，不静默放行）
  - 不在 registry 但命中 `Order/Position/MarketInfo/iClose` 等前缀 → fatal
  - 其它未知名 → 非致命盲区返回 NoneVal
- `OP_CALL_BUILTIN` 在 `popN` 后立即检查 `fatalError`（VM-AUDIT-2026-08-27-4：栈下溢后 early return，防 OrderSend 带空参数产生副作用）。

### 4.3 三级 Severity 落地

实际代码只有两级 `SymbolStatus`：`StatusImplemented` / `StatusUnsupported`（`api_registry.go:21`）。运行时盲区的"三级"由 `SeverityForBuiltin` / `classifySeverity` 在 `interp/analyze.go:196` 提供：**Fatal / Warning / Info**（无 SeveritySkip）。`GetRuntimeBlindSpots` 消费 `runtimeBlindSpots[name]++` 命中计数并附加 Severity（`vm.go:173`）。

### 4.4 Trade 域（`vm_builtin_trade.go`, `vm_builtin_trade_signals.go`, `vm_builtin_mql5_trade.go`）

**signalMode 差异**：
- `signalMode=true`：所有交易 builtin 只构造 `sdk.Signal` 写入 `vm.signal`，**不调用 broker**。`OrderSend` 返回固定票号 `1`；`OrderClose/Modify/Delete/CloseBy` 返回 `true`（`vm_builtin_trade.go:50-69`）。
- `signalMode=false`：直接调用 `vm.ctx.Broker().OrderSend/PositionClose/...`（`vm_builtin_trade.go:71-77`）。

**OrderSelect/OrdersTotal 与缓存交互**：
- `cachedPositions/cachedOrders/cachedHistory` 懒加载；`currentPos/currentOrder` 当前选中。
- `OrderSelect`：`pool==0` 按 index/ticket 在 `cachedPositions + cachedOrders` 中查找；`pool==1` 在 `cachedHistory` 中查找（`vm_builtin_trade.go:151-232`）。
- `OrdersTotal` 返回 `len(cachedPositions) + len(cachedOrders)`；`OrdersHistoryTotal` 返回 `len(cachedHistory)`。
- 属性函数优先读 `currentOrder`，否则读 `currentPos`。
- `invalidateOrderCaches` 在每次事件开始、每次成功交易变更、每个 signal 分支后调用（`vm_helpers.go:17`）。

**CTrade 与 tradeMagic/tradeDeviation**：
- `builtinCTradeBuy/Sell` 调 `ctradeOrder`，构造 `sdk.OrderRequest` 时用 `vm.tradeMagic` 与 `vm.tradeDeviation`（`vm_builtin_trade.go:583-609`）。
- `SetExpertMagicNumber/SetDeviationInPoints` 直接写这两个 VM 字段（`vm_builtin_trade.go:664`）。

**MQL5 vs MQL4**：
- `PositionsTotal` 只数 `cachedPositions`；`PositionGetTicket` 按 index 设 `currentPos`。
- MQL4 `OrderSelect` 混合 positions 与 pending orders；MQL5 `Position*` 只处理 positions，pending orders 由 `OrderGet*` 系列提供（当前均为 stub 返回 0/空，`vm_builtin_mql5_trade.go:28`）。

### 4.5 Indicator 域（`vm_builtin_indicators.go`, `vm_builtin_indicators_ext.go`, `vm_builtin_onarray.go`）

- 每个 builtin 把 MQL 参数位置解析为 period/method/applied_price/shift，调 `vm.ctx.Indicators()`。`maMethodName` 把 `0/1/2/3` 映射到 `sma/ema/smma/lwma`（`vm_builtin_indicators.go:28`）。
- `lastIndicators` 诊断捕获：仅 `shift==0` 时记录，避免历史偏移覆盖当前值。`recordDiag` 用 FNV-1a hash + `diagKeyCache` 避免热路径 `fmt.Sprintf` 分配（`vm_builtin_diag.go:69`）。
- `iADX:MODE_PLUSDI/MODE_MINUSDI` 设 fatalError——SDK 只提供主 ADX 线，不支持 `+DI/-DI` 子线；策略若用这些模式 fail-closed 终止而非返回错误值（`vm_builtin_indicators.go:195`）。
- `*OnArray` 变体：`arrayToDecimals` 转用户数组 → `sliceForShift` 截断 → 本地 sma/ema/smma/lwma 计算。数组越界/长度不足统一返回 0 不报错（`vm_builtin_onarray.go:18`）。

### 4.6 Account/Market 域（`vm_builtin_account.go`, `vm_builtin_market.go`, `vm_builtin_mql5_info.go`）

- `AccountBalance/Equity/FreeMargin/Margin/Leverage` 直接读 `vm.ctx.Account()`（`vm_builtin_account.go:13`）。
- live 模式权威值由 `contextImpl.Account()` → `runner.broker.Account()` 提供；harness 模式从 `liveBalance/liveEquity/...` 字符串字段解析（`runner/broker.go:203`）。
- `AccountNumber()` 返回 `vm.ctx.Account().Login`，live/harness下来自 `liveLogin`（VM-TRADE-CONTEXT-6，`runner/context.go:36`）。
- `IsDemo/IsConnected/IsTradeAllowed` 读 `liveIsDemo/liveIsConnected/liveIsTradeAllowed`（VM-API-TRUTH-3，`vm_builtin_checkup.go:16`）。
- `Bid/Ask` 优先用 `tickBid/tickAsk`，无 tick fallback 到 `bars.Close(0)`（`runner/context.go:178`）。

### 4.7 Array/String/Time/Math/Convert 域

- `ArrayResize` 修改 `args[0].Array` 长度；`ArrayCopy` 在 `dst[dstStart:]` 写入；`ArraySetAsSeries` 返回 `true` 的 no-op（VM 数组永远 series 索引，`vm_builtin_impls.go:224`）。
- `StringFormat` 用 `fmt.Sprintf`，Decimal 转 `InexactFloat64`；`StringFind` 用 `strings.Index`；`StringSubstr` 支持越界返回 `""`。
- `TimeCurrent` 返回 `ServerTime()/1000`（秒级）；`TimeLocal` 复用 `TimeCurrent`（无本地时区偏移）；`TimeToStruct` 按 Unix 秒构造 `MqlDateTime`。
- `NormalizeDouble(value, digits)` = `value.Round(int32(digits))`；`DoubleToString` = `d.String()`。

### 4.8 Globals/Checkup/Diag 域

- `GlobalVariable*` 用**包级全局 map** `gvStore map[string]float64`，受 `gvStoreMu` 保护（`vm_builtin_globals.go:13`）。不是持久化，不是 VM globals 数组；同进程多 VM 共享。
- `GetLastError/ResetLastError` 注册为 no-op（固定返回 0/None）；`SetUserError` 返回 None 无状态修改（`vm_builtin_impls.go:73`）。**关键盲区**：VM 不维护 `GetLastError` 状态；`OrderSend` 返回的 broker error 直接 fatal-out，无法通过 `GetLastError` 读取。
- `diagKeyCache` 用 FNV-1a hash 参数得 uint64，先查缓存命中零分配，未命中首次构建 key 并缓存（`vm_builtin_diag.go:69`）。

---

## 5. Mutation Coordinator 子系统（实盘交易屏障）

### 5.1 trade_barrier.go 状态机

```
idle  ──(Acquire)──>  submitting
submitting  ──(NotifyBrokerAccepted(ticket))──>  acceptedUnconfirmed
submitting  ──(NotifyDeterministicRejected)──>  deterministicRejected
submitting/acceptedUnconfirmed  ──(NotifyOutcomeUnknown)──>  outcomeUnknown
acceptedUnconfirmed  ──(NotifyConfirmationEvent: ticket+magic+updateType 匹配)──>  confirmed
confirmed / deterministicRejected  ──(Release)──>  idle
outcomeUnknown  ──(Reconcile(true/false))──>  confirmed / deterministicRejected  ──(Release)──> idle
outcomeUnknown  ──(无 Reconcile)──>  永久锁定（fail-closed）
```

- `Acquire` 全程在 `b.mu` 下（单一 mutex 下的 compare-and-set 等价），仅 `idle` 状态允许进入 `submitting`（`trade_barrier.go:162`）。每个 `ActiveSession` 只挂一个 `TradeBarrier`（`session_registry.go:52`），故 `Acquire` 返回 false 即"该 session 已有未确认 mutation"——I1 落地。
- `WaitConfirmed` 同步阻塞在 `cond.Wait()`，直到 `cond.Broadcast()` 或 ctx 取消。这直接恢复 MT4 EA `OrderSend` 单线程语义——I2 落地。
- **pre-listen caching**：`coordinateMutation` 在 broker RPC 前先 `SubscribePositionSnapshots` 并起 goroutine 消费。若 barrier 仍在 `submitting`（RPC 未返回 ticket），事件存入 `eventCache`（上限 16 条）。之后 `NotifyBrokerAccepted(ticket)` 检查缓存，命中且 `updateType` 兼容时直接 `submitting → confirmed`（`trade_barrier.go:191`）。处理 order-event-before-RPC-response 真实竞态。
- `Release` 只由 `confirmed`/`deterministicRejected`/ctx 取消触发。**outcomeUnknown 下不 Release**，barrier 永久锁定除非 `Reconcile` 迁移——I4 落地。

### 5.2 mutation_coordinator.go 5 路径共享协议

`coordinateMutation` 9 步协议（`mutation_coordinator.go:70-120`）：
1. `barrier.Acquire`
2. Pre-listen `OnOrderUpdate`（`SubscribePositionSnapshots`，`defer confirmUnsub()`）
3. `context.WithTimeout` broker RPC
4. `mthub.ClassifyMutationError` 分类
5. `barrier.NotifyBrokerAccepted(ticket)`
6. `waitForConfirmation`：push 等待
7. 无 push 则单次 `OpenedOrders` read-after-write（I6）
8. 再次分类 `confirmed / rejected / unknown`
9. `defer` 结束 listener

`mutationSpec` 字段：`action`（open/close/modify/cancel，决定验证器语义与 updateType 兼容表）、`clientID`（MtHub idempotency）、`expectedMagic`（`strategyMagic(ScheduleID)`）、`expectedTicket`（open 为 0，其余已知）、`brokerCall`（实际 RPC 闭包）、`verifyReadAfterWrite`（`verifyTicketPresent/Absent/Modified`）。

`defaultConfirmationConfig`（`trade_barrier.go:422`）：`pushWait=5s`、`readAfterWriteTimeout=10s`、`mutationRPCTimeout=30s`、`recoveryDelay=10s`。

### 5.3 typed error classify（I5 落地）

分类器在 `mthub/mutation_outcome.go:112`：
- 确定性 sentinel（`ErrBrokerRejected/ErrKillSwitchEngaged/ErrRateLimited/...`）→ `deterministic_rejected`
- `MutationError.Phase == PhasePreBroker` → `deterministic_rejected`
- `MutationError.Phase == PhaseBroker` → `outcome_unknown`
- 其它默认 `outcome_unknown`

`DeadlineExceeded`：pre-broker 阶段（preTradeChecks/gate/accountStateProvider 失败）→ `PhasePreBroker` → `deterministic_rejected`；真正 broker RPC 超时 → `PhaseBroker` → `outcome_unknown` → fail-closed 不重下。**I5 落地**。

### 5.4 read-after-write 验证器（`mutation_helpers.go:87`）

- `verifyTicketPresent(ticket)`：ticket 在 `OpenedOrders` 中 → open 确认
- `verifyTicketAbsent(ticket)`：ticket 不在 → close/cancel 确认
- `verifyTicketModified(ticket, sl, tp, price)`：ticket 存在且**显式传入**字段与 broker 返回一致；`nil` 指针表示"不检查"（R5-⑤，`parseDecimalPtr` 区分"未提供"vs"显式零"）

### 5.5 mutation_recovery.go

`recoverFromOutcomeUnknown`（`mutation_recovery.go:41`）：
- `select { case <-time.After(conf.recoveryDelay): case <-ctx.Done(): return }`——VM-AUDIT-2026-08-27-7：恢复等待期间 ctx 取消立即退出，不阻塞 10s。
- 检查 barrier 仍 `outcomeUnknown` → `OpenedOrders` 查询 → `verify` → `Reconcile(true/false)` → `Release` → `SetCircuitOpen(false)`。
- **只有 `expectedTicket != 0` 才启动 recovery**；`open`（ticket 未知）不 recovery，outcomeUnknown 是 fail-closed（需人工介入）。

### 5.6 live_dispatch.go 5 路径

| 信号 | 调用 | 最终 |
|------|------|------|
| `buy/sell` | `dispatchMarketOrder → submitOrder` | `coordinateMutation` |
| 挂单 | `dispatchPendingOrder → submitOrder` | `coordinateMutation` |
| `close` | `dispatchCloseOrder` | `coordinateMutation` |
| `close_all` | `dispatchCloseAll` | 循环 `coordinateMutation`，任何 outcomeUnknown 立即 return |
| `modify` | `dispatchModifyOrder` | `coordinateMutation` |
| `cancel` | `dispatchCancelOrder` | `coordinateMutation` |

`dispatchPaperSignal` 不走 coordinator——paper 无真实 broker，不需要 barrier/read-after-write。

### 5.7 position_cache.go（DATA-TRUTH-5）

`PositionSnapshot` 字段：`Leverage/Balance/Credit/Equity/Margin/FreeMargin/MarginLevel/Profit` + `FinancialsAuthoritative/FinancialsSource/CapturedAt` + `PositionsAuthoritative/Positions/PositionsCapturedAt/PositionsSource` + `UpdateTicket/UpdateType/UpdateMagic`（一次性触发元数据，不保留给 replay）。

- `GetFreshTradingSnapshot`：金融 + 持仓都须 fresh（90s），VM/Risk Gate 用。
- `GetFreshFinancialSnapshot`：只看金融 freshness。
- merge 逻辑：positions-only 更新替换 Positions/PendingOrders 保留旧金融；financials-only 更新保留 Positions/PendingOrders 只更新金融字段。
- 拒绝非权威/无 provenance：`CapturedAt.IsZero() || FinancialsSource == ""` → log.Warn + return（`position_cache.go:114`）。

### 5.8 不变量 I1-I8 落地证据

| 不变量 | 落地证据 |
|--------|----------|
| I1 每 ActiveSession 最多一个未确认 mutation | `TradeBarrier` 每 session 唯一 + `Acquire` 只接受 idle（`trade_barrier.go:165`） |
| I2 恢复 trade command 串行语义 | `coordinateMutation` 同步阻塞到 `WaitConfirmed`（`mutation_coordinator.go:230`） |
| I3 broker ticket ≠ positions caught up | `NotifyBrokerAccepted` 只设 ticket 不确认，须等 `NotifyConfirmationEvent` 或 read-after-write |
| I4 barrier 只由确定性 outcome 释放 | `Release` 只由 confirmed/rejected/ctx 取消触发；outcomeUnknown 不 Release |
| I5 transport timeout = outcome unknown → 不重下 | `brokerError` 包装 → `PhaseBroker` → `outcome_unknown`（`service_orders.go:54`） |
| I6 禁 per-tick 轮询 | `waitForConfirmation` 仅无 push 时单次 `OpenedOrders`（`mutation_coordinator.go:300`） |
| I7 保留 TickSeq ClientID 语义 | `TickSeq.Add(1)` + `strategyOrderClientID` 保证同 run+bar+signal 相同 client ID |
| I8 OrdersTotal() 保留 native MQL 账户级语义 | `backfillContextStrings` 不按 magic 过滤，全量 Positions+PendingOrders 进 VM context |

---

## 6. 指标缓存子系统

### 6.1 SDK 接口层

`sdk.IndicatorSet`（`strategy/sdk/indicators.go:8`）定义 ~45 个指标方法（MA/EMA/RSI/MACD/ATR/Bollinger/Stochastic/CCI/ADX/MFI/OBV/SAR/StdDev/WPR/Momentum/ICustom/Alligator/Ichimoku/Envelopes/DeMarker/OsMA/RVI/Force/Fractals/Gator/AC/AD/AO/BearsPower/BullsPower/BWMFI/AMA/DEMA/TEMA/FrAMA/VIDyA/TriX/ADXWilder/Chaikin/Volumes）。

`sdk.BarSeries`（`strategy/sdk/series.go:5`）MQL 反向索引：`[0]=current, [1]=previous`。`barSlice.at(shift)` = `bars[len-1-shift]`，越界返回零值 `Bar{}`。

### 6.2 SeriesCache（`strategy/indicators/cache_core.go`）

`SeriesCache` 持有 17 个 series map（ema/smma/sma/lwma/rsi/atr/adx/macd/chaikin/ad/obv/sar/force/ama/dema/tema）+ n/lastRev/hasRev。

`EnsureUpdated()` 协议（`cache_core.go:69`）：
1. Revisioned source 且 revision 变化 → `reset()` + lazy rebuild
2. 非 revisioned source 且 `n < c.n` → `reset()`
3. 否则只处理新增 bar，`O(new bars)`

`reset()` 覆盖全部 17 个 series map + n/lastRev/hasRev（`cache_core.go:102`）。

`processBar` 按 `i = n-c.n-1` 递减到 0 处理（由旧到新），对每个已存在 series map 增量更新（`cache_core.go:124`）。

### 6.3 RevisionedBarSource（LIVE-INDICATOR-1 修复）

`runnerBarSource` 实现 `RevisionedBarSource`（`Revision() uint64`），`Revision()` 返回 `runner.barRevision()`（`bar_source.go:54`）。

`Runner.barRev atomic.Uint64` 只在 `OnBar` 中 `Add(1)`，`OnTick/OnTrade/OnTimer/OnBookEvent` 不推进（`runner.go:142`）。原因：tick/trade/timer/book 事件发生在同一根 bar 内，不改变 bar 窗口内容；只有 `OnBar` 替换/滚动 bar 序列才需让缓存失效重建。

**修复根因**：实时模式 `maxContextBars=500` 滚动窗口，`appendDedupBar` 每次追加后裁回 500。窗口滚动后 `Len()` 仍 500 但内容已变。老版 `SeriesCache` 只比较 `Len()` 误判"无新 bar" → 指标冻结。修复：`RevisionedBarSource` 检测 revision 变化触发 `reset()` + lazy rebuild。

**对抗测试**：
- `TestSeriesCache_RevisionedRollingWindow`：删 revision reset → 23 指标全 RED
- `TestSeriesCache_RevisionUnchanged_NoRebuild`：同 revision 多次 `EnsureUpdated()`，EMA series 指针不变（tick 热路径不重建）
- `TestRunner_BarRevision_AdvancesOnBarOnly`：删 `barRev.Add(1)` → RED

### 6.4 回测路径

回测用 `btBarSource`，**不实现 `RevisionedBarSource`**，依赖 `Len()` 单调递增触发增量更新，`O(新增 bars)`（`backtest/bar_source.go`）。回测 append-only，无需 revision。

`btBarSeries` 包装器处理 `Volume(0)=1`（模拟 MT4 "Open prices only"，`bt_bar_series.go:24`）。

### 6.5 精度权衡

**缓存内部全部使用 `float64` 计算**，`decimal.Decimal` 进入时 `.Float64()` 丢弃尾数，返回时 `decimal.NewFromFloat(...)` 重新包装（`cache_ma.go:158`）。decimal 精度在热路径上被牺牲以换 float64 性能。无状态/非收盘价指标（Bollinger/StdDev/Stochastic/CCI/MFI/WPR/Momentum/Fractals/DeMarker/RVI/AC/AO/BWMFI/Ichimoku）每次全量扫描 `O(n)`。

`floatSeries.maxLen=1000`，超过丢弃最旧值；`shift > 999` 返回 0。

---

## 7. Python 子集编译器

### 7.1 Python 子集定义

**支持**：`class MyStrategy:` 单继承、`def __init__/on_init/on_bar/on_tick/on_timer/on_trade/on_trade_transaction/on_book_event/on_deinit`、用户函数、`if/elif/else`、`for i in range(...)`、`for pos in ctx.positions:`、`while`、`return/break/continue/pass`、赋值与增量赋值（`+=,-=,*=,/=,%=,//=,**=`）、二元/一元/三元表达式、`and/or/not`、`in/not in`、`True/False/None`、字符串/整数/浮点/十六进制字面量、`from decimal import Decimal`、关键字参数重排、`self.field`、类型注解。

**不支持**（`compile_py_subset.go:174`）：多重继承、嵌套 class、异常（try/except/finally）、上下文管理器（with）、lambda、生成器、装饰器、yield、async/await、global/nonlocal/del/assert/raise、切片、海象运算符、展开（`*args/**kwargs`）、集合字面量（list/tuple/dict/set）、元组/多赋值、模式匹配、f-string、ellipsis、type alias、Python2 print 语句。

### 7.2 编译管线

`CompilePythonToIR`（`compile_py.go:12`）：
1. `MaxSourceSize` 检查 + panic recovery
2. `validatePythonSubset`（文本层）+ `validatePythonCST`（tree-sitter 节点层）
3. `ParsePython`（`github.com/smacker/go-tree-sitter/python`）
4. `pyCompiler.compile(root) → IR`
5. 进入**同一个 `CompileAST`** + **同一个 `VM`**

与 MQL 分歧：MQL 需 `PreprocessMQL`，Python 不需要；Python 有独立 subset validator；Python IR 固定 `Version="python"`。

### 7.3 语句/表达式编译

- `for i in range(...)` 解糖为 C-style `StmtFor`（`compile_py_stmt.go:166`）
- `for pos in ctx.positions:` 解糖为 `range(0, PositionsTotal())` + `PositionGetTicket` + `PositionSelectByTicket`（`compile_py_stmt.go:251`）
- `in/not in` 映射为 `operator_in` builtin 调用（`compile_py_compare.go:73`）
- `True/False/None` → `BoolVal/NoneVal`
- `**` → `MathPow`；`//` 保留为 `ExprBinary Op="//"`
- `**=` 解糖为 `x = MathPow(x, rhs)`

### 7.4 SDK 映射（`compile_py_mapping.go`）

- `ctx.broker.buy/sell/buy_limit/...` → `CTrade.Buy/Sell/BuyLimit/...`
- `ctx.broker.close_all` → `CloseAll`
- `ctx.bars().close(1)` → `Close(1)`
- `ctx.bars_tf("H4").close(0)` → `iClose("", 240, 0)`
- `ctx.account.balance/equity/margin` → `AccountBalance/AccountEquity/AccountMargin`（**仅方法调用触发，`ctx.account.equity` 无括号不走映射**）

### 7.5 与 MQL 共用 VM 的差异点

1. **隐式全局变量**：`compile.go:284` 对 `Version=="python"` 开启隐式全局，函数/事件内 `x=1` 落到 VM 全局槽（仅函数参数进局部槽）。与 Python 局部作用域语义不符——**已知风险**。
2. **`//` floor division**：Python `//` 向负无穷取整（`-7//2=-4`）；MQL `/` 向零截断（`-7/2=-3`）。VM 用 `OP_FLOOR_DIV` + `floorDiv()` 区分（`vm_helpers.go:127`）。
3. **`None` vs MQL `0`**：Python `None` → `NoneVal`；函数无 return 值时 `OP_PUSH_CONST NoneVal()`。
4. **`bool(None)` 语义错误**：`compile_py_expr.go:165` 把 `bool(x)` 硬编码为 `x != 0`，`bool(None)` 得 `true`（与 Python 语义相反）——**已知风险**。

---

## 8. 回测路径

```
executeVMBacktest
  → importedRepo.GetBytecode (缓存)
  → CompileMQLCached (命中校验 SourceHash+Version)
  → 缓存命中时 recompile 恢复 CoverageReport (缓存省略了 coverage)
  → importedRepo.SaveBytecode (持久化新编译)
  → runVMEngine
      → klinesToBars + buildBacktestConfig + applySymbolInfo
      → backtest.New(cfg, vmRunner, bars).Run(ctx)
          → OnInit 一次
          → for i:=1..N: SetBar/检查挂单/SLTP/runStrategySignal(OnBar)
          → 同 bar 或下 bar open 派发 signal (SignalTiming)
          → 每 bar 记 equity、检查 margin call
          → OnDeinit
      → buildBacktestResponse: metrics + trades + equity + blind spots + risk
      → runDiagnostics: rule engine + coverage/runtime blind spots
      → 失败签名持久化 (failureSigRepo)
```

回测 `signalMode=false`，VM 的 Order* builtin 直调 `ctx.Broker()`（SimBroker），VM 返回 nil signal，engine 不二次派发。

`buildBacktestResponse` 执行多项 P0 不变量检查：volume>0、capital conservation、trade field integrity（price positive/side valid/time order）、Defense A violations、fatal blind spots → 任一违反设 `Risk.IsReliable=false`。

---

## 9. 实盘路径

两种模式：
- **单次 RPC** `executeVMLive`：每请求编译+Init+dispatch 一次（无状态）。
- **常驻 session** `VMLiveSession`：首 bar 编译一次 + Init 一次，后续 `SendEvent` 复用。

```
liveRunner 订阅 bar/tick/trade channel (push-first，无轮询)
  → handleBar: appendDedupBar(500 窗) → buildLiveContext → initVMSession(首帧) → session.Start/SendEvent
  → VMLiveSession.dispatch(req)
      → vmHandleBar/Tick/Trade/Timer
          → UpdateLiveState(positions+pendingOrders) + SetLogin + SetAccountStatus + UpdateSymbolInfo
          → parseBarsStrict (VM-TRADE-CONTEXT-6 S3 严格解析，非法 decimal fail-closed)
          → runner.OnBar → VMRunner.OnBar → vm.RunOnBar → runLoop
          → 返回 sdk.Signal → vmSignalToProto
  → dispatchResponse → dispatchLiveSignal
      → paper: paperEngine
      → live: 按 action 分发 → 全部经 coordinateMutation (同步 barrier)
```

实盘 `signalMode=true`，VM 的 Order* builtin 造 pending `sdk.Signal`，runner 返回给上层，由 `live_dispatch.go` 经 `coordinateMutation` 同步派发到 mthub → mtapi.io。**LIVE-ORDER-REENTRY-1 核心修复**：旧代码 `submitOrder` 用 goroutine fire-and-forget，违反 MT4 EA `OrderSend` 单线程语义 → 重复开仓；现改为同步 barrier 恢复串行语义。

**Session 接口传结构体指针**（FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP）：`Session` 接口传 `*antv1.ExecuteLiveRequest/Response` 而非 `[]byte`，避免 proto3 把空 repeated slice 折叠成 nil（"无持仓" vs "数据缺失"不可区分）。

---

## 10. 已落地硬伤修复总览

从 STATE.md / registry 摘要，VM 管线多轮深度审计的关键修复均已 ✅done：

| ID | 问题 | 修复点 |
|----|------|--------|
| VM-CACHE-INTEGRITY-1/2/5 | stale bytecode / Python bytecode 跑 MQL | SourceHash + Version 校验，缓存命中强制 recompile 恢复 coverage |
| VM-COMPILER-SEMANTICS-1/4 | comma_expression / 保留字 / missing initializer | ExprSeq + checkReservedKeywordUsage + hasMissingInitializer |
| VM-TRADE-CONTEXT-1/2/6 | 交易上下文失真 / broker error 静默 / login 来源 | 集中缓存失效 + fail-closed + 服务端权威 login |
| VM-TIMESERIES-SEMANTICS-1 | timeseries 语义 | 8 项对抗证明 |
| VM-RUNTIME-FAILCLOSED-1 | 错误传播 | builtin error → fatalError → runLoop 顶部捕获 |
| LIVE-INDICATOR-1 | 500-bar 滚动窗口指标冻结 | RevisionedBarSource + SeriesCache revision 检测 |
| LIVE-ORDER-REENTRY-1 | 实盘重复开仓 P0 | trade_barrier + mutation_coordinator 同步 barrier |
| LIVE-MQL-ORDER-CONTEXT-1 | MQL order context 字段丢失 | LivePosition/LivePendingOrder proto 全字段 + harness 透传 |
| VM-AUDIT-2026-08-27-1..8 | 8 项 round 4-5 遗留 | Python live SourceHash / fatalError 重置 / MaxStackDepth / popN early return / dispatch default / compileForLive 统一 / recoverFromOutcomeUnknown / PositionCache panic recovery |
| DATA-TRUTH-2b | MT4 margin 缺失 | 从 AccountSummary 补齐 |
| TRUST-1 | demo/real 战绩混展 | adapter + mdtick + service + marketplace 全链路区分 |

---

## 11. 潜在风险与盲区（本次审计识别）

### 11.1 P1 级

| # | 风险 | 位置 | 说明 |
|---|------|------|------|
| P1-1 | `isFatalUnimplemented` 注释与代码不一致 | `vm_helpers.go:200` vs `api_registry.go:64` | 注释说 Object/Chart/File 是"非致命 silent blind spot"，但 registry 标 `StatusUnsupported`，`isFatalUnimplemented` 对 `StatusUnsupported` 返回 `true`。策略若调用这些会 fatal 而非 silent skip，与 ADR §5.4 "Skip 级永久盲区静默跳过"契约不符。 |
| P1-2 | `GetLastError/ResetLastError/SetUserError` 全为 no-op | `vm_builtin_impls.go:73` | MQL 策略常见的 `if(GetLastError()==ERR_TRADE_NOT_ALLOWED)` 永远拿不到真实错误码；`OrderSend` 失败直接 fatal-out 而非返回 -1 + 设置 error。与 MQL4 原生语义不符。 |
| P1-3 | Python 隐式全局变量 | `compile.go:284` | Python 函数/事件内 `x=1` 落到 VM 全局槽（仅函数参数进局部槽）。并发事件间、递归调用间状态互相污染，与 Python 局部作用域语义不符。 |
| P1-4 | `bool(None)` 为 `true` | `compile_py_expr.go:165` | `bool(x)` 硬编码为 `x != 0`，`None` 不等于 0 故 `bool(None)=true`，与 Python 语义相反。 |
| P1-5 | Python 语言检测漏 `from decimal import Decimal` | `strategy/sdk/language.go:60` | `isPythonSource` 未包含 `from decimal import Decimal`。不含 `StrategyBase/from alphaforge/def on_...` 的合法 Python 子集源码会被误判为非 Python，路由到 MQL 路径。 |
| P1-6 | 指标缓存 float64 精度 | `cache_ma.go:158` | 缓存内部全 `float64`，`decimal.Decimal` 进入时 `.Float64()` 丢弃尾数。与纯 decimal 参考实现可能存在尾差；超长序列累计误差、`decimal.NewFromFloat` 非精确性是已知权衡。 |
| P1-7 | `waitForConfirmation` 强制 `return barrierConfirmed` 兜底 | `mutation_coordinator.go:334` | 当 `verify` 成功但 `NotifyConfirmationEvent` 未正常迁移状态时，代码直接 `return barrierConfirmed` 绕过状态机，削弱 barrier 状态机单一真相源。 |
| P1-8 | outcomeUnknown 的 open 路径永久锁仓 | `mutation_coordinator.go:264` | `open` 无 known ticket 无法 recovery；网络长期中断需人工介入。设计选择但需运维可见。 |

### 11.2 P2 级

| # | 风险 | 位置 | 说明 |
|---|------|------|------|
| P2-1 | `ArrayResize/ArrayCopy` side-effect 作用域 | `vm_builtin_string.go:210` | 修改 `args[0].Array`，但 `args` 是按值传入的 slice。若全局数组作为值传入，修改的是本地 `Value` 的 slice header，未必能正确写回 `vm.globals` slot。 |
| P2-2 | `TimeLocal` 与 `TimeCurrent` 相同 | `vm_builtin_impls.go:121` | 无本地时区偏移；`TimeGMTOffset` 用 `time.Now().Local().Zone()`，但 `TimeLocal` 不应用该偏移。 |
| P2-3 | MQL5 pending/history/deal 函数全为 stub | `vm_builtin_mql5_trade.go:28` | `OrderGet*/History*/HistoryDeal*` 全返回 0/空/true，不提供真实数据。 |
| P2-4 | `iADX/iADXWilder` 的 `+DI/-DI` 会 fatal | `vm_builtin_indicators.go:195` | 策略若用这些子线被强制终止而非返回 0。设计选择（fail-closed 优于错误值），但应在覆盖度报告显式标记。 |
| P2-5 | `GlobalVariableName` 顺序不确定 | `vm_builtin_globals.go:84` | Go map 迭代随机，MQL 代码依赖 `GlobalVariablesTotal + GlobalVariableName(i)` 遍历得非稳定结果。 |
| P2-6 | `OrderSend` 参数校验致命化 | `vm_builtin_trade.go:31` | 非法 `cmd` 或 `volume<=0` 返回 error 触发 fatal，与 MQL4 返回 -1 + `GetLastError` 原生行为不同。 |
| P2-7 | `floatSeries.maxLen=1000` | `series_base.go:5` | 策略查询 `shift > 999` 返回 0，可能被误解为"无数据"。 |
| P2-8 | Python 属性访问与调用混用限制 | `compile_py_mapping.go` | SDK 映射仅在方法调用路径触发。`ctx.account.equity`（无括号）不走映射，用户必须写 `ctx.account.equity()`。 |
| P2-9 | Python 无直接 MQL↔Python parity 测试 | `compile_py_test.go` | 可靠性依赖共享 VM 的间接测试，存在跨语言语义漂移未被发现的风险。 |
| P2-10 | `dispatchCloseAll` outcomeUnknown 后未返回已关闭数量 | `live_dispatch.go:186` | 直接 `return`，调用方无法感知哪些已关闭；logging 有但 API 不返回。 |
| P2-11 | `WaitConfirmed` 每次起 watcher goroutine | `trade_barrier.go:288` | 长时间等待产生 goroutine；单 session 同时只有一个未确认 mutation，但建议确认是否可复用。 |
| P2-12 | `actionCompatibleUpdateTypes` 类型不一致 | `trade_barrier.go:91` | 映射 key 是 `string(actionOpen)` 等，但 `cancel` 直接写死字符串 `"cancel"`；`mutationAction` 有 `actionCancel="cancel"` 语义一致但类型转换不一致，refactor 易出错。 |
| P2-13 | `Decimal("...")` 退化为字符串 | `compile_py_expr.go:174` | `Decimal` 调用直接返回第一个参数，`d = Decimal("0.1")` 留 `StringVal("0.1")`，后续算术是否正确解析为 Decimal 取决于 VM `ToDecimal`。 |
| P2-14 | 回测 btBarSource 无 revision | `backtest/bar_source.go` | 依赖 `Len()` 单调递增；若回测出现回退/修改历史 bar 会误判。当前 append-only 安全，但未来若支持 bar 修正需补 revision。 |

---

## 12. 当前 open 债务（VM 相关）

从 STATE.md，VM 管线本身已无 open 条目。仍 open 的是周边：

- **TRON-SECURITY-1** 🟦open：提现冷签 MITM（`tron_client.go:34` 仍 `insecure.NewCredentials()`，P0 资金）——业主暂缓不做。
- **DATA-TRUTH-1** 🟦open：orders 表 reconciliation 只检测不收敛——需架构决策。
- **MQL-COMPILER-LOCAL-ARRAYS** 🟦open：局部动态数组盲区（CHAT-CTX 遗留③立债）。
- **SCHEDULE-HOTLOOP-1** ⚠️待生产部署验收。

本次审计识别的 P1/P2 风险（§11）建议按优先级入 registry 跟踪。

---

## 13. 审计方法与证据

- **主审计方**：Devin CLI 通读核心文件（vm.go/bytecode.go/vm_execute.go/interp_runner.go/compile.go/runner.go/sdk/strategy.go/context.go/vm_live_session.go/backtest_worker_vm.go/live_dispatch.go/vm_live_handlers.go/vm_helpers.go/builtins.go/vm_builtin_wiring.go/compile_interp.go/engine.go/live_runner.go）。
- **并行 subagent 深探**：4 个只读 subagent 分别覆盖 Builtin 实现、Mutation Coordinator、指标缓存、Python 子集编译器，返回结构化事实报告。
- **交叉验证**：subagent 发现与主审计方通读结果一致；关键 file:line 引用已交叉核对。
- **未运行测试**：本次为只读审计，未执行 build/test/gate；历史对抗证明引用自 registry。

---

## 14. 结论

VM 管线实际架构与 ADR-0023 契约一致，四段式（CST→IR→Bytecode→VM）落地完整。多轮深度审计的对抗证明已覆盖缓存完整性、编译语义、交易上下文、fail-closed、滚动窗口指标、重复开仓等关键硬伤。fail-closed 贯穿全管线。本次审计未发现 P0 级新缺陷，识别的 P1/P2 风险建议入 registry 跟踪。

**审计方决定**：本报告为只读深度审计，不涉及代码改动，无需验收。报告落档 `docs/audits/`，变更日志 append 到 `handover-audit-plan.md`。§11 识别的风险条目建议业主决定是否立债施工。

---

> 报告结束。引用的具体代码位置均基于 `/opt/ant/backend` 当前源码（2026-09-16）。
