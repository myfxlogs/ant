"""Account state (interface stub).

契约：docs/adr/0020 · 任务 T0.1
镜像 MT ACCOUNT_* 属性。金额一律 Decimal。
"""

from __future__ import annotations

from dataclasses import dataclass
from decimal import Decimal

from app.sdk.types import AccountMode


@dataclass(frozen=True)
class AccountInfo:
    """账户实时状态快照。"""

    balance: Decimal
    equity: Decimal
    margin: Decimal
    free_margin: Decimal
    margin_level: Decimal       # 百分比；无持仓时由实现定义
    leverage: int
    currency: str
    mode: AccountMode           # netting / hedging
