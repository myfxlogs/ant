# Strategy Workspace 全量修复方案

**日期**: 2026-06-06
**来源**: 功能评估报告 + 合规审查报告（5 路并行审计）
**发现问题**: 功能缺失 26 项 + 合规违规 68 项 → 去重合并后 **42 项需修复** → 审计后修正为 **34 项有效修复**
**审计日期**: 2026-06-06（附录含完整审计结论）
**实施原则**: 按优先级分 4 层（P0-P3），每项含现状/最优解/落地步骤/验收标准。所有方案经最优解审计验证。

---

## 修复优先级总览

| 优先级 | 数量 | 范围 | 判定标准 |
|:------:|:----:|------|---------|
| **P0** | 6 | 正确性/安全性/数据完整性 | 不改会导致：DB 存 nil、风控绕过、内存 OOM、Smart Tuning 不可用 |
| **P1** | 8 | 架构合规/功能可用 | 不改会导致：float64 精度风险、协议违规、轮询浪费、无 trace/metrics |
| **P2** | 13 | 工程健壮性 | 不改会导致：泄漏、静默失败、可维护性持续恶化 |
| **P3** | 7 | 优化/体验 | 不改会导致：UX 不佳、性能退化、代码难读 |

> **注意**: 原 42 项中 4 项已修复（P0-7/P0-9/P0-10/P1-7），1 项撤销（P1-3），2 项降级至 P3（P1-2/P1-6），净有效 34 项。

---

## P0 — 正确性与安全（必须最先修复）

---

### P0-1. 修复 proto.Marshal 错误静默丢弃（5 处）

**现状**: 5 处 `proto.Marshal` 错误被 `_` 丢弃，失败时 nil bytes 写入 DB 列。
**影响**: 回测 config snapshot 丢失、实验参数/评分丢失、候选记录损坏。
**涉及文件**:
- `backend/internal/connect/strategy/strategy_experiment_worker.go:105,106,333`
- `backend/internal/connect/strategy/strategy_experiment_handler.go:107`
- `backend/internal/connect/strategy/python_strategy_backtest_crud.go:73`

**最优解**:
```go
// 改前: paramProto, _ := proto.Marshal(...)
// 改后:
paramProto, err := proto.Marshal(paramsToProto(c.Overrides))
if err != nil {
    w.log.Error("marshal params failed", zap.Error(err), zap.String("expID", exp.ID.String()))
    continue  // 跳过此候选，不影响其他
}
```

**落地步骤**:
1. 逐文件替换 `_, _ := proto.Marshal` → `_, err := proto.Marshal` + 错误处理
2. strategy_experiment_worker.go:105,106 — marshal 失败时 skip 当前候选
3. strategy_experiment_worker.go:333 — marshal 失败时返回 nil + error
4. strategy_experiment_handler.go:107 — marshal 失败时返回 `connect.NewError(CodeInternal, err)`
5. python_strategy_backtest_crud.go:73 — 同上
6. 编译验证: `go build ./internal/connect/strategy/...`

**验收**: 
- CI 新增 lint 规则检测 `_, _ := proto.Marshal`（或 `errcheck` 覆盖）
- 强制 Marshal 失败场景（传不可序列化类型）→ 不会写 nil 到 DB

---

### P0-2. 修复状态更新错误静默丢弃（7 处）

**现状**: `_ = repo.UpdateExperimentStatus(...)` / `_ = repo.UpdateAsyncFields(...)` 丢弃 DB 写错误。写失败时实验/回测状态不更新，可能被重复处理。
**涉及文件**:
- `backend/internal/connect/strategy/strategy_experiment_worker.go:84,90,99,122`
- `backend/internal/connect/strategy/backtest_worker.go:203`

**最优解**:
```go
// 改前: _ = w.repo.UpdateExperimentStatus(ctx, exp.ID, "FAILED")
// 改后:
if err := w.repo.UpdateExperimentStatus(ctx, exp.ID, "FAILED"); err != nil {
    w.log.Error("update experiment status failed", zap.Error(err), zap.String("expID", exp.ID.String()))
}
```

**落地步骤**:
1. 替换所有 `_ = repo.Update*` 为 `if err := ...; err != nil { log.Error(...) }`
2. 关键路径（回测取消）失败时额外用 `log.Error`（非 Warn — 这代表数据不一致）

**验收**: 日志中出现 status update 失败记录但服务继续运行不 panic

---

### P0-3. 修复 MarketRegime UserID 从未设置

**现状**: `market_regime_handler.go:38-45` 创建 `MarketRegime{}` 未填 `UserID`，所有记录分配在 `uuid.Nil` 下，权限检查完全失效。
**涉及文件**: `backend/internal/connect/marketplace/market_regime_handler.go`

**最优解**:
```go
// 在 handler 中从 ctx 提取 userID:
userID, err := uuid.Parse(interceptor.GetUserID(ctx))
if err != nil { return nil, connect.NewError(connect.CodeUnauthenticated, err) }
row.UserID = userID
```

**落地步骤**:
1. 在 `DetectMarketRegime` handler 开头调用 `interceptor.GetUserID(ctx)`
2. 设置 `row.UserID = parsed`
3. 所有读路径（List/Get）确保按 userID 过滤

**验收**: 不同用户创建 MarketRegime → DB 中 `user_id` 列非 nil → List 仅返回自己的记录

---

### P0-4. 修复 orderSend 丢弃后端响应

**现状**: `trading.ts:142-166` 读取了 `response` 但 158-165 返回硬编码 `{order:undefined, error:'', retcode:0}`。风控拒绝也被当作下单成功。
**涉及文件**: `frontend/src/client/trading.ts`

**最优解**: 复用同文件 `orderClose:174-188` 的正确模式：
```ts
const response = await tradingClient.placeOrder(create(PlaceOrderRequestSchema, {...}));
return {
  order: response.order ?? undefined,
  ticket: response.ticket ? Number(response.ticket) : 0,
  error: response.riskError?.message ?? '',
  retcode: response.retcode ?? 0,
  message: response.message ?? '',
  requestId: response.requestId,
  riskError: response.riskError,
};
```

**落地步骤**:
1. 读 `src/gen/ant/v1/trading_pb.ts` 确认 `PlaceOrderResponse` 精确字段名
2. 替换硬编码返回值为 proto response 字段映射
3. 前端所有 `orderSend` 调用方检查 `result.error` 而非只检查 catch

**验收**: 触发风控拒绝（如超额手数）→ 前端 toast 显示拒绝原因，不再显示"下单成功"

---

### P0-5. 沙箱双防线（静态扫描 + 资源限制）

> **⚠️ 已过时：** Python 沙箱已按 ADR-0021 退役，此项不再适用。

**现状**:
- `sandbox_scan.py` 的 `scan_code()` 生产零调用（死代码）
- `backtest_sandbox.py:131-159` 子进程无 `resource.setrlimit`

**涉及文件**:
- `strategy-service/app/sandbox_scan.py`
- `strategy-service/app/engine/backtest_sandbox.py`

**最优解**（参考已正确实现的 `live_sandbox.py:164-170`）:
```python
# backtest_sandbox.py: _backtest_worker 和 _exec_in_child 中:
import resource
mem_bytes = 512 * 1024 * 1024  # 512MB per subprocess
resource.setrlimit(resource.RLIMIT_AS, (mem_bytes, mem_bytes))
resource.setrlimit(resource.RLIMIT_CPU, (30, 30))  # 30s CPU time
resource.setrlimit(resource.RLIMIT_NOFILE, (64, 64))
```
同时在 `compile_and_serialize` 前调用 `scan_code(code)`，violations 直接拒绝。

