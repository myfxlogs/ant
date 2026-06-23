"""SDK Strategy Worker — pure StrategyRuntime path (D7 final).

Replaces the old signal-dict sandbox.  All strategies run through
StrategyRuntime → LiveBroker → intents export → Go gate → mthub.

Entry point: process_bar(code, bar_context) → list[intent_dicts]

No RestrictedPython. No def run(context). No signal dict.
Clean SDK-only execution.
"""

from __future__ import annotations

import json
import sys
from decimal import Decimal
from typing import Any, Dict, List, Optional

from app.engine.live_broker import LiveBroker, build_live_broker_from_proto
from app.sdk import StrategyBase
from app.sdk.runtime import RuntimeContext, StrategyRuntime
from app.sdk.series import Bars, Series


# ── Cached runtime (persists across bars) ──────────────────────────────

_runtime: Optional[StrategyRuntime] = None
_runtime_hash: str = ""


def _load_strategy(code: str) -> type[StrategyBase]:
    """Compile and load a StrategyBase subclass from source code."""
    namespace: Dict[str, Any] = {}
    exec(code, namespace)
    for obj in namespace.values():
        if isinstance(obj, type) and issubclass(obj, StrategyBase) and obj is not StrategyBase:
            return obj
    raise ValueError("No StrategyBase subclass found in code")


def _build_bars_provider(bar_context: dict) -> callable:
    """Build a bars provider from the bar context dict (sent by Go live_runner)."""
    closes = bar_context.get("close", [])
    opens = bar_context.get("open", [])
    highs = bar_context.get("high", [])
    lows = bar_context.get("low", [])
    volumes = bar_context.get("volume", [])
    times = bar_context.get("bar_times_ms", [])

    class _LiveSeries(Series):
        def __init__(self, data):
            self._data = list(data)
        def __getitem__(self, shift):
            idx = len(self._data) - 1 - shift
            if idx < 0 or idx >= len(self._data):
                raise IndexError
            return self._data[idx]
        def __len__(self):
            return len(self._data)
        def slice(self, count):
            return self._data[-count:] if count <= len(self._data) else list(self._data)

    def provider(timeframe=None):
        b = Bars()
        b.timeframe = timeframe or bar_context.get("timeframe", "")
        n = len(closes)
        b.open = _LiveSeries(opens) if opens else _LiveSeries([0.0] * n)
        b.high = _LiveSeries(highs) if highs else _LiveSeries([0.0] * n)
        b.low = _LiveSeries(lows) if lows else _LiveSeries([0.0] * n)
        b.close = _LiveSeries(closes) if closes else _LiveSeries([0.0] * n)
        b.volume = _LiveSeries(volumes) if volumes else _LiveSeries([0.0] * n)
        b.time = _LiveSeries(times) if times else _LiveSeries([0.0] * n)
        b.total = lambda: len(b.close)
        return b
    return provider


def _build_indicators(bars_provider: callable) -> object:
    """Build a simple indicators implementation backed by the bars provider."""
    import numpy as np
    from app.sdk.indicators import Indicators

    class LiveIndicators(Indicators):
        def _get_close(self):
            bars = bars_provider()
            return np.array([bars.close[i] for i in range(len(bars.close)-1, -1, -1)])

        def ma(self, period=14, shift=0, method="sma"):
            data = self._get_close()
            if len(data) < period + shift: return 0.0
            window = data[:len(data)-shift] if shift > 0 else data
            if method in ("sma", "simple"):
                return float(np.mean(window[-period:]))
            if method in ("ema", "exponential"):
                alpha = 2.0 / (period + 1)
                result = window[0]
                for v in window[1:]: result = alpha * v + (1 - alpha) * result
                return float(result)
            return float(np.mean(window[-period:]))

        def ema(self, period=14, shift=0):
            return self.ma(period, shift, "ema")

        def rsi(self, period=14, shift=0):
            data = self._get_close()
            if len(data) < period + shift + 1: return 50.0
            window = data[:len(data)-shift] if shift > 0 else data
            deltas = np.diff(window[-period-1:])
            gains = np.sum(deltas[deltas > 0])
            losses = -np.sum(deltas[deltas < 0])
            if losses == 0: return 100.0 if gains > 0 else 50.0
            rs = gains / losses
            return float(100.0 - 100.0 / (1.0 + rs))

        def bands(self, period=20, deviation=2.0, shift=0):
            data = self._get_close()
            if len(data) < period: return (0.0, 0.0, 0.0)
            middle = self.ma(period, shift, "sma")
            window = data[:len(data)-shift] if shift > 0 else data
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

    return LiveIndicators()


def process_bar(code: str, bar_context: dict) -> Dict[str, Any]:
    """Process one bar event through the SDK strategy.

    Called by Go live_runner for each new bar.
    Returns {"intents": [...], "error": ""} or {"intents": [], "error": "..."}.
    """
    global _runtime, _runtime_hash

    code_hash = str(hash(code))
    try:
        # Initialize or reuse runtime.
        if _runtime is None or _runtime_hash != code_hash:
            strategy_cls = _load_strategy(code)
            broker = build_live_broker_from_proto(bar_context)
            bars_fn = _build_bars_provider(bar_context)
            indicators = _build_indicators(bars_fn)

            _runtime = StrategyRuntime(
                strategy_class=strategy_cls,
                broker=broker,
                bars_provider=bars_fn,
                indicators=indicators,
                symbol=bar_context.get("symbol", ""),
                timeframe=bar_context.get("timeframe", ""),
                params={p.get("key"): p.get("value") for p in bar_context.get("params", [])},
            )
            _runtime.init()
            _runtime_hash = code_hash

        # Update broker state from latest bar context.
        if hasattr(_runtime._broker, 'update_state'):
            acct = bar_context.get("account")
            if acct:
                from app.sdk.account import AccountInfo, AccountMode
                _runtime._broker.update_state(
                    account=AccountInfo(
                        balance=Decimal(str(acct.get("balance", 0))),
                        equity=Decimal(str(acct.get("equity", 0))),
                        margin=Decimal(str(acct.get("margin", 0))),
                        free_margin=Decimal(str(acct.get("free_margin", 0))),
                        margin_level=Decimal(str(acct.get("margin_level", 0))),
                        leverage=acct.get("leverage", 100),
                        currency=acct.get("currency", "USD"),
                        mode=AccountMode.HEDGING,
                    ),
                )

        # Drive strategy.
        _runtime.on_bar(bar_context.get("timeframe", ""))

        # Export intents.
        intents = _runtime.export_intents()
        return {"intents": intents, "error": ""}

    except Exception as e:
        return {"intents": [], "error": str(e)}


def reset_runtime():
    """Reset cached runtime — called on strategy stop."""
    global _runtime, _runtime_hash
    _runtime = None
    _runtime_hash = ""
