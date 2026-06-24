"""LiveBroker — Broker interface backed by live MT gateway (T3.1).

Implements ``app.sdk.broker.Broker`` by proxying order intents to the
Go-side MT gateway via the strategy signal return path.

Architecture:
  Strategy → self.broker.order_send() → LiveBroker records intent
  Strategy returns → signal dict → Go extracts intent → MT gateway RPC

Key: LiveBroker runs inside the Python sandbox subprocess.  It cannot
make outbound gRPC calls itself.  Instead, it records order intents and
lets the Go side execute them via the existing signal dispatch path.

The Go side feeds account/position state into the LiveStrategyContext;
LiveBroker reads from that cached state for queries.
"""

from __future__ import annotations

from decimal import Decimal
from typing import Callable, Dict, List, Optional

from app.sdk.account import AccountInfo, AccountMode
from app.sdk.broker import Broker
from app.sdk.symbol import SymbolInfo
from app.sdk.types import (
    OrderRequest,
    OrderResult,
    OrderType,
    PendingOrder,
    Position,
    PositionSide,
    Retcode,
    TypeFilling,
)


class LiveBroker(Broker):
    """Live/paper trading broker that records intents for Go-side execution.

    State is fed in from Go's LiveStrategyContext at the start of each
    bar/tick callback.  Order intents are recorded and returned to Go
    via the signal dict after the strategy callback completes.
    """

    def __init__(
        self,
        account_state: Optional[AccountInfo] = None,
        position_list: Optional[List[Position]] = None,
        pending_orders: Optional[List[PendingOrder]] = None,
        symbol_infos: Optional[Dict[str, SymbolInfo]] = None,
        server_time_fn: Optional[Callable[[], int]] = None,
    ) -> None:
        self._account = account_state or AccountInfo(
            balance=Decimal("0"), equity=Decimal("0"),
            margin=Decimal("0"), free_margin=Decimal("0"),
            margin_level=Decimal("0"), leverage=100,
            currency="USD", mode=AccountMode.HEDGING,
        )
        self._positions: List[Position] = list(position_list or [])
        self._pending: List[PendingOrder] = list(pending_orders or [])
        self._symbols: Dict[str, SymbolInfo] = dict(symbol_infos or {})
        self._server_time_fn = server_time_fn or (lambda: 0)

        # Intent log — returned to Go after each callback.
        self._intents: List[OrderIntent] = []
        self._close_intents: List[CloseIntent] = []
        self._modify_intents: List[ModifyIntent] = []
        self._delete_intents: List[int] = []  # ticket numbers

        self._next_virtual_ticket = 1000000  # high range to avoid collision

    # ── State injection (called by runtime before each callback) ────────

    def update_state(
        self,
        account: Optional[AccountInfo] = None,
        positions: Optional[List[Position]] = None,
        pending: Optional[List[PendingOrder]] = None,
        server_time_ms: int = 0,
    ) -> None:
        """Update cached state from Go's LiveStrategyContext."""
        if account is not None:
            self._account = account
        if positions is not None:
            self._positions = list(positions)
        if pending is not None:
            self._pending = list(pending)
        if server_time_ms > 0:
            self._server_time_fn = lambda: server_time_ms

    def clear_intents(self) -> None:
        """Clear the intent log (called after Go consumes intents)."""
        self._intents.clear()
        self._close_intents.clear()
        self._modify_intents.clear()
        self._delete_intents.clear()

    # ── Broker interface ────────────────────────────────────────────────

    def order_send(self, request: OrderRequest) -> OrderResult:
        """Record an order intent for Go-side execution."""
        ticket = self._next_virtual_ticket
        self._next_virtual_ticket += 1

        # Record the intent.
        self._intents.append(OrderIntent(
            ticket=ticket,
            request=request,
        ))

        return OrderResult(
            retcode=Retcode.DONE,
            ticket=ticket,
            price=request.price,
            volume=request.volume,
            comment="live intent recorded",
        )

    def position_modify(
        self, ticket: int, sl: Optional[Decimal] = None, tp: Optional[Decimal] = None
    ) -> OrderResult:
        """Record a modify intent."""
        self._modify_intents.append(ModifyIntent(ticket=ticket, sl=sl, tp=tp))

        # Update cached position optimistically.
        for pos in self._positions:
            if pos.ticket == ticket:
                if sl is not None:
                    pos.sl = sl
                if tp is not None:
                    pos.tp = tp
                return OrderResult(retcode=Retcode.DONE, ticket=ticket)

        return OrderResult(
            retcode=Retcode.REJECTED, comment=f"position {ticket} not found"
        )

    def position_close(
        self, ticket: int, volume: Optional[Decimal] = None
    ) -> OrderResult:
        """Record a close intent."""
        self._close_intents.append(CloseIntent(ticket=ticket, volume=volume))

        # Optimistically remove from cached positions.
        if volume is None:
            self._positions = [p for p in self._positions if p.ticket != ticket]
        else:
            for pos in self._positions:
                if pos.ticket == ticket:
                    pos.volume -= volume
                    if pos.volume <= Decimal("0"):
                        self._positions.remove(pos)
                    break

        return OrderResult(
            retcode=Retcode.DONE if volume is None else Retcode.DONE_PARTIAL,
            ticket=ticket,
            comment="close intent recorded",
        )

    def order_delete(self, ticket: int) -> OrderResult:
        """Record a cancel intent."""
        self._delete_intents.append(ticket)
        self._pending = [o for o in self._pending if o.ticket != ticket]
        return OrderResult(retcode=Retcode.DONE, ticket=ticket)

    def positions(
        self, symbol: Optional[str] = None, magic: Optional[int] = None
    ) -> List[Position]:
        """Return cached positions, optionally filtered."""
        result = self._positions
        if symbol is not None:
            result = [p for p in result if p.symbol == symbol]
        if magic is not None:
            result = [p for p in result if p.magic == magic]
        return list(result)

    def orders(
        self, symbol: Optional[str] = None, magic: Optional[int] = None
    ) -> List[PendingOrder]:
        """Return cached pending orders, optionally filtered."""
        result = self._pending
        if symbol is not None:
            result = [o for o in result if o.symbol == symbol]
        if magic is not None:
            result = [o for o in result if o.magic == magic]
        return list(result)

    def account(self) -> AccountInfo:
        """Return cached account state."""
        return self._account

    def symbol_info(self, symbol: str) -> SymbolInfo:
        """Return symbol metadata; fallback if unknown."""
        if symbol in self._symbols:
            return self._symbols[symbol]
        return SymbolInfo(
            name=symbol, digits=5, point=Decimal("0.00001"),
            tick_size=Decimal("0.00001"), tick_value=Decimal("1.0"),
            contract_size=Decimal("100000"),
            volume_min=Decimal("0.01"), volume_max=Decimal("100"),
            volume_step=Decimal("0.01"),
            stops_level=0, freeze_level=0,
            swap_long=Decimal("0"), swap_short=Decimal("0"),
            margin_rate=Decimal("0.01"),
        )

    def server_time(self) -> int:
        """Return cached server time."""
        return self._server_time_fn()

    # ── Intent accessors (for Go-side consumption) ──────────────────────

    @property
    def order_intents(self) -> List[OrderIntent]:
        return list(self._intents)

    @property
    def close_intents(self) -> List[CloseIntent]:
        return list(self._close_intents)

    @property
    def modify_intents(self) -> List[ModifyIntent]:
        return list(self._modify_intents)

    @property
    def delete_intents(self) -> List[int]:
        return list(self._delete_intents)

    def export_intents(self) -> List[dict]:
        """Export all intents as serializable dicts, then clear.

        Called by StrategyRuntime.export_intents() after each bar/tick.
        Returns a flat list of intent dicts ready for Go gate evaluation.
        """
        result: List[dict] = []
        for intent in self._intents:
            result.append(intent.to_signal_dict())
        for intent in self._close_intents:
            result.append(intent.to_signal_dict())
        for intent in self._modify_intents:
            result.append(intent.to_signal_dict())
        for ticket in self._delete_intents:
            result.append({"action": "cancel", "ticket": str(ticket)})
        self.clear_intents()
        return result


