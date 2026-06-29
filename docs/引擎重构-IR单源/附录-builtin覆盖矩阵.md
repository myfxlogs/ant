# 附录：解释器 Builtin 覆盖度矩阵

> 审计基准：`backend/tools/mql2go/interp/` 全部 `builtin_*.go` + `builtins.go` + `eval.go` + `exec.go` + `series.go`
> 审计日期：2025-07-13
> 状态定义：**实现** = 有完整逻辑 / **stub** = 有 dispatch 但 SDK 层返回 0 或固定值 / **缺失** = 无 dispatch，命中 `callBuiltin` 末尾的 unimplemented 路径

---

## 1. Math 函数（builtinTable）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| MathAbs | 共享 | 实现 | Math |
| MathMax | 共享 | 实现 | Math |
| MathMin | 共享 | 实现 | Math |
| MathSqrt | 共享 | 实现（float64 内部） | Math |
| MathPow | 共享 | 实现（float64 内部） | Math |

## 2. Platform 函数（builtinTable）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| Print | 共享 | 实现 | Platform |
| Alert | 共享 | 实现（alias Print） | Platform |
| Comment | 共享 | 实现（alias Print） | Platform |
| Sleep | 共享 | 实现（no-op） | Platform |

## 3. Array 函数（builtinTable，builtin_tools.go init 注册）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| ArrayResize | 共享 | 实现 | Array |
| ArraySize | 共享 | 实现 | Array |
| ArrayCopy | 共享 | 实现 | Array |
| ArraySetAsSeries | 共享 | 实现（no-op） | Array |
| ArrayMaximum | 共享 | 实现 | Array |
| ArrayMinimum | 共享 | 实现 | Array |
| ArraySort | 共享 | 实现 | Array |
| ArrayInitialize | 共享 | 实现 | Array |

## 4. String 函数（builtinTable）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| StringConcatenate | 共享 | 实现 | String |
| StringFind | 共享 | 实现 | String |
| StringSubstr | 共享 | 实现 | String |
| StringLen | 共享 | 实现 | String |
| StringReplace | 共享 | 实现 | String |
| StringSplit | 共享 | 实现 | String |
| StringTrimLeft | 共享 | 实现 | String |
| StringTrimRight | 共享 | 实现 | String |

## 5. Conversion 函数（builtinTable）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| DoubleToString | 共享 | 实现 | Conversion |
| IntegerToString | 共享 | 实现 | Conversion |
| StringToDouble | 共享 | 实现 | Conversion |
| StringToInteger | 共享 | 实现 | Conversion |
| NormalizeDouble | 共享 | 实现 | Conversion |

## 6. Datetime 函数（builtinTable）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| TimeToString | 共享 | 实现（简化：返回 unix 秒字符串） | Datetime |
| TimeCurrent | 共享 | 实现 | Datetime |

## 7. Market Data 函数（callMarketData switch）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| Ask | 共享 | 实现 | Market |
| Bid | 共享 | 实现 | Market |
| Point / _Point | 共享 | 实现 | Market |
| Symbol / _Symbol | 共享 | 实现 | Market |
| Digits | 共享 | 实现 | Market |
| Period | 共享 | 实现 | Market |

## 8. Time Series 访问（evalSubscript，非 builtinTable）

| 访问方式 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| Close[n] | 共享 | 实现 | Series |
| Open[n] | 共享 | 实现 | Series |
| High[n] | 共享 | 实现 | Series |
| Low[n] | 共享 | 实现 | Series |
| Volume[n] | 共享 | 实现 | Series |
| Time[n] | 共享 | 实现 | Series |

