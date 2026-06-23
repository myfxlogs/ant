"""Sample 4 — Hedge Twins (simultaneous long + short in hedging mode).

MQL pattern: two independent signal tracks run on the same symbol — one
  trend-following (long), one mean-reversion (short).  In hedging mode
  both can coexist.  Each track uses its own magic number.
Demonstrates: AccountMode.HEDGING, multiple concurrent positions on same symbol,
  magic-number isolation, partial close (position_close with volume),
  position_modify (SL/TP adjustment), broker.positions filtering by magic,
  multi-timeframe via ctx.bars("H1").
"""

from decimal import Decimal

from app.sdk import (
    AccountMode,
    OrderRequest,
    OrderResult,
    OrderType,
    PositionSide,
    StrategyBase,
    TypeFilling,
)


class HedgeTwins(StrategyBase):
    """Runs two independent strategies on the same symbol in hedging mode:
    - Twin A (magic 4001): trend-following on H1 — rides long trends.
    - Twin B (magic 4002): mean-reversion on M15 — shorts overbought spikes.
    Both can hold positions simultaneously because the account is HEDGING."""

    def on_init(self) -> None:
        # Twin A params (trend).
        self.trend_ma_period: int = int(self.ctx.param("trend_ma", 50))
        self.trend_lot: Decimal = Decimal(str(self.ctx.param("trend_lot", "0.20")))
        self.magic_trend: int = 4001

        # Twin B params (mean-reversion).
        self.mr_rsi_period: int = int(self.ctx.param("mr_rsi_period", 7))
        self.mr_lot: Decimal = Decimal(str(self.ctx.param("mr_lot", "0.10")))
        self.magic_mr: int = 4002

        # Verify we're in hedging mode.
        self._hedging: bool = False

    def on_bar(self, timeframe: str) -> None:
        account = self.broker.account()
        if account.mode != AccountMode.HEDGING:
            if not self._hedging:
                # In netting mode, only one twin can run.
                pass
            return
        self._hedging = True

        bars_primary = self.ctx.bars()  # primary timeframe

        # ── Twin A: trend-following on H1 ──────────────────────
        if timeframe == "H1":
            self._run_trend_twin(bars_primary)

        # ── Twin B: mean-reversion on primary (M15) ────────────
        if timeframe == bars_primary.timeframe:
            self._run_mean_reversion_twin(bars_primary)

    def _run_trend_twin(self, bars) -> None:
        """EMA-based trend following. Long when price > EMA, flat when below."""
        ema = self.indicators.ema(self.trend_ma_period, shift=1)
        if ema <= 0.0:
            return

        try:
            current_close = Decimal(str(bars.close[0]))
        except (IndexError, TypeError):
            return

        trend_positions = self.broker.positions(magic=self.magic_trend)
        is_long = any(p.side == PositionSide.BUY for p in trend_positions)

        if current_close > Decimal(str(ema)) and not is_long:
            # Uptrend — go long.
            req = OrderRequest(
                symbol=self.ctx.symbol,
                type=OrderType.BUY,
                volume=self.trend_lot,
                magic=self.magic_trend,
                comment="trend_long",
                type_filling=TypeFilling.IOC,
            )
            result = self.broker.order_send(req)
            if result.retcode.value in ("done", "done_partial") and result.ticket:
                # Set initial SL at 2× ATR below entry.
                atr = self.indicators.atr(14, shift=1)
                if atr > 0.0 and result.price:
                    sl = result.price - Decimal(str(atr * 2))
                    self.broker.position_modify(result.ticket, sl=sl, tp=None)

        elif current_close <= Decimal(str(ema)) and is_long:
            # Downtrend — close trend long.
            for pos in trend_positions:
                self.broker.position_close(pos.ticket)

    def _run_mean_reversion_twin(self, bars) -> None:
        """RSI-based mean reversion. Short when RSI > 70, cover when RSI < 50."""
        rsi = self.indicators.rsi(self.mr_rsi_period, shift=1)
        if rsi <= 0.0:
            return

        mr_positions = self.broker.positions(magic=self.magic_mr)
        is_short = any(p.side == PositionSide.SELL for p in mr_positions)

        if rsi > 70.0 and not is_short:
            # Overbought — go short.
            req = OrderRequest(
                symbol=self.ctx.symbol,
                type=OrderType.SELL,
                volume=self.mr_lot,
                magic=self.magic_mr,
                comment="mr_short",
                type_filling=TypeFilling.IOC,
            )
            result = self.broker.order_send(req)
            if result.retcode.value in ("done", "done_partial") and result.ticket:
                atr = self.indicators.atr(14, shift=1)
                if atr > 0.0 and result.price:
                    sl = result.price + Decimal(str(atr * 2))
                    self.broker.position_modify(result.ticket, sl=sl, tp=None)

        elif rsi < 50.0 and is_short:
            # Oversold — cover short (partial close to scale out).
            for pos in mr_positions:
                half_volume = pos.volume / Decimal("2")
                if half_volume >= self.broker.symbol_info(self.ctx.symbol).volume_min:
                    # Partial close: scale out half.
                    self.broker.position_close(pos.ticket, volume=half_volume)
                else:
                    # Volume too small for partial — close fully.
                    self.broker.position_close(pos.ticket)

    def on_deinit(self, reason: str) -> None:
        for magic in [self.magic_trend, self.magic_mr]:
            for pos in self.broker.positions(magic=magic):
                self.broker.position_close(pos.ticket)
