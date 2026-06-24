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

_backtest_memory: Optional[StrategyMemory] = None


def get_backtest_memory() -> StrategyMemory:
    global _backtest_memory
    if _backtest_memory is None:
        _backtest_memory = StrategyMemory("backtest_memory")
    return _backtest_memory
