"""Sample 1 — Single MA Cross (EMA crossover trend-follower).

MQL pattern: two EMAs, cross-up → buy, cross-down → close+reverse.
Demonstrates: on_init, on_bar, indicators (ema), broker.order_send (market),
  broker.position_close, broker.positions, ctx.param, AccountMode check.
"""

from decimal import Decimal

from app.sdk import (
    AccountMode,
    OrderRequest,
    OrderType,
    PositionSide,
    StrategyBase,
    TypeFilling,
)


class SingleMACross(StrategyBase):
    """Trend-following EA: EMA(fast) crosses above EMA(slow) → buy;
    crosses below → close long + open short.  One position at a time."""

    def on_init(self) -> None:
        # Strategy parameters (equivalent to MQL extern/input).
        self.fast_period: int = int(self.ctx.param("fast_period", 12))
        self.slow_period: int = int(self.ctx.param("slow_period", 26))
        self.lot_size: Decimal = Decimal(str(self.ctx.param("lot_size", "0.10")))
        self.magic: int = 1001

        # State — persists across callbacks (like MQL globals).
        self._prev_fast: float = 0.0
        self._prev_slow: float = 0.0
        self._has_position: bool = False

    def on_bar(self, timeframe: str) -> None:
        # Only trade on primary timeframe.
        if timeframe != self.ctx.bars().timeframe:
            return

        fast = self.indicators.ema(self.fast_period, shift=1)
        slow = self.indicators.ema(self.slow_period, shift=1)

        if fast <= 0.0 or slow <= 0.0:
            return  # not enough bars

        cross_up = self._prev_fast <= self._prev_slow and fast > slow
        cross_down = self._prev_fast >= self._prev_slow and fast < slow

        self._prev_fast = fast
        self._prev_slow = slow

        if cross_up:
            self._close_existing()
            self._open(PositionSide.BUY, "ema_cross_up")
        elif cross_down:
            self._close_existing()
            # Netting mode: opening a SELL closes the BUY.
            # Hedging mode: we explicitly close first, then open.
            self._open(PositionSide.SELL, "ema_cross_down")

    def _close_existing(self) -> None:
        """Close all positions tagged with this EA's magic number."""
        for pos in self.broker.positions(magic=self.magic):
            result = self.broker.position_close(pos.ticket)
            if result.retcode.value in ("done", "done_partial"):
                self._has_position = False

    def _open(self, side: PositionSide, comment: str) -> None:
        """Open a market order."""
        symbol = self.ctx.bars().timeframe and self.ctx.symbol or ""

        account = self.broker.account()
        if account.mode == AccountMode.NETTING:
            # In netting mode, opening the opposite side closes the existing.
            # Check if we already have a position in the same direction.
            existing = self.broker.positions(magic=self.magic)
            if existing and existing[0].side == side:
                return  # already positioned

        ot = OrderType.BUY if side == PositionSide.BUY else OrderType.SELL
        req = OrderRequest(
            symbol=symbol,
            type=ot,
            volume=self.lot_size,
            magic=self.magic,
            comment=comment,
            type_filling=TypeFilling.IOC,
        )
        result = self.broker.order_send(req)
        if result.retcode.value in ("done", "done_partial"):
            self._has_position = True

    def on_deinit(self, reason: str) -> None:
        if reason in ("user_stop", "kill_switch", "schedule_end"):
            self._close_existing()
