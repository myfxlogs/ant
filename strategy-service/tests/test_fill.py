"""Tests for app/engine/fill.py."""

from __future__ import annotations

import pytest

from app.engine.cost import CostModel
from app.engine.fill import FillModel
from app.engine.types import (
    CostProfile,
    Order,
    OrderStatus,
    OrderType,
    SlippageMode,
    Tick,
)


def _cost(slip_rate: float = 0.0, commission: float = 0.0) -> CostModel:
    return CostModel(
        CostProfile(
            commission_per_lot=commission,
            slippage_mode=SlippageMode.FIXED,
            slippage_rate=slip_rate,
        )
    )


def _fm(**kw) -> FillModel:
    return FillModel(cost=_cost(**{k: v for k, v in kw.items() if k != "max_fill"}),
                     max_fill_volume=kw.get("max_fill", 0.0))


# --- enqueue & queue state ----------------------------------------------


def test_enqueue_assigns_id_and_marks_pending():
    fm = _fm()
    o = Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0)
    fm.enqueue(o)
    assert o.id > 0
    assert o.status is OrderStatus.PENDING
    assert len(fm.pending) == 1


def test_enqueue_replace_same_type_evicts_existing():
    fm = _fm()
    a = Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0)
    b = Order(id=0, type=OrderType.BUY_LIMIT, volume=2.0, price=1.1)
    fm.enqueue(a)
    fm.enqueue(b, replace_same_type=True)
    assert len(fm.pending) == 1
    assert fm.pending[0].volume == 2.0


def test_enqueue_replace_only_touches_same_type():
    fm = _fm()
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0))
    fm.enqueue(Order(id=0, type=OrderType.SELL_STOP, volume=1.0, price=0.9))
    fm.enqueue(
        Order(id=0, type=OrderType.BUY_LIMIT, volume=3.0, price=1.0),
        replace_same_type=True,
    )
    # Sell_stop survives; only the prior buy_limit is gone.
    types = sorted(o.type.value for o in fm.pending)
    assert types == ["buy_limit", "sell_stop"]


def test_cancel_all_returns_count_and_empties_queue():
    fm = _fm()
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0))
    fm.enqueue(Order(id=0, type=OrderType.SELL_STOP, volume=1.0, price=0.9))
    assert fm.cancel_all() == 2
    assert fm.pending == []


# --- expiration ---------------------------------------------------------


def test_expired_orders_are_dropped_before_matching():
    fm = _fm()
    o = Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0, expiration=1_000)
    fm.enqueue(o)
    fills = fm.process_on_tick(Tick(ts=2_000, bid=0.9, ask=0.9))
    assert fills == []
    assert o.status is OrderStatus.EXPIRED
    assert fm.pending == []


def test_order_at_exact_expiration_is_expired():
    fm = _fm()
    o = Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0, expiration=1_000)
    fm.enqueue(o)
    fills = fm.process_on_tick(Tick(ts=1_000, bid=0.9, ask=0.9))
    assert fills == []


# --- triggers: limit / stop ---------------------------------------------


def test_buy_limit_fires_when_ask_drops_below_price():
    fm = _fm()
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.00))
    fills = fm.process_on_tick(Tick(ts=1, bid=0.99, ask=0.995))
    assert len(fills) == 1
    fill, order = fills[0]
    assert fill.price == 0.995  # fills @ ask
    assert order.status is OrderStatus.FILLED


def test_buy_limit_does_not_fire_when_ask_above():
    fm = _fm()
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.00))
    fills = fm.process_on_tick(Tick(ts=1, bid=1.05, ask=1.06))
    assert fills == []
    assert len(fm.pending) == 1


def test_sell_limit_fires_when_bid_rises_above_price():
    fm = _fm()
    fm.enqueue(Order(id=0, type=OrderType.SELL_LIMIT, volume=1.0, price=1.00))
    fills = fm.process_on_tick(Tick(ts=1, bid=1.02, ask=1.03))
    assert len(fills) == 1
    assert fills[0][0].price == 1.02


def test_buy_stop_fires_when_ask_rises_above_price():
    fm = _fm()
    fm.enqueue(Order(id=0, type=OrderType.BUY_STOP, volume=1.0, price=1.00))
    fills = fm.process_on_tick(Tick(ts=1, bid=1.01, ask=1.02))
    assert len(fills) == 1
    assert fills[0][0].price == 1.02


def test_sell_stop_fires_when_bid_drops_below_price():
    fm = _fm()
    fm.enqueue(Order(id=0, type=OrderType.SELL_STOP, volume=1.0, price=1.00))
    fills = fm.process_on_tick(Tick(ts=1, bid=0.99, ask=1.0))
    assert len(fills) == 1
    assert fills[0][0].price == 0.99


# --- stop_limit two-leg activation --------------------------------------


