# 派工单：VM-LIVE-VENUE-R2 —— venue 交易常量真值化

> 立项背景：F1/F2/F3 验收残余 R2（`fcbfa867`）——VM venue 常量仍硬编码，
> `SymbolParam.TradeMode/FreezeLevel` 已采集但断在 proto 边界。
> 设计 SSOT：`docs/audits/design-vm-venue-const-r2.md`（唯一真相源，坐标冲突以它+代码为准）。
> 约束与目标：D1-D8 全部实现；`-1`=unknown 哨兵全链；回测模型常量保留。
> 边界/不做：不改 CALC_MODE/ORDER_MODE；不动 antv1.SymbolInfo；R1/R4 不立项。

## S1 — SymbolParam + adapter 哨兵

- `internal/mthub/order_types.go` `SymbolParam`（~:37-50）+= `TradeExemode int32`
  （注释：canonical exemode enum 0=instant,1=request,2=market,3=exchange；-1=unknown）。
- `internal/mdgateway/adapter/mt4/orders.go` ~:260-280：Ex 分支内
  `param.TradeExemode = ex.GetExemode()`；**Ex==nil 时** `TradeMode/FreezeLevel/TradeExemode = -1`
  （现 `param.TradeMode = ex.GetTrade()` :280 在 Ex nil 时 Go 零值=0 与 disabled 混淆——先设 -1，Ex 在场再覆写）。
  注意：FreezeLevel 现仅在 Ex 分支赋值（:269），-1 默认值同理。
- `internal/mdgateway/adapter/mt5/orders.go` ~:289-306：`FreezeLevel = -1; TradeExemode = -1`
  （现注释"stays 0"翻正为显式 -1=unknown）；`TradeMode` 保持 `int32(sg.GetTradeMode())`（真枚举）。

## S2 — proto + regen

- `proto/ant/v1/strategy_runtime.proto`：
  - `LiveStrategyContext`（:290 后）+=
    `int32 trade_mode = 41; int32 freeze_level = 42; int32 trade_exemode = 43;`
    （注释：canonical enum；-1=unknown）。
  - `TickContext`（:394 前）+= `int32 trade_mode = 25; int32 freeze_level = 26; int32 trade_exemode = 27;`
- 走仓内 proto 生成链路（`make`/`buf generate`，与 F2 同路径）；
  Go pb + `frontend/src/gen/.../strategy_runtime_pb.ts` 双产出随提交。

## S3 — backfill

- `internal/connect/strategy/live_symbol_backfill.go`：
  - `backfillSymbolInfo`（:22-44）与 `backfillTickSymbolInfo`（:49-71）：
    `param == nil` 早退**之前**先赋 `TradeMode/FreezeLevel/TradeExemode = -1`；
    param 在场 → `lctx.TradeMode = param.TradeMode` 等逐字透传（含 0=disabled，不漂白）。
  - 注意 mt5 TradeMode 0 是**真值**（disabled），只有 param 缺席才 -1。

## S4 — runner 传播

- `strategy/runner/runner.go` `LiveSymbolInfo`（:313-319）+= `TradeMode, FreezeLevel, TradeExemode int32`；
  ctx struct += `liveTradeMode, liveFreezeLevel, liveTradeExemode int32`；
  `UpdateSymbolInfo`（:321-335）直接 int32 赋值。
- `internal/connect/strategy/vm_live_handlers.go` :40-41/:152-153 两处构造
  `LiveSymbolInfo` 处 += 三字段（`lctx.TradeMode`/`tctx.TradeMode` 等）。
- `strategy/runner/broker.go` `brokerImpl.SymbolInfo` harness 分支（:186-201）+=
  `TradeMode: ctx.liveTradeMode, FreezeLevel: ctx.liveFreezeLevel, TradeExemode: ctx.liveTradeExemode`。

## S5 — sdk + SimBroker

- `strategy/sdk/broker.go` `SymbolInfo`（:75-89）+= `TradeMode, FreezeLevel, TradeExemode int32`。
- `strategy/backtest/broker.go` `SimBroker.SymbolInfo`（:476-510）return struct +=
  `TradeMode: 4, FreezeLevel: 0, TradeExemode: 2`（回测 venue 模型常量——全交易/无冻结/市价执行，
  与现行硬编码语义一致，注释标 model claim）。

## S6 — VM builtin 接线

- `tools/mql2go/vm_builtin_account.go`：
  - `SymbolInfoInteger` case 22 → `interp.IntVal(info.TradeMode)`；
    case 26 → `interp.IntVal(info.FreezeLevel)`；case 27 → `interp.IntVal(info.TradeExemode)`。
  - `builtinMarketInfo` case 32 → `interp.DecimalVal(decimal.NewFromInt(int64(info.FreezeLevel)))`。
  - case 22 MODE_TRADEALLOWED（:255-259）→
    `if vm.ctx.Account().IsTradeAllowed && (m == 1 || m == 2 || m == 4)`（m=info.TradeMode）→ 1 else 0
    （注释：symbol 轴"允许开仓"∧账户轴；0/-1/3→0，unknown fail-closed）。

## T — 测试与 mutation（先红后绿）

- T1 adapter parity（mt4 orders_parity_test.go 扩展）：Ex==nil→-1×3 断言；
  Ex 在场→`ex.GetTrade()/GetFreezeLevel()/GetExemode()` 透传。mt5 测试：Freeze/Exemode=-1。
- T2 backfill：param==nil→proto -1；param.TradeMode=0→proto 0（真值不漂白）。
- T3 harness：`UpdateSymbolInfo{TradeMode:4,FreezeLevel:5,TradeExemode:2}` →
  `brokerImpl.SymbolInfo` 断言三字段。
- T4 builtin：`SymbolInfoInteger` 22/26/27 逐字（含 -1）；
  `MarketInfo` TRADEALLOWED 六格：(T∧4)→1、(T∧1)→1、(T∧-1)→0、(T∧0)→0、(T∧3)→0、(F∧4)→0。
- T5 更新 `vm_api_truth_test.go:1034/1136-1141` 旧语义断言→新两轴语义（ctx 给 TradeMode=4）。
- M1：mt4 Ex==nil 回退 0 值 → T1 RED。M2：case 26 回退 `0` → T4 RED。
  M3：TRADEALLOWED 回退纯 `IsTradeAllowed` → T4 (T∧-1) 格 RED。

## 机检门禁

`go build ./...`、`go vet`（影响包）、影响包 `go test`、`check-file-lines --strict`、
`git diff --check`、proto regen 双产出 diff 净。

## 尾部

勿部署、勿触容器、勿实盘单。完成报证据（含 mutation RED→GREEN 实录），
停手等 Devin CLI 独立复审。禁 `--no-verify`。
