# Builder Handoff — VM-ENUM-NUMBERING-1

> 立项：registry `VM-ENUM-NUMBERING-1`（P2）。设计 SSOT + 施工提示词，Devin CLI 出品。
> 状态：**已落档，等前序任务验收后发开工指令**。

## 0. 立项背景（触发 + 证据链）

批次2c 复审时发现 `AccountInfo*` 的 prop 编号是自造紧凑值而非真 MQL5 枚举（已修）。
本债做**全枚举一次性对齐审计**：`SymbolInfoDouble`/`SymbolInfoInteger`/`SymbolInfoString`/`MarketInfo` + `constants.go` 常量表。

**实查新发现（比 registry 原描述更严重）**：`constants.go` 常量值与 builtin switch 的 case 号**互相不自洽**——不是"编号风格不符"，是活的静默错标数据：

| 调用 | 常量值 | 命中 case | 实际返回 |
|---|---|---|---|
| `SymbolInfoDouble(s, SYMBOL_POINT)` | 2 | case2 | `VolumeMax` |
| `SymbolInfoDouble(s, SYMBOL_BID)` | 0 | case0 | `Point` |
| `SymbolInfoDouble(s, SYMBOL_ASK)` | 1 | case1 | `VolumeMin` |
| `SymbolInfoDouble(s, SYMBOL_SWAP_LONG)` | 9 | default | `0` |
| `SymbolInfoInteger(s, SYMBOL_DIGITS)` | 12 | default | `0` |
| `SymbolInfoInteger(s, SYMBOL_SELECT)` | 0 | case0 | `Digits` |
| `SymbolInfoString(s, 任何prop)` | — | 无 switch | 恒返回 `info.Name`（真 MQL5 STRING 枚举无 NAME 成员，全 prop 假） |

`MarketInfo` 的常量/switch **自洽**（MODE_TICKVALUE=17↔case17 等），但整组值偏离真 MQL4 编号（真 TICKVALUE=16），裸数字字面量调用会静默错标。

## 1. 权威枚举参考（设计 SSOT，施工不得偏离）

### 1a. ENUM_SYMBOL_INFO_DOUBLE（MQL5 当前文档序 = 枚举值）

```
BID=0 BIDHIGH=1 BIDLOW=2 ASK=3 ASKHIGH=4 ASKLOW=5
LAST=6 LASTHIGH=7 LASTLOW=8
VOLUME_REAL=9 VOLUMEHIGH_REAL=10 VOLUMELOW_REAL=11
OPTION_STRIKE=12 POINT=13
TRADE_TICK_VALUE=14 TRADE_TICK_VALUE_PROFIT=15 TRADE_TICK_VALUE_LOSS=16
TRADE_TICK_SIZE=17 TRADE_CONTRACT_SIZE=18
TRADE_ACCRUED_INTEREST=19 TRADE_FACE_VALUE=20 TRADE_LIQUIDITY_RATE=21
VOLUME_MIN=22 VOLUME_MAX=23 VOLUME_STEP=24 VOLUME_LIMIT=25
SWAP_LONG=26 SWAP_SHORT=27
SWAP_SUNDAY=28 SWAP_MONDAY=29 SWAP_TUESDAY=30 SWAP_WEDNESDAY=31
SWAP_THURSDAY=32 SWAP_FRIDAY=33 SWAP_SATURDAY=34
MARGIN_INITIAL=35 MARGIN_MAINTENANCE=36
SESSION_VOLUME=37 SESSION_TURNOVER=38 SESSION_INTEREST=39
SESSION_BUY_ORDERS_VOLUME=40 SESSION_SELL_ORDERS_VOLUME=41
SESSION_OPEN=42 SESSION_CLOSE=43 SESSION_AW=44
SESSION_PRICE_SETTLEMENT=45 SESSION_PRICE_LIMIT_MIN=46 SESSION_PRICE_LIMIT_MAX=47
MARGIN_HEDGED=48 PRICE_CHANGE=49 PRICE_VOLATILITY=50 PRICE_THEORETICAL=51
PRICE_DELTA=52 PRICE_THETA=53 PRICE_GAMMA=54 PRICE_VEGA=55 PRICE_RHO=56
PRICE_OMEGA=57 PRICE_SENSITIVITY=58
```

