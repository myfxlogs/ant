"""Fill model: pending-order queue + tick matching.

契约：docs/domains/backtest-system.md §7.4.3 · fill.py

撮合语义严格复刻旧 ``app/backtest.py:175-230``：

* FIFO by insertion sequence.
* 6 种挂单：buy/sell × limit/stop + buy/sell_stop_limit（两段式）。
* 特殊：``CANCEL_PENDING`` 由 runner 调 :py:meth:`cancel_all`，不走队列。
* ``expiration``：tick.ts >= order.expiration 时整单作废，不产生 fill。
* ``replace_same_type=True``：入队前先清掉同 type 已有挂单。
* ``max_fill_volume > 0``：启用部分成交，剩余量重回队列保留原序。
* 成交价后应用 :py:meth:`CostModel.apply_slippage`；佣金由 :py:meth:`CostModel.commission` 计算。

:py:meth:`process_on_tick` 与 :py:meth:`process_market_order` 都返回
``(Fill, Order) `` 对，供 runner 逐对交给 :py:meth:`Portfolio.apply_fill`。
"""

from __future__ import annotations

from typing import List, Optional, Tuple

from app.engine.cost import CostModel
from app.engine.types import (
    Fill,
    Order,
    OrderStatus,
    OrderType,
    Tick,
)


_BUY_PENDING = {
    OrderType.BUY_LIMIT,
    OrderType.BUY_STOP,
    OrderType.BUY_STOP_LIMIT,
}
_SELL_PENDING = {
    OrderType.SELL_LIMIT,
    OrderType.SELL_STOP,
    OrderType.SELL_STOP_LIMIT,
}


def _is_buy_side(order_type: OrderType) -> bool:
    return order_type.value.startswith("buy")


