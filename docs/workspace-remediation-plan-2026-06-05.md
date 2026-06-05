# Strategy Workspace 架构整改方案（供 AI Agent 落地）

**日期**: 2026-06-05
**目标**: 对 `/strategy/workspace` 页面涉及的全部功能做代码级架构审计，确认是否最优解，给出**稳健、强壮、符合项目既有逻辑**的整改方案。
**实施原则**: 不追求快，追求对。每一项整改都**对齐项目已存在的最优范式**（见引用），不引入新风格。
**配套阅读**: 本文与 `workspace-pipeline-audit-2026-06-05.md`（含第十节"追加审阅"）互补——后者是缺陷清单，本文是**架构判定 + 整改落地**。重复项只引用编号不重述。

---

## 一、审计对象（实际依赖树）

```
StrategyWorkspacePage.tsx
└─ useStrategyWorkspaceState()        ← 聚合 hook（god-hook）
   ├─ useStrategyCode()               代码/校验/模板 CRUD/保存
   ├─ useBacktestParams()             回测参数 + Smart Tuning + Gate
   ├─ useAccount() + useWorkspaceStore()   账户选择/布局持久化
   ├─ useAccountFinancials(TQ)        账户资金（SSE→TanStack Query）
   ├─ usePositionsQuery(TQ)           持仓（SSE→TanStack Query + 60s 轮询）
   └─ useTradingStore(Zustand)        ⚠️ hasReceivedData 门（遗留）
   组件：WorkspaceToolbar / WorkspaceCodePanel / AIChatPanel / PriceChart /
        BacktestParamsCard / WorkspaceBacktestPanel(→SmartTuningPanel/GatePanel) /
        QuickTradePanel / MiniPositionsTable / BacktestRunDrawer / SaveTemplateModal
后端：
  pythonStrategyApi → backtest_worker.go（主回测，参数透传正确）→ Python sandbox
  strategyExperimentApi → strategy_experiment_worker.go（Smart Tuning，⚠️ 参数硬编码）
  gateApi → 6-Gate 管线（SSE）
  codeAssistApi → CodeAssist（LLM）
  tradingApi → PlaceOrder/CloseOrder（OMS→MT 网关）
```

---

## 二、架构判定总表

| # | 子系统 | 现状 | 是否最优 | 整改优先级 |
|---|--------|------|:---:|:---:|
| R1 | 实时持仓/资金数据源 | SSE→TanStack Query（桥接正确）**但** workspace 残留 Zustand `hasReceivedData` 门 + 无条件 60s 轮询 | ⚠️ 接近最优，有遗留污染 | **P0** |
| R2 | Smart Tuning 数据流 | 前端不传 code/symbol/tf/range；worker 硬编码 `XAUUSDm/1h/近一月` | ❌ 本质不可用 | **P0** |
| R3 | 下单返回值 | `orderSend` 丢弃后端响应，永远"成功" | ❌ 正确性缺陷 | **P0** |
| R4 | 流/异步生命周期 | 回测 watch、gate、experiment watch 卸载不清理；`runTuning` 无 catch | ❌ 泄漏/无反馈 | **P0** |
| R5 | 沙箱安全 | 回测路径无静态扫描（死代码）+ 无 rlimit | ❌ 双重缺失 | **P0** |
| R6 | 后端 ctx 传播 | `runOneShot` 用 `context.Background()`；`time.Sleep` 轮询 | ❌ 不可取消 | **P1** |
| R7 | 前端状态聚合 | `useStrategyWorkspaceState` god-hook，40+ 字段手工 `bt*` 重映射 | ⚠️ 可维护性差 | **P1** |
| R8 | 主回测 worker | 参数透传正确 ✅，但函数 181 行 + 轮询而非 NOTIFY + 死代码 | ⚠️ 行为对、结构需收敛 | **P1** |
| R9 | 错误处理/类型 | 多处静默 `catch{}` + `as any` 逃逸 proto 类型 | ⚠️ 健壮性 | **P2** |
| R10 | 布局/渲染 | 1360 宽度上限套住 workspace；trades overlay 内联未 memo | ⚠️ 体验/性能 | **P2** |

> ✅ **已是最优、勿动**：ConnectRPC 全协议一致；SSE→TanStack Query 桥接（`bridgeStreamEvents.ts`）；CodeMirror `Compartment` 模式；klinecharts `registerOverlay`；Zustand `persist`（仅 UI 态）；主回测进程隔离沙箱；Gate 6 关 SSE 流式。

---

## 三、整改项（按优先级，每项含：现状/为何非最优/最优解/落地/验收）

### P0 —— 正确性与安全（必须先做）

---

#### R1. 实时数据源统一到 TanStack Query，清除 Zustand 遗留门