def test_buy_stop_limit_activates_and_fills_in_two_ticks():
    fm = _fm()
    fm.enqueue(
        Order(
            id=0,
            type=OrderType.BUY_STOP_LIMIT,
            volume=1.0,
            price=1.05,             # stop trigger
            stop_limit_price=1.04,  # limit price after activation
        )
    )
    # First tick: ask rises above stop → activated, but limit not hit yet (ask>limit).
    t1 = Tick(ts=1, bid=1.05, ask=1.06)
    fills1 = fm.process_on_tick(t1)
    assert fills1 == []
    assert fm.pending[0].activated is True
    assert fm.pending[0].type is OrderType.BUY_LIMIT
    assert fm.pending[0].price == 1.04
    # Second tick: ask falls to the limit → fill.
    t2 = Tick(ts=2, bid=1.03, ask=1.04)
    fills2 = fm.process_on_tick(t2)
    assert len(fills2) == 1
    assert fills2[0][0].price == 1.04


def test_sell_stop_limit_mirrors_buy_side():
    fm = _fm()
    fm.enqueue(
        Order(
            id=0,
            type=OrderType.SELL_STOP_LIMIT,
            volume=1.0,
            price=0.95,
            stop_limit_price=0.96,
        )
    )
    fm.process_on_tick(Tick(ts=1, bid=0.94, ask=0.95))  # stop trigger
    assert fm.pending[0].activated is True
    assert fm.pending[0].type is OrderType.SELL_LIMIT
    fills = fm.process_on_tick(Tick(ts=2, bid=0.96, ask=0.97))
    assert len(fills) == 1


def test_stop_limit_without_stop_limit_price_keeps_stop_price():
    fm = _fm()
    fm.enqueue(
        Order(id=0, type=OrderType.BUY_STOP_LIMIT, volume=1.0, price=1.05)
    )
    fm.process_on_tick(Tick(ts=1, bid=1.05, ask=1.06))
    assert fm.pending[0].activated is True
    assert fm.pending[0].price == 1.05  # unchanged


# --- slippage & commission -----------------------------------------------


def test_slippage_applied_to_fill_price():
    fm = FillModel(cost=_cost(slip_rate=0.01), max_fill_volume=0.0)
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0))
    fills = fm.process_on_tick(Tick(ts=1, bid=0.99, ask=1.0))
    fill, _ = fills[0]
    assert fill.price == pytest.approx(1.0 * 1.01)
    assert fill.slippage > 0.0


def test_commission_populated_on_fill():
    fm = FillModel(cost=_cost(commission=7.0), max_fill_volume=0.0)
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=2.0, price=1.0))
    fills = fm.process_on_tick(Tick(ts=1, bid=0.99, ask=1.0))
    assert fills[0][0].commission == pytest.approx(14.0)


# --- partial fills -------------------------------------------------------


def test_max_fill_volume_creates_partial_fill_and_requeues_remainder():
    fm = FillModel(cost=_cost(), max_fill_volume=0.3)
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0))
    fills = fm.process_on_tick(Tick(ts=1, bid=0.99, ask=1.0))
    assert len(fills) == 1
    assert fills[0][0].volume == pytest.approx(0.3)
    # Remaining 0.7 volume still pending.
    assert len(fm.pending) == 1
    assert fm.pending[0].volume == pytest.approx(0.7)


def test_no_max_fill_volume_fully_fills_and_clears():
    fm = FillModel(cost=_cost(), max_fill_volume=0.0)
    fm.enqueue(Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0))
    fills = fm.process_on_tick(Tick(ts=1, bid=0.99, ask=1.0))
    assert fills[0][0].volume == 1.0
    assert fm.pending == []


# --- FIFO order ---------------------------------------------------------


def test_fifo_fill_order_by_enqueue_sequence():
    fm = _fm()
    a = Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0)
    b = Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0, price=1.0)
    fm.enqueue(a)
    fm.enqueue(b)
    fills = fm.process_on_tick(Tick(ts=1, bid=0.99, ask=0.99))
    assert [f.order_id for f, _ in fills] == [a.id, b.id]


# --- market orders ------------------------------------------------------


def test_market_buy_uses_ask_and_returns_fill():
    fm = FillModel(cost=_cost())
    o = Order(id=0, type=OrderType.BUY, volume=1.0)
    result = fm.process_market_order(o, Tick(ts=5, bid=1.0, ask=1.0002))
    assert result is not None
    fill, order = result
    assert fill.price == 1.0002
    assert fill.ts == 5
    assert order.status is OrderStatus.FILLED


def test_market_sell_uses_bid():
    fm = FillModel(cost=_cost())
    result = fm.process_market_order(
        Order(id=0, type=OrderType.SELL, volume=1.0),
        Tick(ts=5, bid=0.999, ask=1.0),
    )
    fill, _ = result  # type: ignore[misc]
    assert fill.price == 0.999


def test_market_order_rejects_zero_volume():
    fm = FillModel(cost=_cost(), max_fill_volume=0.0)
    o = Order(id=0, type=OrderType.BUY, volume=0.0)
    result = fm.process_market_order(o, Tick(ts=1, bid=1.0, ask=1.0))
    assert result is None
    assert o.status is OrderStatus.REJECTED


def test_market_order_wrong_type_raises():
    fm = FillModel(cost=_cost())
    with pytest.raises(ValueError):
        fm.process_market_order(
            Order(id=0, type=OrderType.BUY_LIMIT, volume=1.0),
            Tick(ts=1, bid=1.0, ask=1.0),
        )
