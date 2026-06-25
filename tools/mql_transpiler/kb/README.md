# MQL → Python Transpiler Knowledge Base

Catalog of MQL4/MQL5 language features mapped to transpiler coverage.
Gap patterns accumulate here; each becomes a fixture for regression testing.

## Confidence levels

| Level | Threshold | Meaning |
|-------|-----------|---------|
| 🟢 FULL | `>= 95%` | Fully handled — no gaps expected |
| 🟡 PARTIAL | `85% – 95%` | Core patterns work, edge cases may gap |
| 🔴 GAP | `< 85%` | Known gaps — manual review required |
| ⚫ UNSUPPORTED | N/A | Not a target (GUI, File I/O, Network, DLL) |

## Feature coverage matrix

### 1. Program structure

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| Entry point | `int start()` | — deprecated | 🟢 FULL |
| Init | `int OnInit()` | `def on_init(self)` | 🟢 FULL |
| Tick | `void OnTick()` | `def on_tick(self)` | 🟢 FULL |
| Timer | `void OnTimer()` | `def on_timer(self)` | 🟢 FULL |
| Trade event | `void OnTrade()` | `def on_trade(self)` | 🟡 PARTIAL |
| Deinit | `void OnDeinit(const int r)` | `def on_deinit(self, reason)` | 🟢 FULL |
| Calculate | `int OnCalculate(...)` | `def on_bar(self, timeframe)` | 🟢 FULL |
| Preprocessor | `#property`, `#include` | — ignored | 🟢 FULL |

### 2. Data types

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| Integers | `int`, `long`, `uint` | `int` | 🟢 FULL |
| Floats | `double`, `float` | `Decimal(str(x))` | 🟢 FULL |
| Boolean | `bool` | `bool` | 🟢 FULL |
| String | `string` | `str` | 🟢 FULL |
| Color | `color` | `str` (hex) | 🟡 PARTIAL |
| Datetime | `datetime` | `int` (unix_ms) | 🟢 FULL |
| Enum | `ENUM_*` | Enum classes | 🟡 PARTIAL |
| Arrays | `double arr[]` | `list` | 🟡 PARTIAL |
| Structs | `struct {...}` | `dataclass` | 🔴 GAP |

### 3. Variables & assignment

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| Global scalar | `double x = 1.0;` | `self.x = 1.0` | 🟢 FULL |
| Extern param | `extern int x = 5;` | `self.ctx.param('x', 5)` | 🟢 FULL |
| Input param | `input double x = 0.1;` | `self.ctx.param('x', 0.1)` | 🟢 FULL |
| Local scalar | `int x = 0;` | `x = 0` | 🟢 FULL |
| Array init | `double arr[10];` | `[0.0]*10` | 🔴 GAP |
| Array resize | `ArrayResize(arr, 20)` | `arr.extend([0]*10)` | 🔴 GAP |

### 4. Control flow

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| If | `if (cond) { ... }` | `if cond:` | 🟢 FULL |
| Else if | `} else if (cond) {` | `elif cond:` | 🟢 FULL |
| Else | `} else {` | `else:` | 🟢 FULL |
| For (int) | `for (int i=0; i<n; i++)` | — see OrdersTotal | 🟡 PARTIAL |
| For (OrdersTotal) | `for (i=0; i<OrdersTotal(); i++)` | `for order in self.broker.orders()` | 🟢 FULL |
| For (PositionsTotal) | `for (i=0; i<PositionsTotal(); i++)` | `for pos in self.broker.positions()` | 🟢 FULL |
| While | `while (cond) { ... }` | `while cond:` | 🟢 FULL |
| Switch | `switch(x) { case 1: ... }` | `if/elif` | 🟡 PARTIAL |
| Break/Continue | `break; continue;` | `break; continue` | 🟢 FULL |
| Return | `return; return x;` | `return; return x` | 🟢 FULL |

### 5. Trading functions

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| Market order | `OrderSend(OP_BUY, ...)` | `self.broker.order_send(OrderRequest(...))` | 🟢 FULL |
| Pending order | `OrderSend(OP_BUYLIMIT, ...)` | `self.broker.order_send(OrderRequest(...))` | 🟢 FULL |
| Close position | `OrderClose(ticket, ...)` | `self.broker.position_close(ticket)` | 🟢 FULL |
| Modify position | `OrderModify(ticket, ...)` | `self.broker.position_modify(ticket)` | 🟢 FULL |
| Delete pending | `OrderDelete(ticket)` | `self.broker.order_delete(ticket)` | 🟢 FULL |
| Close by (MT5) | `OrderCloseBy(ticket, opposite)` | — not implemented | 🔴 GAP |
| OrderSelect (pos) | `OrderSelect(i, SELECT_BY_POS)` | `for order in self.broker.orders()` | 🟢 FULL |
| OrderSelect (ticket) | `OrderSelect(ticket, SELECT_BY_TICKET)` | direct access | 🟡 PARTIAL |
| OrderMagicNumber | `OrderMagicNumber()` | `order.magic` | 🟢 FULL |
| OrderTicket | `OrderTicket()` | `order.ticket` | 🟢 FULL |
| OrderLots | `OrderLots()` | `order.volume` | 🟢 FULL |
| OrderProfit | `OrderProfit()` | `order.profit` | 🟢 FULL |
| OrderOpenPrice | `OrderOpenPrice()` | `order.open_price` | 🟢 FULL |
| OrderStopLoss | `OrderStopLoss()` | `order.sl` | 🟢 FULL |
| OrderTakeProfit | `OrderTakeProfit()` | `order.tp` | 🟢 FULL |
| OrderSymbol | `OrderSymbol()` | `order.symbol` | 🟢 FULL |
| OrderType | `OrderType()` | `order.type` | 🟢 FULL |
| OrderComment | `OrderComment()` | `order.comment` | 🟢 FULL |
| OrderCommission | `OrderCommission()` | `order.commission` | 🟢 FULL |
| OrderSwap | `OrderSwap()` | `order.swap` | 🟢 FULL |
| OrderClosePrice | `OrderClosePrice()` | `order.close_price` | 🟢 FULL |

