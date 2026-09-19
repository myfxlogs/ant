# VM-LIVE-PARITY-F1/F2/F3 立项设计 — 实盘对账修复

> 立项背景：VM-LIVE-PARITY-1 实盘对账（Exness-Trial demo `904d14e6`，BTCUSDm，3 单+2 拒绝用例）实锤三个实盘语义缝。证据：`docs/audits/vm-live-parity-results-2026-09.md`。
> 本文是**设计 SSOT**，不含实现；施工见 `builder-handoff-vm-live-parity-f1f3.md`。
> 设计实查日期：2026-09-19。所有代码坐标以实查时 HEAD 为准。

## 0. 设计总原则

- 任何字段写不出 broker 事实时写"未知"（0/空），**永不回显请求值、永不硬编码 venue 值**。
- 权威源就近取：mtapi 同步响应 > mtapi 回读 > 推导（仅 10^-digits 这类定义性推导，且标注来源）。
- MT4/MT5 adapter 不共享实现——两侧独立修、独立验，枚举语义逐平台核对。
- 不扩大 diff：不顺手改 VM/OMS/前端；信号模式 sentinel 架构不重设计。

---

## 1. F1 — OrderSend 结果回显请求值非 broker 事实（P2）

### 1.1 实查事实链（全部已证实）

| 层 | 坐标 | 实况 |
|---|---|---|
| mtapi 同步响应 | `mt4/mt4.pb.go:7967` `OrderSendReply.GetResult() *Order` | **返回完整 Order**（Ticket/OpenPrice/Lots/SL/TP/Comment/MagicNumber/OpenTime/Profit），非仅 ticket。mt5 同型 |
| mt4 adapter | `internal/mdgateway/adapter/mt4/orders.go:61-90` | 丢弃 Result 全部字段，只 `GetTicket()`；**且不传 Comment/Slippage**（pb 字段都存在） |
| mt5 adapter | `internal/mdgateway/adapter/mt5/orders.go:19-65` | 同丢 Result（`GetTicket()` only）；Comment 已传、Slippage 未传 |
| mthub Executor 接口 | `internal/mthub/order_types.go:112` | `PlaceOrder(ctx, req) (int64, error)`——签名只许回 ticket |
| mthub service | `internal/mthub/service_orders.go:78` | 返回 `&OrderRecord{Ticket, State: Pending}`——诚实（不回显价量）但丢弃了 adapter 本可给的事实；且 market fill 也标 Pending |
| runner OrderExecutor | `strategy/runner/runner.go:322-330` | 散参 `PlaceOrder(...) (int64, error)`，**无 deviation 形参** |
| 造假点 | `strategy/runner/broker.go:48-53` | `sdk.OrderResult{RetCode: RetDone, Ticket, Volume: req.Volume, Price: req.Price}`——回显请求值 |
| executor 接线 | 全仓 grep | **生产零装配**（注释 "set by LiveRunner" 但无人调）→ 非 harness 路径仅测试可达；造假是地雷非现役 |
| 实盘真路径 | `tools/mql2go/vm_builtin_trade.go:70-87` | signalMode→sentinel `IntVal(1)`→`vm.signal`→live_dispatch→coordinateMutation（同步 barrier 等确认）→mtHub.PlaceOrder。fill 事实经 position 快照→position cache→OrderSelect 抵达策略（设计内） |
| deviation | `vm_builtin_trade.go:65` `req.Deviation=deviation` 有值 → runner 接口无形参丢一次；signal 转换器（vm_live_handlers.go:392）不映射 Deviation 丢第二次；`mthub.OrderRequest` 无 Deviation 字段丢第三次；adapter 永不设 Slippage | **四层全丢** |
| OrderSend sentinel | `vm_builtin_trade.go:85` | `IntVal(1)` 是 async 设计契约（代码已注释）；改造=VM 内阻塞等 barrier，属架构重设计 |

### 1.2 实盘证据（vm-live-parity-results）

- 请求 `price=0` → 实际 fill `81262.24`；请求 `volume=0.015` → broker 归一化 `0.02`。任何把请求值当结果返回的路径都在陈述假事实。

### 1.3 设计决策

