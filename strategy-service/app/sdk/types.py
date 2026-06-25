"""SDK value types: enums, order request/result, position (interface stubs).

契约：docs/adr/0020 · 任务 T0.2
金额/价格字段一律 Decimal，禁止 float。
枚举语义对齐 MT4/MT5 交易服务器。
"""

from __future__ import annotations

from dataclasses import dataclass, field
from decimal import Decimal
from enum import Enum
from typing import Optional


class PositionSide(Enum):
    """持仓方向（与 MT 多空一致）。"""

    BUY = "buy"
    SELL = "sell"


class OrderType(Enum):
    """订单类型（市价 + 全挂单类型，对齐 MT ORDER_TYPE_*）。"""

    BUY = "buy"
    SELL = "sell"
    BUY_LIMIT = "buy_limit"
    SELL_LIMIT = "sell_limit"
    BUY_STOP = "buy_stop"
    SELL_STOP = "sell_stop"
    BUY_STOP_LIMIT = "buy_stop_limit"
    SELL_STOP_LIMIT = "sell_stop_limit"


class TypeFilling(Enum):
    """成交模式（对齐 MT ORDER_FILLING_*）。"""

    FOK = "fok"  # Fill or Kill
    IOC = "ioc"  # Immediate or Cancel
    RETURN = "return"


class AccountMode(Enum):
    """账户持仓模式。"""

    NETTING = "netting"
    HEDGING = "hedging"


class DealType(Enum):
    """成交类型（用于 on_trade 回调与成交回执）。"""

    ENTRY = "entry"
    EXIT = "exit"
    MODIFY = "modify"
    PARTIAL_CLOSE = "partial_close"


class Retcode(Enum):
    """下单返回码（对齐 MT TRADE_RETCODE_* 的核心子集）。"""

    DONE = "done"
    DONE_PARTIAL = "done_partial"
    REQUOTE = "requote"
    REJECTED = "rejected"
    NO_MONEY = "no_money"
    INVALID_VOLUME = "invalid_volume"
    INVALID_STOPS = "invalid_stops"
    MARKET_CLOSED = "market_closed"
    RISK_BLOCKED = "risk_blocked"  # 被 Go 风控门拦截 (D6)


@dataclass(frozen=True)
class OrderRequest:
    """下单意图。所有价格/手数用 Decimal。

    type 决定市价/挂单；price 对市价单可为空（由 broker 取当前价）。
    """

    symbol: str
    type: OrderType
    volume: Decimal
    price: Optional[Decimal] = None
    sl: Optional[Decimal] = None
    tp: Optional[Decimal] = None
    deviation: Optional[int] = None  # slippage in points
    magic: int = 0
    comment: str = ""
    type_filling: TypeFilling = TypeFilling.RETURN
    stop_limit_price: Optional[Decimal] = None  # 仅 *_STOP_LIMIT


@dataclass(frozen=True)
class OrderResult:
    """下单回执。"""

    retcode: Retcode
    ticket: Optional[int] = None
    price: Optional[Decimal] = None  # 实际成交价
    volume: Optional[Decimal] = None  # 实际成交量（部分成交时 < 请求量）
    comment: str = ""


@dataclass
class Position:
    """持仓快照。"""

    ticket: int
    symbol: str
    side: PositionSide
    volume: Decimal
    open_price: Decimal
    sl: Optional[Decimal] = None
    tp: Optional[Decimal] = None
    profit: Decimal = field(default_factory=lambda: Decimal("0"))
    swap: Decimal = field(default_factory=lambda: Decimal("0"))
    magic: int = 0
    comment: str = ""
    open_time_ms: int = 0


@dataclass
class PendingOrder:
    """挂单快照。"""

    ticket: int
    symbol: str
    type: OrderType
    volume: Decimal
    price: Decimal
    sl: Optional[Decimal] = None
    tp: Optional[Decimal] = None
    magic: int = 0
    comment: str = ""
    placed_time_ms: int = 0


@dataclass
class Deal:
    """Deal (成交记录) — MQL5 HistoryDealGet* 等价物。

    Proto source: reference/grpc/mt5.proto → DealInternal (32 fields).
    """

    ticket: int
    order_ticket: int = 0
    position_ticket: int = 0
    symbol: str = ""
    type: int = 0          # DealType enum
    direction: int = 0     # 0=Buy, 1=Sell
    volume: Decimal = Decimal("0")
    open_price: Decimal = Decimal("0")
    price: Decimal = Decimal("0")
    open_time_ms: int = 0
    history_time_ms: int = 0
    profit: Decimal = Decimal("0")
    swap: Decimal = Decimal("0")
    commission: Decimal = Decimal("0")
    fee: Decimal = Decimal("0")
    sl: Decimal = Decimal("0")
    tp: Decimal = Decimal("0")
    comment: str = ""
    contract_size: Decimal = Decimal("0")
    digits: int = 0
    login: int = 0
    expert_id: int = 0
