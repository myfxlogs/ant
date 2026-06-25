"""Unified Strategy SDK (interface stubs).

契约冻结：见 docs/adr/0020-ea-replacement-strategy-sdk-and-broker.md (D1-D7)
任务：docs/plan/2026-06-23-EA完全替代-实施说明书(DeepSeek).md  T0.1 / T0.2

本包是 **接口桩**：仅定义签名与数据契约，不含实现逻辑。
任何实现（SimBroker/LiveBroker/runtime）必须遵守此处冻结的接口；
修改接口签名需在 PR 标注 CONTRACT-CHANGE 并经人类确认。

关键约定：
- 金额 / 价格一律使用 ``decimal.Decimal``，禁止 float（CLAUDE.md）。
- Series 采用 MQL 逆序索引：``close[0]`` = 当前 bar，``close[1]`` = 上一根。
- 策略以类承载（``class Strategy(StrategyBase)``），承接 EA 的全局状态。
- 回测与实盘注入同一份 SDK 对象；策略代码不感知 broker 实现（D3）。
"""

from __future__ import annotations

from app.sdk.account import AccountInfo
from app.sdk.broker import Broker
from app.sdk.context import Context
from app.sdk.indicators import Indicators
from app.sdk.runtime import RuntimeContext, StrategyRuntime
from app.sdk.series import Series
from app.sdk.strategy_base import StrategyBase
from app.sdk.symbol import SymbolInfo
from app.sdk.types import (
    AccountMode,
    Deal,
    DealType,
    OrderRequest,
    OrderResult,
    OrderType,
    PendingOrder,
    Position,
    PositionSide,
    Retcode,
    TypeFilling,
)

__all__ = [
    "AccountInfo",
    "AccountMode",
    "Broker",
    "Context",
    "Deal",
    "DealType",
    "Indicators",
    "OrderRequest",
    "OrderResult",
    "OrderType",
    "PendingOrder",
    "Position",
    "PositionSide",
    "Retcode",
    "RuntimeContext",
    "Series",
    "StrategyBase",
    "StrategyRuntime",
    "SymbolInfo",
    "TypeFilling",
]
