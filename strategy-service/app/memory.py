"""
策略记忆系统（BM25 实现）

移植自 TradingAgents FinancialSituationMemory，针对 Forex/MT4/MT5 策略场景改造。
记忆以「市场情境摘要 → 策略表现+改进建议」的键值对形式持久化到本地 JSON。

用法：
    mem = StrategyMemory("strategy_memory")
    mem.record(situation, performance_summary, advice)
    results = mem.query(current_situation, n=3)
"""

import json
import logging
import os
import re
import time
from pathlib import Path
from typing import List, Tuple, Optional

try:
    from rank_bm25 import BM25Plus
    _BM25_AVAILABLE = True
except ImportError:
    _BM25_AVAILABLE = False


MEMORY_DIR = os.getenv("MEMORY_DIR", "/app/data/memory")
MAX_MEMORY_ENTRIES = int(os.getenv("MAX_MEMORY_ENTRIES", "500"))
logger = logging.getLogger(__name__)


class StrategyMemory:
    """
    基于 BM25 的策略情境记忆。

    每条记忆包含：
      - situation: 市场情境描述（品种、周期、指标快照、策略逻辑摘要）
      - advice    : 该情境下的策略表现总结 + 改进建议
      - timestamp : 写入时间
    """

    def __init__(self, name: str = "strategy_memory"):
        self.name = name
        self._docs: List[str] = []
        self._advices: List[str] = []
        self._timestamps: List[float] = []
        self._bm25: Optional["BM25Okapi"] = None
        self._storage_path = Path(MEMORY_DIR) / f"{name}.json"
        self._load()

    # ------------------------------------------------------------------
    # 公开接口
    # ------------------------------------------------------------------

    def record(self, situation: str, advice: str) -> None:
        """写入一条记忆，若条目超限则淘汰最旧的。"""
        self._docs.append(situation)
        self._advices.append(advice)
        self._timestamps.append(time.time())

        if len(self._docs) > MAX_MEMORY_ENTRIES:
            self._docs = self._docs[-MAX_MEMORY_ENTRIES:]
            self._advices = self._advices[-MAX_MEMORY_ENTRIES:]
            self._timestamps = self._timestamps[-MAX_MEMORY_ENTRIES:]

        self._rebuild_index()
        self._save()

    def query(self, situation: str, n: int = 3) -> List[dict]:
        """
        检索最相似的 n 条历史记忆。

        返回列表，每项：
          {"situation": str, "advice": str, "score": float, "timestamp": float}
        """
        if not self._docs or not _BM25_AVAILABLE or self._bm25 is None:
            return []

        tokens = self._tokenize(situation)
        scores = self._bm25.get_scores(tokens)

        top_indices = sorted(range(len(scores)), key=lambda i: scores[i], reverse=True)[:n]

        # BM25Plus 分数始终 >= 0，直接用 max 归一化
        max_score = max(scores[i] for i in top_indices) or 1.0

        results = []
        for idx in top_indices:
            results.append({
                "situation": self._docs[idx],
                "advice": self._advices[idx],
                "score": float(scores[idx] / max_score),
                "timestamp": self._timestamps[idx],
            })
        return results

    def size(self) -> int:
        return len(self._docs)

    def clear(self) -> None:
        self._docs = []
        self._advices = []
        self._timestamps = []
        self._bm25 = None
        self._save()

    # ------------------------------------------------------------------
    # 内部方法
    # ------------------------------------------------------------------

    def _tokenize(self, text: str) -> List[str]:
        return re.findall(r'\b\w+\b', text.lower())

    def _rebuild_index(self) -> None:
        if not _BM25_AVAILABLE or not self._docs:
            self._bm25 = None
            return
        tokenized = [self._tokenize(doc) for doc in self._docs]
        self._bm25 = BM25Plus(tokenized)

    def _save(self) -> None:
        try:
            self._storage_path.parent.mkdir(parents=True, exist_ok=True)
            data = {
                "name": self.name,
                "entries": [
                    {"situation": s, "advice": a, "timestamp": t}
                    for s, a, t in zip(self._docs, self._advices, self._timestamps)
                ],
            }
            tmp = self._storage_path.with_suffix(".tmp")
            tmp.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
            tmp.replace(self._storage_path)
        except Exception as exc:
            logger.warning("failed to save strategy memory %s: %s", self.name, exc)

    def _load(self) -> None:
        try:
            if not self._storage_path.exists():
                return
            data = json.loads(self._storage_path.read_text(encoding="utf-8"))
            for entry in data.get("entries", []):
                self._docs.append(entry["situation"])
                self._advices.append(entry["advice"])
                self._timestamps.append(float(entry.get("timestamp", 0.0)))
            self._rebuild_index()
        except Exception as exc:
            logger.warning("failed to load strategy memory %s: %s", self.name, exc)


