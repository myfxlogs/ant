"""Sample 3 — Martingale (double-after-loss, reset-after-win).

MQL pattern: fixed entry signal (e.g. RSI oversold → buy), but position
  size doubles after each loss; resets to base after each win.
Demonstrates: on_trade (detect trade outcome), account query (balance/equity),
  position sizing with Decimal math, magic number for trade tracking,
  broker.position_close, RSI indicator, tracking realized PnL manually.
"""

from decimal import Decimal, ROUND_HALF_UP

from app.sdk import (
    AccountInfo,
    OrderRequest,
    OrderResult,
    OrderType,
    Position,
    PositionSide,
    Retcode,
    StrategyBase,
    TypeFilling,
)


class Martingale(StrategyBase):
    """RSI-based entry with martingale position sizing.
    Doubles lot size after each loss; resets to base after a win.
    Hard cap prevents account-destroying sequences."""

    def on_init(self) -> None:
        self.rsi_period: int = int(self.ctx.param("rsi_period", 14))
        self.rsi_oversold: float = float(self.ctx.param("rsi_oversold", "30.0"))
        self.rsi_overbought: float = float(self.ctx.param("rsi_overbought", "70.0"))
        self.base_lot: Decimal = Decimal(str(self.ctx.param("base_lot", "0.10")))
        self.max_lot: Decimal = Decimal(str(self.ctx.param("max_lot", "5.0")))
        self.magic: int = 3001

        self._current_lot: Decimal = self.base_lot
        self._last_trade_was_loss: bool = False
        self._consecutive_losses: int = 0
        # Track PnL of positions opened by this EA.
        self._trade_entry_equity: Decimal = Decimal("0")
        self._in_trade: bool = False

    def on_bar(self, timeframe: str) -> None:
        if timeframe != self.ctx.bars().timeframe:
            return

        rsi = self.indicators.rsi(self.rsi_period, shift=1)
        if rsi <= 0.0:
            return  # not enough data

        # Check if we already have an open position for this magic.
        positions = self.broker.positions(magic=self.magic)
        if positions:
            return  # one trade at a time

        if rsi < self.rsi_oversold:
            self._open(PositionSide.BUY, f"martingale_rsi_{rsi:.1f}")
        elif rsi > self.rsi_overbought:
            self._open(PositionSide.SELL, f"martingale_rsi_{rsi:.1f}")

    def on_trade(self) -> None:
        """Detect when our position closes and adjust lot size."""
        positions = self.broker.positions(magic=self.magic)

        if positions and not self._in_trade:
            # Position opened — record entry equity.
            self._in_trade = True
            self._trade_entry_equity = self.broker.account().equity

        elif not positions and self._in_trade:
            # Position closed — evaluate outcome.
            self._in_trade = False
            current_equity = self.broker.account().equity
            pnl = current_equity - self._trade_entry_equity

            if pnl > Decimal("0"):
                # Win — reset to base lot.
                self._current_lot = self.base_lot
                self._consecutive_losses = 0
                self._last_trade_was_loss = False
            elif pnl < Decimal("0"):
                # Loss — double the lot (capped).
                self._consecutive_losses += 1
                self._last_trade_was_loss = True
                doubled = (self._current_lot * Decimal("2")).quantize(
                    Decimal("0.01"), rounding=ROUND_HALF_UP
                )
                self._current_lot = min(doubled, self.max_lot)
            # pnl == 0 (breakeven) — no change.

    def _open(self, side: PositionSide, comment: str) -> None:
        """Open with current martingale lot size."""
        ot = OrderType.BUY if side == PositionSide.BUY else OrderType.SELL
        req = OrderRequest(
            symbol=self.ctx.symbol,
            type=ot,
            volume=self._current_lot,
            magic=self.magic,
            comment=comment,
            type_filling=TypeFilling.IOC,
        )
        result = self.broker.order_send(req)
        if result.retcode == Retcode.RISK_BLOCKED:
            # Risk gate blocked — don't change lot progression.
            pass

    def on_deinit(self, reason: str) -> None:
        for pos in self.broker.positions(magic=self.magic):
            self.broker.position_close(pos.ticket)