### 1b. ENUM_SYMBOL_INFO_INTEGER（MQL5 当前文档序）

```
SUBSCRIPTION_DELAY=0 SECTOR=1 INDUSTRY=2 CUSTOM=3 BACKGROUND_COLOR=4
CHART_MODE=5 EXIST=6 SELECT=7 VISIBLE=8
SESSION_DEALS=9 SESSION_BUY_ORDERS=10 SESSION_SELL_ORDERS=11
VOLUME=12 VOLUMEHIGH=13 VOLUMELOW=14 TIME=15 TIME_MSC=16
DIGITS=17 SPREAD_FLOAT=18 SPREAD=19 TICKS_BOOKDEPTH=20
TRADE_CALC_MODE=21 TRADE_MODE=22 START_TIME=23 EXPIRATION_TIME=24
TRADE_STOPS_LEVEL=25 TRADE_FREEZE_LEVEL=26 TRADE_EXEMODE=27
SWAP_MODE=28 SWAP_ROLLOVER3DAYS=29 MARGIN_HEDGED_USE_LEG=30
EXPIRATION_MODE=31 FILLING_MODE=32 ORDER_MODE=33 ORDER_GTC_MODE=34
OPTION_MODE=35 OPTION_RIGHT=36
```

（注：SUBSCRIPTION_DELAY 为新版 MQL5 插入的首成员，占 0；若 MetaQuotes 旧版资料给 SELECT=6，以当前文档序为准——我们面向当前枚举。）

### 1c. ENUM_SYMBOL_INFO_STRING（MQL5 当前文档序）

```
BASIS=0 CATEGORY=1 COUNTRY=2 SECTOR_NAME=3 INDUSTRY_NAME=4
CURRENCY_BASE=5 CURRENCY_PROFIT=6 CURRENCY_MARGIN=7 BANK=8
DESCRIPTION=9 EXCHANGE=10 FORMULA=11 ISIN=12 PAGE=13 PATH=14
```

**无 NAME 成员**——symbol 名是查询键不是属性。本函数无任何可接源 prop → 整个重分类。

### 1d. MQL4 MarketInfo MODE_*（docs.mql4.com 权威）

```
MODE_LOW=1 MODE_HIGH=2 MODE_TIME=5 MODE_BID=9 MODE_ASK=10
MODE_POINT=11 MODE_DIGITS=12 MODE_SPREAD=13 MODE_STOPLEVEL=14
MODE_LOTSIZE=15 MODE_TICKVALUE=16 MODE_TICKSIZE=17
MODE_SWAPLONG=18 MODE_SWAPSHORT=19 MODE_STARTING=20 MODE_EXPIRATION=21
MODE_TRADEALLOWED=22 MODE_MINLOT=23 MODE_LOTSTEP=24 MODE_MAXLOT=25
MODE_SWAPINIT=26 MODE_MAINTENANCE=27 MODE_MARGININIT=28
MODE_MARGINMAINTENANCE=29 MODE_MARGINHEDGED=30 MODE_MARGINREQUIRED=31
MODE_FREEZELEVEL=32 MODE_CLOSEBY_ALLOWED=33
（MODE_VOLUME=4 存在；MODE_SWAPTYPE/MODE_SWAPMODE/MODE_PROFITCALCMODE/
MODE_MARGINCALCMODE 不是真 MQL4 MarketInfo 模式 → 常量删除）
```

## 2. 数据源盘点（sdk.SymbolInfo / sdk.Context）

`strategy/sdk/broker.go:67-81` `SymbolInfo{Name,Digits,Point,VolumeMin,VolumeMax,VolumeStep,StopsLevel,Spread,TickValue,TickSize,SwapLong,SwapShort,ContractSize}` — **无 Bid/Ask/Time/字符串类字段**。

`ctx.Ask()/ctx.Bid()`（当前品种报价，`vm_builtin_impls.go:354-358` 同款用法）+ `ctx.ServerTime()`（unix_ms）+ `ctx.Symbol()` + `ctx.Account().IsTradeAllowed` 为权威上下文源。

