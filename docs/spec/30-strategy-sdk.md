# 30 · Strategy SDK Specification

> **关联 ADR**: ADR-0020 (EA 完全替代)
> **关联 spec**: `docs/spec/31-risk-gate.md` (风控门协议)
> **关联任务**: T0.1 / T0.2 (Phase 0 契约冻结)
> **状态**: Frozen (Phase 0 contract)
> **实现落点**: `strategy-service/app/sdk/`

## 1. Purpose

The Strategy SDK is the **single programming surface** that MQL→SDK translation targets, and that the unified strategy runtime (USR) drives. It mirrors the two things every EA depends on:

1. **Event model**: `OnInit / OnTick / OnTimer / OnTrade / OnDeinit`
2. **Broker semantics**: orders, positions, deals, margin, execution rules, netting/hedging

By faithfully modeling these two domains, MQL translation degrades to mechanical mapping, and behavioral fidelity with MT is achieved at the SDK layer rather than per-strategy.

**Key invariant (D3)**: The same strategy code runs under SimBroker (backtest/paper) and LiveBroker (real). The strategy never knows which broker is injected — it only sees the `Broker` interface.

## 2. Package Structure

```
strategy-service/app/sdk/
├── __init__.py        # public re-exports
├── strategy_base.py   # StrategyBase — lifecycle hooks
├── context.py         # Context — bars / params / timer
├── broker.py          # Broker — abstract trade interface (T0.2)
├── symbol.py          # SymbolInfo — instrument metadata
├── account.py         # AccountInfo — account state
├── series.py          # Series / Bars — MQL reverse-indexed price data
├── indicators.py      # Indicators — technical indicator facade
└── types.py           # enums + dataclasses (OrderRequest, Position, …)
```

## 3. Lifecycle (StrategyBase)

```python
class Strategy(StrategyBase):
    def on_init(self) -> None:        ...
    def on_tick(self) -> None:        ...
    def on_bar(self, timeframe: str) -> None:  ...
    def on_timer(self) -> None:       ...
    def on_trade(self) -> None:       ...
    def on_deinit(self, reason: str) -> None:  ...
```

| Hook | MQL Equivalent | Trigger |
|------|---------------|---------|
| `on_init()` | `OnInit` | Worker start. Register params, init state, call `set_timer()`. |
| `on_tick()` | `OnTick` | Every tick from broker feed (live) or tick sim (backtest). Primary entry for event-driven EAs. |
| `on_bar(timeframe)` | `OnCalculate`-equivalent | New bar closed (derived from ticks). Bar-type strategy entry. |
| `on_timer()` | `OnTimer` | Periodic, gated by `ctx.set_timer(seconds)`. |
| `on_trade()` | `OnTrade` | After any trade event (fill, partial close, SL/TP hit). |
| `on_deinit(reason)` | `OnDeinit` | Worker shutdown. `reason` is one of `"user_stop"`, `"kill_switch"`, `"error"`, `"schedule_end"`. |

**Order**: `on_init → (on_tick / on_bar / on_timer / on_trade)* → on_deinit`. The runtime guarantees:
- `on_init` is called exactly once before any other hook.
- `on_deinit` is called exactly once at shutdown (best-effort; may be skipped on SIGKILL).
- Strategy exceptions are caught by the runtime; a crash in one hook does not bring down the worker.

The runtime injects three attributes after construction:
- `self.broker: Broker` — all trade operations
- `self.ctx: Context` — bars, params, timer registration
- `self.indicators: Indicators` — technical indicators

## 4. Context

Available as `self.ctx` inside any lifecycle hook.

| Method | Returns | Description |
|--------|---------|-------------|
| `bars(timeframe=None)` | `Bars` | Price series for a timeframe; `None` = primary. |
| `param(name, default=None)` | `object` | Strategy parameter (equivalent to MQL `extern`/`input`). |
| `set_timer(seconds)` | `None` | Register periodic `on_timer()` callback (≥ 1s). |
| `kill_timer()` | `None` | Unregister timer. |

A secondary timeframe is retrieved as e.g. `self.ctx.bars("H1")`. The runtime must provide at least the primary timeframe; others are best-effort.

## 5. Series & Bars (MQL Reverse Indexing)

