"""Tests for app/engine/metrics.py.

Formulas are pinned to the legacy ``app/backtest.py`` behaviour to preserve
the I9 bit-equal golden standard.
"""

from __future__ import annotations

import math

import numpy as np
import pytest

from app.engine.metrics import build_metrics
from app.engine.types import Bar, CloseReason, Side, Trade


# --- fixtures ------------------------------------------------------------


def _h1_bars(n: int, start_ms: int = 0) -> list[Bar]:
    step = 3_600_000  # 1 hour in ms
    bars = []
    for i in range(n):
        ot = start_ms + i * step
        ct = ot + step
        bars.append(
            Bar(open_time=ot, close_time=ct, open=1.0, high=1.1, low=0.9, close=1.0)
        )
    return bars


def _trade(pnl: float) -> Trade:
    return Trade(
        ticket=1,
        side=Side.BUY,
        volume=1.0,
        open_ts=0,
        open_price=1.0,
        close_ts=1_000,
        close_price=1.0 + pnl,
        pnl=pnl,
        commission=0.0,
        reason=CloseReason.SIGNAL,
    )


# --- total_return / max_drawdown ----------------------------------------


def test_empty_curve_yields_zero_metrics():
    m, r = build_metrics([], [], _h1_bars(2))
    assert m.total_return == 0.0
    assert m.max_drawdown == 0.0
    assert r.is_reliable is False


def test_flat_curve_total_return_zero():
    m, _ = build_metrics([10_000.0, 10_000.0, 10_000.0], [], _h1_bars(3))
    assert m.total_return == 0.0
    assert m.max_drawdown == 0.0
    assert m.sharpe_ratio == 0.0


def test_rising_curve_positive_return_zero_drawdown():
    m, _ = build_metrics([100.0, 110.0, 121.0], [], _h1_bars(3))
    assert m.total_return == pytest.approx(0.21)
    assert m.max_drawdown == 0.0


def test_drawdown_from_peak():
    m, _ = build_metrics([100.0, 150.0, 75.0], [], _h1_bars(3))
    # Peak 150, trough 75 → dd = 0.5
    assert m.max_drawdown == pytest.approx(0.5)


# --- sharpe -------------------------------------------------------------


def test_sharpe_zero_when_std_is_zero():
    # Craft equal per-step returns exactly (flat curve → all zeros → std=0).
    m, _ = build_metrics([100.0, 100.0, 100.0, 100.0], [], _h1_bars(4))
    assert m.sharpe_ratio == 0.0


def test_sharpe_annualized_bars_per_year_from_timeframe():
    # H1 bars: bars_per_year ≈ 8760.
    bars = _h1_bars(5)
    # Crafted returns with known mean/std so we can verify annualization factor.
    curve = [100.0, 101.0, 100.5, 102.0, 103.0]
    m, _ = build_metrics(curve, [], bars)
    eq = np.array(curve)
    step = np.diff(eq) / eq[:-1]
    expected = np.mean(step) / np.std(step) * np.sqrt(8760)
    assert m.sharpe_ratio == pytest.approx(expected)


# --- annual_return ------------------------------------------------------


def test_annual_return_over_one_year_equals_total():
    # 1 year H1 bars ≈ 8760 bars.
    bars = _h1_bars(8760)
    m, _ = build_metrics([100.0, 110.0], [], bars)
    # years ≈ 1, so annual ≈ total_return.
    assert m.annual_return == pytest.approx(0.1, rel=0.01)


def test_annual_return_negative_total_returns_minus_one_when_below_threshold():
    bars = _h1_bars(2)
    m, _ = build_metrics([100.0, 0.0], [], bars)
    assert m.total_return == -1.0
    assert m.annual_return == -1.0


# --- profit_factor / win_rate -------------------------------------------


def test_all_winners_profit_factor_capped_at_99():
    trades = [_trade(1.0), _trade(2.0)]
    m, _ = build_metrics([100.0, 103.0], trades, _h1_bars(2))
    assert m.profit_factor == 99.0
    assert m.win_rate == 1.0
    assert m.total_trades == 2


def test_all_losers_profit_factor_zero():
    trades = [_trade(-1.0), _trade(-0.5)]
    m, _ = build_metrics([100.0, 98.5], trades, _h1_bars(2))
    assert m.profit_factor == 0.0
    assert m.win_rate == 0.0


def test_mixed_profit_factor_computed():
    trades = [_trade(2.0), _trade(-1.0), _trade(3.0), _trade(-1.0)]
    m, _ = build_metrics([100.0, 103.0], trades, _h1_bars(2))
    # gross_profit = 5, gross_loss = 2 → PF = 2.5.
    assert m.profit_factor == pytest.approx(2.5)
    assert m.win_rate == pytest.approx(0.5)


def test_profit_factor_large_value_capped_at_99():
    # 100 winners of 1, 1 loser of 0.001 → PF = 100_000 > 99 → 99.
    # Use 1-year horizon to avoid annual_return overflow.
    trades = [_trade(1.0)] * 100 + [_trade(-0.001)]
    m, _ = build_metrics([100.0, 199.0], trades, _h1_bars(8760))
    assert m.profit_factor == 99.0


# --- trade stats averages -----------------------------------------------


def test_average_profit_and_loss_symmetry():
    trades = [_trade(2.0), _trade(4.0), _trade(-1.0), _trade(-3.0)]
    m, _ = build_metrics([100.0, 102.0], trades, _h1_bars(2))
    assert m.average_profit == pytest.approx(3.0)
    assert m.average_loss == pytest.approx(2.0)
    assert m.winning_trades == 2
    assert m.losing_trades == 2


# --- risk assessment ----------------------------------------------------


def test_risk_score_clamped_and_level_maps():
    # 0% dd → score 100 → low.
    _, r = build_metrics([100.0, 101.0], [], _h1_bars(2))
    assert r.score == 100
    assert r.level == "low"


def test_risk_score_high_dd_maps_to_high():
    _, r = build_metrics([100.0, 150.0, 10.0], [], _h1_bars(3))
    # dd = (150-10)/150 ≈ 0.933 → score = 100 - 186 → clamp to 0 → high.
    assert r.score == 0
    assert r.level == "high"


def test_warnings_fire_for_small_sample():
    _, r = build_metrics([100.0, 101.0], [_trade(1.0)], _h1_bars(2))
    assert any("样本数据较少" in w for w in r.warnings)
    assert r.is_reliable is False


def test_warnings_fire_for_negative_sharpe():
    # Descending curve produces negative sharpe.
    _, r = build_metrics([100.0, 95.0, 90.0, 85.0, 80.0], [], _h1_bars(5))
    assert any("夏普比率为负" in w for w in r.warnings)


def test_warnings_fire_for_deep_drawdown():
    _, r = build_metrics([100.0, 150.0, 50.0], [], _h1_bars(3))
    assert any("偏高" in w for w in r.warnings)


def test_is_reliable_threshold():
    trades = [_trade(1.0)] * 10
    _, r = build_metrics([100.0, 110.0], trades, _h1_bars(2))
    assert r.is_reliable is True


# --- dimensional sanity -------------------------------------------------


def test_metrics_fields_are_floats_or_ints():
    m, _ = build_metrics([100.0, 105.0], [_trade(5.0)], _h1_bars(2))
    assert isinstance(m.total_return, float)
    assert isinstance(m.total_trades, int)
    assert math.isfinite(m.sharpe_ratio)