## 3. 分支处置表（逐 prop 裁定）

### 3a. SymbolInfoDouble（保持 implemented，重编号 + 分支处置）

| prop | case | 处置 | 源/理由 |
|---|---|---|---|
| BID=0 | 0 | **实接** `ctx.Bid()` | 仅当 `sym==ctx.Symbol()`；sym 不同或 Bid==0 → error（非当前品种无报价源） |
| ASK=3 | 3 | **实接** `ctx.Ask()` | 同上守卫 |
| LAST=6 LASTHIGH=7 LASTLOW=8 | 6,7,8 | venue 0 | forex 无 last-deal 概念，真 MT5 亦返 0（同 iRealVolume 裁定） |
| VOLUME_REAL=9 VOLUMEHIGH_REAL=10 VOLUMELOW_REAL=11 | 9,10,11 | venue 0 | 同上 |
| POINT=13 | 13 | **实接** `info.Point` | |
| TRADE_TICK_VALUE=14 | 14 | **实接** `info.TickValue` | |
| TRADE_TICK_SIZE=17 | 17 | **实接** `info.TickSize` | |
| TRADE_CONTRACT_SIZE=18 | 18 | **实接** `info.ContractSize` | |
| VOLUME_MIN=22 | 22 | **实接** `info.VolumeMin` | |
| VOLUME_MAX=23 | 23 | **实接** `info.VolumeMax` | |
| VOLUME_STEP=24 | 24 | **实接** `info.VolumeStep` | |
| SWAP_LONG=26 | 26 | **实接** `info.SwapLong` | |
| SWAP_SHORT=27 | 27 | **实接** `info.SwapShort` | |
| 其余全部（BIDHIGH/BIDLOW/ASKHIGH/ASKLOW/OPTION_STRIKE/TICK_VALUE_PROFIT/LOSS/ACCRUED_INTEREST/FACE_VALUE/LIQUIDITY_RATE/VOLUME_LIMIT/SWAP_SUNDAY..SATURDAY/MARGIN_INITIAL/MARGIN_MAINTENANCE/SESSION_*/MARGIN_HEDGED/PRICE_*）+ default | — | `fmt.Errorf("SymbolInfoDouble: unsupported prop %d")` | 无源；TICK_VALUE_PROFIT/LOSS 不冒充=TickValue（对称性无数据支撑） |

### 3b. SymbolInfoInteger（保持 implemented）

| prop | case | 处置 | 源/理由 |
|---|---|---|---|
| CUSTOM=3 | 3 | venue 0 | 品种均 broker 源，非 synthetic |
| CHART_MODE=5 | 5 | venue `SYMBOL_CHART_MODE_BID`=0 | 回测按 Bid 建模 |
| EXIST=6 | 6 | venue 1 | 到达 case 即 broker 已解析成功（err→error 在前） |
| SELECT=7 | 7 | venue 1 | 同 `SymbolSelect`→true 既有裁定（vm_builtin_mql5_info.go:32） |
| VISIBLE=8 | 8 | venue 1 | 同上 |
| TIME=15 | 15 | **实接** `ctx.ServerTime()/1000` | 仅 `sym==ctx.Symbol()` 且 ServerTime≠0，否则 error |
| TIME_MSC=16 | 16 | **实接** `ctx.ServerTime()` | 同上守卫（毫秒精度为源原生精度） |
| DIGITS=17 | 17 | **实接** `info.Digits` | |
| SPREAD_FLOAT=18 | 18 | venue 0 | 回测固定 spread 模型事实；live 待 LIVE-ACCOUNT-FIELDS-1 复核 |
| SPREAD=19 | 19 | **实接** `info.Spread` | |
| TICKS_BOOKDEPTH=20 | 20 | venue 0 | 无 DOM 队列 = venue 事实 |
| TRADE_CALC_MODE=21 | 21 | venue `SYMBOL_CALC_MODE_FOREX`=0 | venue 模型 |
| TRADE_MODE=22 | 22 | venue `SYMBOL_TRADE_MODE_FULL`=4 | venue 全向交易 |
| START_TIME=23 | 23 | venue 0 | 永续品种无上市日（真 MT5 对非期货亦返 0） |
| EXPIRATION_TIME=24 | 24 | venue 0 | 同上 |
| TRADE_STOPS_LEVEL=25 | 25 | **实接** `info.StopsLevel` | |
| TRADE_FREEZE_LEVEL=26 | 26 | venue 0 | venue 无 freeze 模型 |
| TRADE_EXEMODE=27 | 27 | venue `SYMBOL_TRADE_EXECUTION_MARKET`=2 | 回测市价成交模型 |
| SWAP_MODE=28 | 28 | venue `SYMBOL_SWAP_MODE_POINTS`=1 | MT swap 参数按点值建模 |
| ORDER_MODE=33 | 33 | venue mask `ORDER_MARKET\|LIMIT\|STOP\|STOP_LIMIT\|SL\|TP`=63 | OrderSend 支持市价+挂单+SL/TP（vm_builtin_trade.go:88-97） |
| ORDER_GTC_MODE=34 | 34 | venue `SYMBOL_ORDERS_GTC`=0 | venue 无订单过期模型 |
| 其余（SUBSCRIPTION_DELAY=0/SECTOR=1/INDUSTRY=2/BACKGROUND_COLOR=4/SESSION_*=9-11/VOLUME*=12-14/EXPIRATION_MODE=31/FILLING_MODE=32/OPTION_*=35,36）+ default | — | error | 无源；VOLUME*=日 tick 量聚合非 venue 0（真 MT5 forex 此 prop 有值） |
| **注**：SWAP_ROLLOVER3DAYS=29 与 MARGIN_HEDGED_USE_LEG=30 | — | error | 无 swap 计费引擎/无对冲腿模型；命名常量同步删除（见 S2）使源码期即拒 |

