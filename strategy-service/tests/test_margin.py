"""Tests for app/engine/margin.py."""

from __future__ import annotations

import math

import pytest

from app.engine.margin import MarginModel
from app.engine.portfolio import Portfolio
from app.engine.types import Fill, Order, OrderType, Side, Tick


def _open(p: Portfolio, side: OrderType, price=1.0, volume=1.0) -> None:
    p.apply_fill(
        Fill(order_id=1, ts=0, price=price, volume=volume),
        Order(id=1, type=side, volume=volume),
        Tick(ts=0, bid=price, ask=price),
    )


# --- no-op disabled mode -------------------------------------------------


def test_zero_leverage_is_disabled():
    m = MarginModel(leverage=0.0)
    assert m.enabled() is False


def test_disabled_model_never_triggers_margin_call():
    m = MarginModel(leverage=0.0)
    p = Portfolio(1_000.0, legacy_pnl=False, contract_size=100_000.0)
    _open(p, OrderType.BUY, price=1.0, volume=10.0)  # huge notional
    assert m.is_margin_call(p, Tick(ts=0, bid=0.1, ask=0.1001)) is False


def test_disabled_model_required_margin_zero():
    m = MarginModel(leverage=0.0)
    p = Portfolio(1_000.0)
    _open(p, OrderType.BUY)
    assert m.required_margin(p.positions[0], 1.0) == 0.0


def test_disabled_model_margin_level_inf():
    m = MarginModel(leverage=0.0)
    p = Portfolio(1_000.0)
    _open(p, OrderType.BUY)
    assert math.isinf(m.margin_level(p, Tick(ts=0, bid=1.0, ask=1.0001)))


# --- enabled calculations -----------------------------------------------


def test_required_margin_formula():
    m = MarginModel(leverage=100.0, contract_size=100_000.0)
    p = Portfolio(10_000.0)
    _open(p, OrderType.BUY, price=1.2, volume=0.5)
    # 0.5 * 100_000 * 1.2 / 100 = 600
    assert m.required_margin(p.positions[0], 1.2) == pytest.approx(600.0)


def test_used_margin_sums_all_positions():
    m = MarginModel(leverage=100.0, contract_size=100_000.0)
    p = Portfolio(10_000.0)
    _open(p, OrderType.BUY, price=1.0, volume=1.0)
    _open(p, OrderType.SELL, price=1.0, volume=1.0)
    t = Tick(ts=0, bid=1.0, ask=1.0)
    used = m.used_margin(p, t)
    # Each: 1 * 100_000 * 1 / 100 = 1000 → 2000 total
    assert used == pytest.approx(2000.0)


def test_no_positions_yields_infinite_margin_level():
    m = MarginModel(leverage=100.0)
    p = Portfolio(10_000.0)
    assert math.isinf(m.margin_level(p, Tick(ts=0, bid=1.0, ask=1.0001)))


def test_margin_call_fires_when_level_below_100():
    # Setup: leverage 100, buy 1 lot @ 1.0 (legacy_pnl False for realistic pnl),
    # cash 500 < margin 1000; price barely moves → equity < used margin.
    m = MarginModel(leverage=100.0, contract_size=100_000.0)
    p = Portfolio(500.0, legacy_pnl=False, contract_size=100_000.0)
    _open(p, OrderType.BUY, price=1.0, volume=1.0)
    t = Tick(ts=0, bid=0.999, ask=0.9991)  # tiny adverse move
    assert m.is_margin_call(p, t) is True


def test_margin_call_does_not_fire_when_well_funded():
    m = MarginModel(leverage=100.0, contract_size=100_000.0)
    p = Portfolio(100_000.0, legacy_pnl=False, contract_size=100_000.0)
    _open(p, OrderType.BUY, price=1.0, volume=0.1)
    t = Tick(ts=0, bid=1.0, ask=1.0001)
    assert m.is_margin_call(p, t) is False


def test_enabled_required_margin_ignores_zero_price():
    m = MarginModel(leverage=100.0)
    p = Portfolio(10_000.0)
    _open(p, OrderType.BUY)
    assert m.required_margin(p.positions[0], 0.0) == 0.0


def test_enabled_required_margin_ignores_negative_price():
    m = MarginModel(leverage=100.0)
    p = Portfolio(10_000.0)
    _open(p, OrderType.BUY)
    assert m.required_margin(p.positions[0], -0.5) == 0.0
