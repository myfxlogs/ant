"""ConnectRPC handler for PythonStrategyService.Execute / Validate.
Replaces the previous FastAPI /api/strategy/execute and /api/strategy/validate REST endpoints.
Protocol: POST /ant.v1.PythonStrategyService/{Execute,Validate}"""

import asyncio
import json
import logging
import os
import time

import numpy as np
from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse
from google.protobuf.json_format import MessageToDict, Parse

from app.python_strategy_pb2 import (
    ExecuteStrategyRequest, ExecuteStrategyResponse, StrategySignal,
    ValidateStrategyRequest, ValidateStrategyResponse,
)
from app.engine.params_extractor import extract_required_params
from app.engine.sandbox import StrategyRunner, StrategyRuntimeError, validate_strategy_code

logger = logging.getLogger(__name__)
router = APIRouter()


async def _parse_request(request: Request, req_cls):
    """Parse request body as proto binary or JSON based on Content-Type."""
    ct = request.headers.get("content-type", "")
    if "application/proto" in ct or "application/grpc" in ct:
        return req_cls.FromString(await request.body())
    body = await request.json()
    return Parse(json.dumps(body), req_cls(), ignore_unknown_fields=True)


def _respond(proto_resp, request: Request) -> Response:
    """Return proto binary with ConnectRPC headers."""
    ct = request.headers.get("content-type", "")
    if "application/proto" in ct or "application/grpc" in ct:
        return Response(
            content=proto_resp.SerializeToString(),
            media_type="application/proto",
            headers={"Connect-Protocol-Version": "1"},
        )
    return JSONResponse(
        content=MessageToDict(proto_resp, preserving_proto_field_name=True),
        headers={"Connect-Protocol-Version": "1"},
    )


@router.post("/ant.v1.PythonStrategyService/Validate")
async def validate_strategy_connect(request: Request):
    """Validate strategy code syntax via ConnectRPC."""
    req = await _parse_request(request, ValidateStrategyRequest)
    try:
        result = validate_strategy_code(req.code or "")
        params = extract_required_params(req.code or "") if result.valid else []
        resp = ValidateStrategyResponse(
            valid=result.valid,
            errors=list(result.errors),
            warnings=list(result.warnings),
        )
    except Exception as e:
        resp = ValidateStrategyResponse(valid=False, errors=[f"验证错误: {e}"])
    return _respond(resp, request)


@router.post("/ant.v1.PythonStrategyService/Execute")
async def execute_strategy_connect(request: Request):
    """Execute strategy code on paper/live market data via ConnectRPC."""
    req = await _parse_request(request, ExecuteStrategyRequest)
    start_time = time.time()

    try:
        # Build context from proto request — market data is fetched separately
        # by the Go handler. When no klines are provided, use an empty context
        # and let the strategy code handle the graceful degradation.
        context = {
            "symbol": req.symbol or "",
            "timeframe": req.timeframe or "1h",
            "close": np.array([]),
            "open": np.array([]),
            "high": np.array([]),
            "low": np.array([]),
            "volume": np.array([]),
        }

        timeout_seconds = int(os.getenv("BACKTEST_TIMEOUT", "120"))
        runner = StrategyRunner(req.code or "", timeout_ms=timeout_seconds * 1000)
        loop = asyncio.get_event_loop()
        signal_data = await asyncio.wait_for(
            loop.run_in_executor(None, runner.call, context),
            timeout=timeout_seconds + 5,
        )
        elapsed = (time.time() - start_time) * 1000

        if signal_data is None or not isinstance(signal_data, dict):
            return _respond(ExecuteStrategyResponse(
                success=False, error="策略未返回有效信号"), request)

        action = str(signal_data.get("signal", "hold")).strip().lower()
        allowed = {"buy", "sell", "hold", "close", "buy_limit", "sell_limit",
                    "buy_stop", "sell_stop", "buy_stop_limit", "sell_stop_limit",
                    "cancel_pending"}
        if action not in allowed:
            return _respond(ExecuteStrategyResponse(
                success=False, error="signal 字段不支持"), request)

        signal = StrategySignal(
            signal_type=action,
            volume=float(signal_data.get("volume", 0.0)),
            price=float(signal_data.get("price", 0.0)),
            stop_loss=float(signal_data.get("stop_loss", 0.0)),
            take_profit=float(signal_data.get("take_profit", 0.0)),
            reason=signal_data.get("reason", ""),
        )
        resp = ExecuteStrategyResponse(success=True, signal=signal)
    except StrategyRuntimeError as e:
        resp = ExecuteStrategyResponse(success=False, error=str(e))
    except Exception as e:
        resp = ExecuteStrategyResponse(success=False, error=f"服务器错误: {e}")

    return _respond(resp, request)
