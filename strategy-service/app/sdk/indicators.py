"""Indicator facade exposed to strategies (interface stub).

契约：docs/adr/0020 · 任务 T0.1
镜像 MQL iMA/iRSI/... 语义；shift=0 为当前 bar（MQL 约定）。
实现应复用现有 app/engine/indicators.py 的算法（bit-equal），
此处仅定义策略可见的统一门面 + i_custom 自定义指标挂载点。

注：指标返回 float（数值计算域），价格/手数等账户量才用 Decimal。
"""

from __future__ import annotations

from typing import Sequence, Tuple


class Indicators:
    """策略通过 ``self.indicators`` 访问。所有方法读取当前可见 bar 窗口。"""

    def ma(self, period: int = 14, shift: int = 0, method: str = "sma") -> float:
        raise NotImplementedError

    def ema(self, period: int = 14, shift: int = 0) -> float:
        raise NotImplementedError

    def rsi(self, period: int = 14, shift: int = 0) -> float:
        raise NotImplementedError

    def bands(
        self, period: int = 20, deviation: float = 2.0, shift: int = 0
    ) -> Tuple[float, float, float]:
        """返回 (upper, middle, lower)。"""
        raise NotImplementedError

    def macd(
        self, fast: int = 12, slow: int = 26, signal: int = 9, shift: int = 0
    ) -> Tuple[float, float, float]:
        """返回 (macd, signal, histogram)。"""
        raise NotImplementedError

    def atr(self, period: int = 14, shift: int = 0) -> float:
        raise NotImplementedError

    def stochastic(
        self, k_period: int = 5, d_period: int = 3, shift: int = 0
    ) -> Tuple[float, float]:
        """返回 (k, d)。"""
        raise NotImplementedError

    def cci(self, period: int = 14, shift: int = 0) -> float:
        raise NotImplementedError

    def i_custom(self, name: str, params: Sequence[object], buffer: int = 0, shift: int = 0) -> float:
        """自定义指标挂载点（等价 MQL iCustom）。

        name 解析到已注册/已翻译的自定义指标实现。
        无法解析时实现应抛出明确错误（不可静默返回 0）。
        """
        raise NotImplementedError
