# FILL-SIM 施工提示词（审计方 Claude Code → 施工方 Windsurf）

> **身份**：本提示词由**审计方 Claude Code** 撰写（只读验证 + 出 spec + 验收，不直接改代码）。
> **施工方（Windsurf）**：读本文件开工。实现修复 + 回填进度，不重新审计、不扩大范围（one task = one scope）、不自由发挥。
> **依据**：`docs/audits/spec-fill-rule-limit-simulation-mode.md`（先完整读：§1 问题 + §2 设计 + §2.6 前置 + §3 最优性 + §4 清单 + §6 风险 + §7/§8 修订汇总）。**spec 为唯一事实源**，本提示词只重申强制点。
> **状态**：审计方已复核定稿 + 用户评审 3 点已闭环（§2.2 防御性 / §6.4 范围确认 / §2.6 item 4 自包含，commit `b1328ab`）。

---

## 任务

实现 `fill_rule=limit`（market order 转 pending order）与 `simulation_mode=OHLC_PATH`（bar 内 OHLC 路径 SL/TP 顺序判定）。

**背景**：这两个值当前在 `validateBacktestRequest` 被 API 拒绝（EXEC-PARAMS 诚实性闸）。本任务把它们变成真实能力：

- `fill_rule=limit` = market order **转 pending order**（回测专属近似旋钮，实盘 market order 永不转 pending——`runner/broker.go` 不改）
- `simulation_mode` wire 值 `DATASET` **直接改名 `OHLC_PATH`**（string 字段无 wire 兼容约束 + 无旧 snapshot 落盘，零成本；`DATASET` 字符串全库移除）

**⚠️ 必备前置（§2.6，Phase A 先做）**：VM pending order 可见性——审计实测 `builtinOrdersTotal`/`OrderSelect` 只查 Positions、`PositionModify` 只搜 positions（对 pending 静默 RetRejected）、`builtinOrderType` 按 Side 报型。不先修，market→pending 会让**所有 market order EA 的持仓管理静默错乱**（违反 MQL 诚实性红线）。此修复同时治愈原生 OP_BUYLIMIT EA 今日既有的同类静默错（独立价值）。

**范围（one task = one scope）**：仅 FILL-SIM 一个任务，内部按 Phase A→E 顺序执行（每 Phase 独立测试 + 对抗证明，全部完成才交付）。
❌ 不扩大：不拉 M1/tick 数据、不重建 tick 持久化（ADR-0012）、不动实盘下单路径（`runner/broker.go` 不改）、不顺手重构其他代码。

**【REUSE preflight 必做】** 动工前 `bash scripts/cap.sh` 换词查：checkSLTP / pending / OrderModify / ordersTotal / whitelist。已有能力直接复用（`checkPendingOrders` 触发逻辑、`PositionModify` 结构、`exec_params_validation_test.go` 现有测试模式）。PR 描述逐条给 `REUSE:`/`NEW:`。

---

## Phase A【§2.6 必备前置，先做】VM pending 可见性

**文件**：`backend/tools/mql2go/vm_builtin_trade.go` + `backend/strategy/backtest/broker.go`

1. `builtinOrdersTotal`：`len(Positions(0)) + len(Orders(0))`（MQL4 `OrdersTotal` 含 market + pending）
   ⚠️ **红队**：live 路径 `brokerImpl.Positions` 来自 `executor.OpenedOrders`——先核验 adapter（`mdgateway/adapter/mt5/order_history.go` 等）的 `OpenedOrders` 是否已含 pending；若含则 Positions+Orders 双计。实测后二选一：a) 确认 disjoint（多数 MT5 语义 OpenedOrders=positions 仅）；b) 若 MT4 侧含，在测试中锁定 backtest（SimBroker positions/pending 天然 disjoint，`broker.go:135-141` 实证）+ live 侧记录已知边界不双计（写回 spec 或注释，**不静默**）。
2. `builtinOrderSelect` MODE_TRADES：positions 索引段之后追加 pending 段（SELECT_BY_POS 与 SELECT_BY_TICKET 均覆盖；MQL4 order 池按位置索引含 pending）
3. `builtinOrderType`：pending 订单按 `OrderType` 返回 `OP_BUYLIMIT=2`/`OP_SELLLIMIT=3`/`OP_BUYSTOP=4`/`OP_SELLSTOP=5`，**非按 Side**（sideToOrderType 只在 currentPos 为 position 时用）
4. `SimBroker.PositionModify`：追加 pending 扫描段（SL/TP 可改）；`builtinOrderModify` 接住 `args[1]`（price）——MQL4 `OrderModify(ticket, price, sl, tp)` 的 price 仅对 pending 有意义（= 挂单价），对已开仓位该参数 MT4 原生忽略；回测侧映射为 `pending.Price`（§2.6 item 4 已自包含，**与 §6.4 commission/margin 无关**）
5. 缓存一致性：pending 快照与 `cachedPositions` 同生命周期（事件内多次调用一致）

