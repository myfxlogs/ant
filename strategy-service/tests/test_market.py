"""Tests for app/engine/market.py."""

from __future__ import annotations

import numpy as np
import pytest

from app.engine.market import MarketSimulator, TickSimulator
from app.engine.types import Bar, Tick


def _bar(open_t: int, close_t: int, *, ohlc=(1.0, 1.1, 0.9, 1.05), vol: float = 100.0) -> Bar:
    o, h, l, c = ohlc
    return Bar(open_time=open_t, close_time=close_t, open=o, high=h, low=l, close=c, volume=vol)


def _bars(n: int, step_ms: int = 60_000) -> list[Bar]:
    bars = []
    for i in range(n):
        ot = i * step_ms
        ct = ot + step_ms
        bars.append(_bar(ot, ct, ohlc=(1.0 + i * 0.001, 1.0 + i * 0.001 + 0.0005,
                                        1.0 + i * 0.001 - 0.0005, 1.0 + (i + 1) * 0.001)))
    return bars


# --- TickSimulator -------------------------------------------------------


def test_tick_simulator_replays_supplied_ticks():
    ticks = [Tick(ts=10, bid=1.0, ask=1.0001), Tick(ts=20, bid=1.2, ask=1.2001)]
    sim = TickSimulator(bars=[], ticks=ticks)
    assert list(sim) == ticks
    assert len(sim) == 2
    assert sim.synthetic is False


def test_tick_simulator_synthesizes_from_bars_when_ticks_missing():
    bars = _bars(3)
    sim = TickSimulator(bars=bars, ticks=None)
    out = list(sim)
    assert len(out) == 3
    for t, b in zip(out, bars):
        assert t.ts == b.close_time
        assert t.bid == b.close
        assert t.ask == b.close
    assert sim.synthetic is True


def test_tick_simulator_treats_empty_ticks_as_missing():
    bars = _bars(2)
    sim = TickSimulator(bars=bars, ticks=[])
    assert len(sim) == 2
    assert sim.synthetic is True


def test_tick_simulator_defensive_copy():
    ticks = [Tick(ts=10, bid=1.0, ask=1.0001)]
    sim = TickSimulator(bars=[], ticks=ticks)
    ticks.append(Tick(ts=20, bid=2.0, ask=2.0001))
    assert len(sim) == 1  # external mutation must not leak in.


def test_tick_simulator_iterable_is_stable_across_iterations():
    sim = TickSimulator(bars=_bars(3))
    first = list(sim)
    second = list(sim)
    assert first == second


# --- MarketSimulator -----------------------------------------------------


def test_market_simulator_len_matches_bars():
    sim = MarketSimulator(_bars(5))
    assert len(sim) == 5


def test_bar_closed_at_or_before_returns_minus_one_for_empty():
    sim = MarketSimulator([])
    assert sim.bar_closed_at_or_before(100) == -1


def test_bar_closed_at_or_before_before_first_close():
    bars = _bars(3)  # close_times: 60_000, 120_000, 180_000
    sim = MarketSimulator(bars)
    assert sim.bar_closed_at_or_before(bars[0].close_time - 1) == -1


def test_bar_closed_at_or_before_exactly_on_close():
    bars = _bars(3)
    sim = MarketSimulator(bars)
    assert sim.bar_closed_at_or_before(bars[1].close_time) == 1


def test_bar_closed_at_or_before_between_closes():
    bars = _bars(3)
    sim = MarketSimulator(bars)
    ts = bars[1].close_time + 1
    assert sim.bar_closed_at_or_before(ts) == 1


def test_bar_closed_at_or_before_past_end():
    bars = _bars(3)
    sim = MarketSimulator(bars)
    assert sim.bar_closed_at_or_before(bars[-1].close_time + 10_000) == 2


def test_slice_until_returns_inclusive_ndarray():
    bars = _bars(4)
    sim = MarketSimulator(bars)
    view = sim.slice_until(2)
    assert isinstance(view["close"], np.ndarray)
    assert len(view["close"]) == 3
    np.testing.assert_array_equal(view["close"], [b.close for b in bars[:3]])
    # Other columns present
    for k in ("open", "high", "low", "volume"):
        assert len(view[k]) == 3


def test_slice_until_rejects_out_of_range():
    sim = MarketSimulator(_bars(3))
    with pytest.raises(IndexError):
        sim.slice_until(3)
    with pytest.raises(IndexError):
        sim.slice_until(-1)


def test_slice_first_bar_has_length_one():
    sim = MarketSimulator(_bars(3))
    view = sim.slice_until(0)
    assert len(view["open"]) == 1
