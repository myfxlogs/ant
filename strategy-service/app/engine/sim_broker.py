"""SimBroker — Broker interface backed by the existing backtest engine (T1.2).

Implements ``app.sdk.broker.Broker`` by delegating to the mature engine modules:
  - ``portfolio.py`` — position tracking, PnL, SL/TP
  - ``fill.py`` — order queue, tick matching, partial fills
  - ``cost.py`` — commission + slippage
  - ``margin.py`` — leverage and margin calculations
  - ``market.py`` — tick simulation, bar views

Key responsibilities added by SimBroker on top of the engine:
  - Ticket management (monotonic, unique across orders and positions)
  - Magic number + comment tracking (engine types lack these fields)
  - Netting vs hedging account modes
  - Partial position close
  - Position SL/TP modification
  - Pending order cancellation
  - Decimal ↔ float conversion at the SDK boundary
"""

from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal
from typing import Callable, Dict, List, Optional

from app.engine.cost import CostModel
from app.engine.fill import FillModel
from app.engine.margin import MarginModel
from app.engine.market import MarketSimulator
from app.engine.portfolio import Portfolio
from app.engine.types import (
    CloseReason,
    Fill,
    Order,
    OrderStatus,
    OrderType as EngineOrderType,
    Position as EnginePosition,
    Side,
    Tick,
)
from app.sdk.account import AccountInfo, AccountMode
from app.sdk.broker import Broker
from app.sdk.symbol import SymbolInfo
from app.sdk.types import (
    Deal,
    OrderRequest,
    OrderResult,
    OrderType as SDKOrderType,
    PendingOrder,
    Position as SDKPosition,
    PositionSide,
    Retcode,
    TypeFilling,
)

# ── Decimal / float conversion helpers ──────────────────────────────────

_DECIMAL_PRECISION = Decimal("0.00001")  # 5 decimal places for prices


def _to_float(d: Optional[Decimal]) -> float:
    if d is None:
        return 0.0
    return float(d)


def _to_decimal(f: float, precision: Decimal = _DECIMAL_PRECISION) -> Decimal:
    if f == float("inf"):
        return Decimal("999999")
    if f == float("-inf"):
        return Decimal("-999999")
    return Decimal(str(f)).quantize(precision)


def _to_decimal_or_none(f: float) -> Optional[Decimal]:
    if f == 0.0:
        return None
    return _to_decimal(f)


# ── SDK ↔ Engine type mapping ──────────────────────────────────────────

_SDK_TO_ENGINE_ORDER: Dict[SDKOrderType, EngineOrderType] = {
    SDKOrderType.BUY: EngineOrderType.BUY,
    SDKOrderType.SELL: EngineOrderType.SELL,
    SDKOrderType.BUY_LIMIT: EngineOrderType.BUY_LIMIT,
    SDKOrderType.SELL_LIMIT: EngineOrderType.SELL_LIMIT,
    SDKOrderType.BUY_STOP: EngineOrderType.BUY_STOP,
    SDKOrderType.SELL_STOP: EngineOrderType.SELL_STOP,
    SDKOrderType.BUY_STOP_LIMIT: EngineOrderType.BUY_STOP_LIMIT,
    SDKOrderType.SELL_STOP_LIMIT: EngineOrderType.SELL_STOP_LIMIT,
}

_ENGINE_TO_SDK_ORDER: Dict[EngineOrderType, SDKOrderType] = {
    v: k for k, v in _SDK_TO_ENGINE_ORDER.items()
}

_ENGINE_SIDE_TO_SDK: Dict[Side, PositionSide] = {
    Side.BUY: PositionSide.BUY,
    Side.SELL: PositionSide.SELL,
}


def _is_market_order(ot: SDKOrderType) -> bool:
    return ot in (SDKOrderType.BUY, SDKOrderType.SELL)


def _is_pending_order(ot: SDKOrderType) -> bool:
    return not _is_market_order(ot)


# ── Symbol-info derivation ──────────────────────────────────────────────
#
# 原则：digits 从 K 线价格数据推导（不硬编码品种名）；contract_size
# 无法从价格数据推导，必须由经纪商提供。默认值 1 匹配 synthetic 回测模式
# 的 PnL 计算。真实经纪商数据应通过 SimBroker 的 symbol_info_map 传入。
#
# 参见 runner.py:_init_sdk_path — 从 _primary_bars 推导并传入 symbol_info_map。

