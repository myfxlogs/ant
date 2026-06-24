"""Shared SDK strategy loader — compiles code and returns the StrategyBase subclass.

Used by both runner.py (backtest) and sdk_worker.py (live execution).
"""

from __future__ import annotations

from typing import Optional

from app.sdk.strategy_base import StrategyBase


def load_sdk_strategy(code: str) -> type[StrategyBase]:
    """Compile strategy source and return the StrategyBase subclass.

    Uses ``build_sandbox_globals()`` to pre-inject numpy, math, and
    engine indicators into the execution namespace.

    Raises ``ValueError`` if no StrategyBase subclass is found.
    """
    from app.engine.sandbox import build_sandbox_globals

    exec_scope = build_sandbox_globals()
    exec_scope["StrategyBase"] = StrategyBase
    exec(compile(code, "<strategy>", "exec"), exec_scope)

    for obj in exec_scope.values():
        if isinstance(obj, type) and issubclass(obj, StrategyBase) and obj is not StrategyBase:
            return obj

    raise ValueError("No StrategyBase subclass found in code")