**现状（代码级）**
- `usePositionsQuery.ts:21` `refetchInterval: 60_000` 无条件轮询。
- `useStrategyWorkspaceState.ts:96` `if (!tradingStore.hasReceivedData(accountId)) return;` 作为 `fetchTradeHistory` 的门。
- `tradingStore.ts:108,209` `hasReceivedData` 依赖 `accountReceivedData`，而它**只在 `setAccountInfoById` 内被 `.add()`**。

**为何非最优**
- 项目 SSE 已迁移为"**只写 TanStack Query**"（见 `stream-pattern` skill：`bridgeStreamEvents.ts` 写 TQ，不写 Zustand）。`setAccountInfoById` 是**遗留路径**，workspace 不在该路径上 → `hasReceivedData(accountId)` **恒为 false** → `qtRecentTrades` **永远为空**（Quick Trade 最近成交不显示）。
- 这正是 `stream-pattern` skill 记录的"Dashboard 读 Zustand、SSE 写 TQ"同类陷阱的复发。
- 无条件 60s 轮询与 SSE 重复拉取，违反"SSE 为主、轮询仅兜底"。

**最优解（对齐项目既有范式 `AccountDetail`）**
`useAccountDetailData.ts:43` 已确立的正确写法是 **`const hasReceivedData = financialsQ.isSuccess;`**（TQ 成功即视为已收到数据）。workspace 照搬：

```ts
// useStrategyWorkspaceState.ts
const financialsQ = useAccountFinancials(accountId);   // 已有 data，改取 query 对象
// 门改为 TQ 成功标志，删除对 tradingStore.hasReceivedData 的依赖
const fetchTradeHistory = useCallback(async () => {
  if (!accountId || !financialsQ.isSuccess) return;
  if (tradeCacheRef.current.has(accountId)) return;
  ...
}, [accountId, financialsQ.isSuccess]);
```
轮询改条件化（对齐 H3 最优解）：
```ts
// usePositionsQuery.ts
refetchInterval: (q) => {
  const last = q.state.dataUpdatedAt;
  return Date.now() - last > 90_000 ? 60_000 : false;  // SSE 活跃则不轮询
},
```

**落地步骤**
1. `usePositionsQuery` 的 `refetchInterval` 改为函数式条件轮询。
2. `useStrategyWorkspaceState` 移除 `useTradingStore` 依赖（仅剩它处仍需则保留 import），`fetchTradeHistory` 门改 `financialsQ.isSuccess`。
3. 触发 `fetchTradeHistory` 的 `useEffect(..., [accountId])` 增加 `financialsQ.isSuccess` 依赖。

**验收**
- 切换账户后，Quick Trade "最近成交"在资金数据到达后出现（不再恒空）。
- SSE 推送活跃时，Network 面板无 60s 周期性 `OpenedOrders` 轮询；断流 >90s 后才兜底轮询。

---

#### R2. Smart Tuning 打通工作区上下文（最严重）

**现状（代码级，证据三连）**
1. `useBacktestParams.ts:148-154` `submit({ baseTemplateId:'', parameterSpace, searchMethod, maxCandidates, objective })` —— **不传 code/symbol/timeframe/dateRange**。
2. `strategy_experiment_worker.go:82-86` `code := exp.StrategyCode; if code=="" { FAILED }` —— code 为空直接失败。
3. `strategy_experiment_worker.go:203-206` `Symbol:"XAUUSDm", Timeframe:"1h", FromTs:now-1月` —— **硬编码**，无视用户选择。

**为何非最优**：从工作区发起的 Smart Tuning 要么立即 FAILED，要么在错误标的上寻优——功能性 Critical。与主回测（`backtest_worker.go:77-96` 正确透传参数）形成"**两套真相**"。

**最优解（让寻优与主回测共用同一参数来源）**
1. 扩展 `SubmitStrategyExperimentRequest`（proto）：新增 `strategy_code`、`symbol`、`timeframe`、`from`、`to`、`execution_config`（commission/slippage/leverage/direction/strict）。
2. `useBacktestParams.runTuning` 提交工作区当前 `code/symbol/timeframe/startDate/endDate` + `executionConfig`（与 `runBacktest` 同源，避免重复读取）。
3. `strategy_experiment_worker.backtestAndScore` 用 `exp.Symbol/exp.Timeframe/exp.FromTs/exp.ToTs/exp.*` 替换 203-206 硬编码；**复用主回测的参数装配逻辑**（抽出 `buildBacktestRun(exp, overrides)` 共享函数，消除两套真相）。

**落地步骤**：proto 改字段 → `buf generate` → 后端 repository 落字段 → worker 读字段 → 前端提交字段。**注意**：`runTuning` 需从 `useStrategyWorkspaceState` 拿到 symbol/timeframe/dates（当前 `useBacktestParams` 不持有 symbol/timeframe，需由 `handleRunTuning` 传入，类似 `handleRunBacktest`）。