_DEFAULT_CONTRACT_SIZE = 1       # 匹配 synthetic 回测模式
_DEFAULT_DIGITS_FALLBACK = 5     # 无 K 线数据时的回退值


def _count_decimal_places(value: float) -> int:
    """Count meaningful decimal places from a price float.

    Uses ``Decimal(str(value))`` which employs Python's shortest
    round-tripping representation — this naturally avoids both
    trailing-zero loss (``61234.50`` → 1 place) and IEEE-754 noise
    (``61000.27`` → 12 places under ``.12f``).

    >>> _count_decimal_places(1.12345)
    5
    >>> _count_decimal_places(61000.27)
    2
    >>> _count_decimal_places(100.0)
    0
    """
    d = Decimal(str(value))
    t = d.as_tuple()
    if t.exponent >= 0:
        return 0
    return -t.exponent


def _derive_symbol_info_from_bars(
    symbol: str,
    bars,
) -> SymbolInfo:
    """Derive SymbolInfo from K-line price data — no hardcoded symbol names.

    *digits* is inferred from the maximum decimal places observed in
    OHLC prices (sample of up to 500 bars).

    *contract_size* defaults to 1 (matching synthetic-backtest PnL mode).
    Real broker data should be passed via ``symbol_info_map`` when the
    MT4/MT5 connection is available.

    When *bars* is empty, falls back to conservative defaults.
    """
    if not bars:
        return _make_symbol_info(
            symbol, digits=_DEFAULT_DIGITS_FALLBACK,
            contract_size=_DEFAULT_CONTRACT_SIZE,
        )

    # ── digits from price data ─────────────────────────────────────────
    max_decimals = 2  # floor: sub-pip pricing is universal
    sample = bars[:500]
    for b in sample:
        for price in (b.open, b.high, b.low, b.close):
            d = _count_decimal_places(price)
            if d > max_decimals:
                max_decimals = d
    digits = min(max_decimals, 8)  # ceiling

    return _make_symbol_info(
        symbol, digits=digits, contract_size=_DEFAULT_CONTRACT_SIZE,
    )


# ── Shared builder ─────────────────────────────────────────────────────


def _make_symbol_info(
    symbol: str,
    digits: int,
    contract_size: int,
) -> SymbolInfo:
    """Build a SymbolInfo with consistent point/tick_size derived from digits."""
    point = Decimal("1") / Decimal(str(10**digits))
    return SymbolInfo(
        name=symbol,
        digits=digits,
        point=point,
        tick_size=point,
        tick_value=point,
        contract_size=Decimal(str(contract_size)),
        volume_min=Decimal("0.01"),
        volume_max=Decimal("100"),
        volume_step=Decimal("0.01"),
        stops_level=0,
        freeze_level=0,
        swap_long=Decimal("0"),
        swap_short=Decimal("0"),
        margin_rate=Decimal("0.01"),
    )


# ── Position metadata (engine types lack magic/comment/symbol) ─────────


@dataclass
class _PositionMeta:
    symbol: str
    magic: int = 0
    comment: str = ""


@dataclass
class _OrderMeta:
    symbol: str
    magic: int = 0
    comment: str = ""


# ── SimBroker ───────────────────────────────────────────────────────────