**测试**（mql2go 包 + backtest 包）：EA 下 OP_BUYLIMIT → `OrdersTotal()==1`；`OrderSelect(0, MODE_TRADES)` 成功且 `OrderType()==2`；OrderModify 对 pending 成功（SL 变更后 `OrderStopLoss()` 反映）；pending 成交后 `OrdersTotal()` 回落；OrderDelete 对 pending 不回归（已支持）。

**对抗证明**（缺 = Phase A 判失败）：删 `builtinOrdersTotal` 的 `+len(Orders)` 行 → OrdersTotal 测试红；删 `PositionModify` pending 段 → OrderModify 测试红。

---

## Phase B【§2.1 + §6.4】fill_rule=limit

**文件**：`backend/strategy/backtest/broker.go`（+ `engine.go` fill 分支）

1. `OrderSend`：`config.FillRule=="limit" && req.Type==OrderMarket` → 转 OrderLimit。**顺序强制**：price=0→currentPrice 解析必须发生在转换后仍生效（spec §2.1 注：先解析价再转换，防挂单 Price=0 永不触发）。保留 SL/TP/comment/magic 原样。
2. **§6.4 决策①（默认执行，审计已批准）**：commission 从 OrderSend 移至 `checkPendingOrders` fill 分支（成交时刻扣）+ 同点 margin 复检（不足 → 撤单记 `RetNoMoney` 并 log，不 append 进 positions）。
   📌 **范围已确认（2026-08-10 用户评审）**：适用于**全部 pending**（含原生 OP_BUYLIMIT），非仅 limit 转换单——实盘语义对齐（真实 MT4 挂单不成仓不收 commission、成仓时才扣；margin 同理成交时占位）。KLINE_RANGE 下原生挂单 EA"永不成交也扣 commission"是既有失真 → **修正非回归**。审计已核实无现有测试锁定"下单即扣"。
   ⚠️ 红队：若实测发现改造成本不可控，可退回②（保持现状+文档声明），但必须在回填中说明理由，**不能悄悄选**。
3. 不改 `checkPendingOrders` 触发逻辑本身（范围检查保持，§3.4 局限 1a 已接受）。

**测试**（§4.3 前 4 例）：`TestFillRule_Limit_MarketOrderBecomesPending`（same_bar_close 模式）/ `TestFillRule_Limit_PendingFillsOnBarTouch` / `TestFillRule_Limit_NextBarOpen_FillsSameBarAtOpen`（退化行为锁定）/ `TestFillRule_Limit_ExplicitPrice_WaitsForTouch`。

**对抗证明**：删转换行 → pending 空 → `TestFillRule_Limit_MarketOrderBecomesPending` 红；删 fill 分支 commission 行 → commission 断言红。

---

## Phase C【§2.2】OHLC_PATH

**文件**：`backend/strategy/backtest/engine.go`（或新文件 `sltp_path.go`——engine.go 已 495 行，**checkSLTPPath 单独成文件优先**，避免继续膨胀）+ 主循环 3 行切换（`SimulationMode=="OHLC_PATH"` 时替代 `checkSLTP`）

1. 路径构建：阳线 O→H→L→C、阴线 O→L→H→C（Close==Open 归阳线，注释说明）
2. 3 单调段区间包含检查（buy/sell 对称）；SL/TP 落不同段 → 先出现的段先触发；落同一段 → **距段起点近者先触发**（上行段先触较低价位，下行段先触较高价位）；成交价=触发价
   ⚠️ **防御性标注（2026-08-10 用户评审）**：对合法 SL/TP（buy: `SL < entry < TP`；sell 对称）同段场景**不可达**——SL 只由向下穿越触发、TP 只由向上穿越触发，单调段只含单一方向。同段规则仅为非法配置兜底（EA 可设置反向 SL/TP，SimBroker 不校验合法性）：**实现为防御性代码，不承担合法性校验**。
3. gap-at-open 保留：Open 已穿越 SL/TP → 成交价=Open（与现有 `checkBuySLTP`/`checkSellSLTP` 行为一致，spec §2.2 修订 4）
4. `checkPendingOrders` 保持范围检查（§3.4 局限 1a，不扩范围）

**测试**（§4.3 后 7 例）：`TestOHLCPath_Buy_TPBeforeSL_BullishBar` / `TestOHLCPath_Buy_SLBeforeTP_BearishBar` / `TestOHLCPath_Sell_TPBeforeSL_BearishBar` / `TestOHLCPath_SameSegment_NearerFirst`（**用非法 SL/TP 构造**，如 buy `SL>entry`；注释标明"防御性，合法 SL/TP 不可达"）/ `TestOHLCPath_GapOpen_FillsAtOpen` / `TestOHLCPath_NoHit` / `TestKlineRange_BehaviorUnchanged`（回归：默认模式逐位不变）。