**落地步骤**:
1. `backtest_sandbox.py` 子进程 exec 前添加 `setrlimit`（对齐 `live_sandbox.py`）
2. `sandbox_scan.py` 集成到 `compile_and_serialize` 调用链
3. 或确认不需要静态扫描后删除 `sandbox_scan.py` 及误导注释

**验收**: 
- 提交 `while True: x = [0] * 10**8` 策略 → 进程被 OOM-kill 而非整个服务崩溃
- 提交 `import os` 策略 → 编译阶段即拒绝（如启用静态扫描）

---

### P0-6. 打通 Smart Tuning 数据管线（三重断裂）

**现状（三处断裂）**:
1. `runTuning()` 调用 `submit()` 后不回传 experimentId 给 SmartTuningPanel
2. `watchExperiment()` 返回 `{ stream, abort }` 对象，但 SmartTuningPanel 当 AsyncIterable 遍历
3. `AbortController` 创建后 signal 未传入

**涉及文件**:
- `frontend/src/pages/strategy/hooks/useBacktestParams.ts`
- `frontend/src/pages/strategy/components/workspace/SmartTuningPanel.tsx`
- `frontend/src/client/strategyExperiment.ts`

**最优解**:

**Step A — 接口签名变更（跨 3 个文件）**:

`useBacktestParams.ts` — `runTuning` 返回类型 `void` → `Promise<string>`:
```ts
const runTuning = useCallback(async (params: {...}): Promise<string> => {
  setTuningRunning(true);
  try {
    const result = await strategyExperimentApi.submit({...});
    message.success('Smart Tuning started');
    return result.experiment?.id || result.jobId || '';  // 返回 ID
  } catch (e: any) {
    message.error(e?.message || 'Tuning failed');
    return '';
  } finally { setTuningRunning(false); }
}, [...]);
```

`SmartTuningPanel.tsx` — `handleRunTuning` 接收返回值并设置 experimentId:
```ts
const handleRunTuning = useCallback(async () => {
  if (!canRun || tuningRunning) return;
  setCandidates([]); setExperimentId(''); setWatching(true);
  const eid = await onRunTuning();  // onRunTuning 现在返回 Promise<string>
  if (eid) setExperimentId(eid);
  else setWatching(false);
}, [canRun, tuningRunning, onRunTuning]);
```

`useStrategyWorkspaceState.ts` — `handleRunTuning` 透传返回值:
```ts
// 改前: btCtx.runTuning({...});  // void
// 改后:
const handleRunTuning = useCallback(async () => {
  btCtx.setSubTab('tuning');
  return btCtx.runTuning({
    code: codeCtx.code, symbol, timeframe,
    startDate: btCtx.startDate, endDate: btCtx.endDate,
  });
}, [btCtx, codeCtx.code, symbol, timeframe]);
```

**Step B — `SmartTuningPanel.tsx` 中修复 destructure**:
```ts
useEffect(() => {
  if (!watching || !experimentId) return;
  const ctrl = new AbortController();
  (async () => {
    try {
      const { stream } = strategyExperimentApi.watchExperiment(experimentId);
      for await (const event of stream) {  // 遍历 stream，不是对象本身
        if (event.status === 'COMPLETED') {
          setCandidates(event.candidates || []);
          setWatching(false); break;
        }
        if (event.status === 'FAILED') { setWatching(false); break; }
      }
    } catch (e) { if ((e as any)?.name !== 'AbortError') setWatching(false); }
  })();
  return () => { ctrl.abort(); };
}, [watching, experimentId]);

useEffect(() => {
  if (!watching || !experimentId) return;
  const ctrl = new AbortController();
  (async () => {
    try {
      const { stream } = strategyExperimentApi.watchExperiment(experimentId);
      for await (const event of stream) {  // 遍历 stream，不是对象
        if (event.status === 'COMPLETED') {
          setCandidates(event.candidates || []);
          setWatching(false); break;
        }
        if (event.status === 'FAILED') { setWatching(false); break; }
      }
    } catch (e) { if ((e as any)?.name !== 'AbortError') setWatching(false); }
  })();
  return () => { ctrl.abort(); };
}, [watching, experimentId]);
```

**Step C — `strategyExperiment.ts` 中 `watchExperiment` 的 AbortController 接入 signal**:
```ts
watchExperiment: (experimentId: string) => {
  const abortController = new AbortController();
  const stream = strategyExperimentClient.watchExperiment(
    { experimentId },
    { signal: abortController.signal },
  );
  return { stream, abort: () => abortController.abort() };
},
```

**Step D — 后端 worker 使用用户传入的 financial 参数，移除硬编码**:
`strategy_experiment_worker.go:backtestAndScore` 行 235-240 中 `InitialCapital(10000)` / `Commission(0.001)` / `Slippage(0)` / `Leverage(1)` / `TradeDirection("both")` / `StrictMode(true)` **全部硬编码**，完全无视用户在回测参数面板中的设置。

> **审计修正** (2026-06-06): 原审计声称 "symbol/timeframe 硬编码 XAUUSDm/1h"，代码验证显示 symbol/timeframe/dateRange 已从 `exp` 字段正确透传（行 199-223，空值时 fallback 向后兼容）。真正硬编码的是这 6 个金融参数。

修正：
1. Proto `SubmitStrategyExperimentRequest` 增加 `ExecutionConfig execution_config = 10;`（复用已有 `ExecutionConfig` message）
2. 前端 `runTuning` 提交时带 `executionConfig: { commission, slippage, leverage, tradeDirection, strictMode }` + `initialCapital`
3. 后端 worker `backtestAndScore` 从 `exp.ExecutionConfig` 读取，fallback 到当前硬编码值以兼容旧实验
4. Repository `StrategyExperiment` 结构体增加 `ExecutionConfig` 字段 + DB migration 增加对应列

**验收**: 
- 选 EURUSD/15m/自定义区间 → 发起 Smart Tuning → 候选回测在 EURUSD/15m 该区间执行
- SmartTuningPanel 显示实时候选结果
- 组件卸载后 SSE 连接关闭

---

### P0-7. ~~修复 hasReceivedData 门~~ ✅ 已修复

**审计验证**: `useStrategyWorkspaceState.ts:79` 已使用 `isSuccess: financialsReady`（TanStack Query），行 104 检查 `!financialsReady`。`useTradingStore` 已不再 import。此问题在审计时已不存在。**从方案中移除。**

> **审计日期**: 2026-06-06，验证通过。保留此项以供追溯。

---

### P0-8. 修复 Smart Tuning abort 操作（信号未传入）

**现状**: SmartTuningPanel 创建了 `AbortController` 但 signal 从未传给 `watchExperiment`，cleanup 中的 `ctrl.abort()` 是空操作。
**涉及文件**: `frontend/src/pages/strategy/components/workspace/SmartTuningPanel.tsx:58-71`

**最优解**: 已在 P0-6 Step C 中通过 `strategyExperiment.ts` 包装解决。`SmartTuningPanel` → `strategyExperimentApi.watchExperiment(id)` → 返回 `{ stream, abort }`，cleanup 调用 `abort()`。

**验收**: 调优运行中关闭 Smart Tuning tab → Network 面板 SSE 连接关闭

---

### P0-9. ~~修复 runTuning 无 catch 块~~ ✅ 已修复

**审计验证**: `useBacktestParams.ts:177-178` 已有 `catch (e: any) { message.error(e?.message || 'Tuning failed'); }`。此问题在审计时已不存在。**从方案中移除。**

> **审计日期**: 2026-06-06，验证通过。保留此项以供追溯。

---

