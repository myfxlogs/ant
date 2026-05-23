"""Tests for app/engine/portfolio.py."""

from __future__ import annotations

import pytest

from app.engine.portfolio import Portfolio
from app.engine.types import (
    CloseReason,
    Fill,
    Order,
    OrderType,
    Side,
    Tick,
    Trade,
)


def _fill(order_id=1, ts=1_000, price=1.0, volume=1.0, commission=0.0) -> Fill:
    return Fill(order_id=order_id, ts=ts, price=price, volume=volume, commission=commission)


def _order(type_: OrderType, sl=0.0, tp=0.0) -> Order:
    return Order(id=1, type=type_, volume=1.0, sl=sl, tp=tp)


def _tick(ts=1_000, bid=1.0, ask=1.0001) -> Tick:
    return Tick(ts=ts, bid=bid, ask=ask)


# --- open / initial state -----------------------------------------------


def test_initial_state():
    p = Portfolio(10_000.0)
    assert p.cash == 10_000.0
    assert p.positions == []
    assert p.closed_trades == []
    assert not p.has_open()
    assert p.positions_total() == 0


def test_apply_fill_opens_position_and_deducts_commission():
    p = Portfolio(10_000.0)
    pos = p.apply_fill(_fill(commission=7.0), _order(OrderType.BUY), _tick())
    assert pos.ticket == 1
    assert pos.side is Side.BUY
    assert p.cash == 10_000.0 - 7.0
    assert p.has_open()


def test_multiple_fills_get_unique_tickets():
    p = Portfolio(10_000.0)
    a = p.apply_fill(_fill(), _order(OrderType.BUY), _tick())
    b = p.apply_fill(_fill(), _order(OrderType.SELL), _tick())
    assert a.ticket != b.ticket
    assert p.positions_total() == 2


# --- legacy vs real PnL --------------------------------------------------


def test_legacy_pnl_is_bare_price_diff():
    p = Portfolio(10_000.0, legacy_pnl=True)
    p.apply_fill(_fill(price=1.0, volume=2.0), _order(OrderType.BUY), _tick())
    t = p.close_position(1, close_price=1.1, close_ts=2_000, reason=CloseReason.SIGNAL)
    assert t.pnl == pytest.approx(0.2)  # (1.1-1.0) * 2 = 0.2
    assert p.cash == pytest.approx(10_000.2)


def test_real_pnl_includes_contract_size():
    p = Portfolio(10_000.0, legacy_pnl=False, contract_size=100_000.0)
    p.apply_fill(_fill(price=1.0, volume=1.0), _order(OrderType.BUY), _tick())
    t = p.close_position(1, close_price=1.001, close_ts=2_000, reason=CloseReason.SIGNAL)
    assert t.pnl == pytest.approx(100.0)  # 0.001 * 1 * 100000


def test_sell_pnl_symmetry():
    p = Portfolio(10_000.0, legacy_pnl=True)
    p.apply_fill(_fill(price=1.1, volume=1.0), _order(OrderType.SELL), _tick())
    t = p.close_position(1, close_price=1.0, close_ts=2_000, reason=CloseReason.TP)
    assert t.pnl == pytest.approx(0.1)  # (1.1-1.0) * 1 = 0.1 for sell


def test_close_unknown_ticket_raises():
    p = Portfolio(10_000.0)
    with pytest.raises(KeyError):
        p.close_position(42, close_price=1.0, close_ts=0, reason=CloseReason.SIGNAL)


# --- equity & unrealized -------------------------------------------------


def test_unrealized_buy_at_higher_bid_is_positive():
    p = Portfolio(10_000.0, legacy_pnl=True)
    p.apply_fill(_fill(price=1.0, volume=1.0), _order(OrderType.BUY), _tick())
    t = _tick(bid=1.05, ask=1.0501)
    # Unrealized: (bid - open) * vol = 0.05
    pos = p.positions[0]
    assert p.unrealized(pos, t) == pytest.approx(0.05)