### 3c. SymbolInfoString → **整体重分类 StatusUnsupported**

真 STRING 枚举（§1c）无 NAME；`SymbolInfo` 无任何字符串类字段可接 → 全部 15 prop 无源。处置同批次2d：`unsupportedSymbols` 加 `{Name:"SymbolInfoString", Reason: reasonAccountSymbolStub}`；删 `builtinSymbolInfoString` 函数、`builtinRegistry` 注册、wiring 绑定、implemented 条目。

### 3d. MarketInfo（MQL4 API，按真 MQL4 编号）

| mode | case | 处置 | 源/理由 |
|---|---|---|---|
| BID=9 | 9 | **实接** `ctx.Bid()` | 当前品种守卫同 3a |
| ASK=10 | 10 | **实接** `ctx.Ask()` | 同上 |
| TIME=5 | 5 | **实接** `ServerTime()/1000` | 同上守卫 |
| POINT=11 | 11 | **实接** `info.Point` | 现值正确不变 |
| DIGITS=12 | 12 | **实接** `info.Digits` | 不变 |
| SPREAD=13 | 13 | **实接** `info.Spread` | 不变 |
| STOPLEVEL=14 | 14 | **实接** `info.StopsLevel` | 不变 |
| LOTSIZE=15 | 15 | **实接** `info.ContractSize` | 不变 |
| TICKVALUE=16 | 16 | **实接** `info.TickValue` | 编号 17→16 |
| TICKSIZE=17 | 17 | **实接** `info.TickSize` | 编号 18→17 |
| SWAPLONG=18 | 18 | **实接** `info.SwapLong` | 新增 case |
| SWAPSHORT=19 | 19 | **实接** `info.SwapShort` | 新增 case |
| STARTING=20 | 20 | venue 0 | 永续品种 |
| EXPIRATION=21 | 21 | venue 0 | 同上 |
| TRADEALLOWED=22 | 22 | **实接** `ctx.Account().IsTradeAllowed` 0/1 | 新增 case（真源） |
| MINLOT=23 | 23 | **实接** `info.VolumeMin` | 编号 20→23 |
| LOTSTEP=24 | 24 | **实接** `info.VolumeStep` | 编号 22→24 |
| MAXLOT=25 | 25 | **实接** `info.VolumeMax` | 编号 21→25 |
| MARGININIT=28 | 28 | **实接** `ContractSize*ctx.Ask()/leverage` | 同 FreeMarginCheck 权威公式（volume=1）；仅当前品种、lev≠0、Ask≠0，否则 error |
| MARGINREQUIRED=31 | 31 | **实接** 同 MARGININIT | 本 venue 初始=维持保证金模型 |
| FREEZELEVEL=32 | 32 | venue 0 | 无 freeze 模型 |
| CLOSEBY_ALLOWED=33 | 33 | venue 0 | venue 不支持 close-by |
| 其余（LOW=1/HIGH=2/VOLUME=4/SWAPINIT=26/MAINTENANCE=27/MARGINHEDGED=30）+ default | — | error | LOW/HIGH/VOLUME 为日聚合无源；SWAPINIT/MAINTENANCE 期货属性；MARGINHEDGED 无对冲保证金模型 |

