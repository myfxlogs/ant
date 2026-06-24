"""Vectorized strategy runner.

Provides ``DataFrameStrategyRunner`` — an alternative to the event-driven
``StrategyRunner`` that calls ``run_dataframe(df, params)`` once with the
full OHLC DataFrame.  The function returns a DataFrame with signal columns
that are consumed bar-by-bar by the existing main loop.

Signal contract (aligned with QuantDinger indicator workspace):

    # Four-way signals (preferred)
    df["open_long"]   = True   # enter long at this bar
    df["close_long"]  = True   # exit long at this bar
    df["open_short"]  = True   # enter short at this bar
    df["close_short"] = True   # exit short at this bar

    # Legacy two-way signals (backward compatible)
    df["buy"]  = True
    df["sell"] = True
"""

from __future__ import annotations

import ast
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional

import pandas as pd

from app.engine.sandbox import (
    StrategyValidationResult,
    _compile_source,
    build_sandbox_globals,
    code_sha256,
)
from app.engine.types import StrategyCompileError, StrategyRuntimeError

# ── validation ───────────────────────────────────────────────────────────────

_FOUR_WAY_COLS = {"open_long", "close_long", "open_short", "close_short"}
_LEGACY_COLS = {"buy", "sell"}


@dataclass(frozen=True)
class DataFrameValidationResult:
    valid: bool
    errors: List[str]
    warnings: List[str]
    quality_hints: List[Any] = field(default_factory=list)


def validate_dataframe_code(code: str) -> DataFrameValidationResult:
    """AST-level check that code has a valid ``run_dataframe`` entry point."""
    errors: List[str] = []
    warnings: List[str] = []

    try:
        tree = ast.parse(code)
    except SyntaxError as e:
        errors.append(f"语法错误: {e}")
        return DataFrameValidationResult(valid=False, errors=errors, warnings=warnings)

    run_df_defs: List[ast.FunctionDef] = []
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef) and node.name == "run_dataframe":
            run_df_defs.append(node)

    if len(run_df_defs) == 0:
        errors.append(
            "矢量策略必须定义 run_dataframe(df, params) 函数。"
            "df 是包含整个 OHLC 的 DataFrame, params 是参数字典。"
        )
    elif len(run_df_defs) > 1:
        errors.append("只允许定义一个 run_dataframe(df, params) 函数")
    else:
        fn = run_df_defs[0]
        if fn.args.vararg is not None or fn.args.kwarg is not None:
            errors.append("run_dataframe 禁止使用 *args/**kwargs")
        if len(fn.args.args) != 2:
            errors.append(
                "run_dataframe 必须且只能接收两个参数: df, params"
            )

    seen = set()
    deduped: List[str] = []
    for e in errors:
        if e not in seen:
            seen.add(e)
            deduped.append(e)

    return DataFrameValidationResult(
        valid=len(deduped) == 0, errors=deduped, warnings=warnings,
    )


# ── runner ───────────────────────────────────────────────────────────────────