### P0-10. ~~修复 SSE 连接卸载不清理~~ ✅ 已修复

**审计验证**: `useBacktestParams.ts:136-140` 已有 `useEffect(() => () => { gateStopRef.current?.(); backtestWatchRef.current?.(); }, [])`。此问题在审计时已不存在。**从方案中移除。**

> **审计日期**: 2026-06-06，验证通过。保留此项以供追溯。

---

## P1 — 架构合规与功能可用

---

### P1-1. 全管线 float64 → string/Decimal 渐进迁移

**现状**: 前端 `types/trading.ts` Position 接口全 `number` + 后端 `broker_types.go` SSE 类型全 `float64` — 全管线 23 处违规。
**策略**: 分 3 阶段渐进迁移（**先修源头，再修消费端**）。

> **审计修正** (2026-06-06): 原方案 Phase 顺序为 前端→后端→DB。精度丢失的根源在后端 Go `float64` → proto `double`。应先修源头（后端 SSE 类型），再修消费端（前端类型），最后修存储（DB column type）。避免过渡期类型不匹配。

**Phase 1 — 后端 SSE 类型层（源头修正）**:
| 文件 | 改动 |
|------|------|
| `mthub/broker_types.go:8-36` | `PositionSnapshot`/`PositionSnapshotItem`: 金融字段 `float64` → `string`（对齐 proto wire format） |
| `mthub/broker_types.go:83-96` | `BarUpdate`: OHLCV 字段 `float64` → `string` |
| `model/trade.go:57-59` | `KlineData`: `HighPrice`/`LowPrice` `float64` → `decimal.Decimal` |

**Phase 2 — DB/存储层**:
| 文件 | 改动 |
|------|------|
| `market_data_repo.go:26-38` | `KlineBar`: 用 `decimal.Decimal` 替代 `float64`，ClickHouse scan 改用 string 中转解析 |
| `backtest_run_repository.go:38-48` | 列类型 `float8` → `NUMERIC(20,8)`，Go 类型 `*float64` → `*decimal.Decimal` |

**Phase 3 — 前端类型层（消费端适配）**:
| 文件 | 改动 |
|------|------|
| `types/trading.ts:1-33` | `Position` 接口: `volume/openPrice/currentPrice/sl/tp/profit/commission/closePrice` 从 `number` → `string` |
| `useStrategyWorkspaceState.ts:19-27` | `QuickTradePosition`/`RecentTrade`: price/profit 从 `number` → `string` |
| `client/trading.ts:82-88` | `fromProtoOrders`: 同时支持 `string` 和 `number`（`typeof v === 'string' ? v : String(v)`），确保过渡期无 breakage |
| `client/market.ts:94-104` | OHLCV 价格字段: 同上兼容处理 |

**验收**: 
- Phase 1: SSE 推送的 Decimal 值以 string 形式完整到达前端
- Phase 2: DB 中存储的价格无浮点舍入误差
- Phase 3: 前端显示价格不乱码（`string` 直接展示或 `parseFloat` 格式化）

---

### P1-2. 客户端技术指标计算 — 文档化边界

**现状**: klinecharts v9.8 在浏览器中用 `calc` 函数计算 MA/EMA/MACD/RSI/BOLL/ATR 等指标用于图表显示。
**涉及文件**: `PriceChart.tsx:86-100`, `chartIndicatorsStore.ts:31-78`, `BidAskIndicator.ts:49-68`

> **审计修正** (2026-06-06): 原方案提议将指标计算全量迁移到后端（新增 RPC + SSE 流 + 后端计算）。经过"最优解"评估，判定为**过度工程化**：
> - 图表指标是纯视觉显示辅助，不影响任何交易决策（策略代码的指标在 Python 沙箱中独立计算）
> - 新增后端计算管线引入 SSE 延迟、网络依赖性，安全收益为零
> - 行业对标：TradingView、MT4/MT5 客户端均在本地计算图表指标
> - 真正的安全边界在策略执行层（Python 沙箱），不在图表渲染层

**修正后最优解**:
1. **文档化边界**: 在 `chartIndicatorsStore.ts` 顶部添加注释明确声明"图表指标仅用于视觉参考，非交易信号"
2. **代码隔离**: 添加 ESLint `no-restricted-imports` 规则，禁止策略/交易相关代码 import `chartIndicatorsStore` 或 klinecharts `calc` 函数
3. **架构文档**: 在零信任架构文档中明确：`klinecharts 指标渲染` = 展示侧例外（类比：浏览器渲染 HTML 不属于"业务逻辑"）

**落地步骤**:
1. `chartIndicatorsStore.ts` 添加注释头
2. `.eslintrc` 或 `eslint.config` 添加 restricted imports 规则
3. 更新 `platform-hard-rules.md` 零信任章节，增加展示侧例外说明

**验收**: 
- 策略代码无法 import 图表指标数据
- 架构文档明确标注展示侧边界
- 如未来需要后端预计算指标（如自定义指标实时推送），作为独立功能开发而非合规改造

**严重度**: P1 → **P3**（降级）

---

### P1-3. ~~@param 参数范围后端解析~~ 撤销 — 当前已是最优

**现状**: `backtestParamHelpers.ts:62-87` 在前端解析 `@param` 注解生成 UI 搜索空间。

> **审计判定** (2026-06-06): **撤销此项。** 当前方案已是最优：
> - 前端解析 → 即时 UI 响应（用户改代码→参数维度立即更新），无需网络往返
> - 后端 `strategy_experiment_worker.go:88` 的 `ai.ExtractParams(code)` 在提交时**重新解析并验证** — 这是真正的安全边界
> - 前端解析 + 后端重新验证 = 纵深防御，优于纯后端解析（牺牲 UI 响应性而无安全增益）
> - 可选增强（P3）：后端 `SubmitStrategyExperiment` handler 对比前端 `parameterSpace` 与后端 `ExtractParams` 结果，不一致时以服务端为准并 warn

**从方案中移除。** 保留后端重新验证作为既有安全边界。

---

### P1-4. 消除 REST 旁路端点

**现状**: 2 处 REST 端点绕过 ConnectRPC：
1. `handlers_sre.go:117-139` — 7 个 SRE REST 端点与已有 ConnectRPC 重复
2. `ai/api.ts:79` — `fetch('/api/ai/agents')` 绕过 ConnectRPC

**最优解**:
- SRE: 迁移到已有的 `AdminSREService` ConnectRPC handler，删除 REST 端点
- AI agents: 在 proto 中增加 `BatchSetAgents` RPC，前端改用 `aiClient`

**落地步骤**:
1. 确认 `AdminSREService` proto 覆盖所有 7 个 SRE 操作。如有缺失，补 RPC
2. 前端 SRE 调用迁移到 `sreClient`
3. 删除 `HandleFunc` 注册
4. AI: proto 新增 `rpc BatchSetAgents(BatchSetAgentsRequest) returns (BatchSetAgentsResponse)`
5. `ai/api.ts` 移除 `fetch`，改用 `aiClient.batchSetAgents(create(...))`

**验收**: `grep -r "HandleFunc\|mux.Handle" backend/cmd/server/` 只剩 healthz/metrics/auth 例外

---

### P1-5. 回测/实验 worker 用 PG NOTIFY 替代 time.Sleep 轮询

**现状**: 
- `backtest_worker.go:128-149`: 每 5s `time.Ticker` 轮询 DB 检查 `CANCEL_REQUESTED`
- `strategy_experiment_worker.go:253-254`: `time.Sleep(5s)` 循环最长 120 次轮询回测状态

