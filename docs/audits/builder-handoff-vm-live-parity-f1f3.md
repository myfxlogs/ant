# 施工提示词 — VM-LIVE-PARITY-F1/F2/F3 实盘对账修复

> **[角色:施工] 开工令**。设计 SSOT：`docs/audits/design-vm-live-parity-f1f3.md`（@3475e314）——先读设计稿再动手，本文是施工分解，冲突以设计稿为准。
> 触发：VM-LIVE-PARITY-1 实盘对账（Exness-Trial demo `904d14e6`，BTCUSDm，3 单+2 拒绝用例）实锤三缝。证据：`docs/audits/vm-live-parity-results-2026-09.md`。

## 约束与目标

- **真实性红线**：任何字段写不出 broker 事实就写"未知"（0/空），禁止回显请求值、禁止硬编码 venue 值、禁止推导非定义性数据（唯一允许推导=point 的 10^-digits，且标注）。
- MT4/MT5 adapter 不共享实现——两侧独立改、独立测，枚举语义逐平台核对，禁相互抄。
- proto 变更用 `make`（Makefile:101 `buf generate`）；生成文件随 proto 一起提交。
- 严格按 S1–S6 + T 施工，顺序执行；遇设计与实况不符**停手上报**，不自作主张改设计。

## 边界/不做

- 不动 signal-mode sentinel `IntVal(1)` 架构（残余 R1）；不动 MODE_FREEZELEVEL/TRADEALLOWED venue 常量（R2）；不动 resolver/broker_symbols（R3）；不实现 CloseBy（R4）。
- 不动读侧 comment 编码/过滤；不做 UTF-8 校验或规范化。
- 不顺手修其他 open 债；不碰无关文件。
- **禁部署、禁触运行中容器/生产端口、禁下任何真实订单**；全部验证=单测+编译。

---

## S1 — F1-a：`mthub.OrderExecutor.PlaceOrder` 改返回 `(*OrderRecord, error)`

**目标**：adapter 把 mtapi 同步响应里的完整 Order 事实交出来。

- `backend/internal/mthub/order_types.go:112` 接口签名改：`PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error)`。同文件 `OrderRequest` 加 `Deviation int32`（D4，随本步一并改，省二次签名漂移）。
- `backend/internal/mdgateway/adapter/mt4/orders.go:38-90` `Gateway.PlaceOrder`：
  - 签名改 `(*mthub.OrderRecord, error)`。
  - `pb.OrderSendRequest` 补 `Comment: req.Comment`（F3-d 合步）、`Slippage: req.Deviation`。
  - `resp.GetResult()`（`*Order`）填 `mthub.OrderRecord`：`Ticket←GetTicket()`、`OpenPrice←decimal.NewFromFloat(GetOpenPrice())`、`Volume←decimal.NewFromFloat(GetLots())`、`StopLoss/TakeProfit←GetStoploss()/GetTakeprofit()`、`Comment←GetComment()`、`Magic←GetMagicNumber()`、`OpenTime←GetOpenTime().AsTime()`（nil 守）、`AccountID←req.AccountID`、`Canonical/SymbolRaw←req.Canonical`。
  - `State` **显式推导**：`op` 为 OP_BUY/OP_SELL（market）→`OrderStateOpen`；其余（pending）→`OrderStatePending`。禁留零值（零值恰是 Pending，market fill 会撒谎）。
  - 响应缺字段→对应字段零值（=未知），不推导。
- `backend/internal/mdgateway/adapter/mt5/orders.go:19-65` 同型改：`pb.OrderSendRequest` 补 `Slippage`（`uint64(req.Deviation)`，负值钳 0；Comment 已有）；`resp.GetResult()` 同型填 record（mt5 Order 字段名以 pb 为准，逐字段核实映射，不对号的留零）。
- 调用方同步：`internal/mdgateway/adapter/broker_registry.go:45`、`internal/oms/adapter_mt.go:43`——`ticket, err := …PlaceOrder` 改 `rec, err := …` 后用 `rec.Ticket`；`internal/mthub/service_orders.go` `submitToBroker` 返回类型改 `(*OrderRecord, error)`（UpdateTicket 用 `rec.Ticket`），`PlaceOrder` 末尾改为直接返回 adapter record（删现造 `&OrderRecord{Ticket, State: Pending}`）。