**对抗证明**：删路径顺序判定改用 SL 优先 → `TestOHLCPath_Buy_TPBeforeSL_BullishBar` 红。

---

## Phase D【§2.3 + §2.4】wire 改名 + 白名单

**文件**：`strategy_backtest_validate.go` + `exec_params_validation_test.go` + proto/types 注释 + ExecutionAssumptions 注释

1. 白名单校验（spec §2.4）：`fill_rule ∈ {"", bar_close, market, limit}`、`simulation_mode ∈ {"", KLINE_RANGE, OHLC_PATH}`、`signal_timing ∈ {"", next_bar_open, same_bar_close}`（顺带补齐）；非法值 → `invalid_argument`，错误信息列合法值；`"DATASET"` 报错含"已更名 OHLC_PATH"提示
2. 测试翻转：`:118 RejectFillRuleLimit` → 接受断言；`:144 RejectSimulationModeDataset` → 拒绝+改名提示断言；**`:383 CaseSensitiveFillRule` → 翻转**（`"LIMIT"` 大写现在必须被拒——白名单顺带修正大小写 bug，测试同步改）；新增未知值拒绝测试
3. proto/types/ExecutionAssumptions 注释 `DATASET`→`OHLC_PATH`（spec §2.3：wire 值直接改名，无旧快照零成本）

**对抗证明**：删白名单 → `"FOO"` 静默走默认 → 未知值拒绝测试红。

---

## Phase E【§4.2】前端

**文件**：`ExecutionAssumptionsSelectors.tsx`（limit/DATASET 选项 enabled；DATASET value 改 `"OHLC_PATH"`）+ `BacktestParamsModal.tsx` / `useBacktestRunner.ts` / `strategyRuntime.ts` / `useStrategyWorkspaceState.ts`（union `'DATASET'`→`'OHLC_PATH'`）+ i18n（OHLC_PATH 显示名 "OHLC Path"/"K线路径模拟"；limit tooltip 说明 §2.5 退化行为——next_bar_open 下 Price=0 的 market order 转 limit 同 bar open 即成交；same_bar_close 下可能永不成交）

**对抗证明**：改回 `'DATASET'` union → `tsc --noEmit` 红。

---

## Gate（全部 Phase 完成才跑）

```
go build ./...
go test ./strategy/...
go test ./internal/connect/strategy/...
go test ./tools/mql2go/...
go run ./tools/check-file-lines --strict
cd frontend && npm run build
npx tsc --noEmit
npx vitest run
```

全部绿才回填。**测试数据确定性**：固定 epoch（`time.Date(2024, 1, 1, 0, i, 0, 0, time.UTC)`），禁 `time.Now()`（spec 21 §10 Determinism Contract）。

---

## 红队自审（任务级 edge cases，任一不过回去处理，不带债交付）

- **Phase A**：live `OpenedOrders` 是否含 pending（双计风险，先核验再写死）；OrderSelect 索引越界不 panic；OrderType 对 pending 四型映射不串；缓存一致性（事件内多调用）；KLINE_RANGE 原生 pending 路径回归
- **Phase B**：price=0→currentPrice 顺序（转换后仍生效）；next_bar_open 退化行为被测试锁定（同 bar open 成交）非静默；SL/TP 随转换保留；commission/margin 时点变化对 KLINE_RANGE 现有测试的影响（先跑回归确认破坏面）
- **Phase C**：doji（Close==Open 归阳线）；SL/TP 恰在段边界（区间包含用 <=/>= 与现有 checkSLTP 一致）；开仓于 bar 内（pending fill）的仓位同 bar SL/TP——局限 1a 接受，不扩范围；成交价恒为 SL/TP 价/Open（无 spread 混入，fill_rule=limit 非 market）
- **Phase D**：空串合法（默认语义）；大小写敏感（"LIMIT"/"DATASET" 被拒）；错误信息可执行（列合法值）
- **Phase E**：TS union 全链改齐（5 文件，缺一 tsc 红）；i18n 5 语言 key 同步；退化行为 tooltip 必须写（诚实性，spec §2.5）
- **克制**：不改实盘路径；不扩 pending 路径精度（§3.4 局限 1a/2 已接受）；engine.go 超行数时优先新文件

---

## 完工回填（不做 = 任务判失败）

1. `docs/audits/tech-debt-registry.md` FILL-SIM 条目（🟦open → ✅done 标日期）追加：**真实根因/修复方式/对抗证明结果/测试结果**；若实际根因与 spec 假设不同**如实写明**（高价值纠偏）。只改状态列 + 追加，不删条目、不改审计方事实陈述。
2. `docs/audits/handover-audit-plan.md` 变更日志加一行。
3. commit（代码 + 测试 + docs 一并，message 含 FILL-SIM）。
4. **不自行宣告完成**——标 ⚠️待 Claude 复审，等审计方核对状态 + 实测（对抗证明会实测验证）。