**最优解**: 
- 取消: 在 `pglisten` 订阅 `backtest_cancel` 频道，`CancelBacktestRun` handler 发送 `pg_notify`
- 状态: worker 通过 channel 或 pglisten 订阅回测完成事件（已有 30s ticker 兜底模式可复用）

**落地步骤**:
1. `CancelBacktestRun` handler 中 `UPDATE` 后 `SELECT pg_notify('backtest_cancel', runID)`
2. `executeBacktestRun` 中取消轮询 goroutine 替换为 `pglisten.Notify` 接收
3. `backtestAndScore` 中 `time.Sleep` 轮询替换为 `select { case <-ctx.Done(); case <-time.After(5s); case <-notifCh: }`
4. 保留 30s ticker 兜底

**验收**: 取消回测 → worker 立即收到通知（不再等 5s）；回测完成 → experiment worker 即时收到（不再等 sleep 周期）

---

### P1-6. 增强 applyParamsToCode 健壮性（P3 可选）

**现状**: `SmartTuningPanel.tsx:37-49` 用正则替换 `@param` 值。行 44 `key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')` 已正确转义 regex 特殊字符，对预期输入格式（数字或 `range(...)`）覆盖良好。

> **审计修正** (2026-06-06): 当前正则实现已正确处理转义，对预期输入安全。降级为 P3 可选增强。

**可选增强**: 
- 添加 try-catch 包裹替换逻辑，失败时 fallback 到行级替换
- 行级替换作为备选方案（`lines[i] = lines[i].replace(/\bdefault=\S+/, \`default=${value}\`)`）

**严重度**: P1 → **P3**（降级为可选增强）

---

### P1-7. ~~修复 AIChatPanel streamText 字段名过期~~ ❌ 不存在此 bug

**审计验证**: 
- Proto 字段名是 `delta`（`code_assist.proto:32`: `string delta = 1;`）
- `streamText` 是 AIChatPanel 中的 **React 本地 state 变量名**，用于累加 delta 增量显示
- 行 70: `onDelta: (d) => { setStreamText(p => p + d); }` — 正确逻辑
- proto 字段 `delta` 和 React state `streamText` 是两个不同层级的命名，不存在"字段名过期"问题

**从方案中移除。** 此项为误报。

---

### P1-8. 后端 ctx 全链路传播

**现状**: `strategy_experiment_worker.go:160` `runOneShot` 使用 `context.Background()` — 服务器关闭/用户取消时无法停止。
**涉及文件**: `backend/internal/connect/strategy/strategy_experiment_worker.go`

**最优解**:
```go
// 改前: func (w *ExperimentWorker) runOneShot(overridesList []..., code string, exp *...) {
// 改后:
func (w *ExperimentWorker) runOneShot(ctx context.Context, overridesList []..., code string, exp *...) {
    for _, overrides := range overridesList {
        select {
        case <-ctx.Done():
            return results, ctx.Err()
        default:
        }
        r, err := w.backtestAndScore(ctx, code, overrides, exp)
        ...
    }
}
```

**落地步骤**: `runOneShot` 签名加 `ctx` → `runOptimizer` 传入 → `processOne` 传入（已持有 ctx）

**验收**: 用户取消实验 → ctx 传播到子回测循环 → 立即退出

---

### P1-9. 全链路 Trace Context 注入

**现状**: `backend/internal/connect/strategy/` 下所有 zap 日志无 `trace_id`/`user_id`/`request_id`。
**涉及文件**: 所有 strategy handler + worker 文件

**最优解**: 
1. 利用已有的 `interceptor.GetUserID(ctx)` 和 `interceptor.GetRequestID(ctx)`
2. 在每个 handler/worker 入口从 ctx 提取并注入 logger:
```go
logger := s.log.With(
    zap.String("user_id", interceptor.GetUserID(ctx)),
    zap.String("request_id", interceptor.GetRequestID(ctx)),
)
```

**落地步骤**:
1. handler 级别：在 `interceptor` 中统一注入 logger 到 ctx
2. worker 级别：在 `processOne`/`executeBacktestRun` 入口创建带 trace 的 logger
3. 所有后续 `log.Info/Warn/Error` 自动携带 trace context

**验收**: 日志中包含 `user_id` 和 `request_id` → 可从前端请求关联到后端日志

---

### P1-10. 修复 usePositionsQuery 条件轮询

**现状**: `usePositionsQuery.ts:21` `refetchInterval: 60_000` 是固定值，SSE 活跃时也轮询。
**涉及文件**: `frontend/src/queries/usePositionsQuery.ts`

**最优解**:
```ts
refetchInterval: (query) => {
  const lastUpdate = query.state.dataUpdatedAt;
  const now = Date.now();
  // SSE 活跃（90s 内有更新）则不轮询；超过 90s 无更新则 60s 兜底轮询
  return (now - lastUpdate) > 90_000 ? 60_000 : false;
},
```

**验收**: SSE 推送活跃时 Network 无周期性 `OpenedOrders` 请求；断流 >90s 后才出现

---

### P1-11. 添加策略子系统 Prometheus Metrics

**现状**: mdgateway 有 metrics，但 strategy handlers/workers 无任何 metrics。
**涉及文件**: `backend/internal/connect/strategy/`

**最优解**: 添加以下 counters + histograms：
```go
var (
    backtestRunsTotal = prometheus.NewCounterVec(..., []string{"status"})  // started/completed/failed/canceled
    experimentRunsTotal = prometheus.NewCounterVec(..., []string{"status"})
    sseConnectionsActive = prometheus.NewGauge(...)
    backtestDuration = prometheus.NewHistogram(...)
)
```

**验收**: `/metrics` 端点可观测回测/实验/SSE 连接数

---

### P1-12. 添加显式状态机

**现状**: Backtest/Experiment 状态用散落字符串常量，Proto 已有 `BacktestRunStatus` enum 但 Go 代码从不使用。
**涉及文件**: `backend/internal/connect/strategy/backtest_worker.go`, `strategy_experiment_worker.go`

**最优解**:
1. 定义 Go 状态机：
```go
type BacktestRunStateMachine struct { current antv1.BacktestRunStatus }
func (sm *BacktestRunStateMachine) CanTransition(to antv1.BacktestRunStatus) bool {
    valid := map[antv1.BacktestRunStatus][]antv1.BacktestRunStatus{
        antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_PENDING: {
            antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_RUNNING,
            antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCEL_REQUESTED,
        },
        // ...
    }
    for _, v := range valid[sm.current] { if v == to { return true } }
    return false
}
```
2. 所有状态变更前调用 `CanTransition`
3. 字符串 `"SUCCEEDED"` → `antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_SUCCEEDED.String()`

**验收**: 尝试非法状态转移（如 RUNNING→PENDING）→ 日志 warning + 拒绝

---

## P2 — 工程健壮性

---

### P2-1. 复杂度拆分

