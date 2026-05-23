"""Optional margin / leverage model.

契约：docs/domains/backtest-system.md §7.4.3 · margin.py

Leverage ≤ 0 → :py:meth:`enabled` returns False and all checks are no-ops.
This matches the legacy engine's behaviour (no margin call at all).
"""

from __future__ import annotations

import math
from typing import TYPE_CHECKING

from app.engine.types import Position, Side, Tick

if TYPE_CHECKING:  # pragma: no cover
    from app.engine.portfolio import Portfolio


class MarginModel:
    """Simple symmetrical leverage model.

    ``required_margin(pos, price) = pos.volume * contract_size * price / leverage``

    Margin level = equity / used_margin * 100 (percent).
    Margin call fires when level < 100% (i.e. equity <= used_margin).
    """

    def __init__(self, leverage: float, contract_size: float = 100_000.0) -> None:
        self._leverage = float(leverage)
        self._contract_size = float(contract_size)

    def enabled(self) -> bool:
        return self._leverage > 0.0

    # --- calculations ----------------------------------------------------

    def required_margin(self, position: Position, price: float) -> float:
        if not self.enabled() or price <= 0:
            return 0.0
        return position.volume * self._contract_size * price / self._leverage

    def used_margin(self, portfolio: "Portfolio", tick: Tick) -> float:
        total = 0.0
        for pos in portfolio.positions:
            # Mark side: the price we'd owe at if closed right now.
            price = tick.ask if pos.side is Side.BUY else tick.bid
            total += self.required_margin(pos, price)
        return total

    def margin_level(self, portfolio: "Portfolio", tick: Tick) -> float:
        """Return margin level in percent. ``inf`` if no used margin (incl. disabled)."""
        if not self.enabled():
            return math.inf
        used = self.used_margin(portfolio, tick)
        if used <= 0:
            return math.inf
        return portfolio.equity(tick) / used * 100.0

    def is_margin_call(self, portfolio: "Portfolio", tick: Tick) -> bool:
        if not self.enabled():
            return False
        return self.margin_level(portfolio, tick) < 100.0
