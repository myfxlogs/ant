"""Tests for app/engine/context.py."""

from __future__ import annotations

import numpy as np
import pytest

from app.engine.context import build_context
from app.engine.market import MarketSimulator
from app.engine.portfolio import Portfolio
from app.engine.types import (
    Bar,
    Fill,
    Order,
    OrderType,
    RunMode,
    Side,
    Tick,
)


def _bars(n: int) -> list[Bar]:
    bars = []
    for i in range(n):
        ot = i * 60_000
        bars.append(
            Bar(
                open_time=ot,
                close_time=ot + 60_000,
                open=1.0 + i * 0.01,
                high=1.0 + i * 0.01 + 0.001,
                low=1.0 + i * 0.01 - 0.001,
                close=1.0 + (i + 1) * 0.01,
                volume=100.0,
            )
        )
    return bars


def _market(n: int = 5) -> MarketSimulator:
    return MarketSimulator(_bars(n))


def _tick(bid=1.0, ask=1.0001) -> Tick:
    return Tick(ts=1_000, bid=bid, ask=ask)


# --- basic shape --------------------------------------------------------


def test_context_has_all_legacy_keys():
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1",
        _market(3), bar_idx=2, portfolio=Portfolio(10_000.0),
        params={}, current_tick=_tick(),
    )
    for key in (
        "close", "open", "high", "low", "volume",
        "symbol", "timeframe", "current_price",
    ):
        assert key in ctx


def test_context_injects_account_fields():
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1",
        _market(3), bar_idx=0, portfolio=Portfolio(10_000.0),
        params={}, current_tick=_tick(),
    )
    # D8 bug fix: these must now be present.
    for key in (
        "cash", "equity", "position", "positions", "positions_total",
        "account_balance", "account_equity", "params", "mode",
    ):
        assert key in ctx


# --- OHLCV slices -------------------------------------------------------


def test_ohlcv_slice_length_equals_bar_idx_plus_one():
    market = _market(5)
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", market, bar_idx=2,
        portfolio=Portfolio(10_000.0), params={}, current_tick=_tick(),
    )
    assert len(ctx["close"]) == 3
    assert isinstance(ctx["close"], np.ndarray)


def test_first_bar_slice_has_length_one():
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", _market(5), bar_idx=0,
        portfolio=Portfolio(10_000.0), params={}, current_tick=_tick(),
    )
    assert len(ctx["open"]) == 1


def test_out_of_range_bar_idx_raises_via_market_slice():
    with pytest.raises(IndexError):
        build_context(
            RunMode.BACKTEST, "EURUSD", "M1", _market(2), bar_idx=2,
            portfolio=Portfolio(10_000.0), params={}, current_tick=_tick(),
        )


# --- account fields ------------------------------------------------------


def test_no_positions_yields_empty_position_list():
    p = Portfolio(10_000.0)
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", _market(3), bar_idx=0,
        portfolio=p, params={}, current_tick=_tick(),
    )
    assert ctx["position"] is None
    assert ctx["positions"] == []
    assert ctx["positions_total"] == 0
    assert ctx["cash"] == 10_000.0
    assert ctx["equity"] == 10_000.0


def test_one_position_appears_as_dict_with_side_string():
    p = Portfolio(10_000.0)
    p.apply_fill(
        Fill(order_id=1, ts=0, price=1.0, volume=1.0),
        Order(id=1, type=OrderType.BUY, volume=1.0, sl=0.95, tp=1.05),
        _tick(),
    )
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", _market(3), bar_idx=0,
        portfolio=p, params={}, current_tick=_tick(),
    )
    assert ctx["positions_total"] == 1
    assert ctx["position"] is not None
    assert ctx["position"]["side"] == "buy"
    assert ctx["position"]["volume"] == 1.0


def test_equity_reflects_unrealized_gain():
    p = Portfolio(10_000.0)
    p.apply_fill(
        Fill(order_id=1, ts=0, price=1.0, volume=1.0),
        Order(id=1, type=OrderType.BUY, volume=1.0),
        _tick(),
    )
    # Mark tick with higher bid: unrealized = +0.05.
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", _market(3), bar_idx=0,
        portfolio=p, params={}, current_tick=Tick(ts=100, bid=1.05, ask=1.0501),
    )
    assert ctx["equity"] == pytest.approx(p.cash + 0.05)


def test_current_price_is_mid():
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", _market(3), bar_idx=0,
        portfolio=Portfolio(10_000.0), params={},
        current_tick=Tick(ts=1, bid=1.0, ask=1.002),
    )
    assert ctx["current_price"] == pytest.approx(1.001)


# --- mode & params ------------------------------------------------------


def test_mode_is_stringified():
    ctx = build_context(
        RunMode.ADVICE, "EURUSD", "M1", _market(2), bar_idx=0,
        portfolio=Portfolio(10_000.0), params={"a": 1}, current_tick=_tick(),
    )
    assert ctx["mode"] == "advice"


def test_params_is_a_copy():
    params = {"a": 1}
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", _market(2), bar_idx=0,
        portfolio=Portfolio(10_000.0), params=params, current_tick=_tick(),
    )
    ctx["params"]["a"] = 999
    assert params["a"] == 1  # caller's dict untouched


def test_none_params_accepted():
    ctx = build_context(
        RunMode.BACKTEST, "EURUSD", "M1", _market(2), bar_idx=0,
        portfolio=Portfolio(10_000.0), params=None,  # type: ignore[arg-type]
        current_tick=_tick(),
    )
    assert ctx["params"] == {}
