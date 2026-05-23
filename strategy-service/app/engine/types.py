"""Engine core types.

契约：docs/domains/backtest-system.md §7.4.2 / §7.4.4
约束：本文件不得 import 同包其他模块（types 为依赖图根）。
"""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Dict, List, Literal, Optional


# --- Enums ---------------------------------------------------------------


class Side(str, Enum):
    BUY = "buy"
    SELL = "sell"


class OrderType(str, Enum):
    BUY = "buy"
    SELL = "sell"
    BUY_LIMIT = "buy_limit"
    SELL_LIMIT = "sell_limit"
    BUY_STOP = "buy_stop"
    SELL_STOP = "sell_stop"
    BUY_STOP_LIMIT = "buy_stop_limit"
    SELL_STOP_LIMIT = "sell_stop_limit"
    CLOSE = "close"
    CANCEL_PENDING = "cancel_pending"


class OrderStatus(str, Enum):
    PENDING = "pending"
    ACTIVATED = "activated"
    FILLED = "filled"
    CANCELLED = "cancelled"
    EXPIRED = "expired"
    REJECTED = "rejected"


class RunMode(str, Enum):
    ADVICE = "advice"
    BACKTEST = "backtest"


class SlippageMode(str, Enum):
    FIXED = "fixed"
    RANDOM = "random"


class CloseReason(str, Enum):
    SIGNAL = "signal"
    SL = "sl"
    TP = "tp"
    MARGIN_CALL = "margin_call"
    EXPIRED = "expired"
    END_OF_TEST = "end_of_test"


# --- Market data ---------------------------------------------------------


@dataclass(frozen=True)
class Bar:
    """OHLC bar. Timestamps in unix milliseconds (UTC)."""

    open_time: int
    close_time: int
    open: float
    high: float
    low: float
    close: float
    volume: float = 0.0


@dataclass(frozen=True)
class Tick:
    """Quote tick. ts in unix milliseconds (UTC)."""

    ts: int
    bid: float
    ask: float


# --- Cost profile --------------------------------------------------------


@dataclass(frozen=True)
class CostProfile:
    """Backtest cost configuration. Semantics align with legacy engine."""

    commission_per_lot: float = 0.0
    slippage_mode: SlippageMode = SlippageMode.FIXED
    slippage_rate: float = 0.0
    slippage_seed: int = 0
    # Swap (legacy semantics: equity percentage per rollover)
    swap_rate_per_rollover: float = 0.0
    triple_swap_weekday: int = 3
    rollover_hour: int = 0
    server_timezone: str = "UTC"
    # Used only when legacy_pnl == False
    pip_size: float = 0.0001
    contract_size: float = 100_000.0


# --- Orders / fills / positions / trades --------------------------------


@dataclass
class Order:
    id: int
    type: OrderType
    volume: float
    price: float = 0.0
    sl: float = 0.0
    tp: float = 0.0
    stop_limit_price: float = 0.0
    expiration: Optional[int] = None
    activated: bool = False
    status: OrderStatus = OrderStatus.PENDING
    created_at_ts: int = 0


@dataclass
class Fill:
    order_id: int
    ts: int
    price: float
    volume: float
    commission: float = 0.0
    slippage: float = 0.0


@dataclass
class Position:
    ticket: int
    side: Side
    volume: float
    open_price: float
    open_ts: int
    sl: float = 0.0
    tp: float = 0.0


@dataclass
class Trade:
    ticket: int
    side: Side
    volume: float
    open_ts: int
    open_price: float
    close_ts: int
    close_price: float
    pnl: float
    commission: float
    reason: CloseReason


# --- Metrics / risk / snapshot ------------------------------------------


@dataclass
class Metrics:
    total_return: float = 0.0
    annual_return: float = 0.0
    max_drawdown: float = 0.0
    sharpe_ratio: float = 0.0
    win_rate: float = 0.0
    profit_factor: float = 0.0
    total_trades: int = 0
    winning_trades: int = 0
    losing_trades: int = 0
    average_profit: float = 0.0
    average_loss: float = 0.0


@dataclass
class RiskAssessment:
    score: int = 0
    level: Literal["low", "medium", "high"] = "low"
    reasons: List[str] = field(default_factory=list)
    warnings: List[str] = field(default_factory=list)
    is_reliable: bool = False


@dataclass
class RunSnapshot:
    code_sha256: str = ""
    params: dict = field(default_factory=dict)
    cost_profile: Optional[CostProfile] = None
    dataset_id: Optional[str] = None
    bars_count: int = 0
    ticks_count: int = 0


# --- Request / response --------------------------------------------------


@dataclass(frozen=True)
class BacktestRequest:
    """Inputs for one backtest run.

    Loaded by the Go backend worker and dispatched into an engine subprocess.
    """

    run_id: str
    user_id: int
    account_id: int
    symbol: str
    timeframe: str
    start: datetime
    end: datetime
    initial_cash: float
    leverage: float = 0.0
    source: Literal["MT_LIVE", "DATASET"] = "MT_LIVE"
    dataset_id: Optional[str] = None
    strategy_code: str = ""
    strategy_params: dict = field(default_factory=dict)
    cost_profile: CostProfile = field(default_factory=CostProfile)
    bars: List[Bar] = field(default_factory=list)
    ticks: Optional[List[Tick]] = None
    # Multi-symbol support (Phase B2). When ``bars_by_symbol`` is non-empty the
    # runner builds a :class:`MultiSymbolMarket` keyed by symbol and exposes
    # ``closes_by_symbol`` etc. to the sandbox context. Trading execution still
    # targets ``primary_symbol`` (defaults to ``symbol``).
    symbols: List[str] = field(default_factory=list)
    bars_by_symbol: Dict[str, List[Bar]] = field(default_factory=dict)
    primary_symbol: Optional[str] = None
    single_position_only: bool = True
    legacy_pnl: bool = True
    max_fill_volume: float = 0.0
    deadline_ms: int = 120_000


@dataclass
class BacktestResult:
    run_id: str
    success: bool = False
    equity_curve: List[float] = field(default_factory=list)
    events: List[dict] = field(default_factory=list)
    metrics: Metrics = field(default_factory=Metrics)
    risk_assessment: RiskAssessment = field(default_factory=RiskAssessment)
    trades: List[Trade] = field(default_factory=list)
    snapshot: Optional[RunSnapshot] = None
    error: Optional[str] = None


# --- Exceptions ----------------------------------------------------------


class EngineError(Exception):
    """Base class for all engine errors."""


class StrategyCompileError(EngineError):
    """AST whitelist or RestrictedPython compilation rejected the code."""


class StrategyRuntimeError(EngineError):
    """Strategy raised inside the sandbox at runtime."""


class DataUnavailableError(EngineError):
    """Historical data could not be loaded."""


class DeadlineExceededError(EngineError):
    """Hard timeout reached; engine process must be killed."""


class MarginCallError(EngineError):
    """Unused in normal flow (margin call is graceful), reserved for fatal cases."""