## 4. 施工步骤

**S1 `constants.go`（interp 常量表，坐标 `:44-73, :407-435`）——值对齐 + 增删：**

- `MODE_*` 块改真值：FREEZELEVEL 16→32、TICKVALUE 17→16、TICKSIZE 18→17、SWAPLONG 35→18、SWAPSHORT 36→19、STARTING 25→20、EXPIRATION 26→21、TRADEALLOWED 27→22、MINLOT 20→23、LOTSTEP 22→24、MAXLOT 21→25、MARGININIT 30→28、MARGINMAINTENANCE 31→29、MARGINHEDGED 32→30、MARGINREQUIRED 37→31、CLOSEBY_ALLOWED 28→33。**删除非真模式**：`MODE_SWAPTYPE`、`MODE_SWAPMODE`、`MODE_PROFITCALCMODE`、`MODE_MARGINCALCMODE`（编译期拒假 API 面）。
- `SYMBOL_*` INTEGER prop 块（`:408-422`）改真值：SELECT 0→7、VISIBLE 1→8、TIME 5→15、DIGITS 12→17、SPREAD_FLOAT 14→18、SPREAD 13→19、TRADE_CALC_MODE 15→21、TRADE_MODE 16→22、START_TIME 17→23、EXPIRATION_TIME 18→24、TRADE_STOPS_LEVEL 19→25、TRADE_FREEZE_LEVEL 20→26、TRADE_EXEMODE 21→27、SWAP_MODE 22→28；**新增** `SYMBOL_EXIST=6`、`SYMBOL_CUSTOM=3`、`SYMBOL_CHART_MODE=5`、`SYMBOL_TIME_MSC=16`、`SYMBOL_TICKS_BOOKDEPTH=20`、`SYMBOL_ORDER_MODE=33`、`SYMBOL_ORDER_GTC_MODE=34`；**删除** `SYMBOL_SWAP_ROLLOVER3DAYS`（无源，编译期拒）。
- `SYMBOL_*` DOUBLE prop 块（`:423-435`）：ASK 1→3、POINT 2→13、TRADE_TICK_VALUE 3→14、TRADE_TICK_SIZE 4→17、TRADE_CONTRACT_SIZE 5→18、VOLUME_MIN 6→22、VOLUME_MAX 7→23、VOLUME_STEP 8→24、SWAP_LONG 9→26、SWAP_SHORT 10→27；**新增** `SYMBOL_LAST=6`、`SYMBOL_LASTHIGH=7`、`SYMBOL_LASTLOW=8`、`SYMBOL_VOLUME_REAL=9`、`SYMBOL_VOLUMEHIGH_REAL=10`、`SYMBOL_VOLUMELOW_REAL=11`；**删除** `SYMBOL_MARGIN_INITIAL`、`SYMBOL_MARGIN_MAINTENANCE`。
- 返回值枚举常量新增（本批实接分支需要）：`SYMBOL_CHART_MODE_BID=0`、`SYMBOL_CHART_MODE_LAST=1`、`SYMBOL_CALC_MODE_FOREX=0`、`SYMBOL_SWAP_MODE_POINTS=1`、`SYMBOL_EXPIRATION_GTC=1`、`SYMBOL_ORDER_MARKET=1`、`SYMBOL_ORDER_LIMIT=2`、`SYMBOL_ORDER_STOP=4`、`SYMBOL_ORDER_STOP_LIMIT=8`、`SYMBOL_ORDER_SL=16`、`SYMBOL_ORDER_TP=32`、`SYMBOL_ORDER_CLOSEBY=64`、`SYMBOL_ORDERS_GTC=0`、`SYMBOL_ORDERS_DAILY=1`、`SYMBOL_ORDERS_DAILY_EXCLUDING_STOPS=2`。（`SYMBOL_TRADE_MODE_*`/`SYMBOL_TRADE_EXECUTION_*` 已存在勿重复。）
- **check-lines 预警**：constants.go 现 541 行，净增 ~20 行可能越阈——若 --strict 报 error，把 ACCOUNT_/SYMBOL_ 块拆到 `constants_account.go`/`constants_symbol.go` 新文件（同 package）。