## S2 — F1-b：`runner.OrderExecutor` 签名改 + `brokerImpl` 透传化

- `backend/strategy/runner/runner.go:321-330` 接口：`PlaceOrder(ctx, symbol, side, orderType, volume, price, sl, tp decimal.Decimal, comment string, magic int32, deviation int32) (sdk.OrderResult, error)`。
- `backend/strategy/runner/broker.go:35-54` `OrderSend`：nil executor 分支不变；否则 `return b.executor.PlaceOrder(b.orderCtx(), req.Symbol, req.Side, req.Type, req.Volume, req.Price, req.StopLoss, req.TakeProfit, req.Comment, req.Magic, req.Deviation)` 原样透传——**删除 `Volume: req.Volume, Price: req.Price` 回显**。
- `backend/strategy/sdk/types.go:93-99` `OrderResult` 加字段注释：`Volume/Price = broker fill 事实，零值=未知，禁止回显请求值`。
- `backend/strategy/runner/runner_test.go:34` `mockExecutor.PlaceOrder` 签名同步+返回测试事实（构造与 req 不同的 Price/Volume，供 T5 判别）。
- 全仓搜 `PlaceOrder` 其他实现/调用方（含 `OpenedOrders` 路径）一并编译修复；SimBroker 实现的是 `sdk.Broker` 非此接口，不受影响（核实勿动）。

## S3 — F1-c：deviation/comment signal 链接通

- `backend/proto/ant/v1/strategy_signal_messages.proto`：`StrategySignal` 加 `string comment = 18;`（field 17 已用，18 空）。`make` buf generate 重生成。
- `backend/internal/connect/strategy/vm_live_handlers.go:359` `vmSignalToProto`：补 `Deviation: sig.Deviation`、`Comment: sig.Comment`。Magic/OppositeTicket **不映射**（Magic 系统覆盖刻意；CloseBy 无 live dispatch）。
- `backend/internal/connect/strategy/live_dispatch.go:382` `submitOrder`：`req.Comment = sig.GetComment()`、`req.Deviation = sig.GetDeviation()`。
- `sdk.Signal.Comment`/`sdk.OrderRequest.Comment`（`strategy/sdk/strategy.go`/`types.go`）字段注释补："MT4 端 ANSI 码页，非 ASCII 可能不保真"。

## S4 — F2：`SymbolParam` 扩字段 + mt4 Ex 回退 + mt5 修正

- `backend/internal/mthub/order_types.go:34-39` `SymbolParam` 加：`TickValue, TickSize, SwapLong, SwapShort decimal.Decimal`、`FreezeLevel int32`；`TradeMode` 加注释：canonical `0=disabled,1=long_only,2=short_only,3=close_only,4=full`。
- `backend/internal/mdgateway/adapter/mt4/orders.go:212-230` `FetchSymbolParams`：
  - `ex := si.GetEx()`；逐字段 flat→Ex 回退（flat 零值才查 Ex）：`PointValue`、`ContractSize`(含 LotSize 同源)、`StopLevel`。
  - `TradeMode` 改 `ex.GetTrade()`（删 `gp.GetExecution()` 错配）。
  - 新字段：`TickValue←ex.GetTickValue()`、`TickSize←ex.GetTickSize()`、`SwapLong←ex.GetSwapLong()`、`SwapShort←ex.GetSwapShort()`、`FreezeLevel←ex.GetFreezeLevel()`。
  - LotMin/Max/Step 维持 gp 源不动；nil ex 守。
- `backend/internal/mdgateway/adapter/mt5/orders.go:255-267`：`PointValue` 改 `si.GetPoints()`（零则 `si.GetTickSize()`）；`TickValue←si.GetTickValue()`、`TickSize←si.GetTickSize()` 归位；`FreezeLevel`——核实 mt5 pb `SymbolInfo` 剩余 getter，有明确 freeze 语义字段才填，否则留 0 并在代码注释写明 "mt5 mtapi 无 freeze 字段=未知"。

## S5 — F2：live VM 传播链

