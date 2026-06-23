"""Broker abstraction — 方案拱心石 (interface stub).

契约：docs/adr/0020 · 任务 T0.2 (D3/D6)
SimBroker（回测）与 LiveBroker（实盘）实现同一接口；策略不感知是哪种。
**所有 order_send 在到达真实/仿真撮合前，实现内部必须先过 Go 风控门 (D6)；
被拦截时返回 OrderResult(retcode=Retcode.RISK_BLOCKED)。**

所有价格/手数/金额一律 Decimal。
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from decimal import Decimal
from typing import List, Optional

from app.sdk.account import AccountInfo
from app.sdk.symbol import SymbolInfo
from app.sdk.types import OrderRequest, OrderResult, PendingOrder, Position


class Broker(ABC):
    """统一交易接口。SimBroker / LiveBroker 必须完整实现。"""

    @abstractmethod
    def order_send(self, request: OrderRequest) -> OrderResult:
        """提交下单意图（市价或挂单）。内部先过风控门 (D6)。"""
        raise NotImplementedError

    @abstractmethod
    def position_modify(
        self, ticket: int, sl: Optional[Decimal] = None, tp: Optional[Decimal] = None
    ) -> OrderResult:
        """修改持仓 SL/TP（sl/tp 为 Decimal 或 None）。"""
        raise NotImplementedError

    @abstractmethod
    def position_close(self, ticket: int, volume: Optional[Decimal] = None) -> OrderResult:
        """平仓；volume=None 全平，否则部分平仓（Decimal）。"""
        raise NotImplementedError

    @abstractmethod
    def order_delete(self, ticket: int) -> OrderResult:
        """删除挂单。"""
        raise NotImplementedError

    @abstractmethod
    def positions(self, symbol: Optional[str] = None, magic: Optional[int] = None) -> List[Position]:
        """查询持仓（可按品种/magic 过滤）。"""
        raise NotImplementedError

    @abstractmethod
    def orders(self, symbol: Optional[str] = None, magic: Optional[int] = None) -> List[PendingOrder]:
        """查询挂单。"""
        raise NotImplementedError

    @abstractmethod
    def account(self) -> AccountInfo:
        """账户实时状态。"""
        raise NotImplementedError

    @abstractmethod
    def symbol_info(self, symbol: str) -> SymbolInfo:
        """品种交易规格。"""
        raise NotImplementedError

    @abstractmethod
    def server_time(self) -> int:
        """交易服务器时间 (unix_ms)。回测取当前 bar/ tick 时间。"""
        raise NotImplementedError