| 文件 | 函数 | 当前 | 目标 | 拆分方法 |
|------|------|------|------|---------|
| `backtest_worker.go` | `executeBacktestRun` | 181行 / CC 33 | ≤50行 / ≤10 | 提取 `fetchKlineData`, `buildBacktestRequest`, `callPythonEngine` |
| `python_strategy_backtest_crud.go` | `StartBacktestRun` | 92行 / CC 29 | ≤50行 / ≤10 | 提取 `validateAndBuildRun`, `insertAndNotify` |
| `useStrategyWorkspaceState.ts` | hook | 290行 | ≤50行 | 提取 `useQuickTradeData`, `useAutoFix` 独立 hook |
| `useStrategyWorkspaceState.ts` | `handleAutoFix` | 77行 | ≤50行 | 提取 `performAutoFixIteration` |
| `strategy_experiment_handler.go` | `WatchExperiment` | 64行 / CC 17 / depth 7 | ≤50 / ≤10 / ≤4 | 提取 `watchLoop`, `sendExperimentUpdate`，反转 early-return 减少嵌套 |
| `strategy_experiment_worker.go` | `backtestAndScore` | 74行 / CC 13 | ≤50行 / ≤10 | 提取 `waitForCompletion`, `parseResult` |
| `chat.go` | `tryChatCompletionStream` | 63行 / CC 18 | ≤50行 / ≤10 | 提取 `processSSEChunks`, `assembleResponse` |
| `PriceChart.tsx` | 组件 | 247行 / depth 5 | ≤50行 | 提取 `useChartIndicators`, `useChartInteraction` 自定义 hook |
| `BacktestParamsCard.tsx` | 组件 | 203行 / 31 props | ≤50行 / ≤5 | 用 `useBacktestParamsContext` 替代 31 个独立 props |
| `WorkspaceBacktestPanel.tsx` | 组件 | 191行 / 18 props | ≤50行 / ≤5 | 按 subTab 拆分为 `BacktestResultsTab`, `SmartTuningTab`, `GateTab` |

**涉及文件**: 见表
**验收**: `make check-lines-strict` 零 ERROR

---

### P2-2. Props 下钻治理

**现状**: `BacktestParamsCard`(31 props) + `WorkspaceBacktestPanel`(18 props) 深度下钻。
**最优解**: 使用 Zustand slice（扩展 `workspaceStore.ts`），不引入 React Context（Context 导致所有消费者重渲染）。项目已确立 Zustand persist 为状态管理标准。

**落地步骤**:
1. 创建 `workspaceStore` 的 `backtestConfig` 切片（持久化参数）
2. `BacktestParamsCard` 从 store 读写，props 降至 ≤5（仅回调函数）
3. `WorkspaceBacktestPanel` 同理

**验收**: 两个组件 props ≤5 个

---

### P2-3. 前端静默 catch 全部消除

**现状**: 16 处 `catch { /* silent */ }` 或 `catch {}`（空体）在工作区相关文件中。
**涉及文件**: 见合规审查 E14-E17

**最优解**: 
- UI 非关键路径（localStorage、AI model 探测）→ `console.warn('description', e)`
- 数据获取（`getSymbols`, `getKlines`, `fetchTradeHistory`）→ `message.error(...)` 或 `console.error(...)`
- Chart 生命周期 → `console.debug(...)` 保留

**验收**: `grep -r "catch {" frontend/src/pages/strategy/` 零空体

---

### P2-4. SSE watch 循环中 DB 错误传播

**现状**: `WatchExperiment`/`WatchBacktestRun`/`WatchSchedules` 三个 handler 中 DB 查询错误静默 `continue`，可能使 SSE 流静默卡死。
**涉及文件**:
- `strategy_experiment_handler.go:231-233`
- `python_strategy_backtest_crud.go:177`
- `strategy_schedules.go:177-178`

**最优解**: 区分临时错和永久错：
```go
if err != nil {
    if isRetryable(err) {
        s.log.Warn("transient DB error in watch loop", zap.Error(err))
        continue  // 重试
    }
    // 永久错：发送 error 事件给客户端
    s.sendError(stream, err)
    return
}
```

**验收**: 临时 DB 错误后恢复推送；永久错误触发客户端 error 回调

---

### P2-5. 收敛 `useStrategyWorkspaceState` god-hook

**现状**: 返回 ~60 个平铺字段，40+ 行 `bt*` 手工重映射。
**最优解**: 分组返回：
```ts
return {
  account: { activeAccounts, accountId, handleAccountChange, ... },
  code: codeCtx,
  backtest: { submitting: btCtx.submitting, metrics: btCtx.metrics, ... },
  tuning: { tuneMethod: btCtx.tuneMethod, sweepDimensions: btCtx.sweepDimensions, ... },
  gate: { loading: btCtx.gateLoading, gates: btCtx.gateGates, ... },
  quickTrade: { positions, recentTrades, handleClosePosition },
  layout: { codePanelVisible, quickTradeVisible, ... },
  history: { drawerOpen, runId, handleOpenHistory, handleCloseHistory },
};
```
页面用 `ws.backtest.metrics` 代替代替 `ws.btMetrics`。

**涉及文件**: `useStrategyWorkspaceState.ts`, `StrategyWorkspacePage.tsx`
**验收**: hook 返回 ≤8 个分组字段（每组是独立对象）；删除全部 40+ 行重映射

---

### P2-6. 移除 `positionsPanelVisible` 不一致

**现状**: `codePanelVisible`/`quickTradeVisible` 通过 workspaceStore 持久化，但 `positionsPanelVisible` 是本地 useState（刷新后丢失）。
**涉及文件**: `useStrategyWorkspaceState.ts`, `workspaceStore.ts`

**最优解**: 将 `positionsPanelVisible` 加入 workspaceStore 持久化。

**验收**: 打开持仓面板 → 刷新页面 → 面板仍然可见

---

### P2-7. 价格图表 trades overlay 使用 useMemo

**现状**: `StrategyWorkspacePage.tsx` 中 `btMetrics?.trades?.map(...)` 内联创建，每次渲染新数组。
**涉及文件**: `StrategyWorkspacePage.tsx`

**最优解**:
```ts
const chartTrades = useMemo(() => 
  ws.btMetrics?.trades?.map((t: any) => ({
    side: t.side, openPrice: t.price, openTime: t.time, pnl: t.pnl,
  })), [ws.btMetrics?.trades]
);
```
传给 `<PriceChart trades={chartTrades} />`

**验收**: React DevTools Profiler 中 PriceChart 不因 trades 引用变化重渲染

---

### P2-8. 添加 ErrorBoundary

**现状**: workspace 无 `<ErrorBoundary>`，CodeMirror/klinecharts 崩溃会级联到整个页面。
**涉及文件**: `StrategyWorkspacePage.tsx`

**最优解**: 在 Code Panel 和 Chart 区域各包裹 ErrorBoundary：
```tsx
<ErrorBoundary fallback={<ChartErrorFallback />}>
  <PriceChart ... />
</ErrorBoundary>
```

**验收**: `throw new Error()` 从 Chart/Editor 抛出 → 显示 fallback UI，页面其余部分正常

---

### P2-9. 回测结果面板添加进度条

**现状**: 回测运行中只显示 "Running..." Tag，无百分比/步骤进度。
**涉及文件**: `WorkspaceBacktestPanel.tsx`

**最优解**: 利用 `watchBacktestRun` 回调中的 `update.progress` 字段（如 proto 有定义）或状态变化展示步骤进度：
```
[████████░░] 80% — Processing bar 240/300
```

**验收**: 回测中看到逐步更新的进度指示

---

### P2-10. Gate 评估支持选择历史回测

**现状**: Gate 固定评估最近一次 `backtestRunId`，无法对历史回测跑 Gate。
**涉及文件**: `GatePanel.tsx`, `useBacktestParams.ts`

**最优解**: Gate Panel 增加 runId 选择器，从 `listBacktestRuns` 获取已完成回测列表。

**验收**: Gate 下拉可选择任意已完成的回测跑评估

---

### P2-11. 图表指标与策略代码联动

**现状**: 用户在图表上叠加 RSI(14)，但策略代码里用 RSI(7) — 两者完全独立无同步。
**涉及文件**: `chartIndicatorsStore.ts`, `PriceChart.tsx`

**最优解**: 
1. 策略代码验证返回使用的指标列表（从 `@param` 或代码分析提取）
2. 图表侧显示提示："Strategy uses RSI(7), chart shows RSI(14)"
3. 一键同步：点击按钮将图表指标参数同步为策略参数