# ------------------------------------------------------------------
# 全局单例（strategy_memory & backtest_memory 分开管理）
# ------------------------------------------------------------------

_strategy_memory: Optional[StrategyMemory] = None
_backtest_memory: Optional[StrategyMemory] = None


def get_strategy_memory() -> StrategyMemory:
    global _strategy_memory
    if _strategy_memory is None:
        _strategy_memory = StrategyMemory("strategy_memory")
    return _strategy_memory


def get_backtest_memory() -> StrategyMemory:
    global _backtest_memory
    if _backtest_memory is None:
        _backtest_memory = StrategyMemory("backtest_memory")
    return _backtest_memory


def build_situation_key(
    symbol: str,
    timeframe: str,
    strategy_code: str,
    metrics: Optional[dict] = None,
) -> str:
    """
    从回测输入构建「市场情境」文本，用于 BM25 索引。
    只保留「结构特征」：symbol、timeframe、指标名、period 参数。
    metrics 不参与 key（查询时没有 metrics，避免 token 不对齐）。
    """
    parts = [f"symbol {symbol}", f"timeframe {timeframe}"]

    # 从策略代码提取关键词（函数名、指标名）
    indicators = re.findall(
        r'\b(iMA|iRSI|iBands|iMACD|iStochastic|iATR|iCCI|iMomentum|iWPR)\b',
        strategy_code,
    )
    if indicators:
        parts.append("indicators " + " ".join(sorted(set(indicators))))

    # 从策略代码提取 period 数值
    periods = re.findall(r'period\s*=\s*(\d+)', strategy_code)
    if periods:
        parts.append("periods " + " ".join(periods))

    # 提取策略关键逻辑词（buy/sell/hold/crossover 等）
    logic_words = re.findall(r'\b(buy|sell|hold|crossover|cross|breakout|signal)\b', strategy_code.lower())
    if logic_words:
        parts.append("logic " + " ".join(sorted(set(logic_words))))

    return " ".join(parts)


def build_advice_text(metrics: dict, strategy_code: str) -> str:
    """
    将回测结果转化为可读的「建议文本」，写入 BM25 记忆。
    """
    total_return = metrics.get("total_return", 0)
    win_rate = metrics.get("win_rate", 0)
    max_dd = metrics.get("max_drawdown", 0)
    sharpe = metrics.get("sharpe_ratio", 0)
    total_trades = metrics.get("total_trades", 0)
    profit_factor = metrics.get("profit_factor", 0)

    lines = [
        f"回测结果：总收益 {total_return:.2f}%，胜率 {win_rate:.1f}%，"
        f"最大回撤 {max_dd:.2f}%，夏普 {sharpe:.2f}，交易次数 {total_trades}，"
        f"盈亏比 {profit_factor:.2f}。",
    ]

    # 生成改进建议
    suggestions = []
    if win_rate < 45:
        suggestions.append("胜率偏低，考虑加强入场过滤条件（如增加趋势确认）")
    if max_dd > 20:
        suggestions.append("最大回撤过大，建议缩小仓位或收紧止损")
    if total_trades < 10:
        suggestions.append("交易次数过少，策略可能过度过滤信号，可适当放宽条件")
    if profit_factor < 1.2 and profit_factor > 0:
        suggestions.append("盈亏比偏低，建议优化止盈止损比例（目标 ≥ 1.5）")
    if sharpe < 0.5 and sharpe != 0:
        suggestions.append("夏普比率偏低，策略收益波动过大")
    if total_return > 0 and win_rate > 55 and max_dd < 10:
        suggestions.append("策略表现良好，可考虑在相同品种/周期扩大使用")

    if suggestions:
        lines.append("改进建议：" + "；".join(suggestions) + "。")
    else:
        lines.append("策略表现合理，无特别改进建议。")

    return "\n".join(lines)
