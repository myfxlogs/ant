"""Market simulators: Tick time-driver + Bar view.

契约：docs/domains/backtest-system.md §7.4.3 · market.py
"""

from __future__ import annotations

from typing import Dict, Iterator, List, Optional

import numpy as np

from app.engine.types import Bar, Tick


class TickSimulator:
    """Primary time driver.

    If ``ticks`` is provided and non-empty, it is replayed verbatim.
    Otherwise a synthetic tick is emitted at every bar's ``close_time`` with
    ``bid == ask == close`` (this matches the legacy engine fallback in
    ``app/backtest.py`` when no ticks are supplied).
    """

    def __init__(self, bars: List[Bar], ticks: Optional[List[Tick]] = None) -> None:
        self._bars = bars
        self._ticks: List[Tick]
        if ticks:
            # Defensive copy so callers can mutate their input without side-effects.
            self._ticks = list(ticks)
        else:
            self._ticks = [
                Tick(ts=b.close_time, bid=b.close, ask=b.close) for b in bars
            ]

    def __iter__(self) -> Iterator[Tick]:
        return iter(self._ticks)

    def __len__(self) -> int:
        return len(self._ticks)

    @property
    def synthetic(self) -> bool:
        """True if the ticks were synthesized from bar closes."""
        return len(self._ticks) == len(self._bars) and all(
            t.bid == t.ask for t in self._ticks
        )


class MarketSimulator:
    """Read-only bar view.

    Supports two cheap operations the runner loop needs on every tick:

    1. :py:meth:`bar_closed_at_or_before` — binary search the most recent bar
       whose ``close_time <= ts`` (returns ``-1`` if no such bar).
    2. :py:meth:`slice_until` — return numpy OHLCV arrays covering ``[0..idx]``
       for injection into the sandbox ``context``.
    """

    def __init__(self, bars: List[Bar]) -> None:
        self._bars = list(bars)
        # Pre-materialize columns as numpy arrays for O(1) slicing.
        n = len(self._bars)
        self._open = np.fromiter((b.open for b in self._bars), dtype=float, count=n)
        self._high = np.fromiter((b.high for b in self._bars), dtype=float, count=n)
        self._low = np.fromiter((b.low for b in self._bars), dtype=float, count=n)
        self._close = np.fromiter((b.close for b in self._bars), dtype=float, count=n)
        self._volume = np.fromiter(
            (b.volume for b in self._bars), dtype=float, count=n
        )
        self._close_times = np.fromiter(
            (b.close_time for b in self._bars), dtype=np.int64, count=n
        )

    def __len__(self) -> int:
        return len(self._bars)

    @property
    def bars(self) -> List[Bar]:
        return self._bars

    def bar_closed_at_or_before(self, ts: int) -> int:
        """Return the greatest index ``i`` such that ``bars[i].close_time <= ts``.

        Returns ``-1`` if no bar has yet closed at ``ts``.
        """
        if not self._bars:
            return -1
        # ``right`` gives the insertion point such that all values on the left
        # are ``<= ts``. Subtract 1 to land on the last satisfying index.
        idx = int(np.searchsorted(self._close_times, ts, side="right")) - 1
        return idx if idx >= 0 else -1

    def slice_until(self, idx: int) -> dict:
        """Return OHLCV numpy slices for ``[0..idx]`` (inclusive).

        Raises ``IndexError`` if ``idx`` is out of range.
        """
        if idx < 0 or idx >= len(self._bars):
            raise IndexError(f"bar index {idx} out of range [0, {len(self._bars)})")
        end = idx + 1
        return {
            "open": self._open[:end],
            "high": self._high[:end],
            "low": self._low[:end],
            "close": self._close[:end],
            "volume": self._volume[:end],
            "close_time": self._close_times[:end],
        }


class MultiSymbolMarket:
    """Read-only multi-symbol bar view.

    Phase B2 / docs/domains/backtest-system.md §7.4.3. 每个 symbol 独立一个
    :class:`MarketSimulator`；主 symbol 决定时间轴（tick / bar 驱动）。

    其它 symbol 允许 K 线时间轴与主 symbol 不完全对齐：在给定主 symbol 的
    ``bar_idx`` 时，对每个从 symbol 用 ``close_time <= primary_close_time``
    的二分查找取对应索引，避免 look-ahead。
    """

    def __init__(self, bars_by_symbol: Dict[str, List], primary: str) -> None:
        if not bars_by_symbol:
            raise ValueError("bars_by_symbol must be non-empty")
        if primary not in bars_by_symbol:
            raise ValueError(
                f"primary symbol {primary!r} not in bars_by_symbol keys {list(bars_by_symbol)}"
            )
        self._primary = primary
        self._markets: Dict[str, MarketSimulator] = {
            sym: MarketSimulator(bars) for sym, bars in bars_by_symbol.items()
        }

    @property
    def primary(self) -> str:
        return self._primary

    def primary_market(self) -> MarketSimulator:
        return self._markets[self._primary]

    def symbols(self) -> List[str]:
        return list(self._markets.keys())

    def market(self, symbol: str) -> MarketSimulator:
        m = self._markets.get(symbol)
        if m is None:
            raise KeyError(f"unknown symbol: {symbol}")
        return m

    def __len__(self) -> int:
        return len(self.primary_market())

    def bar_closed_at_or_before(self, ts: int) -> int:
        """Drive time axis off the primary market."""
        return self.primary_market().bar_closed_at_or_before(ts)

    def slice_until(self, primary_idx: int) -> Dict[str, dict]:
        """Return {symbol: {open/high/low/close/volume: ndarray}}.

        Secondary symbols are aligned by ``close_time`` of the primary bar to
        prevent look-ahead bias.
        """
        if primary_idx < 0 or primary_idx >= len(self.primary_market()):
            raise IndexError(
                f"primary bar index {primary_idx} out of range "
                f"[0, {len(self.primary_market())})"
            )
        primary_close_ts = int(self.primary_market().bars[primary_idx].close_time)
        out: Dict[str, dict] = {}
        for sym, m in self._markets.items():
            if sym == self._primary:
                out[sym] = m.slice_until(primary_idx)
                continue
            sec_idx = m.bar_closed_at_or_before(primary_close_ts)
            if sec_idx < 0:
                # No data available yet for this secondary symbol; return empty
                # zero-length arrays so the sandbox strategy can detect it.
                out[sym] = {
                    "open": np.empty(0, dtype=float),
                    "high": np.empty(0, dtype=float),
                    "low": np.empty(0, dtype=float),
                    "close": np.empty(0, dtype=float),
                    "volume": np.empty(0, dtype=float),
                }
            else:
                out[sym] = m.slice_until(sec_idx)
        return out