**验收**: 策略代码中的指标参数与图表叠加指标不一致时显示 warning badge

---

### P2-12. 模板支持保存/恢复回测参数

**现状**: 模板只存代码，不含回测参数。加载模板后参数不还原。
**涉及文件**: `strategy.proto`, `WorkspaceTemplateManager.tsx`

**最优解**: `StrategyTemplate` proto 增加 `BacktestConfig` 字段。保存模板时附带当前参数，加载时还原。

**验收**: 保存模板（含 commission=0.002）→ 切换到默认 → 加载模板 → commission 变回 0.002

---

### P2-13. 后端 Worker Start 使用 server context

**现状**: `handlers_sre.go:86,92` 两个 Worker 的 `Start()` 传 `context.Background()`，`Stop()` 存在但从未被调用。
**涉及文件**: `backend/cmd/server/handlers_sre.go`

**最优解**: 传入 `serverShutdownCtx`（从 main 传入），在 shutdown hook 中调用 `Stop()`。

**验收**: 服务关闭 → worker 优雅退出，无 goroutine 泄漏

---

## P3 — 优化与体验

---

### P3-1. Python 自动补全

**现状**: CodeMirror 未集成 `autocompletion()` 扩展。
**涉及文件**: `StrategyCodeEditor.tsx`

**最优解**: 集成 `@codemirror/autocomplete`，提供 Python 关键字 + `@param`/`@strategy` 注解补全。

**验收**: 输入 `@par` → 自动补全 `@param name type=... default=...`

---

### P3-2. AI Generate 功能

**现状**: AI Chat 只能 revise/explain/validate 已有代码，不能从自然语言生成。
**涉及文件**: `AIChatPanel.tsx`, `strategyGen.ts`, 后端 handler

**最优解**: 
1. 检测用户输入为自然语言描述（无策略代码时自动进入 generate 模式）
2. 调用 `strategyGenClient.generateStrategy(...)` SSE 流
3. 生成后自动填充到编辑器 + 触发验证

**验收**: 输入"创建一个基于 RSI 和 MACD 金叉的日内交易策略"→ AI 生成完整代码 → 自动填充编辑区

---

### P3-3. 图表指标用 Ant Design Checkbox 替换原生 input

**现状**: `SmartTuningPanel.tsx:132` 用原生 `<input type="checkbox">`。
**涉及文件**: `SmartTuningPanel.tsx`

**最优解**: 替换为 `<Checkbox>`（已在文件中 import `Table` 从 antd）。

**验收**: 复选框样式与页面其余部分一致

---

### P3-4. 代码编辑器光标位置保持

**现状**: `StrategyCodeEditor.tsx:100-109` value 同步时完全替换文档，光标重置到位置 0。
**涉及文件**: `StrategyCodeEditor.tsx`

**最优解**: 使用 CodeMirror `dispatch` 而非 `setValue` 做增量更新，或在替换前保存/恢复光标位置。

**验收**: 外部修改代码（如 AI 返回结果应用）→ 光标保持在原位

---

### P3-5. 回测运行中禁用账户切换

**现状**: 回测运行中可以切换账户，导致运行中数据与新账户不匹配。
**涉及文件**: `WorkspaceToolbar.tsx`

**最优解**: 回测/调优运行中 disable 账户选择器：
```tsx
<Select disabled={btSubmitting || tuningRunning} ... />
```

**验收**: 回测运行中账户下拉变灰不可选

---

### P3-6. localStorage 保存的参数默认值添加服务端来源

**现状**: `BacktestParamsCard.tsx` 用 localStorage 存储用户偏好默认值，无后端规范来源。
**涉及文件**: `BacktestParamsCard.tsx`

**最优解**: 
1. 后端增加 `GET /api/user/preferences`（或 proto `UserService.GetPreferences`）
2. 前端优先取后端 preferences → fallback localStorage → fallback 硬编码默认值

**验收**: 换设备登录 → 回测参数默认值一致

---

### P3-7. 前端 `strategyExperiment.submit` 使用 `create()`

**现状**: `strategyExperiment.ts:20-33` 传 plain object 而非 `create(Schema, {...})`。
**涉及文件**: `frontend/src/client/strategyExperiment.ts`

**最优解**:
```ts
import { create } from '@bufbuild/protobuf';
import { SubmitStrategyExperimentRequestSchema } from '@/gen/ant/v1/strategy_experiment_pb';

submit: (params: SubmitStrategyExperimentParams) =>
  strategyExperimentClient.submitStrategyExperiment(
    create(SubmitStrategyExperimentRequestSchema, { ... })
  ),
```

**验收**: 所有 API client 调用使用 `create()` 构造 proto message

---

## 实施顺序建议

```
Week 1 (P0):  P0-1 → P0-2 → P0-3 → P0-4 → P0-5
              (proto.Marshal + error discards + MarketRegime + orderSend + sandbox)

Week 2 (P0):  P0-6 → P0-7 → P0-8 → P0-9 → P0-10
              (Smart Tuning 管线 + hasReceivedData + abort + catch + SSE cleanup)

Week 3 (P1):  P1-5 → P1-8 → P1-9 → P1-11 → P1-12
              (Push-First + ctx 传播 + trace context + metrics + state machine)

Week 4 (P1):  P1-1 Phase 1 → P1-3 → P1-6 → P1-7 → P1-10
              (前端 float64 → string + @param 后端化 + regex 修复 + streamText + 条件轮询)

Week 5 (P1):  P1-2 → P1-4
              (指标后端化 + REST 收敛)

Week 6-7 (P2): P2-1 → P2-2 → P2-3 → P2-4 → P2-5 → ... → P2-13
               (复杂度拆分 + props 治理 + catch 消除 + SSE 错误传播 + god-hook 收敛等)

Week 8 (P3):  P3-1 → P3-2 → P3-3 → P3-4 → P3-5 → P3-6 → P3-7
              (自动补全 + AI Generate + UI 细节 + 光标保持等)

P1-1 Phase 2/3 (float64 后端迁移) 可并行进行，与 P2 不冲突
```

---

## 与现有文档的关系

| 文档 | 内容 | 本方案关系 |
|------|------|-----------|
| `workspace-pipeline-audit-2026-06-05.md` | 168 条缺陷发现 | 本方案覆盖 42 条需修复项（去重合并后） |
| `workspace-remediation-plan-2026-06-05.md` | 10 项架构整改（R1-R10） | 本方案集成并扩展 R1-R10，增加 32 项 |
| `strategy-workspace-v31-implementation.md` | v31 功能对齐 5 阶段 | 本方案 P1-2/P3-2 覆盖其 P3/P4 阶段 |

**冲突处理**: 本方案以最新代码状态为准。如与旧方案有差异，以本方案为准。

---

## 附录：修复方案最优解审计（2026-06-06）

对上述 42 项修复方案逐项审计。每项判定：✅ 最优 / ⚠️ 需调整 / ❌ 已修复无需修 / 🔄 降级重定范围。

### 审计方法

1. 重新阅读每项涉及的实际源码（非依赖审计文档的二手描述）
2. 对照项目 `platform-hard-rules` 的"最优解"标准
3. 判断方案是否在所有可行方案中最优（不是"够好"，是"最优"）

---

### 一、已修复项 — 应从方案中移除

以下问题经代码验证 **已在当前代码中修复**：

