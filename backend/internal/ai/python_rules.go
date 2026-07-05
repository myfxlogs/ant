// Package ai — python_rules.go
// Shared Python subset rules and agent discipline prompts.
// Single source of truth used by both the Generator agent (internal/agent/)
// and the chat agent (internal/connect/ai/strategy_plan_handler.go).

package ai

// PythonSubsetRules is the canonical set of Python subset language rules,
// SDK API mappings, and iron constraints. MUST stay in sync with:
//   - compile_py_mapping.go (Python SDK → VM builtin mapping)
//   - interp/builtin_registry.go (VM implementation source of truth)
const PythonSubsetRules = `## ⛔ IRON RULES — violating ANY of these = code REJECTED ⛔

### Required Code Skeleton (EVERY strategy MUST follow this exactly):
` + "```python" + `
from decimal import Decimal

class MyStrategy(StrategyBase):
    def __init__(self, period: int = 14, lot: Decimal = Decimal("0.1")) -> None:
        self.period: int = period
        self.lot: Decimal = lot

    def on_bar(self, ctx: BarContext) -> None:
        # strategy logic here
        pass

    def on_deinit(self, ctx: BarContext, reason: str) -> None:
        pass
` + "```" + `

### Type Annotations — MANDATORY, no exceptions:
- __init__ MUST have -> None return type: def __init__(self, period: int = 14) -> None:
- EVERY __init__ parameter MUST have a type annotation (int, float, bool, str, Decimal)
- EVERY method MUST have a return type annotation (-> None, -> int, -> float, -> Decimal)
- EVERY local variable MUST have a type annotation: ema_fast: float = ctx.indicators.ima(...)

### What IS Allowed:
- from decimal import Decimal (ONLY this import)
- if/elif/else, for i in range(N), while cond, break, continue
- float for indicator values (EMA, RSI, ATR, etc.)
- Decimal for all prices, volumes, P&L, stop-loss, take-profit
- ctx.* SDK calls (indicators, broker, bars, positions)
- None, True, False literals
- Arithmetic, comparison, and logic operators

### What is FORBIDDEN (code will be REJECTED):
- Missing type annotations on __init__ params, methods, or local variables
- import anything other than "from decimal import Decimal"
- list comprehensions, lambda, try/except, with, yield, decorators, async/await
- exec, eval, open, print, len, sorted, sum, enumerate, zip (outside for-loops)
- f-strings, walrus operator (:=), global/nonlocal, del, assert, raise
- slicing, tuple unpacking, *args, **kwargs, multiple inheritance
- float for prices or volumes — use Decimal

## SDK API Mapping
### Market Data
- Close[0] → ctx.bars().close(0)
- Open[0] → ctx.bars().open(0)
- High[0] → ctx.bars().high(0)
- Low[0] → ctx.bars().low(0)
- Volume[0] → ctx.bars().volume(0)
- Time[0] → ctx.bars().time(0)
- Bid → ctx.bid()
- Ask → ctx.ask()
- Point → ctx.point()
- Digits → ctx.digits()
- Spread → ctx.spread()
- Symbol() → ctx.symbol()

### Indicators (all map to ctx.indicators().<name>(ctx.symbol(), period, shift))
- iMA → ctx.indicators().ima(ctx.symbol(), period, shift)
- iRSI → ctx.indicators().irsi(ctx.symbol(), period, shift)
- iATR → ctx.indicators().iatr(ctx.symbol(), period, shift)
- iBands / iBollinger → ctx.indicators().ibands(ctx.symbol(), period, shift)
- iMACD → ctx.indicators().imacd(ctx.symbol(), fast, slow, signal, shift)
- iStochastic → ctx.indicators().istochastic(ctx.symbol(), kperiod, dperiod, shift)
- iCCI → ctx.indicators().icci(ctx.symbol(), period, shift)
- iADX → ctx.indicators().iadx(ctx.symbol(), period, shift)
- iMomentum → ctx.indicators().imomentum(ctx.symbol(), period, shift)
- iWPR → ctx.indicators().iwpr(ctx.symbol(), period, shift)
- iMFI → ctx.indicators().imfi(ctx.symbol(), period, shift)
- iOBV → ctx.indicators().iobv(ctx.symbol(), period, shift)
- iSAR → ctx.indicators().isar(ctx.symbol(), step, max, shift)
- iStdDev → ctx.indicators().istddev(ctx.symbol(), period, shift)
- iAlligator → ctx.indicators().ialligator(ctx.symbol(), jaw, teeth, lips, shift)
- iIchimoku → ctx.indicators().iichimoku(ctx.symbol(), tenkan, kijun, senkou, shift)
- iEnvelopes → ctx.indicators().ienvelopes(ctx.symbol(), period, deviation, shift)
- iDeMarker → ctx.indicators().idemarker(ctx.symbol(), period, shift)
- iOsMA → ctx.indicators().iosma(ctx.symbol(), fast, slow, signal, shift)
- iRVI → ctx.indicators().irvi(ctx.symbol(), period, shift)
- iForce → ctx.indicators().iforce(ctx.symbol(), period, shift)
- iFractals → ctx.indicators().ifractals(ctx.symbol(), shift)
- iGator → ctx.indicators().igator(ctx.symbol(), jaw, teeth, lips, shift)
- iAC → ctx.indicators().iac(shift)
- iAD → ctx.indicators().iad(shift)
- iAO → ctx.indicators().iao(shift)
- iBearsPower → ctx.indicators().ibearspower(ctx.symbol(), period, shift)
- iBullsPower → ctx.indicators().ibullspower(ctx.symbol(), period, shift)
- iBWMFI → ctx.indicators().ibwmfi(ctx.symbol(), period, shift)
### MQL5-only Indicators
- iAMA → ctx.indicators().iama(ctx.symbol(), period, fast, slow, shift)
- iDEMA → ctx.indicators().idema(ctx.symbol(), period, shift)
- iTEMA → ctx.indicators().itema(ctx.symbol(), period, shift)
- iFrAMA → ctx.indicators().iframa(ctx.symbol(), period, shift)
- iVIDyA → ctx.indicators().ividya(ctx.symbol(), period, cmo_period, shift)
- iTriX → ctx.indicators().itrix(ctx.symbol(), period, shift)
- iADXWilder → ctx.indicators().iadxwilder(ctx.symbol(), period, shift)
- iChaikin → ctx.indicators().ichaikin(ctx.symbol(), fast, slow, shift)
- iVolumes → ctx.indicators().ivolumes(ctx.symbol(), shift)

### Trading & Account
- Buy → ctx.broker().buy(lot=Decimal("0.1"))
- Sell → ctx.broker().sell(lot=Decimal("0.1"))
- Close position → ctx.broker().close(ticket)
- Modify position → ctx.broker().modify(ticket, sl, tp)
- Delete order → ctx.broker().delete(ticket)
- Buy Limit → ctx.broker().buy_limit(lot, price, sl, tp)
- Sell Limit → ctx.broker().sell_limit(lot, price, sl, tp)
- Buy Stop → ctx.broker().buy_stop(lot, price, sl, tp)
- Sell Stop → ctx.broker().sell_stop(lot, price, sl, tp)
- Position count → ctx.positions().count()
- Iterate positions → for pos in ctx.positions: pos.ticket, pos.profit, pos.volume, pos.sl, pos.tp
- AccountBalance() → ctx.account().balance()
- AccountEquity() → ctx.account().equity()
- AccountMargin() → ctx.account().margin()
- AccountFreeMargin() → ctx.account().free_margin()
- AccountProfit() → ctx.account().profit()
- AccountLeverage() → ctx.account().leverage()

## ⚠️ PRE-OUTPUT SELF-CHECK — run this checklist BEFORE you output code ⚠️
If your code violates ANY of these, FIX IT before outputting. No exceptions.

1. ✅ Only import is "from decimal import Decimal" — NO open, NO os, NO sys, NO math, NO numpy, NO pandas
2. ✅ NO built-in functions: open, print, exec, eval, len, sorted, sum, range (outside for-loop), enumerate, zip
3. ✅ NO f-strings — use string concatenation with + 
4. ✅ NO list comprehensions — use explicit for loops
5. ✅ NO try/except, with, lambda, decorators, async/await
6. ✅ NO slicing (x[1:3]), tuple unpacking, *args, **kwargs
7. ✅ ALL __init__ params have type annotations, ALL methods have -> return type
8. ✅ Prices/volumes use Decimal, NOT float — float only for indicator return values
9. ✅ Code follows the exact skeleton: class MyStrategy(StrategyBase) with __init__ and on_bar

VIOLATING ANY RULE ABOVE = CODE WILL BE REJECTED. Do not output code you have not verified.`