### 6. Account functions

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| AccountBalance | `AccountBalance()` | `self.broker.account().balance` | 🟢 FULL |
| AccountEquity | `AccountEquity()` | `self.broker.account().equity` | 🟢 FULL |
| AccountFreeMargin | `AccountFreeMargin()` | `self.broker.account().free_margin` | 🟢 FULL |
| AccountMargin | `AccountMargin()` | — | 🔴 GAP |
| AccountLeverage | `AccountLeverage()` | `self.broker.account().leverage` | 🟡 PARTIAL |
| AccountCurrency | `AccountCurrency()` | — | 🔴 GAP |
| AccountInfoDouble (MT5) | `AccountInfoDouble(ACCOUNT_PROFIT)` | — | 🔴 GAP |

### 7. Market data

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| Bid price | `Bid` | `self.ctx.bid` | 🟢 FULL |
| Ask price | `Ask` | `self.ctx.ask` | 🟢 FULL |
| Point size | `Point` | `self.ctx.point` | 🟢 FULL |
| Digits | `Digits` | `self.sym_info.digits` | 🟢 FULL |
| Symbol name | `Symbol()` | `self.ctx.symbol` | 🟢 FULL |
| OHLCV series | `Open[i], Close[i], etc.` | `bars.open[i], bars.close[i]` | 🟡 PARTIAL |
| iClose,iOpen,etc | `iClose(sym,tf,i)` | `bars.close[i]` | 🟡 PARTIAL |
| MarketInfo | `MarketInfo(sym, MODE_*)` | `sym_info.*` | 🔴 GAP |
| SymbolInfoDouble | `SymbolInfoDouble(sym, prop)` | `self.broker.symbol_info(sym)` | 🔴 GAP |

### 8. Technical indicators

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| iMA | `iMA(sym,tf,period,shift,method,price)` | `self.indicators.ma(...)` | 🟢 FULL |
| iRSI | `iRSI(sym,tf,period,price,shift)` | `self.indicators.rsi(...)` | 🟢 FULL |
| iBands | `iBands(sym,tf,period,dev,shift,price)` | `self.indicators.bands(...)` | 🟢 FULL |
| iMACD | `iMACD(sym,tf,fast,slow,sig,price,shift)` | `self.indicators.macd(...)` | 🟢 FULL |
| iATR | `iATR(sym,tf,period,shift)` | `self.indicators.atr(...)` | 🟢 FULL |
| iStochastic | `iStochastic(...)` | `self.indicators.stochastic(...)` | 🟢 FULL |
| iCCI | `iCCI(sym,tf,period,price,shift)` | `self.indicators.cci(...)` | 🟢 FULL |
| iCustom | `iCustom(sym,tf,name,...)` | `self.indicators.i_custom(...)` | 🟢 FULL |
| iADX, iMomentum, iMFI, etc. | 10+ indicators | — not yet mapped | 🔴 GAP |

### 9. Common functions

| Feature | MQL | Python SDK | Status |
|---------|-----|-----------|--------|
| Print | `Print("msg")` | `self.ctx.log("msg")` | 🟢 FULL |
| MathAbs | `MathAbs(x)` | `abs(x)` | 🟢 FULL |
| MathMax/Min | `MathMax(a,b)` | `max(a,b)` | 🟢 FULL |
| MathRound | `MathRound(x)` | `round(x)` | 🟡 PARTIAL |
| String functions | `StringConcatenate, etc.` | `str` methods | 🟡 PARTIAL |
| EventSetTimer | `EventSetTimer(sec)` | `self.ctx.set_timer(sec)` | 🟢 FULL |
| EventKillTimer | `EventKillTimer()` | `self.ctx.kill_timer()` | 🟢 FULL |
| Sleep | `Sleep(ms)` | — not supported | 🔴 GAP |
| GetTickCount | `GetTickCount()` | `time.time()` | 🟡 PARTIAL |

### 10. Explicitly unsupported (⚫ UNSUPPORTED)

These MQL features have no equivalent in the Python SDK and will never be translated:

- **GUI objects**: `ObjectCreate`, `ObjectSet*`, `ObjectGet*`, `Chart*`
- **File I/O**: `FileOpen`, `FileRead`, `FileWrite`, `FileClose`
- **Network**: `WebRequest`, `SendFTP`, `SendMail`, `SendNotification`
- **DLL/External**: `#import`, `#resource`, DLL calls, `IndicatorCreate`
- **Custom events**: `EventChartCustom`, `OnChartEvent`

---

## How to add a new pattern

1. Find the MQL source snippet that produces a TRANSPILER-GAP
2. Create a minimal `.mq4` fixture in `fixtures/<category>/`
3. Create the expected `.py` output in the same directory
4. Add a test in `../tests/test_transpiler.py`
5. Update this catalog — change status from 🔴 to 🟡 or 🟢
6. Run `PYTHONPATH=. python3 -m unittest tools.mql_transpiler.tests.test_transpiler -v`

## Automatic KB updates

The `gap_patterns.json` file in the parent directory auto-accumulates
unknown patterns from every CLI run. Review it periodically to identify
high-frequency gaps that should be prioritized.