def test_equity_equals_cash_plus_unrealized():
    p = Portfolio(10_000.0, legacy_pnl=True)
    p.apply_fill(_fill(price=1.0, volume=1.0, commission=0.0), _order(OrderType.BUY), _tick())
    t = _tick(bid=1.02, ask=1.0201)
    assert p.equity(t) == pytest.approx(p.cash + 0.02)


def test_equity_without_positions_equals_cash():
    p = Portfolio(5_000.0)
    assert p.equity(_tick()) == 5_000.0


# --- SL / TP legacy rules -----------------------------------------------


def test_sl_tp_buy_triggered_by_bid():
    p = Portfolio(10_000.0)
    p.apply_fill(
        _fill(price=1.0), _order(OrderType.BUY, sl=0.95, tp=1.05), _tick()
    )
    # bid drops below SL
    trades = p.check_sl_tp(_tick(bid=0.94, ask=0.9401))
    assert len(trades) == 1
    assert trades[0].reason is CloseReason.SL
    assert trades[0].close_price == 0.94


def test_sl_tp_buy_tp_uses_bid():
    p = Portfolio(10_000.0)
    p.apply_fill(_fill(price=1.0), _order(OrderType.BUY, tp=1.05), _tick())
    trades = p.check_sl_tp(_tick(bid=1.06, ask=1.0601))
    assert trades[0].reason is CloseReason.TP
    assert trades[0].close_price == 1.06


def test_sl_tp_sell_triggered_by_ask():
    p = Portfolio(10_000.0)
    p.apply_fill(
        _fill(price=1.0), _order(OrderType.SELL, sl=1.05, tp=0.95), _tick()
    )
    trades = p.check_sl_tp(_tick(bid=1.05, ask=1.06))
    assert trades[0].reason is CloseReason.SL


def test_sl_takes_priority_when_both_could_trigger():
    # Buy with SL and TP; craft a tick bid that is both <= sl AND >= tp
    # (not truly possible economically, but ensures deterministic ordering).
    p = Portfolio(10_000.0)
    p.apply_fill(_fill(price=1.0), _order(OrderType.BUY, sl=1.1, tp=0.9), _tick())
    # bid = 1.15, satisfies sl (bid <= 1.1? no, 1.15>1.1) → only test ordering when both hit.
    # Use scenario where both legally can hit: sl=1.05, tp=1.05; bid=1.05 ≥ tp and bid<=sl.
    p2 = Portfolio(10_000.0)
    p2.apply_fill(_fill(price=1.0), _order(OrderType.BUY, sl=1.05, tp=1.05), _tick())
    trades = p2.check_sl_tp(_tick(bid=1.05, ask=1.0501))
    assert trades[0].reason is CloseReason.SL  # first branch in legacy rule


def test_check_sl_tp_zero_sl_means_disabled():
    p = Portfolio(10_000.0)
    p.apply_fill(_fill(price=1.0), _order(OrderType.BUY, sl=0.0, tp=0.0), _tick())
    assert p.check_sl_tp(_tick(bid=0.0001, ask=0.0001)) == []  # no trigger


# --- force liquidate -----------------------------------------------------


def test_force_liquidate_closes_all_and_uses_mark_side():
    p = Portfolio(10_000.0, legacy_pnl=True)
    p.apply_fill(_fill(price=1.0), _order(OrderType.BUY), _tick())
    p.apply_fill(_fill(price=1.5), _order(OrderType.SELL), _tick())
    trades = p.force_liquidate_all(_tick(bid=1.05, ask=1.4), CloseReason.END_OF_TEST)
    assert len(trades) == 2
    assert not p.has_open()
    # Buy closed at bid 1.05 → pnl = 0.05; Sell closed at ask 1.4 → pnl = 0.1
    by_side = {t.side: t for t in trades}
    assert by_side[Side.BUY].pnl == pytest.approx(0.05)
    assert by_side[Side.SELL].pnl == pytest.approx(0.1)
    for t in trades:
        assert t.reason is CloseReason.END_OF_TEST


# --- cash adjustments (for swap) ----------------------------------------


def test_adjust_cash_and_set_cash():
    p = Portfolio(10_000.0)
    p.adjust_cash(-50.0)
    assert p.cash == 9_950.0
    p.set_cash(8_000.0)
    assert p.cash == 8_000.0
