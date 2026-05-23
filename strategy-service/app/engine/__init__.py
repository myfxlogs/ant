"""AntTrader v2 Backtest Engine.

对外入口：

    from app.engine import run_backtest
    result = run_backtest(req)

实现按 docs/domains/backtest-system.md §7.4 契约落地。
"""

from app.engine.runner import BacktestRunner, run_backtest
from app.engine.types import (
    BacktestRequest,
    BacktestResult,
    Bar,
    CloseReason,
    CostProfile,
    DataUnavailableError,
    DeadlineExceededError,
    EngineError,
    Fill,
    MarginCallError,
    Metrics,
    Order,
    OrderStatus,
    OrderType,
    Position,
    RiskAssessment,
    RunMode,
    RunSnapshot,
    Side,
    SlippageMode,
    StrategyCompileError,
    StrategyRuntimeError,
    Tick,
    Trade,
)

__all__ = [
    "BacktestRunner",
    "run_backtest",
    "BacktestRequest",
    "BacktestResult",
    "Bar",
    "CloseReason",
    "CostProfile",
    "DataUnavailableError",
    "DeadlineExceededError",
    "EngineError",
    "Fill",
    "MarginCallError",
    "Metrics",
    "Order",
    "OrderStatus",
    "OrderType",
    "Position",
    "RiskAssessment",
    "RunMode",
    "RunSnapshot",
    "Side",
    "SlippageMode",
    "StrategyCompileError",
    "StrategyRuntimeError",
    "Tick",
    "Trade",
]
