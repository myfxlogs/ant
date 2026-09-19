# 设计 SSOT：VM-LIVE-VENUE-R2 —— venue 交易常量真值化

> 派生源：`design-vm-live-parity-f1f3.md` 残余 R2（`fcbfa867` 验收登记）。
> 设计实查完成日期：2026-09（Devin CLI，最终决策席）。
> 范围：仅本文件列出的字段与 builtin；R1/R3/R4 不在本设计。

## 1. 问题

VM 向策略报告的 venue 交易常量是当前唯一仍硬编码的事实类字段：

| builtin | 位置 | 现状（venue 常量） | 真值已存在于 |
|---|---|---|---|
| `MarketInfo` MODE_TRADEALLOWED(22) | vm_builtin_account.go:255-259 | `account.IsTradeAllowed`（账户轴，非 symbol 轴） | `SymbolParam.TradeMode` |
| `MarketInfo` MODE_FREEZELEVEL(32) | vm_builtin_account.go:275-276 | 恒 `0`（"no freeze model"声明） | `SymbolParam.FreezeLevel`（mt4 Ex） |
| `SymbolInfoInteger` SYMBOL_TRADE_MODE(22) | vm_builtin_account.go:181-182 | 恒 `4`（FULL 声明） | `SymbolParam.TradeMode` |
| `SymbolInfoInteger` SYMBOL_TRADE_FREEZE_LEVEL(26) | vm_builtin_account.go:187-188 | 恒 `0` | `SymbolParam.FreezeLevel` |
| `SymbolInfoInteger` SYMBOL_TRADE_EXEMODE(27) | vm_builtin_account.go:189-190 | 恒 `2`（MARKET 声明） | mt4 `SymbolInfoEx.Exemode`（mt5 无字段） |

F2 已把 `TradeMode`/`FreezeLevel` 采集进 `SymbolParam`（order_types.go:40-45），
但链路止于 `live_symbol_backfill.go`——proto/ctx/sdk/builtin 全未接，VM 仍读常量。
策略拿 FULL/no-freeze/market-exec 假事实下单：freeze 区内 modify 会裸撞 broker
拒绝（虽然 fail-closed 兜底，但报告值本身是错的）。

## 2. 设计决策

**D1 — `-1` = unknown 哨兵贯穿全链**（沿用 `mustDecimal` -1 约定）。
枚举 0 是真值（trade_mode 0=disabled；freeze 0=无冻结），"无数据"必须与真值区分。
- mt4：`si.GetEx()==nil` → `TradeMode/FreezeLevel/TradeExemode = -1`；Ex 在场 →
  `ex.GetTrade()/ex.GetFreezeLevel()/ex.GetExemode()`（orders.go:269/280 改造；
  FreezeLevel 已在 Ex 分支内，补 -1 兜底；TradeExemode 新增）。
- mt5：`TradeMode = int32(sg.GetTradeMode())` 保持（真枚举）；`FreezeLevel/TradeExemode = -1`
  （mt5 pb 无字段，orders.go:289-291 注释翻正为显式 -1）。

**D2 — proto 扩字段**（strategy_runtime.proto）：
- `LiveStrategyContext`: `int32 trade_mode = 41; int32 freeze_level = 42; int32 trade_exemode = 43;`
- `TickContext`: `int32 trade_mode = 25; int32 freeze_level = 26; int32 trade_exemode = 27;`

**D3 — backfill 逐字透传 + param==nil → -1**（live_symbol_backfill.go:32-43/59-70）。
nil 分支 return 前置 `-1` 默认值赋值，保证"param 缺席"与"disabled/无冻结"可区分。

**D4 — 传播链**：`runner.LiveSymbolInfo`（runner.go:313-319）+= 3 int32；
ctx += `liveTradeMode/liveFreezeLevel/liveTradeExemode int32`；
`UpdateSymbolInfo`（runner.go:321-335）直接 int32 赋值（非 string/mustDecimal）；
`brokerImpl.SymbolInfo` harness 分支（broker.go:186-201）+= 3 字段直填。

**D5 — `sdk.SymbolInfo`**（sdk/broker.go:75-89）+= `TradeMode, FreezeLevel, TradeExemode int32`。

**D6 — 回测模型常量保留**：`SimBroker.SymbolInfo`（backtest/broker.go:476+）
+= `TradeMode: 4, FreezeLevel: 0, TradeExemode: 2`——回测 venue 是模型（全可交易/
无冻结/市价执行），与现行常量语义一致；仅标注为 model claim，非 broker fact。

**D7 — builtin 接线**（vm_builtin_account.go）：
- `SymbolInfoInteger` 22 → `info.TradeMode`；26 → `info.FreezeLevel`；27 → `info.TradeExemode`
  （-1 逐字上浮=unknown，诚实哨兵与 mustDecimal 一致）。
- `MarketInfo` 32 → `decimal.NewFromInt(int64(info.FreezeLevel))`。
- `MarketInfo` 22 MODE_TRADEALLOWED → `IsTradeAllowed && (mode==1 || mode==2 || mode==4) ? 1 : 0`。
  语义裁决：MQL4 MODE_TRADEALLOWED 是 symbol 轴"允许开仓"——long_only/short_only/full
  均可开仓（方向性拒绝归 broker fail-closed），disabled(0)/close_only(3)/unknown(-1)→0。
  账户许可仍必须∧入（账户轴由 `builtinIsTradeAllowed` 单独承载，不丢）。
- 不改：`SYMBOL_TRADE_CALC_MODE(21)`/`SYMBOL_ORDER_MODE(33)`（无 venue 数据源，保留标注 venue 声明）。

**D8 — 测试更新义务**：`vm_api_truth_test.go:1034/1136-1141` 钉了
TRADEALLOWED=账户旗标的旧语义——改测试构造 TradeMode=4 钉新语义（两轴各验）。

## 3. 对抗测试 + mutation 计划

- T-adapter：mt4 Ex==nil → -1 哨兵断言；Ex 在场 → ex 值透传（含 Exemode 新增）；mt5 Freeze/Exemode=-1。
- T-backfill：param==nil → proto -1；param 值 → 逐字（0=disabled 不漂白为 -1）。
- T-harness：UpdateSymbolInfo→brokerImpl.SymbolInfo 三字段透传。
- T-builtin：TRADE_MODE=4/-1/0 逐字；FREEZE_LEVEL=5→5、-1→-1；
  TRADEALLOWED: (allowed∧mode=4)→1、(allowed∧mode=-1)→0、(allowed∧mode=3 close_only)→0、(!allowed∧mode=4)→0。
- M1：mt4 Ex==nil 分支回退 0 → -1 断言 RED。M2：case 26 回退 0 → freeze 测试 RED。
  M3：TRADEALLOWED 回退纯账户旗标 → (allowed∧mode=-1) 期望 0 得 1 RED。

## 4. 不做（边界）

- 不动 `SYMBOL_TRADE_CALC_MODE`/`SYMBOL_ORDER_MODE`/paper 路径默认值（另行）。
- 不动 `antv1.SymbolInfo`（回测 display 消息，VM 不经它读 venue 常量）。
- R1（signal-mode sentinel）/R4（CloseBy dispatch）各自独立立项，不在本批。