**D1：`mthub.OrderExecutor.PlaceOrder` 改返回 `(*OrderRecord, error)`**——adapter 用同步响应的完整 Order 填事实：
- mt4：`resp.GetResult()` → `Ticket/OpenPrice/Lots→Volume/StopLoss/TakeProfit/Comment/MagicNumber/OpenTime`；`State` 显式推导：market op（OP_BUY/OP_SELL）→`OrderStateOpen`，pending op→`OrderStatePending`（OrderState 零值恰是 Pending——**必须显式赋值**，不能留零值默认）。
- mt5：同型映射（其 `resp.GetResult()` 亦是完整 Order）。
- 响应里某字段缺席→对应字段零值=未知（如 Profit 未回→0）。**不推导**。

**D2：`MtHubService.submitToBroker`/`PlaceOrder` 透传 adapter record**——`PlaceOrder` 直接返回 adapter 的 record（不再现造 `{State:Pending}`）；`omsWriter.UpdateTicket` 用 `record.Ticket` 不变。

**D3：`runner.OrderExecutor.PlaceOrder` 签名改 `(sdk.OrderResult, error)` 并加 `deviation int32` 形参**：
- `brokerImpl.OrderSend` 变纯透传：nil executor→`RetRejected`+error；否则返回 executor 结果原样。造假点物理消除——它再也拿不到可回显的机会。
- 契约钉在 `sdk.OrderResult` 字段注释：Price/Volume=**broker fill 事实**，零值=未知，禁止回显请求值。
- 实现方义务：mockExecutor（测试）填测试事实；未来 live adapter 填 broker 事实。
- 备选项已否决：`OpenedOrders` 回读方案（多一次 RPC 且"not found"语义模糊）；保留 `(int64, error)`+brokerImpl 零填充方案（字段永远未知，浪费接口可承载的事实）。

**D4：deviation 四层接线**：
- `mthub.OrderRequest` 加 `Deviation int32`。
- `runner.OrderExecutor.PlaceOrder` 加 `deviation int32` 形参（D3 同一次签名变更）。
- signal 转换器 `vmSignalToProto`（`vm_live_handlers.go:359`）补映射 `Deviation: sig.Deviation`（**附带发现**：proto 的 Magic/Deviation/OppositeTicket 三字段全部存在但转换器一个都不映射——Magic 是刻意系统覆盖 `strategyMagic()` 不动；Deviation 映射；OppositeTicket **不映射**——`CloseBy` live dispatch 全仓 grep 零命中=未实现，映射了也无消费方，登记残余 R4）。
- `submitOrder`（live_dispatch.go:382）`req.Deviation = sig.GetDeviation()`。
- mt4 `Slippage: req.Deviation`（int32 直传）；mt5 `Slippage: uint64(req.Deviation)`，负值钳 0。
- `broker_registry.go:45`/`oms/adapter_mt.go:43` 的 `mreq` 上游无 deviation 概念→留 0（注释含义 "0 = use default"），不动。

**D5：sentinel `IntVal(1)` 不动**——async 契约已文档化；live 策略读真实 ticket/价量走 OrderSelect→position cache（broker 事实）。登记为**已知残余**：MQL 语义 `ticket=OrderSend()` 拿到的 1 不是真 ticket，策略若存 ticket 做 OrderSelect 会失配——这是信号模式架构的固有妥协，修复=VM 内阻塞等确认（独立设计项，不在本批）。

### 1.4 消费方影响面（全枚举，防漏改）

| 消费方 | 变更 |
|---|---|
| `mthub.OrderExecutor` 实现 | mt4 Gateway、mt5 Gateway 签名+填充 |
| `MtHubService.submitToBroker` | 收 record 透传 |
| `broker_registry.go:45` `a.gw.PlaceOrder` | ticket→record.Ticket |
| `oms/adapter_mt.go:43` `a.executor.PlaceOrder` | 同上 |
| `mthub_service.go:104` RPC | `rec.Ticket` 不变（`Status:"submitted"` 已诚实，不动） |
| `runner.OrderExecutor` 实现 | mockExecutor（runner_test.go）；生产无其他实现 |
| `brokerImpl.OrderSend` | 透传改写 |
| 全部调用 mock/fake | 测试同步改 |