# ── Intent data classes ─────────────────────────────────────────────────


class OrderIntent:
    """Records a strategy's order_send() call for Go-side execution."""
    __slots__ = ("ticket", "request")

    def __init__(self, ticket: int, request: OrderRequest):
        self.ticket = ticket
        self.request = request

    def to_signal_dict(self) -> dict:
        """Convert to a signal-like dict for Go consumption."""
        req = self.request
        return {
            "action": req.type.value,
            "symbol": req.symbol,
            "volume": str(req.volume),
            "price": str(req.price) if req.price else "0",
            "sl": str(req.sl) if req.sl else "0",
            "tp": str(req.tp) if req.tp else "0",
            "magic": str(req.magic),
            "comment": req.comment,
            "deviation": str(req.deviation or 0),
            "type_filling": req.type_filling.value,
            "virtual_ticket": str(self.ticket),
        }


class CloseIntent:
    """Records a strategy's position_close() call."""
    __slots__ = ("ticket", "volume")

    def __init__(self, ticket: int, volume: Optional[Decimal] = None):
        self.ticket = ticket
        self.volume = volume

    def to_signal_dict(self) -> dict:
        return {
            "action": "close",
            "ticket": str(self.ticket),
            "volume": str(self.volume) if self.volume is not None else "full",
        }