class SimBroker(Broker):
    """Broker implementation backed by the existing backtest engine.

    Created once per backtest run and advanced tick-by-tick.  The strategy
    interacts with it through the standard Broker interface; all engine
    interaction is internal.

    Usage sketch inside a runner loop::

        broker = SimBroker(portfolio, fill_model, cost_model, margin, market,
                           tick_source, account_mode=AccountMode.HEDGING)
        for tick in ticks:
            broker.advance_tick(tick)
            # strategy.on_tick() → calls self.broker.order_send(...)
            # ...
    """

    def __init__(
        self,
        portfolio: Portfolio,
        fill_model: FillModel,
        cost_model: CostModel,
        margin_model: MarginModel,
        market: MarketSimulator,
        tick_source: Callable[[], Optional[Tick]],
        account_mode: AccountMode = AccountMode.HEDGING,
        symbol_info_map: Optional[Dict[str, SymbolInfo]] = None,
        initial_balance: Decimal = Decimal("10000"),
    ) -> None:
        self._portfolio = portfolio
        self._fill = fill_model
        self._cost = cost_model
        self._margin = margin_model
        self._market = market
        self._get_tick = tick_source

        self._account_mode = account_mode
        self._symbols: Dict[str, SymbolInfo] = dict(symbol_info_map or {})
        self._initial_balance = initial_balance

        # Ticket counter — shared across orders and positions.
        self._next_ticket = 1

        # Metadata registries (engine types lack magic/comment/symbol).
        self._pos_meta: Dict[int, _PositionMeta] = {}
        self._order_meta: Dict[int, _OrderMeta] = {}

        # Current tick (updated by advance_tick).
        self._current_tick: Optional[Tick] = None
        self._current_bar_idx: int = -1
        self._deals: List[Deal] = []  # closed deal history (MQL5 HistorySelect equivalent)

    # ── Broker interface ────────────────────────────────────────────────

    def order_send(self, request: OrderRequest) -> OrderResult:
        """Submit any order type. Market orders fill immediately; pending
        orders enter the queue and may fill on subsequent ticks."""
        tick = self._get_tick()
        if tick is None:
            return OrderResult(retcode=Retcode.REJECTED, comment="no current tick")

        engine_ot = _SDK_TO_ENGINE_ORDER.get(request.type)
        if engine_ot is None:
            return OrderResult(retcode=Retcode.REJECTED, comment=f"unknown order type: {request.type}")

        # Build engine Order.
        ticket = self._next_ticket
        self._next_ticket += 1

        engine_order = Order(
            id=ticket,
            type=engine_ot,
            volume=_to_float(request.volume),
            price=_to_float(request.price),
            sl=_to_float(request.sl),
            tp=_to_float(request.tp),
            stop_limit_price=_to_float(request.stop_limit_price),
        )
        self._order_meta[ticket] = _OrderMeta(
            symbol=request.symbol,
            magic=request.magic,
            comment=request.comment,
        )

        if _is_market_order(request.type):
            return self._execute_market(engine_order, tick)
        else:
            return self._enqueue_pending(engine_order, tick)

    def position_modify(
        self, ticket: int, sl: Optional[Decimal] = None, tp: Optional[Decimal] = None
    ) -> OrderResult:
        """Modify SL/TP on an open position."""
        pos = self._find_position(ticket)
        if pos is None:
            return OrderResult(
                retcode=Retcode.REJECTED, comment=f"position {ticket} not found"
            )
        if sl is not None:
            pos.sl = _to_float(sl)
        if tp is not None:
            pos.tp = _to_float(tp)
        return OrderResult(retcode=Retcode.DONE, ticket=ticket)

    def position_close(
        self, ticket: int, volume: Optional[Decimal] = None
    ) -> OrderResult:
        """Close a position (fully if volume is None, else partially)."""
        pos = self._find_position(ticket)
        if pos is None:
            return OrderResult(
                retcode=Retcode.REJECTED, comment=f"position {ticket} not found"
            )

        tick = self._get_tick()
        if tick is None:
            return OrderResult(retcode=Retcode.REJECTED, comment="no current tick")

        close_vol = _to_float(volume) if volume is not None else pos.volume
        if close_vol <= 0 or close_vol > pos.volume:
            return OrderResult(
                retcode=Retcode.INVALID_VOLUME,
                comment=f"invalid close volume {close_vol} (position has {pos.volume})",
            )

        mark = tick.bid if pos.side is Side.BUY else tick.ask

        if abs(close_vol - pos.volume) < 1e-12:
            # Full close.
            trade = self._portfolio.close_position(
                ticket, mark, tick.ts, CloseReason.SIGNAL
            )
            meta = self._pos_meta.get(ticket)
            if meta is not None:
                self._record_deal(ticket, mark, close_vol,
                                  getattr(trade, 'pnl', 0), meta.symbol,
                                  meta.comment or "",
                                  getattr(trade, 'open_price', mark), tick.ts)
            self._pos_meta.pop(ticket, None)
            return OrderResult(
                retcode=Retcode.DONE,
                ticket=ticket,
                price=_to_decimal(trade.close_price),
                volume=_to_decimal(trade.volume),
                comment="closed",
            )
        else:
            # Partial close: reduce position volume, create a proportional trade.
            ratio = close_vol / pos.volume
            pnl_portion = self._portfolio._unit_pnl(
                pos.side, pos.open_price, mark
            ) * close_vol

            pos.volume -= close_vol
            self._portfolio._cash += pnl_portion

            return OrderResult(
                retcode=Retcode.DONE_PARTIAL,
                ticket=ticket,
                price=_to_decimal(mark),
                volume=_to_decimal(close_vol),
                comment="partial close",
            )

    def order_delete(self, ticket: int) -> OrderResult:
        """Cancel a pending order by ticket."""
        pending = self._fill.pending
        for i, order in enumerate(pending):
            if order.id == ticket:
                order.status = OrderStatus.CANCELLED
                pending.pop(i)
                self._order_meta.pop(ticket, None)
                return OrderResult(retcode=Retcode.DONE, ticket=ticket)
        return OrderResult(
            retcode=Retcode.REJECTED, comment=f"pending order {ticket} not found"
        )

    def positions(
        self, symbol: Optional[str] = None, magic: Optional[int] = None
    ) -> List[SDKPosition]:
        """Query open positions, optionally filtered."""
        tick = self._get_tick()
        result: List[SDKPosition] = []
        for pos in self._portfolio.positions:
            meta = self._pos_meta.get(pos.ticket)
            if meta is None:
                continue
            if symbol is not None and meta.symbol != symbol:
                continue
            if magic is not None and meta.magic != magic:
                continue
            result.append(self._to_sdk_position(pos, meta, tick))
        return result

    def orders(
        self, symbol: Optional[str] = None, magic: Optional[int] = None
    ) -> List[PendingOrder]:
        """Query pending orders, optionally filtered."""
        result: List[PendingOrder] = []
        for order in self._fill.pending:
            meta = self._order_meta.get(order.id)
            if meta is None:
                continue
            if symbol is not None and meta.symbol != symbol:
                continue
            if magic is not None and meta.magic != magic:
                continue
            sdk_type = _ENGINE_TO_SDK_ORDER.get(order.type)
            if sdk_type is None:
                continue
            result.append(
                PendingOrder(
                    ticket=order.id,
                    symbol=meta.symbol,
                    type=sdk_type,
                    volume=_to_decimal(order.volume),
                    price=_to_decimal(order.price),
                    sl=_to_decimal_or_none(order.sl),
                    tp=_to_decimal_or_none(order.tp),
                    magic=meta.magic,
                    comment=meta.comment,
                    placed_time_ms=order.created_at_ts,
                )
            )
        return result

    def deals(
        self, symbol: Optional[str] = None, magic: Optional[int] = None,
        from_ms: Optional[int] = None, to_ms: Optional[int] = None,
    ) -> List[Deal]:
        """Return closed deal history with optional filtering (MQL5 HistorySelect)."""
        result = self._deals
        if symbol is not None:
            result = [d for d in result if d.symbol == symbol]
        if magic is not None:
            result = [d for d in result if d.order_ticket >= 0]
        if from_ms is not None:
            result = [d for d in result if d.open_time_ms >= from_ms]
        if to_ms is not None:
            result = [d for d in result if d.open_time_ms <= to_ms]
        return result

    def _record_deal(self, ticket: int, close_price: float, close_vol: float,
                     pnl: float, symbol: str, comment: str, open_price: float, ts: int) -> None:
        """Record a closed position as a Deal for history queries (MQL5 HistorySelect)."""
        self._deals.append(Deal(
            ticket=ticket, order_ticket=ticket, position_ticket=ticket,
            symbol=symbol,
            volume=Decimal(str(close_vol)),
            open_price=Decimal(str(open_price)),
            price=Decimal(str(close_price)),
            open_time_ms=ts, history_time_ms=ts,
            profit=Decimal(str(pnl)),
            comment=comment,
        ))

    def account(self) -> AccountInfo:
        """Current account state snapshot."""
        tick = self._get_tick()
        equity = (
            self._portfolio.equity(tick)
            if tick
            else float(self._initial_balance)
        )
        cash = self._portfolio.cash
        margin_used = self._margin.used_margin(self._portfolio, tick) if tick else 0.0
        free_margin = equity - margin_used

        if margin_used > 0:
            margin_level = equity / margin_used * 100.0
        else:
            margin_level = float("inf")

        return AccountInfo(
            balance=_to_decimal(cash),
            equity=_to_decimal(equity),
            margin=_to_decimal(margin_used),
            free_margin=_to_decimal(free_margin),
            margin_level=_to_decimal(margin_level),
            leverage=int(self._margin._leverage) if self._margin._leverage > 0 else 1,
            currency="USD",
            mode=self._account_mode,
        )

    def symbol_info(self, symbol: str) -> SymbolInfo:
        """Return symbol metadata; raises KeyError if unknown."""
        if symbol in self._symbols:
            return self._symbols[symbol]
        # Fallback: conservative defaults (no K-line data available).
        # In production the runner passes symbol_info_map derived from bars.
        return _make_symbol_info(
            symbol, digits=_DEFAULT_DIGITS_FALLBACK,
            contract_size=_DEFAULT_CONTRACT_SIZE,
        )

    def server_time(self) -> int:
        """Current tick time in unix ms."""
        tick = self._get_tick()
        return tick.ts if tick else 0

    # ── Tick advance (called by runner loop) ────────────────────────────

    def advance_tick(self, tick: Tick) -> List[Fill]:
        """Process a tick: match pending orders, check SL/TP, update bar index.

        Returns fills generated by pending-order matching on this tick.
        The caller should update the portfolio for each fill.
        """
        self._current_tick = tick
        fills: List[Fill] = []

        # Match pending orders against this tick.
        for fill, order in self._fill.process_on_tick(tick):
            self._on_fill(fill, order)
            fills.append(fill)

        # SL/TP checks.
        self._portfolio.check_sl_tp(tick)

        # Update bar index.
        new_idx = self._market.bar_closed_at_or_before(tick.ts)
        if new_idx > self._current_bar_idx:
            self._current_bar_idx = new_idx

        return fills

    # ── Internals ───────────────────────────────────────────────────────

    def _execute_market(self, order: Order, tick: Tick) -> OrderResult:
        """Fill a market order immediately at current tick prices."""
        fill_pair = self._fill.process_market_order(order, tick)
        if fill_pair is None:
            order.status = OrderStatus.REJECTED
            self._order_meta.pop(order.id, None)
            return OrderResult(
                retcode=Retcode.REJECTED, comment="market order could not be filled"
            )

        fill, filled_order = fill_pair
        return self._on_fill(fill, filled_order)

    def _enqueue_pending(self, order: Order, tick: Tick) -> OrderResult:
        """Add a pending order to the queue; check if it triggers immediately."""
        self._fill.enqueue(order)

        # Check if it triggers on the current tick.
        fills = self._fill.process_on_tick(tick)
        for fill, matched_order in fills:
            if matched_order.id == order.id:
                return self._on_fill(fill, matched_order)

        # Not triggered yet — remains pending.
        return OrderResult(
            retcode=Retcode.DONE,
            ticket=order.id,
            comment="pending order placed",
        )

    def _on_fill(self, fill: Fill, order: Order) -> OrderResult:
        """Handle a fill: open a position in the portfolio, transfer metadata."""
        meta = self._order_meta.pop(order.id, None)

        pos = self._portfolio.apply_fill(fill, order, self._current_tick)

        if meta is not None:
            self._pos_meta[pos.ticket] = _PositionMeta(
                symbol=meta.symbol,
                magic=meta.magic,
                comment=meta.comment,
            )

        return OrderResult(
            retcode=(
                Retcode.DONE
                if abs(order.volume - fill.volume) < 1e-12
                else Retcode.DONE_PARTIAL
            ),
            ticket=pos.ticket,
            price=_to_decimal(fill.price),
            volume=_to_decimal(fill.volume),
            comment=meta.comment if meta else "",
        )

    def _find_position(self, ticket: int) -> Optional[EnginePosition]:
        for pos in self._portfolio.positions:
            if pos.ticket == ticket:
                return pos
        return None

    def _to_sdk_position(
        self, pos: EnginePosition, meta: _PositionMeta, tick: Optional[Tick]
    ) -> SDKPosition:
        """Convert an engine Position to an SDK Position."""
        profit = Decimal("0")
        if tick:
            mark = tick.bid if pos.side is Side.BUY else tick.ask
            pnl = self._portfolio._unit_pnl(pos.side, pos.open_price, mark) * pos.volume
            profit = _to_decimal(pnl)

        return SDKPosition(
            ticket=pos.ticket,
            symbol=meta.symbol,
            side=_ENGINE_SIDE_TO_SDK[pos.side],
            volume=_to_decimal(pos.volume),
            open_price=_to_decimal(pos.open_price),
            sl=_to_decimal_or_none(pos.sl),
            tp=_to_decimal_or_none(pos.tp),
            profit=profit,
            swap=Decimal("0"),
            magic=meta.magic,
            comment=meta.comment,
            open_time_ms=pos.open_ts,
        )