---

## 2. F2 — SymbolParams 瘦数据链（P3）

### 2.1 实查事实链

| 层 | 坐标 | 实况 |
|---|---|---|
| mt4 `SymbolInfo.Ex` | `mt4/mt4.pb.go:3560-3903` | `SymbolInfoEx` 富结构：`Trade`(3602)/`Spread`/`Exemode`(3735)/`ContractSize`(3777)/`TickValue`(3784)/`TickSize`(3791)/`StopsLevel`(3798)/`Point`(3847)/`FreezeLevel`(3896)/`SwapLong/Short`(3756/3763)/`LongOnly`(3875)/`InstantMaxVolume` 等 54 getter |
| mt4 adapter 映射 | `orders.go:212-230` | 只读平铺 `si.Digits/StopsLevel/Point/ContractSize`+`gp.MinLot/MaxLot/LotStep`；**`si.GetEx()` 从未读** |
| mt4 TradeMode 语义错 | `orders.go:228` | `param.TradeMode = gp.GetExecution()`——**execution-mode 枚举写进 trade-mode 字段**（两个不同枚举）。正确源=`si.GetEx().GetTrade()` |
| mt5 PointValue 语义错 | `mt5/orders.go:259` | `PointValue = si.GetTickValue()`——**货币 tick 值写进 point-size 字段**。正确源=`si.GetPoints()`（mt5.pb.go:5873），兜底 `GetTickSize()`(5915) |
| mt5 TradeMode | `mt5/orders.go:257` | `sg.GetTradeMode()`——语义正确保留 |
| SymbolParam 缺字段 | `order_types.go:34-39` | 无 TickValue/TickSize/SwapLong/SwapShort/FreezeLevel |
| VM sdk.SymbolInfo | `strategy/sdk/broker.go:75-90` | 有 VolumeMin/Max/Step、TickValue/TickSize、SwapLong/Short、ContractSize——模型齐全 |
| live 传播 | `live_context.go:401-437` + `vm_live_handlers.go:38/144` | LiveStrategyContext/TickContext 只载 4 字段→`UpdateSymbolInfo(point,digits,contractSize,stopsLevel)`→`brokerImpl.SymbolInfo` harness 分支只填 4 个→**VM 实盘 VolumeMin/Max/Step/TickValue/TickSize/Swap\* 全 0** |
| backtest 对照 | `backtest_execution.go:299-306` | 回测路径已传播 LotMin/Max/Step 进 `antv1.SymbolInfo`——live 独缺 |
| broker_symbols.trade_mode | `internal/symbol/resolver.go:37-61` | **管理配置表**（migrations 手写），非 adapter 同步→语义修复不影响 resolver |
| 实盘证据 | BTCUSDm 实测 point/lotStep/stopLevel/maxLot/tradeMode 全 0，但 lotStep 实存 0.01（归一化行为证明） |

### 2.2 设计决策

**D6：`mthub.SymbolParam` 扩字段**：`TickValue, TickSize, SwapLong, SwapShort decimal.Decimal` + `FreezeLevel int32`。零值=未知语义沿用（含 ContractSize 不默认 1 的既有注释纪律）。`TradeMode` 加规范注释：canonical 枚举 `0=disabled,1=long_only,2=short_only,3=close_only,4=full`（MT4/MT5 原值同序直传）。

**D7：mt4 adapter 逐字段 flat→Ex 回退**（先平铺权威值，零则查 Ex）：
- `PointValue`: `si.Point`→`ex.Point`
- `ContractSize`/`LotSize`: `si.ContractSize`→`ex.ContractSize`
- `StopLevel`: `si.StopsLevel`→`ex.StopsLevel`
- `TradeMode`: **`ex.Trade`**（语义修正；gp.Execution 写法删除）
- 新字段：`TickValue←ex.TickValue`、`TickSize←ex.TickSize`、`SwapLong/Short←ex.SwapLong/Short`、`FreezeLevel←ex.FreezeLevel`
- LotMin/Max/Step 维持 gp 源（Ex 无对应）。
- 每字段独立回退，真零值保持零=未知，不推导。