## 9. MQL4 交易函数（callTrade switch）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| OrderSend | MQL4 | 实现 | Trade |
| OrderClose | MQL4 | 实现 | Trade |
| OrderModify | MQL4 | 实现 | Trade |
| OrderDelete | 共享 | 实现 | Trade |
| OrdersTotal | MQL4 | 实现 | Trade |
| OrderSelect | MQL4 | 实现 | Trade |
| OrderTicket | MQL4 | 实现 | Trade |
| OrderSymbol | MQL4 | 实现 | Trade |
| OrderType | MQL4 | 实现 | Trade |
| OrderLots | MQL4 | 实现 | Trade |
| OrderOpenPrice | MQL4 | 实现 | Trade |
| OrderStopLoss | MQL4 | 实现 | Trade |
| OrderTakeProfit | MQL4 | 实现 | Trade |
| OrderProfit | MQL4 | 实现 | Trade |
| OrderCommission | MQL4 | 实现 | Trade |
| OrderSwap | MQL4 | 实现 | Trade |
| OrderMagicNumber | MQL4 | 实现 | Trade |
| OrderComment | MQL4 | 实现 | Trade |
| OrderOpenTime | MQL4 | 实现 | Trade |
| OrderCloseTime | MQL4 | 实现 | Trade |
| OrderClosePrice | MQL4 | 实现 | Trade |

**MQL4 交易函数缺失清单**：

| builtin 名 | 版本 | 状态 | 所属类别 | 备注 |
|---|---|---|---|---|
| OrderCloseBy | MQL4 | 缺失 | Trade | 对冲平仓 |
| OrdersHistoryTotal | MQL4 | 缺失 | Trade | 历史订单总数 |
| OrderExpiration | MQL4 | 缺失 | Trade | 挂单过期时间 |
| OrderPrint | MQL4 | 缺失 | Trade | 订单信息打印 |
| MarketInfo | MQL4 | 缺失 | Market | 品种市场信息 |

## 10. MQL5 仓位函数（callTrade switch）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| PositionsTotal | MQL5 | 实现 | Trade |
| PositionGetTicket | MQL5 | 实现 | Trade |
| PositionSelectByTicket | MQL5 | 实现 | Trade |
| PositionGetSymbol | MQL5 | 实现 | Trade |
| PositionGetDouble | MQL5 | 实现 | Trade |
| PositionGetInteger | MQL5 | 实现 | Trade |
| PositionGetString | MQL5 | 实现 | Trade |

## 11. Account 函数（callTrade switch）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| AccountBalance | 共享 | 实现 | Account |
| AccountEquity | 共享 | 实现 | Account |
| AccountFreeMargin | 共享 | 实现 | Account |
| AccountMargin | 共享 | 实现 | Account |
| AccountLeverage | 共享 | 实现 | Account |

## 12. MQL5 CTrade 类方法（execCTrade switch）

| 方法名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| CTrade.Buy | MQL5 | 实现 | Trade |
| CTrade.Sell | MQL5 | 实现 | Trade |
| CTrade.BuyLimit | MQL5 | 实现 | Trade |
| CTrade.SellLimit | MQL5 | 实现 | Trade |
| CTrade.BuyStop | MQL5 | 实现 | Trade |
| CTrade.SellStop | MQL5 | 实现 | Trade |
| CTrade.PositionClose | MQL5 | 实现 | Trade |
| CTrade.PositionClosePartial | MQL5 | 实现 | Trade |
| CTrade.PositionCloseBy | MQL5 | 实现 | Trade |
| CTrade.PositionModify | MQL5 | 实现 | Trade |
| CTrade.OrderDelete | MQL5 | 实现 | Trade |
| CTrade.SetExpertMagicNumber | MQL5 | 实现 | Trade |
| CTrade.SetDeviationInPoints | MQL5 | 实现 | Trade |

## 13. 指标函数 — 完整实现（callIndicator switch）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| iMA | 共享 | 实现 | Indicator |
| iRSI | 共享 | 实现 | Indicator |
| iATR | 共享 | 实现 | Indicator |
| iMACD | 共享 | 实现 | Indicator |
| iBands / iBollinger | 共享 | 实现 | Indicator |
| iStochastic | 共享 | 实现 | Indicator |
| iCCI | 共享 | 实现 | Indicator |
| iADX | 共享 | 实现 | Indicator |
| iMFI | 共享 | 实现 | Indicator |
| iOBV | 共享 | 实现 | Indicator |
| iSAR | 共享 | 实现 | Indicator |
| iStdDev | 共享 | 实现 | Indicator |
| iWPR | 共享 | 实现 | Indicator |
| iMomentum | 共享 | 实现 | Indicator |

