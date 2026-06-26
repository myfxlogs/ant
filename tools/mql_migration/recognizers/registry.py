"""积木识别器注册表。

管理所有识别器的注册、聚合运行、覆盖率追踪、盲区→积木开发触发。

Usage::

    from tools.mql_migration.recognizers.registry import RecognizerRegistry
    registry = RecognizerRegistry()
    registry.register("market_entry", recognize_market_entries)
    registry.register("pending_entry", recognize_pending_entries)

    results = registry.run_all(ast, expr_gen)
    print(results.coverage_summary())
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional, Set, Tuple

from tools.mql_transpiler.ast_nodes import SourceFile

from tools.mql_migration.intent_ir import (
    BlindSpot,
    BlindSpotCategory,
    BlindSpotFingerprint,
    HandlingStrategy,
    Severity,
)


@dataclass
class RecognizerMeta:
    """单个识别器的元信息。"""
    name: str                           # "market_entry"
    description: str = ""
    version: int = 1
    category: str = ""                  # "entry" | "exit" | "sizing" | "risk" | "execution"
    enabled: bool = True
    # Statistics
    total_runs: int = 0
    total_matches: int = 0


@dataclass
class BlockResult:
    """单个识别器的运行结果。"""
    recognizer_name: str
    items: List[Any] = field(default_factory=list)    # 识别到的 EntryRule / ExitRule / ...
    items_count: int = 0
    blind_spots: List[BlindSpot] = field(default_factory=list)


@dataclass
class AggregateResult:
    """所有识别器的聚合结果。"""
    results: Dict[str, BlockResult] = field(default_factory=dict)
    total_blocks_attempted: int = 0
    total_blocks_matched: int = 0
    all_blind_spots: List[BlindSpot] = field(default_factory=list)

    @property
    def coverage(self) -> float:
        if self.total_blocks_attempted == 0:
            return 0.0
        return self.total_blocks_matched / self.total_blocks_attempted

    def coverage_summary(self) -> str:
        """人类可读的覆盖率摘要。"""
        lines = []
        for name, r in sorted(self.results.items()):
            status = "✓" if r.items_count > 0 else "✗"
            lines.append(f"  {status} {name}: {r.items_count} matched")
        lines.append(f"  Coverage: {self.coverage:.0%} "
                      f"({self.total_blocks_matched}/{self.total_blocks_attempted})")
        return "\n".join(lines)


class RecognizerRegistry:
    """积木识别器注册表 — 管理识别器的注册、运行和自学习。"""

    # 盲区触发积木开发的阈值
    DEVELOPMENT_THRESHOLD = 3

    def __init__(self):
        self._recognizers: Dict[str, Tuple[RecognizerMeta, Callable]] = {}
        self._blind_spot_counts: Dict[str, int] = {}  # fingerprint → count

    def register(self, name: str, fn: Callable,
                 description: str = "", category: str = "") -> None:
        """注册一个识别器。

        Args:
            name: 识别器名称，如 "market_entry"
            fn: 识别器函数，签名为 (ast, expr_gen) -> List[X]
            description: 人类可读描述
            category: 分类 entry/exit/sizing/risk/execution
        """
        meta = RecognizerMeta(
            name=name,
            description=description,
            category=category,
        )
        self._recognizers[name] = (meta, fn)

    def unregister(self, name: str) -> None:
        """注销识别器。"""
        self._recognizers.pop(name, None)

    @property
    def registered_names(self) -> List[str]:
        return list(self._recognizers.keys())

    @property
    def entry_count(self) -> int:
        return len(self._recognizers)

    def run_all(self, ast: SourceFile, expr_gen) -> AggregateResult:
        """运行所有已注册的识别器，聚合结果。

        Args:
            ast: MQL 解析后的 AST
            expr_gen: ExpressionGen 实例

        Returns:
            AggregateResult with per-recognizer results
        """
        agg = AggregateResult()

        for name, (meta, fn) in self._recognizers.items():
            if not meta.enabled:
                continue

            try:
                items = fn(ast, expr_gen)
            except Exception as e:
                # 单个识别器失败不应阻断整体流程
                items = []
                agg.all_blind_spots.append(BlindSpot(
                    location=f"recognizer:{name}",
                    category=BlindSpotCategory.CUSTOM_LOGIC,
                    severity=Severity.WARNING,
                    description=f"识别器 {name} 异常: {e}",
                    handling=HandlingStrategy.SKIP_WITH_COMMENT,
                ))

            meta.total_runs += 1
            meta.total_matches += len(items)

            result = BlockResult(
                recognizer_name=name,
                items=items,
                items_count=len(items),
            )
            agg.results[name] = result
            agg.total_blocks_attempted += 1
            if items:
                agg.total_blocks_matched += 1

        agg.all_blind_spots.extend(self._collect_blind_spots(ast))
        return agg

    def track_blind_spot(self, fingerprint: BlindSpotFingerprint) -> bool:
        """追踪盲区，返回是否触发积木开发阈值。

        Args:
            fingerprint: 盲区指纹

        Returns:
            True 如果该盲区出现次数超过 DEVELOPMENT_THRESHOLD
        """
        sig = fingerprint.pattern_signature
        count = self._blind_spot_counts.get(sig, 0) + 1
        self._blind_spot_counts[sig] = count
        return count >= self.DEVELOPMENT_THRESHOLD

    def get_development_candidates(self) -> List[Tuple[str, int]]:
        """获取应触发积木开发的盲区列表。

        Returns:
            [(fingerprint, occurrence_count), ...] 按频次降序
        """
        candidates = [
            (sig, count)
            for sig, count in self._blind_spot_counts.items()
            if count >= self.DEVELOPMENT_THRESHOLD
        ]
        return sorted(candidates, key=lambda x: -x[1])

    def reset_blind_spot(self, fingerprint: str) -> None:
        """重置盲区计数（积木开发完成后）。"""
        self._blind_spot_counts.pop(fingerprint, None)

    def _collect_blind_spots(self, ast: SourceFile) -> List[BlindSpot]:
        """收集全局盲区（未注册的识别器覆盖范围）。"""
        spots = []

        # 检查是否有未覆盖的类别
        covered_categories = set()
        for meta, _ in self._recognizers.values():
            if meta.category:
                covered_categories.add(meta.category)

        all_categories = {"entry", "exit", "sizing", "risk", "execution"}
        missing = all_categories - covered_categories
        for cat in missing:
            spots.append(BlindSpot(
                location="<pipeline>",
                category=BlindSpotCategory.CUSTOM_LOGIC,
                severity=Severity.CRITICAL,
                description=f"未注册 {cat} 类识别器",
                handling=HandlingStrategy.HARD_BLOCK,
                user_action_required=True,
            ))

        return spots

    # ── 自动注册所有内置识别器 ─────────────────────────────────────

    @classmethod
    def create_default(cls) -> RecognizerRegistry:
        """创建预装所有内置识别器的注册表。"""
        from tools.mql_migration.recognizers.market_entry import recognize_market_entries
        from tools.mql_migration.recognizers.pending_entry import recognize_pending_entries
        from tools.mql_migration.recognizers.custom_entry import recognize_custom_entries
        from tools.mql_migration.recognizers.magic_exit import recognize_exit_rules
        from tools.mql_migration.recognizers.margin_check import recognize_margin_checks
        from tools.mql_migration.recognizers.exec_model import (
            recognize_execution_model,
            recognize_meta,
            recognize_params,
            recognize_sizing,
            recognize_state_vars,
            recognize_timer,
        )

        registry = cls()

        # Entry recognizers
        registry.register(
            "market_entry", recognize_market_entries,
            description="市价单入场 (if→OrderSend OP_BUY/SELL)",
            category="entry",
        )
        registry.register(
            "pending_entry", recognize_pending_entries,
            description="挂单入场 (for→OrderSend OP_BUYLIMIT/SELLLIMIT/STOP)",
            category="entry",
        )
        registry.register(
            "custom_entry", recognize_custom_entries,
            description="自定义指标入场 (iCustom→if→OrderSend)",
            category="entry",
        )

        # Exit recognizers
        registry.register(
            "magic_exit", recognize_exit_rules,
            description="Magic 过滤出场 (OrderClose/OrderDelete 循环)",
            category="exit",
        )

        # Structure recognizers
        registry.register(
            "exec_model", recognize_execution_model,
            description="执行模型推断 (on_bar/on_tick/on_init_grid)",
            category="execution",
        )
        registry.register(
            "sizing", recognize_sizing,
            description="手数规则识别 (fixed/martingale)",
            category="sizing",
        )
        registry.register(
            "params", recognize_params,
            description="外部参数提取",
            category="execution",
        )
        registry.register(
            "state_vars", recognize_state_vars,
            description="全局状态变量提取",
            category="execution",
        )
        registry.register(
            "meta", recognize_meta,
            description="策略元信息提取",
            category="execution",
        )
        registry.register(
            "timer", recognize_timer,
            description="定时器规则提取",
            category="risk",
        )
        registry.register(
            "margin_check", recognize_margin_checks,
            description="保证金/风控检查 (AccountFreeMargin → close 模式)",
            category="risk",
        )

        return registry
