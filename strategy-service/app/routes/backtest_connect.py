"""ConnectRPC handler for BacktestService.RunBacktest.
Replaces the previous FastAPI /api/backtest JSON endpoint.
Protocol: POST /ant.v1.BacktestService/RunBacktest."""

import asyncio
import json
import logging
import os
from datetime import datetime, timezone

from fastapi import APIRouter, Request
from fastapi.responses import JSONResponse
from google.protobuf.json_format import MessageToDict, Parse

from app.backtest_service_pb2 import (
    ExecuteBacktestRequest, ExecuteBacktestResponse,
    ExecuteBacktestMetrics, ExecuteBacktestTrade, ExecuteRiskAssessment,
    EngineValidateRequest, EngineValidateResponse,
    EngineRunStrategyRequest, EngineRunStrategyResponse, EngineTradeSignal,
)

from app.engine import (
    BacktestRequest as EngineBacktestRequest,
    Bar as EngineBar,
    run_backtest as engine_run_backtest,
)

logger = logging.getLogger(__name__)
router = APIRouter()
backtest_semaphore = asyncio.Semaphore(int(os.getenv("MAX_BACKTEST_WORKERS", "2")))


def _to_dt(ms: int) -> datetime:
    return datetime.fromtimestamp(ms / 1000.0, tz=timezone.utc)


@router.post("/ant.v1.BacktestService/RunBacktest")
async def run_backtest_connect(request: Request):
    """ConnectRPC handler: supports JSON + protobuf binary."""
    content_type = request.headers.get("content-type", "")
    req = ExecuteBacktestRequest()

    if "application/proto" in content_type or "application/grpc" in content_type:
        body = await request.body()
        req.ParseFromString(body)
    else:
        body = await request.json()
        Parse(json.dumps(body), req, ignore_unknown_fields=True)

    async with backtest_semaphore:
        engine_req = _build_engine_request(req)
        try:
            loop = asyncio.get_event_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
        result = await loop.run_in_executor(None, engine_run_backtest, engine_req)

        resp = ExecuteBacktestResponse(success=result.success)
        if not result.success:
            resp.error = result.error or "backtest failed"
            return JSONResponse(content=MessageToDict(resp, preserving_proto_field_name=True))

        m = result.metrics
        resp.metrics.CopyFrom(ExecuteBacktestMetrics(
            total_return=m.total_return, annual_return=m.annual_return,
            max_drawdown=m.max_drawdown, sharpe_ratio=m.sharpe_ratio,
            win_rate=m.win_rate, profit_factor=m.profit_factor,
            total_trades=m.total_trades, winning_trades=m.winning_trades,
            losing_trades=m.losing_trades, average_profit=m.average_profit,
            average_loss=m.average_loss,
        ))

        ra = result.risk_assessment
        resp.risk.CopyFrom(ExecuteRiskAssessment(
            score=ra.score, level=ra.level,
            reasons=ra.reasons or [], warnings=ra.warnings or [],
            is_reliable=ra.is_reliable,
        ))
        resp.equity_curve.extend(result.equity_curve or [])

        for t in (result.trades or []):
            resp.trades.add().CopyFrom(ExecuteBacktestTrade(
                ticket=t.ticket,
                side=str(t.side.value if hasattr(t.side, "value") else t.side),
                volume=t.volume, open_ts_ms=t.open_ts, open_price=t.open_price,
                close_ts_ms=t.close_ts, close_price=t.close_price,
                pnl=t.pnl, commission=t.commission,
                reason=str(t.reason.value if hasattr(t.reason, "value") else t.reason),
            ))

        return JSONResponse(content=MessageToDict(resp, preserving_proto_field_name=True))


def _build_engine_request(req: ExecuteBacktestRequest) -> EngineBacktestRequest:
    """Convert proto ExecuteBacktestRequest → engine BacktestRequest."""
    bars = [
        EngineBar(open_time=k.open_time_ms, close_time=k.close_time_ms,
                  open=k.open, high=k.high, low=k.low, close=k.close, volume=k.volume)
        for k in req.klines
    ]
    return EngineBacktestRequest(
        run_id=req.strategy_id or "",
        user_id=0, account_id=0,
        symbol=req.symbol or "XAUUSDm",
        timeframe=req.timeframe or "1h",
        start=_to_dt(req.start_date_ms) if req.start_date_ms else datetime(2024, 1, 1, tzinfo=timezone.utc),
        end=_to_dt(req.end_date_ms) if req.end_date_ms else datetime.now(timezone.utc),
        initial_cash=req.initial_capital or 10000.0,
        strategy_code=req.strategy_code or "",
        bars=bars,
        bars_by_symbol={s: bars for s in req.extra_symbols} if req.extra_symbols else {},
    )

# ── ValidateStrategy ──

@router.post("/ant.v1.BacktestService/ValidateStrategy")
async def validate_strategy_connect(request: Request):
    """Validate strategy code syntax."""
    from app.engine.sandbox import validate_strategy_code

    req = EngineValidateRequest()
    ct = request.headers.get("content-type", "")
    if "application/proto" in ct:
        req.ParseFromString(await request.body())
    else:
        body = await request.json()
        Parse(json.dumps(body), req, ignore_unknown_fields=True)

    result = validate_strategy_code(req.strategy_code or "")
    resp = EngineValidateResponse(
        valid=result.valid,
        errors=result.errors or [],
        warnings=result.warnings or [],
    )
    return JSONResponse(content=MessageToDict(resp, preserving_proto_field_name=True))


# ── RunStrategy (live/paper execution) ──

@router.post("/ant.v1.BacktestService/RunStrategy")
async def run_strategy_connect(request: Request):
    """Execute strategy on live/paper market data."""
    from app.engine import (
        Bar as EngineBar,
        run_strategy as engine_run_strategy,
    )

    req = EngineRunStrategyRequest()
    ct = request.headers.get("content-type", "")
    if "application/proto" in ct:
        req.ParseFromString(await request.body())
    else:
        body = await request.json()
        Parse(json.dumps(body), req, ignore_unknown_fields=True)

    bars = [
        EngineBar(open_time=k.open_time_ms, close_time=k.close_time_ms,
                  open=k.open, high=k.high, low=k.low, close=k.close, volume=k.volume)
        for k in req.klines
    ]

    engine_req = EngineBacktestRequest(
        run_id=req.strategy_id or "",
        user_id=0, account_id=0,
        symbol=req.symbol or "XAUUSDm",
        timeframe=req.timeframe or "1h",
        start=datetime(2024, 1, 1, tzinfo=timezone.utc),
        end=datetime.now(timezone.utc),
        initial_cash=10000.0,
        strategy_code=req.strategy_code or "",
        bars=bars,
    )

    try:
        loop = asyncio.get_event_loop()
    except RuntimeError:
        loop = asyncio.new_event_loop()
    result = await loop.run_in_executor(None, engine_run_strategy, engine_req)

    resp = EngineRunStrategyResponse(success=result.success)
    if result.signal:
        resp.signal.CopyFrom(EngineTradeSignal(
            signal=result.signal.get("signal", "hold"),
            symbol=result.signal.get("symbol", ""),
            price=result.signal.get("price", 0.0),
            volume=result.signal.get("volume", 1.0),
            stop_loss=result.signal.get("stop_loss", 0.0),
            take_profit=result.signal.get("take_profit", 0.0),
            confidence=result.signal.get("confidence", 0.0),
            reason=result.signal.get("reason", ""),
        ))
    if result.error:
        resp.error = result.error

    return JSONResponse(content=MessageToDict(resp, preserving_proto_field_name=True))