## 14. 指标函数 — 共享 Stub（callIndicator dispatch，SDK 返回 0）

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| iAlligator | 共享 | stub | Indicator |
| iIchimoku | 共享 | stub | Indicator |
| iEnvelopes | 共享 | stub | Indicator |
| iDeMarker | 共享 | stub | Indicator |
| iOsMA | 共享 | stub | Indicator |
| iRVI | 共享 | stub | Indicator |
| iForce | 共享 | stub | Indicator |
| iFractals | 共享 | stub | Indicator |
| iGator | 共享 | stub | Indicator |
| iAC | 共享 | stub | Indicator |
| iAD | 共享 | stub | Indicator |
| iAO | 共享 | stub | Indicator |
| iBearsPower | 共享 | stub | Indicator |
| iBullsPower | 共享 | stub | Indicator |
| iBWMFI | 共享 | stub | Indicator |

## 15. 指标函数 — MQL5-only Stub

| builtin 名 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| iAMA | MQL5 | stub | Indicator |
| iDEMA | MQL5 | stub | Indicator |
| iTEMA | MQL5 | stub | Indicator |
| iFrAMA | MQL5 | stub | Indicator |
| iVIDyA | MQL5 | stub | Indicator |
| iTriX | MQL5 | stub | Indicator |
| iADXWilder | MQL5 | stub | Indicator |
| iChaikin | MQL5 | stub | Indicator |
| iVolumes | MQL5 | stub | Indicator |

## 16. MQL 预定义常量（mqlConstants map）

| 常量组 | 版本 | 状态 | 所属类别 |
|---|---|---|---|
| OP_BUY / OP_SELL / OP_BUYLIMIT / OP_SELLLIMIT / OP_BUYSTOP / OP_SELLSTOP | MQL4 | 实现 | Constant |
| SELECT_BY_POS / SELECT_BY_TICKET / MODE_TRADES / MODE_HISTORY | MQL4 | 实现 | Constant |
| PRICE_CLOSE / PRICE_OPEN / PRICE_HIGH / PRICE_LOW / PRICE_MEDIAN / PRICE_TYPICAL / PRICE_WEIGHTED | 共享 | 实现 | Constant |
| PERIOD_M1 ~ PERIOD_MN1 | 共享 | 实现 | Constant |
| TRADE_ACTION_DEAL / TRADE_ACTION_PENDING / TRADE_ACTION_SLTP / TRADE_ACTION_PEND_CLOSE | MQL5 | 实现 | Constant |
| ORDER_TYPE_BUY / ORDER_TYPE_SELL / ORDER_TYPE_BUY_LIMIT / ORDER_TYPE_SELL_LIMIT / ORDER_TYPE_BUY_STOP / ORDER_TYPE_SELL_STOP | MQL5 | 实现 | Constant |
| EMPTY / INVALID_HANDLE / CLR_NONE | 共享 | 实现 | Constant |
| true / false | 共享 | 实现 | Constant |

---

## 17. 缺失 Builtin 清单（按版本分类）

### MQL4 缺失