class DataFrameStrategyRunner:
    """Compile once, execute on the full DataFrame.

    Unlike ``StrategyRunner`` which is called per-bar via ``call(ctx)``,
    this runner is called once with the complete OHLC DataFrame and returns
    a signal DataFrame of the same length.
    """

    def __init__(self, source: str, timeout_ms: int = 30_000) -> None:
        validation = validate_dataframe_code(source)
        if not validation.valid:
            raise StrategyCompileError("; ".join(validation.errors))
        # Also run the general sandbox validation to catch import/dunder/etc.
        sandbox_validation = _run_general_validation(source)
        if not sandbox_validation.valid:
            raise StrategyCompileError("; ".join(sandbox_validation.errors))

        self._source = source
        self._timeout_ms = timeout_ms
        try:
            self._bytecode = _compile_source(source)
        except SyntaxError as e:
            raise StrategyCompileError(f"Python 编译失败: {e}") from e

    @property
    def source_sha256(self) -> str:
        return code_sha256(self._source)

    def call(self, ctx: dict) -> Optional[dict]:
        raise NotImplementedError(
            "DataFrameStrategyRunner uses call_dataframe(df, params) "
            "instead of call(ctx). The engine calls call_dataframe once "
            "with the full OHLC DataFrame, then uses extract_signal_at() "
            "to read signals bar-by-bar."
        )

    def shutdown(self) -> None:
        pass

    def call_dataframe(
        self, df: pd.DataFrame, params: Dict[str, Any]
    ) -> pd.DataFrame:
        """Execute ``run_dataframe(df, params)`` and return the signal DataFrame."""
        globals_dict = self._build_globals()
        locals_dict: Dict[str, Any] = {}
        try:
            exec(self._bytecode, globals_dict, locals_dict)
        except Exception as e:
            raise StrategyRuntimeError(f"策略代码执行错误: {e}") from e

        run_df_fn = locals_dict.get("run_dataframe")
        if not callable(run_df_fn):
            raise StrategyRuntimeError(
                "策略代码必须定义 run_dataframe(df, params) 函数"
            )

        try:
            result = run_df_fn(df, params)
        except Exception as e:
            raise StrategyRuntimeError(f"run_dataframe() 抛出异常: {e}") from e

        if not isinstance(result, pd.DataFrame):
            raise StrategyRuntimeError(
                f"run_dataframe() 必须返回 DataFrame，收到 {type(result).__name__}"
            )

        if len(result) != len(df):
            raise StrategyRuntimeError(
                f"run_dataframe() 返回的 DataFrame 长度 ({len(result)}) "
                f"与输入 OHLC ({len(df)}) 不一致"
            )

        return result

    def _build_globals(self) -> dict:
        return build_sandbox_globals()


def extract_signal_at(
    signal_df: pd.DataFrame, idx: int, params: Optional[Dict[str, Any]] = None
) -> Optional[dict]:
    """Extract a signal dict at bar ``idx`` from the vectorized signal DataFrame.

    Returns None for HOLD, or a dict compatible with the existing
    ``_dispatch_signal`` contract.
    """
    if idx < 0 or idx >= len(signal_df):
        return None

    row = signal_df.iloc[idx]

    # Four-way signals take priority over legacy two-way.
    if _FOUR_WAY_COLS & set(signal_df.columns):
        # close_short before open_long, close_long before open_short
        # so that reversals are handled correctly (close existing first).
        cs = bool(row.get("close_short", False) if "close_short" in signal_df.columns else False)
        cl = bool(row.get("close_long", False) if "close_long" in signal_df.columns else False)
        os_ = bool(row.get("open_short", False) if "open_short" in signal_df.columns else False)
        ol = bool(row.get("open_long", False) if "open_long" in signal_df.columns else False)

        # Close existing before opening new (reversal-safe ordering).
        if cs and cl:
            return {"signal": "close"}
        if cs:
            return {"signal": "close", "side": "short"}
        if cl:
            return {"signal": "close", "side": "long"}

        # Open new positions.
        if ol and os_:
            # Ambiguous: both directions. Default to buy.
            return {"signal": "buy"}
        if ol:
            return {"signal": "buy"}
        if os_:
            return {"signal": "sell"}

        return None

    # Legacy two-way signals.
    buy = bool(row.get("buy", False) if "buy" in signal_df.columns else False)
    sell = bool(row.get("sell", False) if "sell" in signal_df.columns else False)

    if buy and sell:
        return {"signal": "buy"}
    if buy:
        return {"signal": "buy"}
    if sell:
        return {"signal": "sell"}

    return None


# ── helpers ──────────────────────────────────────────────────────────────────


def _run_general_validation(source: str) -> StrategyValidationResult:
    """Run the general sandbox validation (no duplicate import, etc.).

    Imported lazily to avoid circular import at module level.
    """
    from app.engine.sandbox import validate_strategy_code
    return validate_strategy_code(source)


def detect_strategy_type(code: str) -> str:
    """Auto-detect whether code uses run_dataframe or run(context).

    Returns "run_dataframe" if the code defines ``run_dataframe``,
    "run_context" otherwise.
    """
    try:
        tree = ast.parse(code)
    except SyntaxError:
        return "run_context"
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef) and node.name == "run_dataframe":
            return "run_dataframe"
    return "run_context"
