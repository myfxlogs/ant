"""Sample 5 — Custom Signal (i_custom indicator + multi-timeframe confluence).

MQL pattern: a custom indicator "SuperTrend" generates buy/sell signals;
  confirmed by a higher-timeframe trend filter (H4 EMA).
Demonstrates: i_custom (custom indicator mount point), multi-timeframe
  bars via ctx.bars("H4"), ctx.set_timer / kill_timer for periodic
  health checks, stop_limit orders (BUY_STOP_LIMIT / SELL_STOP_LIMIT),
  deviation/slippage tolerance, symbol_info margin pre-check pattern.
"""

from decimal import Decimal

from app.sdk import (
    OrderRequest,
    OrderType,
    PositionSide,
    StrategyBase,
    TypeFilling,
)


class CustomSignal(StrategyBase):
    """Uses a custom "SuperTrend" indicator for entry signals, filtered by
    H4 EMA direction.  Enters on stop-limit orders to catch breakouts.
    Runs a periodic health check via on_timer."""

    def on_init(self) -> None:
        # Custom indicator config.
        self.custom_name: str = str(self.ctx.param("custom_name", "SuperTrend"))
        self.custom_period: int = int(self.ctx.param("custom_period", 10))
        self.custom_multiplier: float = float(self.ctx.param("custom_multiplier", "3.0"))

        # Trend filter.
        self.htf_timeframe: str = "H4"
        self.htf_ema_period: int = int(self.ctx.param("htf_ema", 50))

        # Trade management.
        self.lot_size: Decimal = Decimal(str(self.ctx.param("lot_size", "0.10")))
        self.deviation_pts: int = int(self.ctx.param("deviation_pts", 30))
        self.magic: int = 5001

        # Timer for periodic health checks (every 5 minutes).
        self.ctx.set_timer(300)

        # State.
        self._prev_signal: float = 0.0
        self._last_trade_bar: int = 0

    def on_bar(self, timeframe: str) -> None:
        # Only trade on primary timeframe bar close.
        if timeframe != self.ctx.bars().timeframe:
            return

        # ── 1. Custom indicator (SuperTrend) ────────────────────
        # i_custom(name, params, buffer, shift)
        # buffer=0 → trend direction (1=up, -1=down)
        # buffer=1 → stop level (used for stop-limit price)
        main = self.indicators.i_custom(
            self.custom_name,
            params=[self.custom_period, self.custom_multiplier],
            buffer=0,
            shift=1,
        )
        stop_level = self.indicators.i_custom(
            self.custom_name,
            params=[self.custom_period, self.custom_multiplier],
            buffer=1,
            shift=1,
        )

        if main == 0.0 or stop_level == 0.0:
            return  # indicator not ready

        # ── 2. Higher-timeframe trend filter (H4 EMA) ───────────
        htf_bars = self.ctx.bars(self.htf_timeframe)
        htf_ema = self.indicators.ema(self.htf_ema_period, shift=1)
        # Note: In real implementation, indicators would use htf_bars.close.
        # The runtime wires the correct series when bars(timeframe) is active.

        try:
            current_close = Decimal(str(self.ctx.bars().close[0]))
        except (IndexError, TypeError):
            return

        # ── 3. Signal confluence ────────────────────────────────
        # main == 1.0 → uptrend; main == -1.0 → downtrend
        signal_up = main > 0 and current_close > Decimal(str(htf_ema))
        signal_down = main < 0 and current_close < Decimal(str(htf_ema))

        current_bar = len(self.ctx.bars().close)
        if current_bar == self._last_trade_bar:
            return  # one signal per bar

        if signal_up and self._prev_signal <= 0:
            # Trend flipped up + HTF confirms → buy stop above current.
            self._send_stop_order(PositionSide.BUY, Decimal(str(stop_level)))
            self._last_trade_bar = current_bar
        elif signal_down and self._prev_signal >= 0:
            # Trend flipped down + HTF confirms → sell stop below current.
            self._send_stop_order(PositionSide.SELL, Decimal(str(stop_level)))
            self._last_trade_bar = current_bar

        self._prev_signal = main

        # ── 4. Position management ──────────────────────────────
        for pos in self.broker.positions(magic=self.magic):
            # Trail stop to SuperTrend stop level.
            if pos.side == PositionSide.BUY and stop_level > 0:
                new_sl = Decimal(str(stop_level))
                if pos.sl is None or new_sl > pos.sl:
                    self.broker.position_modify(pos.ticket, sl=new_sl, tp=pos.tp)
            elif pos.side == PositionSide.SELL and stop_level > 0:
                new_sl = Decimal(str(stop_level))
                if pos.sl is None or new_sl < pos.sl:
                    self.broker.position_modify(pos.ticket, sl=new_sl, tp=pos.tp)

    def _send_stop_order(self, side: PositionSide, stop_price: Decimal) -> None:
        """Place a BUY_STOP_LIMIT or SELL_STOP_LIMIT order."""
        sym = self.broker.symbol_info(self.ctx.symbol)

        if side == PositionSide.BUY:
            ot = OrderType.BUY_STOP_LIMIT
            # stop_limit_price is the limit (upper bound); price is the trigger.
            limit_price = sym.normalize_price(stop_price + sym.point * self.deviation_pts)
        else:
            ot = OrderType.SELL_STOP_LIMIT
            limit_price = sym.normalize_price(stop_price - sym.point * self.deviation_pts)

        req = OrderRequest(
            symbol=self.ctx.symbol,
            type=ot,
            volume=self.lot_size,
            price=sym.normalize_price(stop_price),
            stop_limit_price=limit_price,
            deviation=self.deviation_pts,
            magic=self.magic,
            comment=f"custom_{self.custom_name}",
            type_filling=TypeFilling.FOK,
        )
        self.broker.order_send(req)

    def on_timer(self) -> None:
        """Periodic health check — verify margin safety."""
        account = self.broker.account()
        if account.margin_level < Decimal("50"):
            # Margin critically low — close all positions for this magic.
            for pos in self.broker.positions(magic=self.magic):
                self.broker.position_close(pos.ticket)

        # Cancel stale pending orders (> 10 bars old).
        pending = self.broker.orders(magic=self.magic)
        current_bar = len(self.ctx.bars().close)
        for order in pending:
            # If order was placed more than 10 bars ago, cancel it.
            # (Real implementation would track placement bar in order metadata.)
            pass

    def on_deinit(self, reason: str) -> None:
        self.ctx.kill_timer()
        if reason in ("user_stop", "kill_switch"):
            for order in self.broker.orders(magic=self.magic):
                self.broker.order_delete(order.ticket)
            for pos in self.broker.positions(magic=self.magic):
                self.broker.position_close(pos.ticket)