| builtin 名 | 所属类别 | 备注 |
|---|---|---|
| OrderCloseBy | Trade | 对冲平仓 |
| OrdersHistoryTotal | Trade | 历史订单总数 |
| OrderExpiration | Trade | 挂单过期时间 |
| OrderPrint | Trade | 订单信息打印 |
| MarketInfo | Market | 品种市场信息（点差、最小手数等） |
| iBars / iClose / iOpen / iHigh / iLow / iTime / iVolume | Indicator/Market | 多品种序列访问 |
| iBarShift | Indicator | 按时间查找 bar 索引 |
| iHighest / iLowest | Indicator | 最高/最低价 bar 索引 |
| iCustom | Indicator | 自定义指标（永久盲区） |
| *OnArray 变体 | Indicator | 基于数组的指标计算 |
| MathFloor / MathCeil / MathRound / MathMod | Math | 取整与取模 |
| MathLog / MathExp | Math | 对数与指数 |
| MathSin / MathCos / MathTan / MathArctan / MathArcsin / MathArccos | Math | 三角函数 |
| StringFormat | String | 格式化字符串 |
| StringCompare | String | 字符串比较 |
| StringToLower / StringToUpper | String | 大小写转换 |
| CharToStr / StrToDouble / StrToInteger / StrToTime | String | 旧式字符串转换 |
| TimeDay / TimeMonth / TimeYear / TimeHour / TimeMinute / TimeSeconds | Datetime | 时间分量分解 |
| TimeDayOfWeek / TimeDayOfYear | Datetime | 星期/年中第几天 |
| Day / DayOfWeek / DayOfYear / Hour / Minute / Month / Year / Seconds | Datetime | 当前时间分量 |
| AccountCurrency / AccountCompany / AccountName / AccountNumber / AccountProfit / AccountServer | Account | 账户扩展信息 |
| AccountFreeMarginCheck | Account | 预扣保证金检查 |
| IsTesting / IsOptimization / IsVisualMode / IsDemo / IsLive | Platform | 运行环境状态 |
| IsTradeAllowed / IsTradeContextBusy | Platform | 交易权限状态 |
| GetLastError / ResetLastError | Platform | 错误处理 |
| EventSetTimer / EventSetMillisecondTimer / EventKillTimer | Platform | 定时器事件（recognizer 解析但解释器无 builtin） |
| FileOpen / FileClose / FileWrite / FileRead / FileDelete | File | 文件 I/O |
| ObjectCreate / ObjectDelete / ObjectSet / ObjectGet | Chart | 图表对象 |
| SetIndexBuffer / IndicatorCounted | Indicator | 指标缓冲区 |
| SendMail / SendNotification / PlaySound | Alert | 通知与声音 |

### MQL5 缺失

| builtin 名 | 所属类别 | 备注 |
|---|---|---|
| OrderSend(MqlTradeRequest, MqlTradeResult) | Trade | 原生 MQL5 下单（盲区，CTrace 已覆盖常用场景） |
| OrderSelect / OrderGetTicket / OrderGetDouble / OrderGetInteger / OrderGetString | Trade | 挂单管理 |
| OrdersTotal（MQL5 挂单） | Trade | 挂单总数（与 MQL4 语义不同） |
| HistoryOrderSelect / HistoryOrderGetTicket / HistoryOrderGet* | Trade | 历史订单 |
| HistoryDealSelect / HistoryDealGetTicket / HistoryDealGet* | Trade | 历史成交 |
| HistoryDealsTotal / HistoryOrdersTotal | Trade | 历史总数 |
| SymbolInfoDouble / SymbolInfoInteger / SymbolInfoString | Market | 品种信息 |
| SymbolInfoTick | Market | 最新 tick |
| SymbolSelect | Market | 市场观察品种选择 |
| CopyRates / CopyClose / CopyOpen / CopyHigh / CopyLow / CopyTime / CopyVolume | Market | 多品种/多周期数据复制 |
| CopyBuffer | Indicator | 指标缓冲区复制 |
| iBars / iClose / iOpen / iHigh / iLow / iTime / iVolume | Indicator/Market | 多品种序列访问（MQL5 版） |
| iBarShift / iHighest / iLowest | Indicator | bar 工具函数 |
| iCustom | Indicator | 自定义指标（永久盲区） |
| IndicatorCreate / IndicatorRelease / IndicatorParameters | Indicator | 指标句柄管理 |
| EventSetTimer / EventSetMillisecondTimer / EventKillTimer | Platform | 定时器事件 |
| OnTrade / OnTradeTransaction / OnBookEvent | Event | 事件回调（永久盲区） |
| OnTesterInit / OnTesterDeinit / OnTesterPass | Event | 测试器事件（永久盲区） |
| MathFloor / MathCeil / MathRound / MathMod | Math | 取整与取模 |
| MathLog / MathExp | Math | 对数与指数 |
| StringFormat | String | 格式化字符串 |
| TimeDay / TimeMonth / TimeYear / TimeHour / TimeMinute | Datetime | 时间分量分解 |
| AccountInfoDouble / AccountInfoInteger / AccountInfoString | Account | MQL5 账户信息 |
| TerminalInfoString / TerminalInfoInteger | Platform | 终端信息 |
| MqlDateTime / MqlRates / MqlTick 结构体字段访问 | Struct | 结构体字段读写 |
| CPositionInfo / CSymbolInfo / CAccountInfo / COrderInfo 类 | Class | 包装类方法 |
| ChartOpen / ChartClose / ChartNavigate | Chart | 图表操作 |
| FileOpen / FileClose / FileWrite / FileRead | File | 文件 I/O |
| ObjectCreate / ObjectDelete / ObjectSet | Chart | 图表对象 |
| SendMail / SendNotification / PlaySound | Alert | 通知与声音 |

