"""Strategy runtime context (interface stub).

契约：docs/adr/0020 · 任务 T0.1
策略通过 ``self.ctx`` 访问行情/账户/参数；通过 ``self.broker`` 交易。
注意：这是 **SDK 上下文对象**，不同于 legacy 的 dict context（engine/context.py）。
迁移期 SimBroker 负责把 legacy dict 适配为此对象。
"""

from __future__ import annotations

from typing import Optional

from app.sdk.series import Bars


class Context:
    """单次回调内策略可见的运行时上下文。"""

    symbol: str
    timeframe: str

    def bars(self, timeframe: Optional[str] = None) -> Bars:
        """取某时间框的序列集合；None 取主时间框。"""
        raise NotImplementedError

    def param(self, name: str, default: object = None) -> object:
        """读取策略参数（等价 MQL extern/input）。"""
        raise NotImplementedError

    def set_timer(self, seconds: int) -> None:
        """注册 on_timer 周期（等价 MQL EventSetTimer）。"""
        raise NotImplementedError

    def kill_timer(self) -> None:
        """注销定时器（等价 MQL EventKillTimer）。"""
        raise NotImplementedError
