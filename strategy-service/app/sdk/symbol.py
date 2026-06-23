"""Symbol metadata (interface stub).

契约：docs/adr/0020 · 任务 T0.1
镜像 MT SYMBOL_* 属性。所有价格/货币量用 Decimal。
"""

from __future__ import annotations

from dataclasses import dataclass
from decimal import Decimal


@dataclass(frozen=True)
class SymbolInfo:
    """品种交易规格（成交规则/保证金/swap 计算依赖此处）。"""

    name: str
    digits: int                 # 报价小数位
    point: Decimal              # 最小报价变动 (10**-digits)
    tick_size: Decimal          # 最小价格步长
    tick_value: Decimal         # 每 tick 价值（账户货币）
    contract_size: Decimal      # 合约规模（1 lot 对应的基础单位）
    volume_min: Decimal
    volume_max: Decimal
    volume_step: Decimal
    stops_level: int            # SL/TP 距现价的最小点数
    freeze_level: int           # 冻结距离（点）
    swap_long: Decimal
    swap_short: Decimal
    margin_rate: Decimal        # 保证金率

    def normalize_price(self, price: Decimal) -> Decimal:
        """按 digits 规整价格。"""
        raise NotImplementedError

    def normalize_volume(self, volume: Decimal) -> Decimal:
        """按 volume_step / min / max 规整手数。"""
        raise NotImplementedError