**验收**：选 EURUSD/15m + 自定义区间发起 Smart Tuning → 候选回测在 EURUSD/15m/该区间上执行（查 `backtest_runs` 记录的 symbol/timeframe/from_ts）；空 code 时前端禁用按钮并提示。

---

#### R3. `orderSend` 使用真实后端响应（镜像同文件 `orderClose`）

**现状**：`trading.ts:142` 读取了 `response` 但 158-165 返回硬编码 `{order:undefined,error:'',retcode:0,...}`。风控拒绝也被当作成功。
**为何非最优**：同文件 `orderClose:174-188` 已正确读 `response.status/message`——存在正确范式却未套用。
**最优解**：
```ts
const response = await tradingClient.placeOrder(create(PlaceOrderRequestSchema, {...}));
return {
  order: response.order,
  error: response.riskError ? (response.riskError.message ?? 'risk rejected') : '',
  retcode: response.retcode ?? 0,
  message: response.message ?? '',
  requestId: response.requestId,
  riskError: response.riskError,
};
```
（字段名以 `PlaceOrderResponse` proto 实际为准，先读 `src/gen/.../trading_pb.ts` 确认。）
**验收**：触发风控拒绝（如超额手数）时，前端 toast 显示拒绝原因，不再误报成功。

---

#### R4. 流/异步生命周期清理（卸载不泄漏 + 失败有反馈）

**现状**：`useBacktestParams.ts` 中回测 watch 句柄(98)是局部 const 未 ref 化；`gateStopRef`(129) 有 ref 但无卸载清理；`runTuning`(138-156) 无 catch；`strategyExperiment.ts:43` `watchExperiment` 返回裸 AsyncIterable 无 abort。
**为何非最优**：组件卸载后 SSE/watch 持续 setState（React 警告 + 泄漏）；寻优异常静默无反馈。项目其他流客户端（`gate.ts`/`stream.ts`）均挂 `AbortController`。
**最优解**：
1. 回测 watch ref 化 + 卸载清理（见 audit 第十节 A3 代码骨架）。
2. `runTuning` 补 `catch (e){ message.error(...) }`。
3. `watchExperiment` 包装 `AbortController` 返回 `() => void`，`SmartTuningPanel` 卸载调用之（修复 L3 空操作 abort）。
4. 统一卸载清理：
```ts
useEffect(() => () => { gateStopRef.current?.(); backtestWatchRef.current?.(); experimentWatchRef.current?.(); }, []);
```
**验收**：回测/寻优运行中切走路由，无 "setState on unmounted" 警告，Network 中 SSE 连接关闭。

---

#### R5. 沙箱双防线（静态扫描 + 资源限制）

**现状**：`sandbox_scan.py:34` `scan_code` 生产零调用（死代码，注释却称"executing 前调用"）；`backtest_sandbox.py:131-159` 子进程无 `resource.setrlimit`。
**为何非最优**：回测路径对用户代码**既无静态扫描又无内存/CPU 上限**；`sandbox_scan.py:15` 自己把 `resource` 列入禁用，反证设计预期有 rlimit 却没接。
**最优解（两者必须同时）**：
1. `compile_and_serialize`（或 `BacktestSandbox` 提交前）调用 `scan_code(code)`，violations 直接拒绝并返回 `compile_error`。
2. `_backtest_worker` 子进程 exec 前设限，**对齐 `live_sandbox.py:164-170`**：
```python
import resource
resource.setrlimit(resource.RLIMIT_AS, (mem_bytes, mem_bytes))
resource.setrlimit(resource.RLIMIT_CPU, (cpu_secs, cpu_secs))
resource.setrlimit(resource.RLIMIT_NOFILE, (64, 64))
```
失败时记录而非静默吞（修复 M39/L80）。
**验收**：提交 `while True: x=[0]*10**8` 的策略 → 进程被 rlimit 终止并返回明确错误，服务不被 OOM-kill；提交 `import os` → 静态扫描阶段即拒绝。

---

### P1 —— 健壮性与结构

---

#### R6. 后端 ctx 全链路传播（可取消）

**现状**：`strategy_experiment_worker.go:160` `runOneShot` 用 `context.Background()`；`backtestAndScore:228-229` `time.Sleep(5s)` 轮询。
**最优解**：`runOneShot(ctx,...)` 透传（`processOne`/`runOptimizer` 已带 ctx，**只改 runOneShot**）；轮询改 `select{ case <-ctx.Done(): return ctx.Err(); case <-time.After(5*time.Second): }`，一处同时解 C1+H8。
**验收**：取消实验或关服时，寻优子回测循环立即退出。

---

#### R7. 收敛 `useStrategyWorkspaceState` god-hook

