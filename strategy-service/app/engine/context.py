"""Execution context builder.

契约：docs/domains/backtest-system.md §7.4.3 · context.py

Produces the ``dict`` that gets passed to the sandbox ``run(context)``. Keys
are aligned with the legacy ``app/backtest.py`` contract **plus** the fields
previously missing (``cash``, ``equity``, ``position``, ``positions``,
``positions_total`` — see bug D8). Keeping it a plain dict preserves
backwards compatibility with existing strategies that do
``context['close']`` / ``context.get('positions_total', 0)`` etc.
"""

from __future__ import annotations

from dataclasses import asdict
from typing import Optional, Union

from app.engine.market import MarketSimulator, MultiSymbolMarket
from app.engine.portfolio import Portfolio
from app.engine.types import Position, RunMode, Tick


def _position_to_dict(pos: Optional[Position]) -> Optional[dict]:
    if pos is None:
        return None
    d = asdict(pos)
    # Enum → string for strategies reading raw dicts.
    d["side"] = pos.side.value
    return d


def build_context(
    mode: RunMode,
    symbol: str,
    timeframe: str,
    market: Union[MarketSimulator, MultiSymbolMarket],
    bar_idx: int,
    portfolio: Portfolio,
    params: dict,
    current_tick: Tick,
) -> dict:
    """Build the per-bar ``context`` dict.

    ``bar_idx`` must satisfy ``0 <= bar_idx < len(primary_market)``.

    When ``market`` is a :class:`MultiSymbolMarket` the returned context also
    carries ``*_by_symbol`` ndarrays aligned to the primary bar close (no
    look-ahead). The primary OHLCV is still exposed as ``close/open/...`` so
    existing single-symbol strategies remain unaffected.
    """
    if isinstance(market, MultiSymbolMarket):
        primary_sym = market.primary
        all_slices = market.slice_until(bar_idx)
        primary_slices = all_slices[primary_sym]
        closes_by_symbol = {s: sl["close"] for s, sl in all_slices.items()}
        opens_by_symbol = {s: sl["open"] for s, sl in all_slices.items()}
        highs_by_symbol = {s: sl["high"] for s, sl in all_slices.items()}
        lows_by_symbol = {s: sl["low"] for s, sl in all_slices.items()}
        volumes_by_symbol = {s: sl["volume"] for s, sl in all_slices.items()}
        symbols_list = market.symbols()
    else:
        primary_sym = symbol
        primary_slices = market.slice_until(bar_idx)
        closes_by_symbol = {symbol: primary_slices["close"]}
        opens_by_symbol = {symbol: primary_slices["open"]}
        highs_by_symbol = {symbol: primary_slices["high"]}
        lows_by_symbol = {symbol: primary_slices["low"]}
        volumes_by_symbol = {symbol: primary_slices["volume"]}
        symbols_list = [symbol]

    positions = portfolio.positions
    latest: Optional[Position] = positions[-1] if positions else None
    equity = portfolio.equity(current_tick)
    current_price = (current_tick.bid + current_tick.ask) / 2.0
    ctx = {
        # Legacy-shaped OHLCV (numpy ndarrays, primary symbol).
        "close": primary_slices["close"],
        "open": primary_slices["open"],
        "high": primary_slices["high"],
        "low": primary_slices["low"],
        "volume": primary_slices["volume"],
        # Bar close times for EA-like time-driven strategies (DCA etc.).
        "bar_times_ms": primary_slices.get("close_time"),
        "bar_time_ms": int(primary_slices.get("close_time")[-1]) if primary_slices.get("close_time") is not None and len(primary_slices.get("close_time")) > 0 else None,
        # Multi-symbol views (Phase B2). Always present; single-symbol runs
        # expose only the primary symbol.
        "closes_by_symbol": closes_by_symbol,
        "opens_by_symbol": opens_by_symbol,
        "highs_by_symbol": highs_by_symbol,
        "lows_by_symbol": lows_by_symbol,
        "volumes_by_symbol": volumes_by_symbol,
        "symbols": symbols_list,
        "primary_symbol": primary_sym,
        # Market metadata.
        "symbol": symbol,
        "timeframe": timeframe,
        "current_price": current_price,
        # Account fields (injected — previously missing in legacy bug D8).
        "cash": portfolio.cash,
        "equity": equity,
        "position": _position_to_dict(latest),
        "positions": [_position_to_dict(p) for p in positions],
        "positions_total": len(positions),
        # Aliases expected by iMA/iRSI helpers that read the context.
        "account_balance": portfolio.cash,
        "account_equity": equity,
        # Strategy-specific parameters + mode (so advisory strategies behave
        # differently from backtest).
        "params": dict(params or {}),
        "mode": mode.value,
    }
    return ctx