**Critical fidelity point**: Series use **MQL reverse indexing** where `series[0]` = current bar, `series[1]` = previous bar, etc. This matches MQL's default `series=true` array behavior and is essential for correct translation.

```python
bars = self.ctx.bars()           # primary timeframe
bars.close[0]                     # current bar close
bars.close[1]                     # previous bar close
bars.close.slice(14)              # last 14 closes (oldest→newest, for indicators)
bars.total()                      # available bar count
```

Each `Bars` object provides:
| Sequence | Type | Description |
|----------|------|-------------|
| `open` | `Series` | Bar open prices |
| `high` | `Series` | Bar high prices |
| `low` | `Series` | Bar low prices |
| `close` | `Series` | Bar close prices |
| `volume` | `Series` | Bar tick volumes |
| `time` | `Series` | Bar open timestamps (unix_ms) |

`Series` supports:
- `series[shift: int] -> float` — reverse-indexed access
- `len(series) -> int` — total bars available
- `series.slice(count: int) -> List[float]` — most recent `count` bars, forward order (for indicator input)

**Note**: Series values are `float` (matching MQL's `double` precision for indicator computation). Financial amounts (order prices, volumes, balances) use `Decimal` at the broker boundary. This distinction is intentional: indicator math prefers float performance; money math requires Decimal exactness.

## 6. Symbol Metadata (SymbolInfo)

Immutable dataclass representing instrument trading specifications. All price/amount fields are `Decimal`.

| Field | Type | MQL Equivalent | Description |
|-------|------|---------------|-------------|
| `name` | `str` | `Symbol()` | Canonical broker symbol |
| `digits` | `int` | `Digits()` | Decimal places in quote |
| `point` | `Decimal` | `Point()` | Min price increment (`10^-digits`) |
| `tick_size` | `Decimal` | `SYMBOL_TRADE_TICK_SIZE` | Min price step |
| `tick_value` | `Decimal` | `SYMBOL_TRADE_TICK_VALUE` | Account-currency value per tick |
| `contract_size` | `Decimal` | `SYMBOL_TRADE_CONTRACT_SIZE` | Base units per lot |
| `volume_min` | `Decimal` | `SYMBOL_VOLUME_MIN` | Min order volume |
| `volume_max` | `Decimal` | `SYMBOL_VOLUME_MAX` | Max order volume |
| `volume_step` | `Decimal` | `SYMBOL_VOLUME_STEP` | Volume granularity |
| `stops_level` | `int` | `SYMBOL_TRADE_STOPS_LEVEL` | Min SL/TP distance in points |
| `freeze_level` | `int` | `SYMBOL_TRADE_FREEZE_LEVEL` | Freeze distance in points |
| `swap_long` | `Decimal` | `SYMBOL_SWAP_LONG` | Overnight swap (long) |
| `swap_short` | `Decimal` | `SYMBOL_SWAP_SHORT` | Overnight swap (short) |
| `margin_rate` | `Decimal` | `SYMBOL_MARGIN_INITIAL` | Initial margin rate |

Two convenience methods (to be implemented):
- `normalize_price(price: Decimal) -> Decimal` — round to `digits`
- `normalize_volume(volume: Decimal) -> Decimal` — round to `volume_step`, clamp to `[volume_min, volume_max]`

## 7. Account State (AccountInfo)

Immutable dataclass. All amounts are `Decimal`.

| Field | Type | MQL Equivalent |
|-------|------|---------------|
| `balance` | `Decimal` | `AccountBalance()` |
| `equity` | `Decimal` | `AccountEquity()` |
| `margin` | `Decimal` | `AccountMargin()` |
| `free_margin` | `Decimal` | `AccountFreeMargin()` |
| `margin_level` | `Decimal` | `AccountMarginLevel()` (percent; ∞ → implementation-defined sentinel when no positions) |
| `leverage` | `int` | `AccountLeverage()` |
| `currency` | `str` | `AccountCurrency()` |
| `mode` | `AccountMode` | `netting` or `hedging` |

## 8. Indicators

Available as `self.indicators` inside any lifecycle hook. All methods read the currently visible bar window. Return values are `float` (computational domain).

| Method | MQL Equivalent | Signature |
|--------|---------------|-----------|
| `ma(period, shift, method)` | `iMA()` | `(period=14, shift=0, method="sma") -> float` |
| `ema(period, shift)` | `iMA(…, MODE_EMA)` | `(period=14, shift=0) -> float` |
| `rsi(period, shift)` | `iRSI()` | `(period=14, shift=0) -> float` |
| `bands(period, deviation, shift)` | `iBands()` | `(period=20, deviation=2.0, shift=0) -> (upper, middle, lower)` |
| `macd(fast, slow, signal, shift)` | `iMACD()` | `(fast=12, slow=26, signal=9, shift=0) -> (macd, signal, histogram)` |
| `atr(period, shift)` | `iATR()` | `(period=14, shift=0) -> float` |
| `stochastic(k_period, d_period, shift)` | `iStochastic()` | `(k=5, d=3, shift=0) -> (k, d)` |
| `cci(period, shift)` | `iCCI()` | `(period=14, shift=0) -> float` |
| `i_custom(name, params, buffer, shift)` | `iCustom()` | `(name, params=[], buffer=0, shift=0) -> float` |

**`i_custom` semantics**: Resolves a named custom indicator. If the indicator is not registered/translated, the implementation MUST raise a clear error — never silently return 0. `params` is a sequence of indicator parameters (periods, levels, etc.). `buffer` selects the indicator line (0 = main).

**`method` in `ma()`**: One of `"sma"`, `"ema"`, `"smma"`, `"lwma"` (matching MQL `ENUM_MA_METHOD`).

Additional indicators (ADX, MFI, OBV, etc.) may be added in Phase 1+; they are not required for Phase 0 contract freeze.

## 9. Trade API (Broker Interface)

All trade operations go through `self.broker`. The `Broker` abstract base class defines the contract; SimBroker and LiveBroker each implement it.

### 9.1 Order Types

| Enum Value | MT Equivalent | Description |
|------------|---------------|-------------|
| `OrderType.BUY` | `ORDER_TYPE_BUY` | Market buy |
| `OrderType.SELL` | `ORDER_TYPE_SELL` | Market sell |
| `OrderType.BUY_LIMIT` | `ORDER_TYPE_BUY_LIMIT` | Buy limit pending |
| `OrderType.SELL_LIMIT` | `ORDER_TYPE_SELL_LIMIT` | Sell limit pending |
| `OrderType.BUY_STOP` | `ORDER_TYPE_BUY_STOP` | Buy stop pending |
| `OrderType.SELL_STOP` | `ORDER_TYPE_SELL_STOP` | Sell stop pending |
| `OrderType.BUY_STOP_LIMIT` | `ORDER_TYPE_BUY_STOP_LIMIT` | Buy stop-limit pending |
| `OrderType.SELL_STOP_LIMIT` | `ORDER_TYPE_SELL_STOP_LIMIT` | Sell stop-limit pending |

### 9.2 Order Request

```python
@dataclass(frozen=True)
class OrderRequest:
    symbol: str
    type: OrderType
    volume: Decimal
    price: Optional[Decimal] = None      # None for market orders
    sl: Optional[Decimal] = None
    tp: Optional[Decimal] = None
    deviation: Optional[int] = None       # slippage tolerance in points
    magic: int = 0
    comment: str = ""
    type_filling: TypeFilling = TypeFilling.RETURN
    stop_limit_price: Optional[Decimal] = None  # only for *_STOP_LIMIT
```

### 9.3 Filling Modes

| Enum Value | MT Equivalent |
|------------|---------------|
| `TypeFilling.FOK` | `ORDER_FILLING_FOK` — Fill or Kill |
| `TypeFilling.IOC` | `ORDER_FILLING_IOC` — Immediate or Cancel |
| `TypeFilling.RETURN` | `ORDER_FILLING_RETURN` — Return (partial fills allowed, remainder canceled on session close) |

### 9.4 Order Result & Return Codes

```python
@dataclass(frozen=True)
class OrderResult:
    retcode: Retcode
    ticket: Optional[int] = None
    price: Optional[Decimal] = None    # actual fill price
    volume: Optional[Decimal] = None   # actual filled volume (≤ requested for partials)
    comment: str = ""
```

| Retcode | MT Equivalent | Meaning |
|---------|---------------|---------|
| `DONE` | `TRADE_RETCODE_DONE` | Fully executed |
| `DONE_PARTIAL` | `TRADE_RETCODE_DONE_PARTIAL` | Partially filled |
| `REQUOTE` | `TRADE_RETCODE_REQUOTE` | Price changed, must resend |
| `REJECTED` | `TRADE_RETCODE_REJECT` | Generic rejection |
| `NO_MONEY` | `TRADE_RETCODE_NO_MONEY` | Insufficient margin |
| `INVALID_VOLUME` | `TRADE_RETCODE_INVALID_VOLUME` | Volume out of range |
| `INVALID_STOPS` | `TRADE_RETCODE_INVALID_STOPS` | SL/TP too close |
| `MARKET_CLOSED` | `TRADE_RETCODE_MARKET_CLOSED` | Outside trading hours |
| `RISK_BLOCKED` | *(our extension)* | Blocked by Go risk gate (D6) |

### 9.5 Broker Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `order_send(request: OrderRequest)` | `OrderResult` | Submit any order type (market or pending). Must call risk gate internally (D6). |
| `position_modify(ticket, sl, tp)` | `OrderResult` | Modify SL/TP of an open position. |
| `position_close(ticket, volume=None)` | `OrderResult` | Close position; `volume=None` = full close, else partial. |
| `order_delete(ticket)` | `OrderResult` | Cancel a pending order. |
| `positions(symbol=None, magic=None)` | `List[Position]` | Query open positions; optionally filtered. |
| `orders(symbol=None, magic=None)` | `List[PendingOrder]` | Query pending orders; optionally filtered. |
| `account()` | `AccountInfo` | Current account state snapshot. |
| `symbol_info(symbol)` | `SymbolInfo` | Trading specifications for one instrument. |
| `server_time()` | `int` | Broker server time (unix_ms). Backtest returns current bar/tick time. |

### 9.6 Position & Pending Order

```python
@dataclass
class Position:
    ticket: int
    symbol: str
    side: PositionSide        # BUY or SELL
    volume: Decimal
    open_price: Decimal
    sl: Optional[Decimal] = None
    tp: Optional[Decimal] = None
    profit: Decimal = Decimal("0")
    swap: Decimal = Decimal("0")
    magic: int = 0
    comment: str = ""
    open_time_ms: int = 0

@dataclass
class PendingOrder:
    ticket: int
    symbol: str
    type: OrderType
    volume: Decimal
    price: Decimal
    sl: Optional[Decimal] = None
    tp: Optional[Decimal] = None
    magic: int = 0
    comment: str = ""
    placed_time_ms: int = 0
```

## 10. Account Modes

| Mode | Description |
|------|-------------|
| `NETTING` | One position per symbol; new trades net against existing (MT5 default, MT4 optional). |
| `HEDGING` | Multiple positions per symbol; each trade opens a separate position (MT4 default). |

Both SimBroker and LiveBroker must support both modes. The strategy can query `self.broker.account().mode` to adapt behavior.

## 11. Execution Modes

| Mode | Description |
|------|-------------|
| `MARKET` | Broker executes at current market price (default). |
| `INSTANT` | Broker requests quote, trader accepts/rejects (MT "Instant Execution"). |
| `REQUEST` | Request execution at specified price (MT "Request Execution"). |
| `EXCHANGE` | Order sent to exchange (MT "Exchange Execution"). |

Execution mode is a property of the broker connection, not per-order. The broker implementation decides which mode is active; the strategy SDK does not expose it as a parameter at Phase 0.

## 12. Anti-Float Rule

Per CLAUDE.md, all prices, volumes, and monetary amounts pass through the SDK as `Decimal`, never `float`. The only exception is indicator return values and Series element access, which use `float` to match MQL's `double` precision in the computational domain (indicator math, price comparisons within strategies). At the broker boundary (`OrderRequest`, `OrderResult`, `Position`, `AccountInfo`, `SymbolInfo`), all values are `Decimal`.

## 13. Verification

```bash
# Import check
cd strategy-service && python3 -c "from app.sdk import *; print('OK')"

# Type check (mypy)
python3 -m mypy app/sdk/ --ignore-missing-imports

# Verify all stubs are implementation-free
grep -rn "def " app/sdk/ | grep -v "raise NotImplementedError\|pass\|...\|-> None:\s*$"
# (should have no output — every method must be a stub)

# Verify no float in financial types
grep -rn "float" app/sdk/types.py app/sdk/account.py app/sdk/symbol.py
# (should have no output)
```
