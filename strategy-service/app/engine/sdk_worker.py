"""SDK Strategy Worker — pure StrategyRuntime path (D7 final).

Replaces the old signal-dict sandbox.  All strategies run through
StrategyRuntime → LiveBroker → intents export → Go gate → mthub.

Entry point: process_bar(code, bar_context) → list[intent_dicts]

No RestrictedPython. No def run(context). No signal dict.
Clean SDK-only execution.
"""

from __future__ import annotations

import hashlib
import json
import sys
import threading
from decimal import Decimal
from typing import Any, Dict, List, Optional

from app.engine.live_broker import LiveBroker, build_live_broker_from_proto
from app.engine.sdk_indicators import SDKIndicators
from app.engine.sdk_loader import load_sdk_strategy
from app.sdk.account import AccountInfo, AccountMode
from app.sdk.runtime import RuntimeContext, StrategyRuntime
from app.sdk.series import Bars, Series


# ── Cached runtime pool (per-strategy, thread-safe) ─────────────────────

_runtimes: Dict[str, StrategyRuntime] = {}
_lock = threading.Lock()


def _get_or_create_runtime(code: str, bar_context: dict) -> StrategyRuntime:
    """Get or create a StrategyRuntime for the given code hash (thread-safe)."""
    import hashlib
    code_hash = hashlib.sha256(code.encode()).hexdigest()

    with _lock:
        runtime = _runtimes.get(code_hash)
        if runtime is not None:
            return runtime

        # Create new runtime under lock
        strategy_cls = load_sdk_strategy(code)
        broker = build_live_broker_from_proto(bar_context)
        bars_fn = _build_bars_provider(bar_context)
        indicators = SDKIndicators(bars_fn)

        runtime = StrategyRuntime(
            strategy_class=strategy_cls,
            broker=broker,
            bars_provider=bars_fn,
            indicators=indicators,
            symbol=bar_context.get("symbol", ""),
            timeframe=bar_context.get("timeframe", "1h"),
            params=bar_context.get("params", {}),
        )
        runtime.init()
        _runtimes[code_hash] = runtime
        return runtime


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


def process_bar(code: str, bar_context: dict) -> Dict[str, Any]:
    """Process one bar event through the SDK strategy.

    Called by Go live_runner for each new bar.
    Returns {"intents": [...], "error": ""} or {"intents": [], "error": "..."}.
    """
    try:
        runtime = _get_or_create_runtime(code, bar_context)

        # Update broker state from latest bar context.
        if hasattr(runtime._broker, 'update_state'):
            # Account state — check for sentinel values (account disconnected).
            equity = bar_context.get("equity", 0)
            balance = bar_context.get("balance", 0)
            if equity == -1.0:
                equity = 0
            if balance == -1.0:
                balance = 0
            runtime._broker.update_state(
                account=AccountInfo(
                    balance=Decimal(str(balance)),
                    equity=Decimal(str(equity)),
                    margin=Decimal("0"),
                    free_margin=Decimal("0"),
                    margin_level=Decimal("0"),
                    leverage=100, currency="USD", mode=AccountMode.HEDGING,
                ),
            )
            # Positions from MT4 backfill.
            positions_raw = bar_context.get("positions", [])
            if positions_raw:
                from app.sdk.types import Position, PositionSide
                positions = []
                for lp in positions_raw:
                    positions.append(Position(
                        ticket=int(lp.get("ticket", 0)),
                        symbol=bar_context.get("symbol", ""),
                        side=PositionSide.BUY if lp.get("side", "buy") == "buy" else PositionSide.SELL,
                        volume=Decimal(str(lp.get("volume", 0))),
                        open_price=Decimal(str(lp.get("openPrice", lp.get("open_price", 0)))),
                        sl=Decimal(str(lp.get("sl", 0))) if lp.get("sl", 0) else None,
                        tp=Decimal(str(lp.get("tp", 0))) if lp.get("tp", 0) else None,
                        profit=Decimal("0"),
                        swap=Decimal("0"),
                        open_time_ms=0,
                    ))
                runtime._broker.update_state(positions=positions)

        # Drive strategy.
        runtime.on_bar(bar_context.get("timeframe", ""))

        # Export intents.
        intents = runtime.export_intents()
        return {"intents": intents, "error": ""}

    except Exception as e:
        return {"intents": [], "error": str(e)}


def reset_runtime():
    """Reset cached runtime — called on strategy stop."""
    global _runtime, _runtime_hash
    _runtime = None
    _runtime_hash = ""