**现状**：返回 ~60 字段，40+ 行手工把 `btCtx.*` 重映射成 `bt*` 前缀；页面通过 `ws.xxx` 平铺取用。
**为何非最优**：违反项目"聚焦 hook"约定（`useStrategyCode`/`useBacktestParams` 已是好粒度，却在聚合层被打平）；新增字段需三处改（hook 内、return、page 解构），易漏。
**最优解（分组透传，不打平）**：聚合 hook 返回**分组对象**而非平铺：
```ts
return { account, code: codeCtx, backtest: btCtx, quickTrade, layout, history };
```
页面 `ws.backtest.commission` 取代 `ws.btCommission`，删除 40+ 行重映射。组件 props 也按组传（缓解 audit M4 的 25-prop 下钻）。
**注意**：这是较大重构，**保持行为不变**，建议在 P0 完成且有回归验证后进行；分多次小 commit。
**验收**：页面渲染与交互完全不变；`useStrategyWorkspaceState` 行数显著下降，无 `bt*` 手工映射。

---

#### R8. 主回测 worker 结构收敛 + NOTIFY 驱动

**现状**：`backtest_worker.go:56-236` `executeBacktestRun` 181 行；`:70` `_ = time.Now()` 死代码；worker 轮询取任务（M1/M21）。参数透传本身**正确**，不要动语义。
**最优解**：
1. 拆 `startCancelWatcher` / `fetchBacktestData` / `callPythonEngine`（纯结构，行为不变）。
2. 删 `:70` 死代码。
3. 取消轮询/取任务改 `pglisten` LISTEN/NOTIFY 驱动 + 低频兜底（项目已有 `pglisten` 包，且 `WatchBacktestRun` 对前端已用 NOTIFY，worker 侧补齐即一致）。
**验收**：单测覆盖拆出的三函数；回测延迟从"≤3s 轮询"降到"NOTIFY 近实时"，断 NOTIFY 时兜底轮询仍工作。

---

### P2 —— 质量与体验

---

#### R9. 错误处理与类型

- 静默 catch（`useStrategyWorkspaceState.ts:107,139`、`useStrategyCode.ts:32`、`market.ts`）：区分"无数据"（正常空态）与"请求失败"（toast/日志）。模板加载失败应给可见提示而非空列表。
- `as any` 逃逸 proto（trades map、`getOrderHistory` 结果、`watchBacktestRun` 回调）：改用 `src/gen` 的 proto 类型；`tsconfig.app.json:34` 排除 `src/gen` 应评估恢复类型检查。
- **最优解原则**：前端只信 proto 类型，转换集中在 `dataAdapter`，不在业务 hook 里 `as any`。

#### R10. 布局与渲染

- **全宽**：`ContentContainer.tsx` 的 `maxWidth:1360` 套住 workspace（详见 `strategy-workspace-v31-visual-replica.md` 第 0.5 节方案 A）——加 `fluid` 开关，workspace 路由全宽。
- **trades overlay memo**：`StrategyWorkspacePage.tsx:130-133` 内联 `btMetrics.trades.map(...)` 每次渲染新数组 → PriceChart overlay 重建。用 `useMemo` 包裹后传入。
- **可访问性**：可点击 div 补 `role/aria-expanded/空格键`（audit L9/L11/L12）。

---

## 四、给 AI Agent 的实施顺序

```
阶段 0（P0 正确性，互相独立可并行）
  R3 orderSend 真实响应      ← 单文件，先做，立即可验
  R4 生命周期清理            ← 防泄漏，影响所有后续验证
  R5 沙箱双防线              ← 安全，后端+Python
阶段 1（P0 数据流，需 proto/后端协同）
  R2 Smart Tuning 打通       ← proto 改 + 前后端，最大工作量
  R1 数据源统一 TQ           ← 前端，依赖对 stream-pattern 的理解
阶段 2（P1 健壮性）
  R6 ctx 传播（后端）
  R8 主回测 worker 收敛 + NOTIFY
  R7 god-hook 收敛（大重构，需回归保护，放最后）
阶段 3（P2 质量）
  R9 错误/类型  +  R10 布局/渲染
```

**纪律**：
- 每项改动**先读引用的"既有正确范式"文件**，照其风格改，不自创。
- 改 proto 必须 `buf generate` 并同步前后端。
- 重构（R7/R8）保持行为不变，分小 commit，配回归验证。
- 不弱化/删除现有测试；为 R2/R3/R5 新增针对性测试。

---

## 五、与既有文档的关系

- 缺陷明细与严重性分级：见 `workspace-pipeline-audit-2026-06-05.md`（C1–C8 / H1–H10 / M / L）。
- 对该审计的更正与遗漏补充：见其**第十节"追加审阅"**（R2/R4/R5/R6 的根因均在此有代码级论证）。
- 本文 = 架构判定 + 最优解 + 落地顺序，三者配合交付 AI Agent。