| 原编号 | 问题 | 证据 |
|:------:|------|------|
| **P0-7** | hasReceivedData 门恒为 false | `useStrategyWorkspaceState.ts:79` 已用 `isSuccess: financialsReady`，`useTradingStore` 已不再 import。行 104 检查 `!financialsReady` 而非 `!tradingStore.hasReceivedData()` |
| **P0-9** | runTuning 无 catch 块 | `useBacktestParams.ts:177-178` 已有 `catch (e: any) { message.error(e?.message \|\| 'Tuning failed'); }` |
| **P0-10** | SSE 连接卸载不清理 | `useBacktestParams.ts:136-140` 已有 `useEffect(() => () => { gateStopRef.current?.(); backtestWatchRef.current?.(); }, [])` |
| **P1-7** | AIChatPanel streamText 字段名过期 | **不存在此 bug**。Proto 字段名是 `delta`（`code_assist.proto:32`），前端本地 state 变量名 `streamText` 是 React state 名，不是 proto 字段名。行 70 `onDelta: (d) => { setStreamText(p => p + d) }` — 正确累加 delta。proto 字段 `delta` 和 React state `streamText` 是两个不同的东西 |

**结论**: 移除 P0-7、P0-9、P0-10、P1-7 四项。P0 从 10 项减为 7 项，P1 从 12 项减为 11 项。总计从 42 项减为 **38 项**。

---

### 二、严重度过高项 — 降级处理

#### P0-6 Step D: Smart Tuning "后端硬编码 XAUUSDm/1h" — 部分误报

**审计发现**: 经代码验证，`strategy_experiment_worker.go:193-242` 中：
- `exp.Symbol` / `exp.Timeframe` / `exp.FromTsUnixMs` / `exp.ToTsUnixMs` **已被正确读取**（行 199-223），仅空值时 fallback 到 `XAUUSDm/1h/近一月`（向后兼容旧实验）
- `strategyCode` / `symbol` / `timeframe` / `fromTsUnixMs` / `toTsUnixMs` **已从前端传入**（`useBacktestParams.ts:170-174`）
- ❌ **真正的问题**: `InitialCapital`(10000) / `Commission`(0.001) / `Slippage`(0) / `Leverage`(1) / `TradeDirection`("both") / `StrictMode`(true) — 这 6 个金融参数**完全硬编码**（行 235-240），完全无视用户在回测参数面板中的设置

**修正方案**: P0-6 Step D 的范围从"传 symbol/timeframe/dateRange"改为"传 executionConfig"：
1. Proto `SubmitStrategyExperimentRequest` 增加 `ExecutionConfig execution_config` 字段（已有 `ExecutionConfig` message 定义）
2. 前端 `runTuning` 提交时带 `executionConfig: { commission, slippage, leverage, tradeDirection, strictMode }`（与 `runBacktest` 同源）
3. 后端 worker 从 `exp.ExecutionConfig` 读取（fallback 到当前硬编码值以兼容旧实验）
4. 还需增加 `initialCapital` 到 ExecutionConfig（或实验单独字段）

**严重度**: P0（原判）→ 维持 P0（具体修复内容修正）

---

#### P1-2: klinecharts 客户端指标后端化 — 过度工程化

**审计发现**: 
- 违规确实存在：klinecharts 在浏览器中计算 MA/EMA/MACD/RSI
- 但：这些指标是**纯图表显示**，不影响任何交易决策。策略代码的指标计算在 Python 沙箱中独立运行
- 提议方案（新增 RPC + SSE 流 + 后端计算 + 前端纯渲染）工程量巨大，新增 latency 依赖，而**安全收益为零**
- 对比：TradingView、MT4/MT5 客户端均在本地计算图表指标 — 这是行业标准做法

**最优解修正**: 
- 不迁移到后端计算。改为**文档化边界**：
  1. 在 `chartIndicatorsStore.ts` 顶部添加注释：`// 图表指标仅用于视觉参考，非交易信号。策略交易逻辑在后端 Python 沙箱中独立运行。`
  2. 禁止任何交易逻辑读取图表指标数据（添加 ESLint `no-restricted-imports` 规则）
  3. 在架构文档中明确：`klinecharts 指标渲染层` = 零信任边界的**展示侧例外**
- 如果未来有后端预计算需求（如自定义指标的实时推送），再新增 RPC，但那应是独立功能而非合规改造

**严重度**: P1（原判）→ **P3**（降级为架构文档化 + lint 规则）

---

#### P1-3: @param 后端解析 — 过度工程化

**审计发现**:
- `parseParamsFromCode` 在前端解析 Python 注释中的 `@param` 注解，生成参数搜索空间
- 提议方案：移到后端 `validateExtended` 返回
- 但：前端解析是**即时的 UI 响应**（用户改代码→参数维度立即更新），后端化会增加网络延迟，降低交互体验
- 后端在 `strategy_experiment_worker.go:88` 已用 `ai.ExtractParams(code)` **重新解析并验证** — 这是真正的安全边界
- 前端解析 + 后端重新验证 = 纵深防御，已经是最优模式

**最优解修正**:
- 保持前端解析（UI 响应），但增加注释说明"后端在 submit 时重新验证"
- 可选增强：后端 `SubmitStrategyExperiment` handler 中对比前端传的 `parameterSpace` 和后端 `ExtractParams` 结果，不一致时以服务端为准并 warn

**严重度**: P1（原判）→ **撤销**（当前方案已是最优）

---

#### P1-6: applyParamsToCode 正则脆弱 — 当前实现已处理

**审计发现**:
- 当前代码行 44 `key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')` **已正确转义** regex 特殊字符
- 正则 `(@param\\s+${escaped}\\s+)([\\d.-]+|range\\([^)]+\\))` 对预期输入格式（数字或 `range(...)`）覆盖良好
- 唯一残余风险：如果 `@param` 的值格式是 `range(0.01, 0.10, 0.01)` 而用户想替换为 `0.05`（单个数字替换 range），正则会正确匹配 `range(...)` 部分

**最优解修正**:
- 当前正则实现已达到实际使用场景的安全级别
- 可选增强（P3）：加 try-catch 包裹替换逻辑，失败时 fallback 到行级替换
- 不移除正则方案，因为当前已正确处理

**严重度**: P1（原判）→ **P3**（降级为可选增强）

---

### 三、方案需调整项

#### P0-6 Step A+B: experimentId 回传 + destructure 修复

**审计发现**: 
- Step A（`runTuning` 返回 experimentId）：`strategyExperimentApi.submit()` 返回 `SubmitStrategyExperimentResponse`，其中有 `experiment` 和 `jobId` 字段。但 `submit()` 是 `await` 的，返回的是 response，可以直接取 ID。✅ 方案正确
- Step B（destructure）：`watchExperiment` 返回 `{ stream, abort }`，SmartTuningPanel 需要 `for await (const event of stream)`。✅ 方案正确
- **但**: 当前 `handleRunTuning` 调用 `onRunTuning()` — 这是一个 `void` 函数。需要改签名为返回 `Promise<string>`。还需修改 `useStrategyWorkspaceState.ts` 中的 `handleRunTuning`（行 67-76），它调用 `btCtx.runTuning(...)` 但没有处理返回值。
- 另外：`useBacktestParams.ts` 的 `runTuning` 函数没有返回 experimentId，需要改 `await strategyExperimentApi.submit(...)` → `const res = await ...; return res.experiment?.id \|\| res.jobId \|\| ''`

**修正**: 在原方案基础上增加明确的接口签名变更：
```
useBacktestParams.ts runTuning: 返回类型 void → Promise<string>
SmartTuningPanel handleRunTuning: 接收返回值 setExperimentId(eid)
useStrategyWorkspaceState handleRunTuning: 改为 const eid = await btCtx.runTuning(...)
```

