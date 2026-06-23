"""Sample 2 — Grid Trader (pending-order grid with magic-number tracking).

MQL pattern: on_init places a grid of buy_limit / sell_limit orders;
  on_trade replaces filled orders; on_timer rechecks grid integrity.
Demonstrates: on_init, on_tick, on_trade, on_timer, broker.order_send (pending),
  broker.order_delete, broker.orders, broker.positions, magic number,
  symbol_info (volume_step, stops_level, normalize_price),
  pending order lifecycle (BUY_LIMIT, SELL_LIMIT, BUY_STOP, SELL_STOP).
"""

from decimal import Decimal

from app.sdk import (
    AccountMode,
    OrderRequest,
    OrderResult,
    OrderType,
    PendingOrder,
    PositionSide,
    Retcode,
    StrategyBase,
    TypeFilling,
)


class GridTrader(StrategyBase):
    """Places a grid of pending orders above/below current price.
    Each filled order triggers a take-profit order at the next grid level.
    Orders are identified by magic number so multiple grids can coexist."""

    def on_init(self) -> None:
        self.grid_spacing_pips: int = int(self.ctx.param("grid_spacing_pips", 20))
        self.grid_levels: int = int(self.ctx.param("grid_levels", 5))
        self.lot_size: Decimal = Decimal(str(self.ctx.param("lot_size", "0.10")))
        self.tp_pips: int = int(self.ctx.param("tp_pips", 20))
        self.magic_base: int = 2001

        self._grid_placed: bool = False

    def on_tick(self) -> None:
        if self._grid_placed:
            return

        sym = self.broker.symbol_info(self.ctx.symbol)
        if sym is None:
            return

        current_bid = self._current_price()
        if current_bid is None:
            return

        # Calculate grid levels.
        pip = sym.point * (10 ** (sym.digits - 1)) if sym.digits > 1 else sym.point * Decimal("10")
        # If point is e.g. 0.00001 (5-digit), 1 pip = 0.00010
        # If point is 0.001 (3-digit JPY), 1 pip = 0.01
        # Simple heuristic: pip is 10×point for forex pairs.
        # For indices/commodities, point == pip.
        point_val = sym.point
        pip_size = point_val * Decimal("10") if sym.digits > 1 else point_val

        grid_step = pip_size * self.grid_spacing_pips
        tp_distance = pip_size * self.tp_pips

        for i in range(1, self.grid_levels + 1):
            buy_price = sym.normalize_price(current_bid - grid_step * i)
            sell_price = sym.normalize_price(current_bid + grid_step * i)

            # Buy limit below market.
            buy_req = OrderRequest(
                symbol=self.ctx.symbol,
                type=OrderType.BUY_LIMIT,
                volume=self.lot_size,
                price=buy_price,
                tp=buy_price + tp_distance,
                magic=self.magic_base + i,
                comment=f"grid_buy_{i}",
                type_filling=TypeFilling.RETURN,
            )
            self.broker.order_send(buy_req)

            # Sell limit above market.
            sell_req = OrderRequest(
                symbol=self.ctx.symbol,
                type=OrderType.SELL_LIMIT,
                volume=self.lot_size,
                price=sell_price,
                tp=sell_price - tp_distance,
                magic=self.magic_base + 100 + i,
                comment=f"grid_sell_{i}",
                type_filling=TypeFilling.RETURN,
            )
            self.broker.order_send(sell_req)

        self._grid_placed = True

    def on_trade(self) -> None:
        """After any trade event, check if filled grid orders need companion TP orders.
        In a real grid EA, this is where you'd place the opposite limit to capture profit.
        """
        open_positions = self.broker.positions(magic=None)  # all magics
        pending = self.broker.orders()

        # For each filled grid position, ensure its opposite grid level exists.
        for pos in open_positions:
            if not (self.magic_base <= pos.magic <= self.magic_base + 200):
                continue  # not our grid

            # Check if the companion order is still pending.
            companion_magic = self._companion_magic(pos.magic)
            has_companion = any(o.magic == companion_magic for o in pending)

            if not has_companion:
                # Replenish the grid level that was consumed.
                self._replenish_grid_level(pos)

    def on_timer(self) -> None:
        """Periodically verify grid integrity — cancel orders too far from market."""
        sym = self.broker.symbol_info(self.ctx.symbol)
        current = self._current_price()
        if current is None:
            return

        pip_size = sym.point * Decimal("10") if sym.digits > 1 else sym.point
        max_distance = pip_size * self.grid_spacing_pips * (self.grid_levels + 2)

        for order in self.broker.orders(magic=None):
            if not (self.magic_base <= order.magic <= self.magic_base + 200):
                continue
            # Cancel orders that drifted too far from market.
            distance = abs(order.price - current)
            if distance > max_distance:
                self.broker.order_delete(order.ticket)

    def on_deinit(self, reason: str) -> None:
        if reason in ("user_stop", "kill_switch"):
            # Cancel all pending grid orders.
            for order in self.broker.orders():
                if self.magic_base <= order.magic <= self.magic_base + 200:
                    self.broker.order_delete(order.ticket)
            # Close all grid positions.
            for pos in self.broker.positions():
                if self.magic_base <= pos.magic <= self.magic_base + 200:
                    self.broker.position_close(pos.ticket)

    def _current_price(self) -> Decimal | None:
        """Best-effort current price from series or symbol_info."""
        bars = self.ctx.bars()
        try:
            return Decimal(str(bars.close[0]))
        except (IndexError, TypeError):
            return None

    def _companion_magic(self, magic: int) -> int:
        """Buy grid level N has magic M; its companion sell is M+100 or M-100."""
        if 1 <= (magic - self.magic_base) <= 100:
            return magic + 100
        return magic - 100

    def _replenish_grid_level(self, filled_pos) -> None:
        """Place a new pending order to replace the filled grid level."""
        sym = self.broker.symbol_info(self.ctx.symbol)
        pip_size = sym.point * Decimal("10") if sym.digits > 1 else sym.point
        grid_step = pip_size * self.grid_spacing_pips
        tp_distance = pip_size * self.tp_pips

        current = self._current_price()
        if current is None:
            return

        is_buy = filled_pos.side == PositionSide.BUY
        level = filled_pos.magic - self.magic_base
        if level > 100:
            level -= 100

        if is_buy:
            price = sym.normalize_price(current - grid_step * level)
            ot = OrderType.BUY_LIMIT
            tp = price + tp_distance
        else:
            price = sym.normalize_price(current + grid_step * level)
            ot = OrderType.SELL_LIMIT
            tp = price - tp_distance

        req = OrderRequest(
            symbol=self.ctx.symbol,
            type=ot,
            volume=self.lot_size,
            price=price,
            tp=tp,
            magic=filled_pos.magic,  # reuse same magic
            comment=f"grid_replenish_{level}",
            type_filling=TypeFilling.RETURN,
        )
        self.broker.order_send(req)
