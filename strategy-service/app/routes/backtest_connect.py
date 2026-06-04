"""ConnectRPC handler for BacktestService.RunBacktest.
Replaces the previous FastAPI /api/backtest JSON endpoint.
Protocol: POST /ant.v1.BacktestService/RunBacktest — JSON body → proto → engine → proto → JSON."""

import asyncio
import logging
import os
import sys
from datetime import datetime

from fastapi import APIRouter, Request, Response
from google.protobuf.json_format import Parse, MessageToDict
from google.protobuf.timestamp_pb2 import Timestamp

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "gen", "proto"))
from backtest_service_pb2 import (
    ExecuteBacktestRequest,
    ExecuteBacktestResponse,
    ExecuteBacktestMetrics,
    ExecuteBacktestTrade,
    ExecuteRiskAssessment,
)

from app.engine import (
    BacktestRequest as EngineBacktestRequest,
    Bar as EngineBar,
    CostProfile as EngineCostProfile,
    SlippageMode as EngineSlippageMode,
    run_backtest as engine_run_backtest,
)

logger = logging.getLogger(__name__)
router = APIRouter()
backtest_semaphore = asyncio.Semaphore(int(os.getenv("MAX_BACKTEST_WORKERS", "2")))


def _msg_to_dict(msg) -> dict:
    """Convert protobuf message to JSON-serializable dict, handling proto3 defaults."""
    return MessageToDict(msg, preserving_proto_field_name=True)


@router.post("/ant.v1.BacktestService/RunBacktest")
async def run_backtest_connect(request: Request):
    """ConnectRPC handler: JSON→proto→engine→proto→JSON."""
    body = await request.json()

    # Parse JSON body → proto ExecuteBacktestRequest
    req = ExecuteBacktestRequest()
    Parse(request.json(), req, ignore_unknown_fields=True)

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
            return Response(
                content=resp.SerializeToString(),
                media_type="application/proto",
                headers={"Content-Type": "application/json"},
            )

        # Build proto response from engine result
        m = result.metrics
        resp.metrics.CopyFrom(ExecuteBacktestMetrics(
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
        ))

        ra = result.risk_assessment
        resp.risk.CopyFrom(ExecuteRiskAssessment(
            score=ra.score,
            level=ra.level,
            reasons=ra.reasons or [],
            warnings=ra.warnings or [],
            is_reliable=ra.is_reliable,
        ))

        resp.equity_curve.extend(result.equity_curve or [])

        for t in (result.trades or []):
            resp.trades.add().CopyFrom(ExecuteBacktestTrade(
                ticket=t.ticket,
                side=str(t.side.value if hasattr(t.side, "value") else t.side),
                volume=t.volume,
                open_ts_ms=t.open_ts,
                open_price=t.open_price,
                close_ts_ms=t.close_ts,
                close_price=t.close_price,
                pnl=t.pnl,
                commission=t.commission,
                reason=str(t.reason.value if hasattr(t.reason, "value") else t.reason),
            ))

        # Return JSON (ConnectRPC protocol: JSON content type)
        resp_json = _msg_to_dict(resp)
        from fastapi.responses import JSONResponse
        return JSONResponse(content=resp_json)


def _build_engine_request(req: ExecuteBacktestRequest) -> EngineBacktestRequest:
    """Convert proto ExecuteBacktestRequest → engine BacktestRequest."""
    def _to_bars(klines) -> list:
        return [
            EngineBar(
                open_time=k.open_time_ms,
                close_time=k.close_time_ms,
                open=k.open,
                high=k.high,
                low=k.low,
                close=k.close,
                volume=k.volume,
            )
            for k in klines
        ]

    bars = _to_bars(req.klines)
    bars_by_symbol: dict = {}
    for sym in req.extra_symbols:
        bars_by_symbol[sym] = bars

    return EngineBacktestRequest(
        strategy_id=req.strategy_id or "",
        strategy_code=req.strategy_code or "",
        symbol=req.symbol or "",
        timeframe=req.timeframe or "1h",
        start_date_ms=req.start_date_ms or 0,
        end_date_ms=req.end_date_ms or 0,
        initial_capital=req.initial_capital or 10000.0,
        commission=req.commission,
        spread=req.spread,
        swap_rate=req.swap_rate,
        server_timezone=req.server_timezone or "UTC",
        rollover_hour=req.rollover_hour,
        triple_swap_weekday=req.triple_swap_weekday or 3,
        slippage_mode=EngineSlippageMode(req.slippage_mode or "fixed"),
        slippage_rate=req.slippage_rate,
        slippage_seed=req.slippage_seed,
        bars=bars,
        bars_by_symbol=bars_by_symbol,
        ticks=[],
    )