class ModifyIntent:
    """Records a strategy's position_modify() call."""
    __slots__ = ("ticket", "sl", "tp")

    def __init__(self, ticket: int, sl: Optional[Decimal] = None, tp: Optional[Decimal] = None):
        self.ticket = ticket
        self.sl = sl
        self.tp = tp

    def to_signal_dict(self) -> dict:
        return {
            "action": "modify",
            "ticket": str(self.ticket),
            "sl": str(self.sl) if self.sl else "0",
            "tp": str(self.tp) if self.tp else "0",
        }


def build_live_broker_from_proto(lctx) -> LiveBroker:
    """Build a LiveBroker from a proto LiveStrategyContext.

    Called by the Python sandbox runtime to inject live state into
    the strategy's broker.
    """
    from app.sdk.account import AccountInfo, AccountMode
    from app.sdk.symbol import SymbolInfo
    from app.sdk.types import Position, PositionSide, PendingOrder, OrderType

    # Account state.
    account = AccountInfo(
        balance=Decimal(str(lctx.balance)) if hasattr(lctx, 'balance') else Decimal("0"),
        equity=Decimal(str(lctx.equity)) if hasattr(lctx, 'equity') else Decimal("0"),
        margin=Decimal("0"),
        free_margin=Decimal("0"),
        margin_level=Decimal("0"),
        leverage=100,
        currency="USD",
        mode=AccountMode.HEDGING,
    )

    # Positions.
    positions: List[Position] = []
    if hasattr(lctx, 'positions') and lctx.positions:
        for lp in lctx.positions:
            positions.append(Position(
                ticket=lp.ticket,
                symbol=getattr(lp, 'symbol', ''),
                side=PositionSide.BUY if getattr(lp, 'side', 'buy') == 'buy' else PositionSide.SELL,
                volume=Decimal(str(getattr(lp, 'volume', 0))),
                open_price=Decimal(str(getattr(lp, 'open_price', 0))),
                sl=Decimal(str(lp.sl)) if getattr(lp, 'sl', 0) else None,
                tp=Decimal(str(lp.tp)) if getattr(lp, 'tp', 0) else None,
                profit=Decimal("0"),
                swap=Decimal("0"),
                magic=int(getattr(lp, 'magic', 0)),
                comment=getattr(lp, 'comment', ''),
                open_time_ms=int(getattr(lp, 'open_time_ms', 0)),
            ))

    return LiveBroker(account_state=account, position_list=positions)