---

## 18. 统计摘要

| 分类 | 实现 | stub | 缺失 | 合计 |
|---|---|---|---|---|
| Math | 5 | 0 | 10 | 15 |
| Platform | 4 | 0 | 12 | 16 |
| Array | 8 | 0 | 0 | 8 |
| String | 8 | 0 | 7 | 15 |
| Conversion | 5 | 0 | 0 | 5 |
| Datetime | 2 | 0 | 12 | 14 |
| Market | 6 | 0 | 8 | 14 |
| Series | 6 | 0 | 0 | 6 |
| Trade (MQL4) | 21 | 0 | 5 | 26 |
| Trade (MQL5 Position) | 7 | 0 | 0 | 7 |
| Trade (MQL5 CTrade) | 13 | 0 | 1 | 14 |
| Trade (MQL5 Order/History) | 0 | 0 | 12 | 12 |
| Account | 5 | 0 | 9 | 14 |
| Indicator (实现) | 14 | 0 | 0 | 14 |
| Indicator (stub) | 0 | 15+9=24 | 0 | 24 |
| Indicator (缺失) | 0 | 0 | 6 | 6 |
| Constant | 30+ | 0 | 0 | 30+ |
| **合计** | **134** | **24** | **82** | **240** |

> **覆盖率**：已实现 134 + stub 24 = 158 / 240 ≈ **65.8%**（含 stub）；纯实现 134 / 240 ≈ **55.8%**
> **有效覆盖率**（排除永久盲区 iCustom/OnArray/事件回调/FileIO/ChartObject）：约 **78%**

---

## 19. 关键发现

1. **CTrade 覆盖完整**：MQL5 `CTrade` 类的常用交易方法全部实现，仅缺原生 `OrderSend(MqlTradeRequest)` — 但实际 EA 使用 `CTrade` 覆盖率极高，原生 `OrderSend` 可标为低优先级盲区。

2. **Stub 指标有 dispatch 无计算**：24 个 stub 指标在解释器层有正确的 `callIndicator` dispatch，但 SDK 层 `runner/indicators.go` 和 `backtest/engine.go` 返回 0。策略中使用这些指标会得到 0 值，不会崩溃但结果错误。

3. **EventSetTimer 是解释器盲区**：`recognizer_misc.go` 在 gen.go 路径解析了 `EventSetTimer`，但解释器 `builtinTable` 中无此函数。MQL 源码中 `EventSetTimer(60)` 在 OnInit 中调用时，解释器会触发 unimplemented 路径。`OnTimer` 回调本身已实现（`exec.go:357`），但定时器设置函数缺失。

4. **多品种数据访问完全缺失**：`iBars`/`iClose`/`iOpen`/`iHigh`/`iLow`/`iTime`/`iVolume` 和 MQL5 `CopyRates`/`CopyClose` 等多品种/多周期函数均未实现。当前仅支持当前品种的 `Close[n]`/`Open[n]` 等 series 访问。

5. **Math 函数缺口大**：MQL 有 ~20 个 Math 函数，解释器仅实现 5 个。缺少 `MathFloor`/`MathCeil`/`MathRound` 等常用函数。

6. **Datetime 分解函数缺失**：`TimeDay`/`TimeHour`/`TimeMonth` 等时间分量提取函数完全缺失，影响基于时间的策略逻辑。

7. **版本隔离正确**：MQL4 交易函数（`Order*`）和 MQL5 仓位函数（`Position*`）在同一个 `callTrade` switch 中，但通过不同的函数名自然隔离。`CTrade` 方法通过 `dispatchClassMethod` 独立路径分发。无交叉污染。