- `backend/proto/ant/v1/strategy_runtime.proto`：`LiveStrategyContext` 与 `TickContext` 各加 `lot_min,lot_max,lot_step,tick_value,tick_size,swap_long,swap_short`（string，取下一空 field 号）。buf generate。
- `backend/internal/connect/strategy/live_context.go:401-437`：两 backfill 填新字段（`param.LotMin.String()` 等）；抽共享 helper `pointOrDerived(param *mthub.SymbolParam) string`——`PointValue` 零且 `Digits>0` → `decimal.New(1, -param.Digits).String()`，注释 "definitional derivation（非 broker 值）"；两处调用替换原 `param.PointValue.String()`。
- `backend/strategy/runner/runner.go:310` `UpdateSymbolInfo` 改聚合形参：新 `runner.LiveSymbolInfo` 结构（Point/Digits/ContractSize/StopsLevel/LotMin/LotMax/LotStep/TickValue/TickSize/SwapLong/SwapShort，价格量 string、Digits/levels int32）；ctx 对应字段扩展；`brokerImpl.SymbolInfo`（broker.go:187-199 harness 分支）填全字段（新增字段 `mustDecimal` 同式解析）。
- `backend/internal/connect/strategy/vm_live_handlers.go:38` 与 `:144` 两调用点改传聚合结构。

## S6 — 测试（先红后绿纪律）

| # | 位置 | 断言 |
|---|---|---|
| T1 | mt4 adapter 测试（fake trading client 捕获 `pb.OrderSendRequest`） | `Comment==req.Comment`、`Slippage==req.Deviation`；fake 返回带 OpenPrice/Lots/Comment 的 Order→record 全映射+market op `State==OrderStateOpen`、pending op `State==OrderStatePending` |
| T2 | mt4 `FetchSymbolParams` fake（flat=0、Ex 有值） | 各字段取 Ex 值；flat 有值时取 flat；双零→0 |
| T3 | 同上 fake `gp.Execution=9, ex.Trade=4` | `param.TradeMode==4`（语义源正确） |
| T4 | mt5 fake `Points=0.01, TickValue=1.5, TickSize=0.01` | `PointValue=="0.01"`、`TickValue=="1.5"` |
| T5 | runner_test：`mockExecutor` 返回 `{Price:81262.24, Volume:0.02}`，req 给 `{Price:0, Volume:0.015}` | `OrderSend` 结果 `Price==81262.24 && Volume==0.02`（executor 值非 req 值） |
| T6 | `pointOrDerived` 单测 | PointValue=0+Digits=2→"0.01"；有值不推导；Digits=0→"0" |
| T7 | `vmSignalToProto` 单测 | `sdk.Signal{Comment:"PARITY-X",Deviation:3}`→`pb.GetComment()=="PARITY-X" && pb.GetDeviation()==3` |
| T8 | submitOrder 测试——`NewHub`+`NewMtHubService` 挂 fake `mthub.OrderExecutor` 捕获 `*OrderRequest`（先例 deploy_live_test.go:212-213）；或等价注入点 | `req.Comment`/`req.Deviation` 透传到 executor |

- **mutation 证据（先红后绿）**：M1 `brokerImpl` 复辟 `req.Price/req.Volume`→T5 RED→restore GREEN；M2 mt4 删 `ex` 回退→T2/T3 RED→restore；M3 mt4 删 `Comment: req.Comment`→T1 RED→restore。
- fake client 基建：mt5_test.go:633 `TestFetchSymbolParams_WithMock` 已有 mock 先例；mt4 侧若无 trading client fake 则按同模式建（查 `scripts/cap.sh` 先复核可复用 fake）。

## 验收门禁

- `cd backend && go build ./...` / `go vet ./...` / `go test ./internal/mdgateway/... ./internal/mthub/... ./strategy/... ./internal/connect/strategy/... ./internal/oms/...` / `go test -race` 同包 / `go run ./tools/check-file-lines --strict` 0 errors / `git diff --check`。
- proto 改动：`make` buf generate 后 `git status` 确认生成文件随提交。
- registry/STATE 更新为 `⚠️待独立复审`；**勿部署，勿 push，停手报证据等 Devin CLI 复审**。
