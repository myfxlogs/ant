import asyncio
import logging
import os
from datetime import datetime
from typing import Dict, List, Optional

from fastapi import APIRouter

from app.engine import (
    BacktestRequest as EngineBacktestRequest,
    Bar as EngineBar,
    CostProfile as EngineCostProfile,
    SlippageMode as EngineSlippageMode,
    Tick as EngineTick,
    run_backtest as engine_run_backtest,
)
from app.memory import build_advice_text, build_situation_key, get_backtest_memory
from app.schemas import BacktestMetrics, BacktestRequest, BacktestResponse, BacktestTrade, KlineData, RiskAssessment

logger = logging.getLogger(__name__)
router = APIRouter()
backtest_semaphore = asyncio.Semaphore(int(os.getenv('MAX_BACKTEST_WORKERS', '2')))


@router.post("/api/backtest", response_model=BacktestResponse)
async def run_backtest(request: BacktestRequest):
    async with backtest_semaphore:
        engine_req = _build_engine_request(request)
        try:
            loop = asyncio.get_event_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
        result = await loop.run_in_executor(None, engine_run_backtest, engine_req)
        if not result.success:
            return BacktestResponse(success=False, error=result.error or "回测失败", equity_curve=[])
        m = result.metrics
        metrics = BacktestMetrics(
            total_return=m.total_return,
            annual_return=m.annual_return,
            max_drawdown=m.max_drawdown,
            sharpe_ratio=m.sharpe_ratio,
            win_rate=m.win_rate,
            profit_factor=m.profit_factor,
            total_trades=m.total_trades,
            winning_trades=m.winning_trades,
            losing_trades=m.losing_trades,
            average_profit=m.average_profit,
            average_loss=m.average_loss,
        )
        ra = result.risk_assessment
        risk_assessment = RiskAssessment(score=ra.score, level=ra.level, reasons=ra.reasons, warnings=ra.warnings, is_reliable=ra.is_reliable)
        _record_backtest_memory(request, m)
        trades_out = [
            BacktestTrade(
                ticket=t.ticket,
                side=str(t.side.value if hasattr(t.side, 'value') else t.side),
                volume=t.volume,
                open_ts=t.open_ts,
                open_price=t.open_price,
                close_ts=t.close_ts,
                close_price=t.close_price,
                pnl=t.pnl,
                commission=t.commission,
                reason=str(t.reason.value if hasattr(t.reason, 'value') else t.reason),
            )
            for t in (result.trades or [])
        ]
        return BacktestResponse(success=True, metrics=metrics, risk_assessment=risk_assessment, equity_curve=result.equity_curve, events=result.events, trades=trades_out)


def _to_ms(dt: datetime) -> int:
    return int(dt.timestamp() * 1000)


def _build_engine_request(req: BacktestRequest) -> EngineBacktestRequest:
    def _to_bars(klines: List[KlineData]) -> list:
        return [EngineBar(open_time=_to_ms(k.open_time), close_time=_to_ms(k.close_time), open=k.open_price, high=k.high_price, low=k.low_price, close=k.close_price, volume=k.volume) for k in klines]

    bars = _to_bars(req.klines)
    bars_by_symbol: Dict[str, list] = {}
    if req.klines_by_symbol or req.extra_symbols:
        bars_by_symbol[req.symbol] = bars
        for sym, ks in (req.klines_by_symbol or {}).items():
            if sym and sym != req.symbol:
                bars_by_symbol[sym] = _to_bars(ks)
    ticks: list[EngineTick] = []
    for t in req.ticks or []:
        dt = _parse_tick_time(t.get("time"))
        if dt is None:
            continue
        try:
            ticks.append(EngineTick(ts=_to_ms(dt), bid=float(t.get("bid")), ask=float(t.get("ask"))))
        except (TypeError, ValueError):
            continue
    max_fill = _float_env("ANTRADER_BACKTEST_MAX_FILL_VOLUME", 0.0)
    timeout_seconds = int(os.getenv('BACKTEST_TIMEOUT', '120'))
    cost = EngineCostProfile(
        commission_per_lot=req.commission,
        slippage_mode=EngineSlippageMode.RANDOM if (req.slippage_mode or "fixed").lower() == "random" else EngineSlippageMode.FIXED,
        slippage_rate=req.slippage_rate,
        slippage_seed=int(req.slippage_seed),
        swap_rate_per_rollover=req.swap_rate,
        triple_swap_weekday=req.triple_swap_weekday,
        rollover_hour=req.rollover_hour,
        server_timezone=req.server_timezone or "UTC",
    )
    return EngineBacktestRequest(
        run_id=req.strategy_id,
        user_id=0,
        account_id=0,
        symbol=req.symbol,
        timeframe=req.timeframe,
        start=req.start_date,
        end=req.end_date,
        initial_cash=req.initial_capital,
        leverage=0.0,
        source="MT_LIVE",
        dataset_id=None,
        strategy_code=req.strategy_code,
        strategy_params={},
        cost_profile=cost,
        bars=bars,
        ticks=ticks or None,
        symbols=list(bars_by_symbol.keys()) if bars_by_symbol else [],
        bars_by_symbol=bars_by_symbol,
        primary_symbol=req.symbol if bars_by_symbol else None,
        single_position_only=True,
        legacy_pnl=True,
        max_fill_volume=max_fill,
        deadline_ms=timeout_seconds * 1000,
    )


def _parse_tick_time(value) -> Optional[datetime]:
    if isinstance(value, str):
        try:
            return datetime.fromisoformat(value.replace("Z", "+00:00"))
        except ValueError:
            return None
    if isinstance(value, datetime):
        return value
    return None


def _float_env(name: str, default: float) -> float:
    try:
        return float(os.getenv(name, str(default)) or default)
    except ValueError:
        return default


def _record_backtest_memory(request: BacktestRequest, metrics) -> None:
    try:
        mem = get_backtest_memory()
        metrics_dict = {
            "total_return": metrics.total_return,
            "annual_return": metrics.annual_return,
            "max_drawdown": metrics.max_drawdown,
            "sharpe_ratio": metrics.sharpe_ratio,
            "win_rate": metrics.win_rate,
            "profit_factor": metrics.profit_factor,
            "total_trades": metrics.total_trades,
            "winning_trades": metrics.winning_trades,
            "losing_trades": metrics.losing_trades,
            "average_profit": metrics.average_profit,
            "average_loss": metrics.average_loss,
        }
        mem.record(build_situation_key(request.symbol, request.timeframe, request.strategy_code, metrics_dict), build_advice_text(metrics_dict, request.strategy_code))
    except Exception as exc:
        logger.warning("failed to record backtest memory: %s", exc)
