"""Portfolio: cash book, open positions, closed trades.

契约：docs/domains/backtest-system.md §7.4.3 · portfolio.py

重要语义差异（对齐旧 ``backtest.py``）：

* 旧引擎的 ``current_capital`` 实际上是**已实现现金**（balance）。开仓扣佣金，
  平仓加 PnL，swap 直接乘到它上。未实现盈亏**不**计入它；策略回调后追加到
  ``equity_curve`` 的也是这个值。所以 :py:attr:`cash` 用来给 runner 写权益曲线，
  而 :py:meth:`equity` 在需要真实 margin level 时才用（cash + 未实现）。

* PnL 公式：``legacy_pnl=True`` 时 ``Δp * volume``（裸价差 × 手数，单位错但旧
  引擎就这样）；``legacy_pnl=False`` 时 ``Δp * volume * contract_size``。

* SL/TP 触发（见 ``app/backtest.py:232-259``）：
    - buy 仓位：``tick.bid <= sl`` → SL 成交价 = bid；``tick.bid >= tp`` → TP @ bid
    - sell 仓位：``tick.ask >= sl`` → SL @ ask；``tick.ask <= tp`` → TP @ ask
  （FIFO，单 tick 同仓位 SL 优先于 TP）
"""

from __future__ import annotations

from typing import List, Optional, Tuple

from app.engine.types import (
    CloseReason,
    Fill,
    Order,
    OrderType,
    Position,
    Side,
    Tick,
    Trade,
)


def _order_side(order_type: OrderType) -> Side:
    v = order_type.value
    if v.startswith("buy"):
        return Side.BUY
    if v.startswith("sell"):
        return Side.SELL
    raise ValueError(f"order type {order_type} has no side")


class Portfolio:
    """Mutable account state carried through the tick loop."""

    def __init__(
        self,
        initial_cash: float,
        legacy_pnl: bool = True,
        contract_size: float = 100_000.0,
    ) -> None:
        self._cash = float(initial_cash)
        self._legacy_pnl = bool(legacy_pnl)
        self._contract_size = float(contract_size)
        self._positions: List[Position] = []
        self._closed: List[Trade] = []
        self._next_ticket = 1

    # --- state -----------------------------------------------------------

    @property
    def cash(self) -> float:
        """Running realised-cash balance (matches legacy ``current_capital``)."""
        return self._cash

    @property
    def positions(self) -> List[Position]:
        return list(self._positions)

    @property
    def closed_trades(self) -> List[Trade]:
        return list(self._closed)

    @property
    def legacy_pnl(self) -> bool:
        return self._legacy_pnl

    def has_open(self) -> bool:
        return bool(self._positions)

    def positions_total(self) -> int:
        return len(self._positions)

    # --- PnL helpers ------------------------------------------------------

    def _unit_pnl(self, side: Side, open_price: float, close_price: float) -> float:
        diff = (close_price - open_price) if side is Side.BUY else (open_price - close_price)
        if self._legacy_pnl:
            return diff
        return diff * self._contract_size

    def unrealized(self, position: Position, tick: Tick) -> float:
        """Mark-to-market PnL for one position at ``tick``.

        Exit side determines the mark price:
          - buy → tick.bid (we would sell to close)
          - sell → tick.ask (we would buy to close)
        """
        mark = tick.bid if position.side is Side.BUY else tick.ask
        return self._unit_pnl(position.side, position.open_price, mark) * position.volume

    def equity(self, tick: Tick) -> float:
        """True equity = cash + Σ unrealized."""
        return self._cash + sum(self.unrealized(p, tick) for p in self._positions)

    # --- mutations -------------------------------------------------------

    def adjust_cash(self, delta: float) -> None:
        """Add ``delta`` (signed) to cash. Used by CostModel swap application."""
        self._cash += float(delta)

    def set_cash(self, new_cash: float) -> None:
        """Replace cash outright (used when runner applies a swap multiplier to equity)."""
        self._cash = float(new_cash)

    def apply_fill(self, fill: Fill, order: Order, tick: Tick) -> Position:
        """Open a new position from a matched fill.

        Deducts commission from cash (旧引擎仅开仓扣).
        """
        side = _order_side(order.type)
        pos = Position(
            ticket=self._next_ticket,
            side=side,
            volume=float(fill.volume),
            open_price=float(fill.price),
            open_ts=int(fill.ts),
            sl=float(order.sl),
            tp=float(order.tp),
        )
        self._next_ticket += 1
        self._positions.append(pos)
        self._cash -= float(fill.commission)
        return pos

    def close_position(
        self,
        ticket: int,
        close_price: float,
        close_ts: int,
        reason: CloseReason,
        commission: float = 0.0,
    ) -> Trade:
        idx = next(
            (i for i, p in enumerate(self._positions) if p.ticket == ticket), -1
        )
        if idx < 0:
            raise KeyError(f"position {ticket} not open")
        pos = self._positions.pop(idx)
        pnl = self._unit_pnl(pos.side, pos.open_price, float(close_price)) * pos.volume
        self._cash += pnl - float(commission)
        trade = Trade(
            ticket=pos.ticket,
            side=pos.side,
            volume=pos.volume,
            open_ts=pos.open_ts,
            open_price=pos.open_price,
            close_ts=int(close_ts),
            close_price=float(close_price),
            pnl=pnl,
            commission=float(commission),
            reason=reason,
        )
        self._closed.append(trade)
        return trade

    def check_sl_tp(self, tick: Tick) -> List[Trade]:
        """Scan open positions for SL/TP hits using legacy rules (FIFO, SL先于TP)."""
        closed: List[Trade] = []
        # Iterate over a snapshot so we can mutate _positions safely via close_position.
        for pos in list(self._positions):
            close_price: Optional[float] = None
            reason: Optional[CloseReason] = None
            if pos.side is Side.BUY:
                if pos.sl and tick.bid <= pos.sl:
                    close_price, reason = tick.bid, CloseReason.SL
                elif pos.tp and tick.bid >= pos.tp:
                    close_price, reason = tick.bid, CloseReason.TP
            else:  # SELL
                if pos.sl and tick.ask >= pos.sl:
                    close_price, reason = tick.ask, CloseReason.SL
                elif pos.tp and tick.ask <= pos.tp:
                    close_price, reason = tick.ask, CloseReason.TP
            if close_price is not None and reason is not None:
                closed.append(self.close_position(pos.ticket, close_price, tick.ts, reason))
        return closed

    def force_liquidate_all(self, tick: Tick, reason: CloseReason) -> List[Trade]:
        """Close every open position at the relevant mark side of ``tick``."""
        closed: List[Trade] = []
        for pos in list(self._positions):
            mark = tick.bid if pos.side is Side.BUY else tick.ask
            closed.append(self.close_position(pos.ticket, mark, tick.ts, reason))
        return closed
