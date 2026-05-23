"""MQL-style technical indicators for the sandbox.

Implementations are a straight port of the logic previously embedded in
``app/executor.py`` so that strategies produce identical values before and
after the v2 engine migration (契约 I: test_indicators bit-equal).

All functions are pure: they take price arrays and return floats/tuples.
``shift=0`` means "latest bar"; ``shift=1`` means "one bar back" (MQL convention).
"""

from __future__ import annotations

from typing import Tuple

import numpy as np

# Re-export the sizing helpers so the sandbox can inject a single module.
from app.risk_sizing import atr_size, kelly_size, risk_size  # noqa: F401

__all__ = [
    "iMA",
    "iRSI",
    "iBands",
    "iMACD",
    "iStochastic",
    "iATR",
    "iCCI",
    "iMomentum",
    "iWPR",
    "OrdersTotal",
    "AccountBalance",
    "AccountEquity",
    "risk_size",
    "atr_size",
    "kelly_size",
]


def _arr(prices) -> np.ndarray:
    return np.asarray(prices, dtype=float)


def iMA(prices, period: int = 14, shift: int = 0, method: str = "sma") -> float:
    a = _arr(prices)
    if len(a) < period + shift:
        return 0.0
    window = a[: len(a) - shift] if shift > 0 else a
    if method in ("sma", "simple"):
        return float(np.mean(window[-period:]))
    if method in ("ema", "exponential"):
        k = 2.0 / (period + 1)
        ema = float(np.mean(window[:period]))
        for v in window[period:]:
            ema = v * k + ema * (1 - k)
        return ema
    if method in ("wma", "weighted"):
        weights = np.arange(1, period + 1, dtype=float)
        return float(np.dot(window[-period:], weights) / weights.sum())
    return float(np.mean(window[-period:]))


def iRSI(prices, period: int = 14, shift: int = 0) -> float:
    a = _arr(prices)
    if shift > 0:
        a = a[:-shift]
    if len(a) < period + 1:
        return 50.0
    deltas = np.diff(a[-(period + 1):])
    gains = np.where(deltas > 0, deltas, 0.0)
    losses = np.where(deltas < 0, -deltas, 0.0)
    avg_gain = np.mean(gains)
    avg_loss = np.mean(losses)
    if avg_loss == 0:
        return 100.0
    rs = avg_gain / avg_loss
    return float(100.0 - 100.0 / (1.0 + rs))


def iBands(
    prices,
    period: int = 20,
    deviation: float = 2.0,
    shift: int = 0,
) -> Tuple[float, float, float]:
    a = _arr(prices)
    if shift > 0:
        a = a[:-shift]
    if len(a) < period:
        mid = float(a[-1]) if len(a) > 0 else 0.0
        return mid, mid, mid
    window = a[-period:]
    mid = float(np.mean(window))
    std = float(np.std(window))
    return mid + deviation * std, mid, mid - deviation * std


def _ema_series(arr: np.ndarray, n: int) -> np.ndarray:
    k = 2.0 / (n + 1)
    result = [float(np.mean(arr[:n]))]
    for v in arr[n:]:
        result.append(v * k + result[-1] * (1 - k))
    return np.array(result)


def iMACD(
    prices,
    fast: int = 12,
    slow: int = 26,
    signal_period: int = 9,
    shift: int = 0,
) -> Tuple[float, float, float]:
    a = _arr(prices)
    if shift > 0:
        a = a[:-shift]
    if len(a) < slow + signal_period:
        return 0.0, 0.0, 0.0
    fast_ema = _ema_series(a, fast)
    slow_ema = _ema_series(a, slow)
    min_len = min(len(fast_ema), len(slow_ema))
    macd_line = fast_ema[-min_len:] - slow_ema[-min_len:]
    if len(macd_line) < signal_period:
        return float(macd_line[-1]), 0.0, 0.0
    sig_line = _ema_series(macd_line, signal_period)
    macd_val = float(macd_line[-1])
    sig_val = float(sig_line[-1])
    return macd_val, sig_val, macd_val - sig_val


def iStochastic(
    high,
    low,
    close,
    k_period: int = 5,
    d_period: int = 3,
    shift: int = 0,
) -> Tuple[float, float]:
    h = _arr(high)
    l = _arr(low)
    c = _arr(close)
    if shift > 0:
        h = h[:-shift]
        l = l[:-shift]
        c = c[:-shift]
    if len(c) < k_period:
        return 50.0, 50.0
    highest = np.max(h[-k_period:])
    lowest = np.min(l[-k_period:])
    denom = highest - lowest
    k = float((c[-1] - lowest) / denom * 100) if denom != 0 else 50.0
    d = k
    return k, d


def iATR(high, low, close, period: int = 14, shift: int = 0) -> float:
    h = _arr(high)
    l = _arr(low)
    c = _arr(close)
    if shift > 0:
        h = h[:-shift]
        l = l[:-shift]
        c = c[:-shift]
    if len(c) < 2:
        return 0.0
    tr = np.maximum(
        h[1:] - l[1:],
        np.maximum(np.abs(h[1:] - c[:-1]), np.abs(l[1:] - c[:-1])),
    )
    if len(tr) < period:
        return float(np.mean(tr)) if len(tr) > 0 else 0.0
    return float(np.mean(tr[-period:]))


def iCCI(high, low, close, period: int = 14, shift: int = 0) -> float:
    h = _arr(high)
    l = _arr(low)
    c = _arr(close)
    if shift > 0:
        h = h[:-shift]
        l = l[:-shift]
        c = c[:-shift]
    if len(c) < period:
        return 0.0
    tp = (h[-period:] + l[-period:] + c[-period:]) / 3.0
    mean_tp = np.mean(tp)
    mean_dev = np.mean(np.abs(tp - mean_tp))
    if mean_dev == 0:
        return 0.0
    return float((tp[-1] - mean_tp) / (0.015 * mean_dev))


def iMomentum(prices, period: int = 14, shift: int = 0) -> float:
    a = _arr(prices)
    if shift > 0:
        a = a[:-shift]
    if len(a) <= period:
        return 100.0
    return float(a[-1] / a[-period - 1] * 100.0)


def iWPR(high, low, close, period: int = 14, shift: int = 0) -> float:
    h = _arr(high)
    l = _arr(low)
    c = _arr(close)
    if shift > 0:
        h = h[:-shift]
        l = l[:-shift]
        c = c[:-shift]
    if len(c) < period:
        return -50.0
    highest = np.max(h[-period:])
    lowest = np.min(l[-period:])
    denom = highest - lowest
    if denom == 0:
        return -50.0
    return float((highest - c[-1]) / denom * -100.0)


# --- Context queries (read from the dict the sandbox builds) -------------


def OrdersTotal(context: dict) -> int:
    return int(context.get("positions_total", 0))


def AccountBalance(context: dict) -> float:
    return float(context.get("account_balance", 10000.0))


def AccountEquity(context: dict) -> float:
    return float(context.get("account_equity", 10000.0))