**D8：mt5 adapter 修正**：`PointValue`←`si.GetPoints()`，零则 `si.GetTickSize()`；`TickValue`/`TickSize` 各归其位（`si.GetTickValue()`/`si.GetTickSize()`）；`FreezeLevel` mt5 若无字段→0=未知（施工时核实 mt5 pb `GetLimitPoints` 等剩余 getter 语义，不对号则不填）。

**D9：live VM 传播（proto 加字段）**——`LiveStrategyContext` 与 `TickContext` 各加：`lot_min,lot_max,lot_step,tick_value,tick_size,swap_long,swap_short`（string）。**不加** trade_mode/freeze_level——sdk.SymbolInfo 无法表达，MODE_FREEZELEVEL/TRADEALLOWED 的 venue-0 声明属另一笔债（登记残余）。
- `backfillSymbolInfo`/`backfillTickSymbolInfo` 填新字段。
- `runner.UpdateSymbolInfo` 签名改 struct 形参（新 `runner.LiveSymbolInfo` 聚合，避免 10 形参）+ctx 字段扩展；`vm_live_handlers.go:38/144` 两调用点同步。
- `brokerImpl.SymbolInfo` harness 分支填全字段。
- **Point 定义性推导**：共享 helper `pointOrDerived(param)`——`param.PointValue` 零且 `Digits>0` → `10^-Digits`，注释标 "definitional derivation（非 broker 值）"；两个 backfill 共用。adapter 输出保持原始 broker 值（0=未知），推导只发生在 VM 边界——RPC/风险门消费方仍见原始值。

### 2.3 边界

- 不改 `symbol/resolver.go` 语义（admin 表 0=disabled 语义独立）。
- 不改 MODE_FREEZELEVEL=0/TRADEALLOWED 的 VM venue 声明（sdk.SymbolInfo 无字段可载，登记残余）。
- `hub_estimator.go`/`mthub_service.go` RPC 透传新字段与否：RPC `SymbolParams` 响应 proto 若有对应字段则顺手补，无则不动（施工核实，禁造字段）。

---

## 3. F3 — comment 在途丢失+UTF-8 损耗（P4）

### 3.1 实查事实链

| 层 | 坐标 | 实况 |
|---|---|---|
| VM signal | `vm_builtin_trade.go:75-83` | `sdk.Signal{Comment: comment}` 已捕获 ✓ |
| proto | `strategy_signal_messages.proto` | StrategySignal 有 Magic(15)/Deviation(16)/OppositeTicket(17)，**无 comment 字段** |
| 转换器 | `vm_live_handlers.go:392-400` | `sdk.Signal→antv1.StrategySignal` 只映射 Symbol/SignalType/Volume/Price/SL/TP/ExecutedTicket——**Comment/Deviation/Magic/OppositeTicket 全丢** |
| submitOrder | `live_dispatch.go:382-390` | `mthub.OrderRequest` 不设 `Comment` |
| mt4 adapter | `orders.go:61-69` | `pb.OrderSendRequest` **不设 Comment**（pb 字段存在 pb.go:7901） |
| mt5 adapter | `mt5/orders.go:41` | `Comment: &req.Comment` 已传 ✓ |
| RPC 路径 | `mthub_service.pb.go:341` + `mthub_service.go:109` | `PlaceOrderRequest.comment` 字段存在且已接 `req.Comment`——RPC 下单只被 mt4 adapter 丢 |
| 读侧 | `order_history.go:73/152`、`order_stream.go:155/178` | `o.GetComment()` 原样透传 ✓（888 持仓 `�` 乱码=MT4 侧 ANSI 真实字节，透传即 broker 事实，非我方损坏） |
| 实盘证据 | PARITY-* comment 到 broker 全空；magic 正常 | 丢链在 send 侧三层 |

### 3.2 设计决策

**D10：proto 加 `string comment = 18`**（`strategy_signal_messages.proto`）+ `make` buf generate 重生成。

**D11：转换器补映射** `Comment: sig.Comment`（+D4 的 Deviation 同修）。

**D12：`submitOrder` 补** `req.Comment = sig.GetComment()`。