**判定**: ⚠️ 调整（接口签名变更需明确标注）

---

#### P1-1: float64 → string/Decimal 渐进迁移

**审计发现**: 三阶段方案架构上正确，但**实施顺序错误**：
- 原方案: Phase 1（前端类型）→ Phase 2（后端类型）→ Phase 3（DB 存储）
- **最优顺序**: Phase 2（后端 SSE 类型 string 化）→ Phase 3（DB NUMERIC 化）→ Phase 1（前端移除 Number()）
- 理由: 精度丢失的**根源**在后端 Go `float64` → proto `double`。如果后端先修（SSE 推送 string 而非 float64），前端可以逐步移除 `Number()` 包装，不会出现类型不匹配。如果前端先改（期望 string 但收到 double），会出现过渡期混乱

**修正**: 
1. Phase 1: `mthub/broker_types.go` SSE 类型 `float64` → `string`（源头修正）
2. Phase 2: `backtest_run_repository.go` + DB migration `float8` → `NUMERIC(20,8)`
3. Phase 3: 前端 `types/trading.ts` 移除 `Number()` + `Position` 接口 `number` → `string`
4. Phase 1 和 Phase 3 之间：前端 `fromProtoOrders` 同时支持 `string` 和 `number`（`typeof v === 'string' ? v : String(v)`），确保过渡期无 breakage

**判定**: ⚠️ 调整（Phase 顺序反转）

---

#### P2-2: Props 下钻治理

**审计发现**: 原方案说"React Context 或 Zustand 切片"。Context 会导致所有消费者重渲染。项目中 `workspaceStore` 已是 Zustand persist 模式，应统一使用 Zustand。

**修正**: 明确使用 Zustand slice（扩展 `workspaceStore.ts`），不引入 Context

**判定**: ⚠️ 调整（技术选型明确化为 Zustand）

---

### 四、方案确认最优项（无需修改）

以下 19 项方案经验证为最优解，无需调整：

| 编号 | 项 | 判定 |
|:----:|----|:----:|
| P0-1 | proto.Marshal 错误处理 | ✅ 对齐 Go best practice |
| P0-2 | 状态更新错误处理 | ✅ 对齐 worker 重试模式 |
| P0-3 | MarketRegime UserID | ✅ 对齐所有 handler 的 GetUserID 模式 |
| P0-4 | orderSend 使用真实响应 | ✅ 对齐同文件 orderClose 模式 |
| P0-5 | 沙箱双防线 | ✅ 对齐 live_sandbox.py rlimit 模式 |
| P0-6 A-C | experimentId + destructure + abort | ✅ 对齐 gate.ts AbortController 模式（见上方接口签名调整） |
| P0-8 | Smart Tuning abort | ✅ 已纳入 P0-6 Step C |
| P1-4 | REST 旁路消除 | ✅ 对齐 ConnectRPC 协议要求 |
| P1-5 | time.Sleep → PG NOTIFY | ✅ 对齐 pglisten 已有模式 |
| P1-8 | ctx 全链路传播 | ✅ 对齐 Go context 标准模式 |
| P1-9 | trace context 注入 | ✅ 对齐 zap 结构化日志标准 |
| P1-10 | usePositionsQuery 条件轮询 | ✅ 对齐 push-first 架构 |
| P1-11 | Prometheus metrics | ✅ 对齐 mdgateway 已有 metrics 模式 |
| P1-12 | 显式状态机 | ✅ 对齐 proto enum 定义 |
| P2-1 | 复杂度拆分 | ✅ 对齐项目文件/函数限制 |
| P2-5 | god-hook 收敛 | ✅ 对齐项目 hook 粒度约定 |
| P2-8 | ErrorBoundary | ✅ 对齐 React 错误边界最佳实践 |
| P2-13 | Worker Start 使用 server ctx | ✅ 对齐 Go server 优雅关闭模式 |
| P3 全部 7 项 | UI/UX 优化 | ✅ 对齐各自领域最佳实践 |

---

### 五、缺失项 — 原方案未覆盖但应加入

审计过程中发现以下问题未被任何已有文档覆盖：

#### M1. `strategy_experiment_worker.go` `runOneShot` 使用 `context.Background()` 

**现状**: 行 157-168，`runOneShot` 不接受 ctx 参数。虽然已在 P1-8 中提及，但具体修复方法不够明确。需确认 `runOneShot` 签名加 `ctx` → `runOptimizer` switch case 传入 → `processOne` 传入。

**纳入**: 已在 P1-8 范围内，确认方案正确。

---

#### M2. 回测 worker 取消轮询中 `backtestAndScore` 的 `time.Sleep` 阻塞

**现状**: `strategy_experiment_worker.go:254` `time.Sleep(5 * time.Second)` 在 for 循环内（最长 120 次迭代 = 10 分钟）。即使 ctx 取消，也需要等到下一个 sleep 结束才能检测到。

**方案**: 
```go
select {
case <-ctx.Done():
    return candidateResult{}, ctx.Err()
case <-time.After(5 * time.Second):
}
```
已在 P1-5 范围内，确认覆盖。

---

#### M3. Smart Tuning 调优实验中 `Commission`/`Slippage` 硬编码 `f64Ptr` 使用 float64

**现状**: `strategy_experiment_worker.go:236-237` `f64Ptr(0.001)` / `f64Ptr(0)` — 当 P1-1 迁移 `BacktestRun` 的 commission/slippage 到 Decimal 时，这些硬编码也必须同步迁移。

**纳入**: 在 P0-6 Step D（修正后）中覆盖 — executionConfig 改为 Decimal 相关类型。

---

### 六、审计总结

| 审计结论 | 数量 | 涉及项 |
|:--------:|:----:|------|
| ✅ 最优（无需修改） | 19 | P0-1~P0-5, P0-8, P1-4~P1-12, P2-1, P2-5, P2-8, P2-13, P3 全部 |
| ⚠️ 需调整 | 4 | P0-6（接口签名）, P1-1（Phase 顺序）, P2-2（Zustand 明确）, P0-6 Step D（修正范围） |
| 🔄 降级/撤销 | 3 | P1-2（P1→P3）, P1-3（撤销）, P1-6（P1→P3） |
| ❌ 已修复移除 | 4 | P0-7, P0-9, P0-10, P1-7 |

**净有效修复项**: 42 → 移除 4 已修复 + 撤销 1（P1-3）+ P0-8 并入 P0-6 = **36 项需执行**（P0: 6, P1: 8, P2: 13, P3: 9）

> 注：P1-2 和 P1-6 降级至 P3 但仍需执行（方案已修正）

**总体判定**: 方案整体质量高。19/27（70%）的非撤销项方案无需修改即为最优解。主要修正：
1. 4 项依赖审计文档二手信息，未验证当前代码实际状态（已修复项仍列入方案）→ 已标记移除
2. P1-2（klinecharts 指标后端化）过度工程化 → 改为文档化边界 + lint 规则
3. P1-3（@param 后端解析）当前已是最优 → 撤销
4. P1-1 Phase 顺序反转（先修源头后端，再修消费端前端）
5. P0-6 Step D 修正范围（symbol/timeframe 已透传，需修的是 financial 参数硬编码）

**总体判定**: 方案整体质量高，19/27（70%）的非撤销项方案无需修改即为最优解。主要问题：
1. 4 项依赖审计文档二手信息，未验证当前代码实际情况（导致已修复项仍列入方案）
2. 2 项（P1-2/P1-3）过度工程化，为实现合规而合规，忽略了实际安全边界和用户体验
3. P1-1 Phase 顺序不当，应先修源头（后端）再修消费端（前端）
