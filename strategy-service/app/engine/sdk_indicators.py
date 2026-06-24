"""Shared SDK Indicators — single implementation for backtest and live.

Both runner.py and sdk_worker.py previously had their own copies.
This module provides the canonical implementation.
"""

from __future__ import annotations

import numpy as np


class SDKIndicators:
    """Indicators backed by engine data, implementing the SDK Indicators interface.

    ``bars_provider`` must be a callable ``(timeframe=None) -> Bars``
    that returns Bars with MQL reverse-indexed Series (bars.close[0]
    is current bar, bars.close[1] is previous).
    """

    def __init__(self, bars_provider):
        self._bars_provider = bars_provider

    def _get_close(self) -> np.ndarray:
        bars = self._bars_provider()
        return np.array([bars.close[i] for i in range(len(bars.close) - 1, -1, -1)])

    def ma(self, period=14, shift=0, method="sma"):
        data = self._get_close()
        if len(data) < period + shift:
            return 0.0
        window = data[:len(data) - shift] if shift > 0 else data
        if method in ("ema", "exponential"):
            alpha = 2.0 / (period + 1)
            result = float(window[-period])
            for v in window[-period + 1:]:
                result = alpha * v + (1 - alpha) * result
            return result
        return float(np.mean(window[-period:]))

    def ema(self, period=14, shift=0):
        return self.ma(period, shift, "ema")

    def rsi(self, period=14, shift=0):
        data = self._get_close()
        if len(data) < period + shift + 1:
            return 50.0
        window = data[:len(data) - shift] if shift > 0 else data
        deltas = np.diff(window[-period - 1:])
        gains = np.sum(deltas[deltas > 0])
        losses = -np.sum(deltas[deltas < 0])
        if losses == 0:
            return 100.0 if gains > 0 else 50.0
        return float(100.0 - 100.0 / (1.0 + gains / losses))

    def bands(self, period=20, deviation=2.0, shift=0):
        data = self._get_close()
        if len(data) < period:
            return (0.0, 0.0, 0.0)
        middle = self.ma(period, shift, "sma")
        window = data[:len(data) - shift] if shift > 0 else data
        std = float(np.std(window[-period:]))
        return (middle + deviation * std, middle, middle - deviation * std)

    def macd(self, fast=12, slow=26, signal=9, shift=0):
        return (0.0, 0.0, 0.0)

    def atr(self, period=14, shift=0):
        return 0.001

    def stochastic(self, k_period=5, d_period=3, shift=0):
        return (50.0, 50.0)

    def cci(self, period=14, shift=0):
        return 0.0

    def i_custom(self, name, params=(), buffer=0, shift=0):
        return 0.0