**D13：mt4 adapter 补** `Comment: req.Comment`（pb 字段直传）。

**D14：编码策略=原样透传+如实文档**：MT4 comment 走 broker ANSI 码页，任意 UTF-8（中文/emoji）到 MT4 可能被转换/截断——这是 venue 限制。策略：不校验、不规范化、不声称保存成功；OrderResult/record 的 Comment 字段一律以 broker 回读为准（D1 后 adapter 返回的是 broker 回执 comment）。在 `OrderRequest.Comment`/`Signal.Comment` 字段注释写明 "MT4 端 ANSI 码页，非 ASCII 可能不保真"。

### 3.3 边界

- 不动读侧编码（透传=事实）。
- 不做评论内容校验/过滤（venue 会拒绝则错误上浮）。
- MQL4 `OrderClose`/`OrderModify` 本无 comment 形参——send 侧 comment 只修 OrderSend 链。

---

## 4. 测试与对抗证据要求

| # | 测试 | 判别面 |
|---|---|---|
| T1 | mt4 adapter PlaceOrder fake trading client 断言 `pb.OrderSendRequest.Comment==req.Comment` 且 `Slippage==req.Deviation`；返回带 OpenPrice/Lots/Comment 的 Order→record 字段全映射、State=Open | 删 Comment 行→RED；填错字段→RED |
| T2 | mt4 FetchSymbolParams fake 返回 flat=0+Ex 有值→param 取 Ex；flat 有值→取 flat；双零→0 | 删 Ex 回退→RED |
| T3 | mt4 TradeMode 源断言 `ex.Trade`（fake 给 gp.Execution=9、ex.Trade=4→param=4） | 复辟 gp.Execution→RED |
| T4 | mt5 `PointValue` fake `Points=0.01,TickValue=1.5`→param.PointValue=0.01 | 复辟 GetTickValue→RED |
| T5 | runner brokerImpl：fake executor 返回 `sdk.OrderResult{Price:81262.24,Volume:0.02}`→OrderSend 原样透传；`Volume/Price` 不来自 req | 改回 `req.Price/req.Volume` 填充→RED（构造 req 值≠executor 值） |
| T6 | `pointOrDerived`：PointValue=0+Digits=2→point=0.01；PointValue 有值→不推导；Digits=0→0 | 删推导→RED |
| T7 | 转换器：`sdk.Signal{Comment:"PARITY-X",Deviation:3}`→antv1 字段齐 | 删映射→RED |
| T8 | submitOrder→`req.Comment`/`req.Deviation` 透传（mock mtHub 捕获 req） | 删赋值→RED |
| mutation | F1：brokerImpl 复辟 `req.Price/req.Volume`→T5 RED；F2：删 ex 回退→T2 RED；F3：删 mt4 Comment→T1 RED | 三处先红后绿 |

## 5. 明确不做

- 不改 signal-mode sentinel 架构/VM 内阻塞等确认（残余登记）。
- 不动读侧 comment 编码、不做 UTF-8 过滤。
- 不改 resolver/broker_symbols 语义。
- 不部署、不下实盘测试单；本批纯代码+单测修复，实盘复验另立协议。
- 不顺手修其他 open 债。

## 6. 残余登记（本批不修）

- **R1**：signal-mode `OrderSend` sentinel `IntVal(1)`——真 ticket 无法同步返回（async 契约）；策略存 ticket 做 OrderSelect 会失配。修法=VM 事件内阻塞等 coordinateMutation 确认后回填，独立设计项。
- **R2**：`MODE_FREEZELEVEL=0`/`MODE_TRADEALLOWED` 等 VM venue 常量声明（vm_builtin_account.go）——sdk.SymbolInfo 无 FreezeLevel/TradeMode 字段，需模型扩展另立。
- **R3**：`broker_symbols.trade_mode` 与 `SymbolParam.TradeMode` 语义虽同名不同源（admin vs broker），文档已辨，无动作。
- **R4**：`ActionCloseBy` live dispatch 未实现（全仓零命中），`StrategySignal.opposite_ticket` 字段无消费方——CloseBy 实盘支持是独立功能债，非本批。
