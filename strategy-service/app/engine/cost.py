"""Cost model: commission, slippage and rollover swaps.

契约：docs/domains/backtest-system.md §7.4.3 · cost.py

所有公式与旧 ``app/backtest.py`` 完全一致，以保证 I9 金标准下 bit-equal。
"""

from __future__ import annotations

import random
from datetime import datetime, timedelta, timezone
from typing import Optional, Tuple

try:
    from zoneinfo import ZoneInfo
except ImportError:  # pragma: no cover - Python < 3.9 fallback
    ZoneInfo = None  # type: ignore[assignment]

from app.engine.types import CostProfile, SlippageMode


def _ms_to_utc(ms: int) -> datetime:
    return datetime.fromtimestamp(ms / 1000.0, tz=timezone.utc)


def _utc_to_ms(dt: datetime) -> int:
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return int(dt.timestamp() * 1000)


class CostModel:
    """Legacy-compatible commission / slippage / swap model."""

    def __init__(self, profile: CostProfile) -> None:
        self._p = profile
        # Local Random instance so tests don't pollute global random state.
        self._rng = random.Random(profile.slippage_seed)
        # Resolve broker timezone once.
        self._tz: Optional[object]
        if profile.server_timezone and ZoneInfo is not None:
            try:
                self._tz = ZoneInfo(profile.server_timezone)
            except Exception:
                self._tz = None
        else:
            self._tz = None

    # --- commission ------------------------------------------------------

    def commission(self, volume: float) -> float:
        """Per-side commission charged on open (matches legacy behaviour)."""
        return max(0.0, self._p.commission_per_lot) * float(volume)

    # --- slippage --------------------------------------------------------

    def apply_slippage(self, price: float, is_buy_side: bool) -> float:
        """Return fill price after applying configured slippage.

        - FIXED: ``price * (1 ± rate)``
        - RANDOM: ``price * (1 ± rng.random() * rate)`` (seeded → reproducible)
        """
        rate = self._p.slippage_rate
        if rate <= 0:
            return price
        if self._p.slippage_mode is SlippageMode.RANDOM:
            slip = self._rng.random() * rate
        else:
            slip = rate
        if is_buy_side:
            return price * (1.0 + slip)
        return price * (1.0 - slip)

    # --- rollover swaps --------------------------------------------------

    def _to_local(self, dt: datetime) -> datetime:
        if self._tz is None:
            return dt
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt.astimezone(self._tz)

    def _next_rollover_utc(self, dt: datetime) -> datetime:
        """Return the next rollover timestamp (UTC) strictly after ``dt``.

        Rollover occurs daily at ``rollover_hour`` in the broker's local time.
        """
        local_tz = self._tz if self._tz is not None else timezone.utc
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        local = dt.astimezone(local_tz)
        cand = datetime(
            local.year,
            local.month,
            local.day,
            self._p.rollover_hour,
            0,
            0,
            tzinfo=local_tz,
        )
        if cand <= local:
            cand += timedelta(days=1)
        return cand.astimezone(timezone.utc)

    def apply_rollover_swaps(
        self,
        current_equity: float,
        cur_tick_ts: int,
        rollover_cursor: Optional[datetime],
    ) -> Tuple[float, datetime]:
        """Advance rollover cursor up to ``cur_tick_ts`` and apply swaps.

        Returns the adjusted equity and the new cursor.

        Exactly mirrors the legacy loop:

        .. code-block:: python

            if cursor is None:
                cursor = next_rollover_utc(tick_time)
            while cursor <= tick_time:
                mult = 3 if local(cursor).weekday() == triple else 1
                equity *= 1 - rate * mult
                cursor = next_rollover_utc(cursor + 1s)
        """
        rate = self._p.swap_rate_per_rollover
        if rate <= 0:
            # Even when rate is zero, still return a valid cursor so the caller
            # can avoid special-casing on its side.
            if rollover_cursor is None:
                rollover_cursor = self._next_rollover_utc(_ms_to_utc(cur_tick_ts))
            return current_equity, rollover_cursor

        tick_utc = _ms_to_utc(cur_tick_ts)
        cursor = rollover_cursor or self._next_rollover_utc(tick_utc)
        equity = current_equity
        while cursor <= tick_utc:
            local_roll = self._to_local(cursor)
            mult = 3.0 if local_roll.weekday() == self._p.triple_swap_weekday else 1.0
            equity *= 1.0 - rate * mult
            cursor = self._next_rollover_utc(cursor + timedelta(seconds=1))
        return equity, cursor