**S2 `vm_builtin_account.go:88-200` 三函数重写：**
按 §3 表逐 case 实现。模式沿用批次2c/2e 先例：无源分支 `fmt.Errorf("<fn>: unsupported prop %d", prop)`；`ctx.Broker()==nil`/`SymbolInfo err` 现状改 `fmt.Errorf`（现静默 0 违反 fail-closed，顺手修）；BID/ASK/TIME/TIME_MSC 的当前品种守卫：`sym != vm.ctx.Symbol()` → error、`ctx.Bid().IsZero()`/`ServerTime()==0` → error。`builtinSymbolInfoString` 整函数删除。

**S3 registry 摘挂：**
- `unsupportedSymbols` 加 `{Name:"SymbolInfoString", Status:StatusUnsupported, Category:CatFunction, Reason:reasonAccountSymbolStub}`。
- 从 `implemented*` 列表移除 `SymbolInfoString`（定位后删行）；`SymbolInfoDouble/Integer`、`MarketInfo` 保持 implemented。

**S4 wiring/registration：**
删 `builtins.go` 与 `vm_builtin_wiring.go` 中 `SymbolInfoString` 的注册/绑定；`SymbolInfoDouble/Integer`、`MarketInfo` 保留。

**S5 测试（`vm_api_truth_test.go` 同文件追加，S6 命名顺延或新段）：**

- S6a 编译期拒绝 `SymbolInfoString` + registry 一致性（StatusUnsupported/Reason 非空）。
- S6b **错标矩阵**：每个真分支断言返回值=注入字段值（`SymbolInfoDouble(s,SYMBOL_POINT)==Point`、`SYMBOL_VOLUMEMIN==VolumeMin`、Integer DIGITS/SPREAD/STOPS_LEVEL、MarketInfo 全 case 含重编号值 16/17/18/19/23/24/25/28/31），**不得再命中邻 case**——关键用例：`SymbolInfoDouble(s, SYMBOL_POINT)==Point`（此前返 VolumeMax）、`SymbolInfoDouble(s, SYMBOL_BID)==Bid`（此前返 Point）。
- S6c venue 分支断言值（LAST/LASTHIGH/LASTLOW/VOLUME_REAL*=0、CUSTOM/SPREAD_FLOAT/TICKS_BOOKDEPTH=0、EXIST/SELECT/VISIBLE=1、TRADE_MODE=4、EXEMODE=2、SWAP_MODE=1、ORDER_MODE=63、ORDER_GTC_MODE=0、MarketInfo STARTING/EXPIRATION/FREEZELEVEL/CLOSEBY_ALLOWED=0、TRADEALLOWED↔IsTradeAllowed）。
- S6d fail-closed：`SymbolInfoDouble(s, SYMBOL_MARGIN_HEDGED)`/`SymbolInfoInteger(s, SYMBOL_SECTOR)`/`MarketInfo(s, MODE_LOW)` → OnInit err 非 nil；`SYMBOL_SWAP_ROLLOVER3DAYS`/`MODE_SWAPTYPE`/`MODE_PROFITCALCMODE`/`SYMBOL_MARGIN_INITIAL` 引用 → 编译期拒。
- S6e 当前品种守卫：`SymbolInfoDouble("OTHER",SYMBOL_BID)` → err；`ctx.ServerTime()==0` 时 `SymbolInfoInteger(s,SYMBOL_TIME)` → err。
- S6f 不误伤：`SymbolInfoDouble(s, SYMBOL_ASK)==Ask`、`MarketInfo(Symbol(),MODE_STOPLEVEL)` 等 e2e_real_grid_test 既有断言全绿。

