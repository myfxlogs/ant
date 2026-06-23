"""Price series with MQL reverse-indexing (interface stub).

契约：docs/adr/0020 · 任务 T0.1
**MQL 逆序索引**：series[0] = 当前 bar，series[1] = 上一根，依此类推。
这是翻译保真的关键——MQL 数组默认 series=true。
"""

from __future__ import annotations

from typing import List


class Series:
    """单一价格序列的逆序视图（close/open/high/low/volume/time 各一个）。

    实现应在底层持有正序 ndarray，__getitem__ 做逆序映射，避免拷贝。
    """

    def __getitem__(self, shift: int) -> float:
        """shift=0 当前 bar，shift=1 上一根。越界由实现定义（建议抛 IndexError）。"""
        raise NotImplementedError

    def __len__(self) -> int:
        raise NotImplementedError

    def slice(self, count: int) -> List[float]:
        """取最近 count 根（正序，供指标函数消费）。"""
        raise NotImplementedError


class Bars:
    """一个时间框下的全部序列集合。"""

    timeframe: str
    open: Series
    high: Series
    low: Series
    close: Series
    volume: Series
    time: Series  # bar 开盘时间 (unix_ms)

    def total(self) -> int:
        """可用 bar 数（等价 MQL iBars / Bars）。"""
        raise NotImplementedError
