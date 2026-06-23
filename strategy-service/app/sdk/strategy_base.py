"""Strategy lifecycle base class (interface stub).

契约：docs/adr/0020 · 任务 T0.1 (D2)
镜像 MQL 事件模型。翻译后的 EA 继承 StrategyBase 并覆写需要的回调。
runtime（T1.1）按事件驱动调用这些钩子，并注入 broker / ctx / indicators。

生命周期顺序：
  on_init() → (on_tick / on_bar / on_timer / on_trade)* → on_deinit(reason)
"""

from __future__ import annotations

from app.sdk.broker import Broker
from app.sdk.context import Context
from app.sdk.indicators import Indicators


class StrategyBase:
    """所有翻译/手写策略的基类。

    注入项由 runtime 在实例化后设置（不在 __init__ 里要求，便于翻译）：
      self.broker / self.ctx / self.indicators
    """

    broker: Broker
    ctx: Context
    indicators: Indicators

    def on_init(self) -> None:
        """等价 OnInit。初始化参数、注册定时器等。"""

    def on_tick(self) -> None:
        """等价 OnTick。逐 tick 驱动（事件型 EA 的主入口）。"""

    def on_bar(self, timeframe: str) -> None:
        """新 bar 收盘事件（由 tick 派生）。bar 型策略入口。"""

    def on_timer(self) -> None:
        """等价 OnTimer。由 set_timer 周期触发。"""

    def on_trade(self) -> None:
        """等价 OnTrade。成交/持仓变化后触发。"""

    def on_deinit(self, reason: str) -> None:
        """等价 OnDeinit。清理资源。"""