**S6 Mutation 证明（每项 RED→restore→GREEN）：**
- M1：S1 常量改回旧值之一（如 SYMBOL_POINT 回 2）→ S6b RED。
- M2：注释 unsupportedSymbols 的 SymbolInfoString 行 → S6a RED×2。
- M3：MarketInfo 删 MARGININIT case（28）→ S6b RED。
- M4：SymbolInfoInteger EXIST 分支改返 0 → S6c RED。

## 5. 验收门禁

`go build ./...` / `go test ./tools/mql2go/` / `go test -race -count=3 ./tools/mql2go/` / `go vet` / `gofmt` / `go run ./tools/check-file-lines --strict`（0 errors）/ `git diff --check`。**勿部署，完成报六段式证据等复审。**

## 6. 边界/不做

- 不碰 `AccountInfo*`（批次2c 已对齐）、不碰 OrderSend/其余函数。
- 不实现 handle/session/DOM/margin-rate 子系统（VM-API-TRUTH-1 已裁定显式限制）。
- `SymbolInfoString` 重分类不在原 registry 描述内——**设计实查新证据**：真枚举无 NAME、无字符串字段可接，属本债"假分支处置"权限内。
- live 场景 `SPREAD_FLOAT`/venue 值后续由 LIVE-ACCOUNT-FIELDS-1 复核，本批只保证回测 venue 语义诚实。

---

## 修订记录（2026-09-18 Devin CLI 复审 R1）——S1/S2 局部修订 1 处

**复审结论：施工方实现与派工单逐格相符、门禁全绿，但发现 1 处 spec 自身缺陷（设计责任在 Devin CLI，非施工方偏离），返修后重验。**

**缺陷**：§3b 裁定 `TIME_MSC=16 → 实接 ctx.ServerTime()`。`ctx.ServerTime()` 返回 unix **毫秒**（int64，当前 ~1.757e12），`Value.Int` 为 **int32**（上限 2.147e9）→ `IntVal(int32(ServerTime()))` 在生产真实时间戳下**截断回绕成假值**（可为负）。施工方测试用 `serverTime:1000000` 小值规避——正确实现 spec，但生产路径静默错误，正是本批消灭的缺陷类。`ValDatetime` 位宽够但为死类型（VM 无生产者/消费者），不能作为通道。

**修订处置（TIME_MSC 转不可实现类）**：
1. `constants.go`：删除 `"SYMBOL_TIME_MSC"` 常量——与本批不可实现 prop 无命名常量的约定一致（同 SECTOR 等无名）；源码引用 → 前端隐式全局 0 → prop 0 落 default → error（端到端 fail-closed 保留，同 ROLLOVER3DAYS 链路）。
2. `vm_builtin_account.go`：删除 `case 16`——裸 `IntVal(16)` 调用落 `default → "unsupported prop 16"`（prop 16 在本 VM 不可支持，消息诚实）。
3. 测试同步：S6b 值断言表删 `{"SYMBOL_TIME_MSC", 1000000}` 行；S6e 的 TIME_MSC 守卫断言改为 `LookupMQLConstant("SYMBOL_TIME_MSC")` 不可解析 + 裸 `IntVal(16)` → err（或并入 `const/` 删名循环的 runtimeErrorCases）；`SYMBOL_TIME=15`（秒，int32 至 2038 有效）保持不变。
4. 该分支补 mutation 证据（恢复 IntVal 截断返回 → 相关断言 RED）。

**其余验收项全部成立**（保留复审记录）：枚举值与 §1a-1d 逐值相符（DOUBLE 19 项/INTEGER 21 项/MODE_* 26 项+删 4 非真名+15 返回值枚举）；§3a-3d 处置表逐格落地（实接带源、venue 事实值、无源 error、BID/ASK/TIME 当前品种+非零守卫、MARGININIT/REQUIRED 公式+leverage 守卫）；SymbolInfoString 摘除四路干净；机检 build/680/race×3 2040/vet/gofmt/check-lines 0 errors/diff --check 全绿。
