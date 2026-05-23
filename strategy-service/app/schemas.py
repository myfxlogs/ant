from datetime import datetime
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class KlineData(BaseModel):
    open_time: datetime
    close_time: datetime
    open_price: float
    high_price: float
    low_price: float
    close_price: float
    volume: float


class MarketData(BaseModel):
    symbol: str
    timeframe: str
    klines: List[KlineData]
    current_price: Optional[float] = None


class StrategyExecuteRequest(BaseModel):
    strategy_id: str
    strategy_code: str
    market_data: MarketData
    context: Optional[Dict[str, Any]] = None


class TradeSignal(BaseModel):
    signal: str
    symbol: str
    price: Optional[float] = None
    volume: Optional[float] = None
    stop_loss: Optional[float] = None
    take_profit: Optional[float] = None
    confidence: float = Field(ge=0, le=1)
    reason: Optional[str] = None
    risk_level: str


class StrategyExecuteResponse(BaseModel):
    success: bool
    signal: Optional[TradeSignal] = None
    error: Optional[str] = None
    runtime: Optional[Dict[str, Any]] = None
    execution_time_ms: float
    logs: List[str]


class StrategyValidateRequest(BaseModel):
    strategy_code: str


class StrategyValidateResponse(BaseModel):
    valid: bool
    errors: List[str]
    warnings: List[str]
    parameters: List[Dict[str, Any]] = Field(default_factory=list)


class BacktestRequest(BaseModel):
    strategy_id: str
    strategy_code: str
    symbol: str
    timeframe: str
    start_date: datetime
    end_date: datetime
    initial_capital: float
    commission: float
    spread: float = 0.0
    swap_rate: float = 0.0
    server_timezone: str = "UTC"
    rollover_hour: int = 0
    triple_swap_weekday: int = 3
    slippage_mode: str = "fixed"
    slippage_rate: float = 0.0
    slippage_seed: int = 0
    klines: List[KlineData]
    ticks: List[dict] = []
    extra_symbols: List[str] = Field(default_factory=list)
    klines_by_symbol: Dict[str, List[KlineData]] = Field(default_factory=dict)


class BacktestMetrics(BaseModel):
    total_return: float
    annual_return: float
    max_drawdown: float
    sharpe_ratio: float
    win_rate: float
    profit_factor: float
    total_trades: int
    winning_trades: int
    losing_trades: int
    average_profit: float
    average_loss: float


class RiskAssessment(BaseModel):
    score: int
    level: str
    reasons: List[str]
    warnings: List[str]
    is_reliable: bool


class BacktestTrade(BaseModel):
    ticket: int
    side: str
    volume: float
    open_ts: int
    open_price: float
    close_ts: int
    close_price: float
    pnl: float
    commission: float
    reason: str


class BacktestResponse(BaseModel):
    success: bool
    metrics: Optional[BacktestMetrics] = None
    risk_assessment: Optional[RiskAssessment] = None
    equity_curve: List[float]
    events: List[Dict[str, Any]] = []
    trades: List[BacktestTrade] = []
    error: Optional[str] = None


class MemoryRecordRequest(BaseModel):
    symbol: str
    timeframe: str
    strategy_code: str
    metrics: Dict[str, Any]
    extra_advice: Optional[str] = None


class MemoryRecordResponse(BaseModel):
    success: bool
    total_entries: int


class MemoryQueryRequest(BaseModel):
    symbol: str
    timeframe: str
    strategy_code: str
    n: int = 3


class MemoryEntry(BaseModel):
    situation: str
    advice: str
    score: float


class MemoryQueryResponse(BaseModel):
    memories: List[MemoryEntry]


class ExternalMacroResponse(BaseModel):
    data: Dict[str, Any] = {}


class NewsItem(BaseModel):
    title: str
    summary: Optional[str] = None
    sentiment: Optional[str] = None
    date: Optional[str] = None


class ExternalNewsResponse(BaseModel):
    items: List[NewsItem]
    status: str = "reserved"
    provider_status: str = "not_configured"
    market: str = ""
    symbol: str = ""
    intended_use: str = "future_financial_news_data_for_ai_strategy_authoring"


class ObjectiveScoreRequest(BaseModel):
    symbol: str
    timeframe: str
    klines: List[KlineData]


class RSISignal(BaseModel):
    value: float
    signal: str


class MACDSignal(BaseModel):
    value: float
    signal_line: float
    histogram: float
    signal: str
    trend: str


class MASignal(BaseModel):
    ma5: float
    ma10: float
    ma20: float
    trend: str


class ObjectiveSignals(BaseModel):
    rsi: RSISignal
    macd: MACDSignal
    ma: MASignal


class ObjectiveScoreResponse(BaseModel):
    decision: str
    overall_score: float
    technical_score: float
    signals: ObjectiveSignals