class FillModel:
    """Pending-order book + tick matcher."""

    def __init__(self, cost: CostModel, max_fill_volume: float = 0.0) -> None:
        self._cost = cost
        self._max_fill_vol = float(max_fill_volume)
        self._pending: List[Order] = []
        self._seq = 0  # monotonic: used as Order.id when strategy didn't set one

    # --- queue management -----------------------------------------------

    @property
    def pending(self) -> List[Order]:
        return list(self._pending)

    def enqueue(self, order: Order, replace_same_type: bool = False) -> None:
        """Add an order to the pending queue."""
        if replace_same_type:
            self._pending = [o for o in self._pending if o.type is not order.type]
        # Assign id if the caller left it at default 0.
        self._seq += 1
        if order.id == 0:
            order.id = self._seq
        order.created_at_ts = order.created_at_ts or 0
        order.status = OrderStatus.PENDING
        self._pending.append(order)

    def cancel_all(self) -> int:
        n = len(self._pending)
        for o in self._pending:
            o.status = OrderStatus.CANCELLED
        self._pending.clear()
        return n

    # --- tick processing ------------------------------------------------

    def process_on_tick(self, tick: Tick) -> List[Tuple[Fill, Order]]:
        """Expire, then match pending orders against ``tick``.

        Returns a list of ``(Fill, Order)`` pairs in FIFO activation order.
        Partially-filled orders remain in the queue with reduced volume.
        """
        if not self._pending:
            return []

        # 1. Expiration sweep.
        self._pending = [
            o for o in self._pending if not self._is_expired(o, tick.ts)
        ]

        fills: List[Tuple[Fill, Order]] = []
        remain: List[Order] = []
        for order in self._pending:
            triggered, raw_price = self._check_trigger(order, tick)
            if not triggered:
                remain.append(order)
                continue

            # Apply slippage (旧引擎在触发后对成交价做 slip)
            fill_price = self._cost.apply_slippage(
                raw_price, is_buy_side=_is_buy_side(order.type)
            )
            fill_vol = self._fill_volume(order.volume)
            if fill_vol <= 0:
                # max_fill_volume == 0 and volume == 0 → drop.
                order.status = OrderStatus.REJECTED
                continue

            commission = self._cost.commission(fill_vol)
            slippage = abs(fill_price - raw_price)
            fill = Fill(
                order_id=order.id,
                ts=tick.ts,
                price=fill_price,
                volume=fill_vol,
                commission=commission,
                slippage=slippage,
            )
            fills.append((fill, order))

            remaining = order.volume - fill_vol
            if remaining > 1e-12:
                # Partial fill: keep the order queued with the smaller volume.
                order.volume = remaining
                remain.append(order)
            else:
                order.status = OrderStatus.FILLED

        self._pending = remain
        return fills

    def process_market_order(
        self, order: Order, tick: Tick
    ) -> Optional[Tuple[Fill, Order]]:
        """Immediate market fill for a ``BUY`` / ``SELL`` order from the strategy.

        Market orders don't enter the pending queue. Returns ``None`` if the
        volume rounds to zero.
        """
        if order.type not in (OrderType.BUY, OrderType.SELL):
            raise ValueError(f"process_market_order only handles BUY/SELL, got {order.type}")

        # Assign an id if the caller left it unset so fills can cross-reference.
        if order.id == 0:
            self._seq += 1
            order.id = self._seq

        is_buy = order.type is OrderType.BUY
        raw_price = tick.ask if is_buy else tick.bid
        fill_price = self._cost.apply_slippage(raw_price, is_buy_side=is_buy)
        fill_vol = self._fill_volume(order.volume)
        if fill_vol <= 0:
            order.status = OrderStatus.REJECTED
            return None

        commission = self._cost.commission(fill_vol)
        slippage = abs(fill_price - raw_price)
        fill = Fill(
            order_id=order.id,
            ts=tick.ts,
            price=fill_price,
            volume=fill_vol,
            commission=commission,
            slippage=slippage,
        )
        order.status = OrderStatus.FILLED
        return fill, order

    # --- helpers --------------------------------------------------------

    @staticmethod
    def _is_expired(order: Order, ts: int) -> bool:
        if order.expiration is None:
            return False
        if ts >= order.expiration:
            order.status = OrderStatus.EXPIRED
            return True
        return False

    def _check_trigger(self, order: Order, tick: Tick) -> Tuple[bool, float]:
        """Return ``(triggered, raw_fill_price)``.

        Mutates ``order`` when a ``stop_limit`` leg activates.
        """
        t = order.type
        if t is OrderType.BUY_LIMIT:
            return (tick.ask <= order.price, tick.ask)
        if t is OrderType.SELL_LIMIT:
            return (tick.bid >= order.price, tick.bid)
        if t is OrderType.BUY_STOP:
            return (tick.ask >= order.price, tick.ask)
        if t is OrderType.SELL_STOP:
            return (tick.bid <= order.price, tick.bid)
        if t is OrderType.BUY_STOP_LIMIT:
            if not order.activated and tick.ask >= order.price:
                order.activated = True
                order.status = OrderStatus.ACTIVATED
                if order.stop_limit_price > 0:
                    order.price = order.stop_limit_price
                order.type = OrderType.BUY_LIMIT
            if order.activated and tick.ask <= order.price:
                return (True, tick.ask)
            return (False, 0.0)
        if t is OrderType.SELL_STOP_LIMIT:
            if not order.activated and tick.bid <= order.price:
                order.activated = True
                order.status = OrderStatus.ACTIVATED
                if order.stop_limit_price > 0:
                    order.price = order.stop_limit_price
                order.type = OrderType.SELL_LIMIT
            if order.activated and tick.bid >= order.price:
                return (True, tick.bid)
            return (False, 0.0)
        # Market / close / cancel types don't sit in the queue.
        return (False, 0.0)

    def _fill_volume(self, requested: float) -> float:
        if self._max_fill_vol > 0:
            return min(float(requested), self._max_fill_vol)
        return float(requested)
