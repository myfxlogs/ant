"""Tests for app/engine/indicators.py.

Indicator formulas were ported verbatim from the legacy executor (now deleted);
these tests pin their numerical output so future refactors cannot silently drift.
"""

from __future__ import annotations

import math

import numpy as np
import pytest

from app.engine.indicators import (
    AccountBalance,
    AccountEquity,
    OrdersTotal,
    iATR,
    iBands,
    iCCI,
    iMA,
    iMACD,
    iMomentum,
    iRSI,
    iStochastic,
    iWPR,
    atr_size,
    kelly_size,
    risk_size,
)


# Deterministic price fixture (30 bars, arithmetic progression 1.0 -> 1.29).
@pytest.fixture
def ramp() -> np.ndarray:
    return np.round(np.linspace(1.0, 1.29, 30), 6)


@pytest.fixture
def ramp_high(ramp: np.ndarray) -> np.ndarray:
    return ramp + 0.005


@pytest.fixture
def ramp_low(ramp: np.ndarray) -> np.ndarray:
    return ramp - 0.005


# --- iMA ----------------------------------------------------------------


def test_iMA_sma_matches_numpy_mean(ramp):
    assert iMA(ramp, period=5, method="sma") == pytest.approx(float(np.mean(ramp[-5:])))


def test_iMA_shift_excludes_latest(ramp):
    assert iMA(ramp, period=5, shift=1, method="sma") == pytest.approx(
        float(np.mean(ramp[-6:-1]))
    )


def test_iMA_wma_matches_weighted_formula(ramp):
    w = np.arange(1, 6, dtype=float)
    expected = float(np.dot(ramp[-5:], w) / w.sum())
    assert iMA(ramp, period=5, method="wma") == pytest.approx(expected)


def test_iMA_ema_is_finite(ramp):
    v = iMA(ramp, period=5, method="ema")
    assert math.isfinite(v)
    assert v > 1.0


def test_iMA_short_history_returns_zero():
    assert iMA([1.0, 1.0], period=5) == 0.0


# --- iRSI ---------------------------------------------------------------


def test_iRSI_monotonic_rise_returns_100(ramp):
    # ramp is strictly increasing -> no losses -> RSI saturates at 100.
    assert iRSI(ramp, period=14) == 100.0


def test_iRSI_short_history_returns_50():
    assert iRSI([1.0], period=14) == 50.0


def test_iRSI_known_mixed_series():
    # Two down deltas among fifteen total -> avg_gain > 0, avg_loss > 0.
    series = [1.0] * 15 + [1.02, 1.01, 1.03, 1.02, 1.04]
    v = iRSI(series, period=14)
    assert 0.0 < v < 100.0


# --- iBands -------------------------------------------------------------


def test_iBands_returns_ordered_triplet(ramp):
    upper, mid, lower = iBands(ramp, period=20, deviation=2.0)
    assert lower <= mid <= upper
    assert mid == pytest.approx(float(np.mean(ramp[-20:])))


def test_iBands_short_history_collapses_to_mid():
    upper, mid, lower = iBands([1.0, 1.1], period=20)
    assert upper == mid == lower == 1.1


# --- iMACD --------------------------------------------------------------


def test_iMACD_returns_three_floats(ramp):
    # Need at least slow + signal = 35 bars; short input must early-return zeros.
    assert iMACD(ramp, fast=12, slow=26, signal_period=9) == (0.0, 0.0, 0.0)


def test_iMACD_long_series_is_finite():
    prices = np.linspace(1.0, 2.0, 100)
    macd, sig, hist = iMACD(prices)
    assert all(math.isfinite(x) for x in (macd, sig, hist))
    assert hist == pytest.approx(macd - sig)


# --- iStochastic --------------------------------------------------------


def test_iStochastic_at_top_of_range(ramp, ramp_high, ramp_low):
    k, d = iStochastic(ramp_high, ramp_low, ramp, k_period=5)
    assert k == pytest.approx(d)
    assert 0.0 <= k <= 100.0


def test_iStochastic_flat_returns_50():
    flat = [1.0] * 10
    k, d = iStochastic(flat, flat, flat, k_period=5)
    assert k == 50.0 and d == 50.0


# --- iATR ---------------------------------------------------------------


def test_iATR_constant_spread(ramp_high, ramp_low, ramp):
    v = iATR(ramp_high, ramp_low, ramp, period=14)
    assert math.isfinite(v)
    assert v > 0.0


def test_iATR_short_history_returns_zero():
    assert iATR([1.0], [1.0], [1.0], period=14) == 0.0


# --- iCCI ---------------------------------------------------------------


def test_iCCI_constant_returns_zero():
    flat = [1.0] * 20
    assert iCCI(flat, flat, flat, period=14) == 0.0


def test_iCCI_on_ramp_is_positive(ramp, ramp_high, ramp_low):
    # Increasing typical price with mean_dev > 0 -> positive CCI.
    assert iCCI(ramp_high, ramp_low, ramp, period=14) > 0.0


# --- iMomentum ----------------------------------------------------------


def test_iMomentum_ratio(ramp):
    v = iMomentum(ramp, period=14)
    assert v == pytest.approx(ramp[-1] / ramp[-15] * 100.0)


def test_iMomentum_short_history_returns_100():
    assert iMomentum([1.0, 1.1], period=14) == 100.0


# --- iWPR ---------------------------------------------------------------


def test_iWPR_ramp_near_top_of_range(ramp, ramp_high, ramp_low):
    v = iWPR(ramp_high, ramp_low, ramp, period=14)
    assert -100.0 <= v <= 0.0


def test_iWPR_flat_returns_minus_50():
    flat = [1.0] * 20
    assert iWPR(flat, flat, flat, period=14) == -50.0


# --- context queries ----------------------------------------------------


def test_orders_total_reads_from_context():
    assert OrdersTotal({"positions_total": 3}) == 3
    assert OrdersTotal({}) == 0


def test_account_balance_and_equity_defaults():
    assert AccountBalance({}) == 10000.0
    assert AccountEquity({}) == 10000.0
    assert AccountBalance({"account_balance": 5000.0}) == 5000.0
    assert AccountEquity({"account_equity": 7500.0}) == 7500.0


# --- risk sizing sanity (re-exports from app.risk_sizing) ---------------


def test_risk_size_rejects_zero_equity():
    assert risk_size(0.0, 0.01, 1.0, 0.99) == 0.0


def test_atr_size_floor_on_zero_atr():
    assert atr_size(10_000.0, 0.01, atr_value=0.0) > 0.0  # returns min_lot


def test_kelly_size_invalid_returns_min_lot():
    assert kelly_size(10_000.0, win_rate=0.0, avg_win=1.0, avg_loss=1.0) > 0.0
