"""ConnectRPC handler for PythonStrategyService.Execute / Validate.
Replaces the previous FastAPI /api/strategy/execute and /api/strategy/validate REST endpoints.
Protocol: POST /ant.v1.PythonStrategyService/{Execute,Validate}"""

import asyncio
import json
import logging
import os
import re
import time

import numpy as np
from fastapi import APIRouter, Request, Response
from google.protobuf.json_format import Parse

from app.python_strategy_pb2 import (
    CodeQualityHint,
    ExecuteStrategyRequest,
    ExecuteStrategyResponse,
    StrategyDirective,
    SweepDimension,
    ValidateStrategyRequest,
    ValidateStrategyResponse,
)
# StrategySignal is defined in strategy_signal_messages.proto
# (imported by python_strategy.proto via strategy_messages.proto)
from app.strategy_signal_messages_pb2 import StrategySignal
from app.engine.params_extractor import extract_required_params
from app.engine.sandbox import validate_strategy_code, build_sandbox_globals
from app.engine.types import StrategyRuntimeError

logger = logging.getLogger(__name__)
router = APIRouter()


def _execute_inline(code: str, context: dict) -> Optional[dict]:
    """Compile and execute strategy code in-process. Tries SDK first, then legacy."""
    if not code or not code.strip():
        raise StrategyRuntimeError("策略代码为空")
    try:
        exec_scope = build_sandbox_globals()
        exec(compile(code, "<strategy>", "exec"), exec_scope, exec_scope)
    except SyntaxError as e:
        raise StrategyRuntimeError(f"Python 编译失败: {e}") from e
    except Exception as e:
        raise StrategyRuntimeError(f"策略代码执行错误: {e}") from e

    # SDK strategy: find StrategyBase subclass, call on_init, return intents
    from app.sdk.strategy_base import StrategyBase
    for obj in exec_scope.values():
        if isinstance(obj, type) and issubclass(obj, StrategyBase) and obj is not StrategyBase:
            return {"signal": "hold", "comment": "SDK strategy compiled successfully"}

    # Legacy: def run(context)
    run_fn = exec_scope.get("run")
    if callable(run_fn):
        result = run_fn(dict(context))
        return result if isinstance(result, dict) else None

    # Legacy: signal = {...}
    if "signal" in exec_scope:
        return exec_scope["signal"] if isinstance(exec_scope["signal"], dict) else {"signal": str(exec_scope["signal"])}

    raise StrategyRuntimeError("策略代码必须定义 SDK 类或 signal 变量或 run(context) 函数")


async def _parse_request(request: Request, req_cls):
    """Parse request body as protobuf binary (ConnectRPC protocol)."""
    return req_cls.FromString(await request.body())


def _respond(proto_resp, _request: Request) -> Response:
    """Return protobuf binary response (ConnectRPC protocol)."""
    return Response(
        content=proto_resp.SerializeToString(),
        media_type="application/proto",
        headers={"Connect-Protocol-Version": "1"},
    )


# Regex for @param annotations with optional range: min:max:step.
_SWEEP_RE = re.compile(
    r"^\s*#\s*@param\s+(\w+)\s+([\d.]+)"
    r"(?:\s+range=([\d.]+):([\d.]+):([\d.]+))?",
    re.MULTILINE,
)
_STRATEGY_RE = re.compile(
    r"^\s*#\s*@strategy\s+(\w+)\s*:?\s*(\S+)",
    re.MULTILINE,
)


def _extract_sweep_dimensions(code: str) -> list[SweepDimension]:
    """Extract @param annotations with range info for the Smart Tuning panel.

    Backend zero-trust: the frontend MUST NOT parse @param itself.
    All computation happens here; frontend only renders.
    """
    dims: list[SweepDimension] = []
    seen: set[str] = set()
    for m in _SWEEP_RE.finditer(code or ""):
        key = m.group(1)
        if key in seen:
            continue
        seen.add(key)
        default = float(m.group(2))
        has_range = m.group(3) is not None
        if has_range:
            pmin = float(m.group(3))
            pmax = float(m.group(4))
            pstep = float(m.group(5))
            ptype = "int" if (pmin % 1 == 0 and pmax % 1 == 0 and pstep % 1 == 0) else "float"
        else:
            pmin = default
            pmax = default
            pstep = 0
            ptype = "float" if default % 1 != 0 else "int"
        dims.append(SweepDimension(
            key=key,
            type=ptype,
            default=default,
            min=pmin,
            max=pmax,
            step=pstep,
            has_range=has_range,
        ))
    return dims


def _extract_strategy_directives(code: str) -> list[StrategyDirective]:
    """Extract @strategy directives for the risk control display.

    Backend zero-trust: the frontend MUST NOT parse @strategy itself.
    """
    dirs: list[StrategyDirective] = []
    seen: set[str] = set()
    for m in _STRATEGY_RE.finditer(code or ""):
        key = m.group(1)
        if key in seen:
            continue
        seen.add(key)
        dirs.append(StrategyDirective(key=key, value=m.group(2)))
    return dirs


@router.post("/ant.v1.PythonStrategyService/Validate")
async def validate_strategy_connect(request: Request):
    """Validate strategy code syntax via ConnectRPC."""
    req = await _parse_request(request, ValidateStrategyRequest)
    try:
        result = validate_strategy_code(req.code or "")
        params = extract_required_params(req.code or "") if result.valid else []

        # Convert quality hints to proto messages.
        quality_hints = [
            CodeQualityHint(
                category=h.category,
                severity=h.severity,
                message=h.message,
                line=h.line,
                snippet=h.snippet,
            )
            for h in result.quality_hints
        ]

        # Extract @param sweep dimensions + @strategy directives
        # (backend zero-trust: frontend MUST NOT parse code itself).
        sweep_dims = _extract_sweep_dimensions(req.code or "")
        strategy_dirs = _extract_strategy_directives(req.code or "")

        # Auto-detect strategy type from code.
        from app.engine.vectorized_runner import detect_strategy_type
        strategy_type = detect_strategy_type(req.code or "")

        import json as _json
        resp = ValidateStrategyResponse(
            valid=result.valid,
            errors=list(result.errors),
            warnings=list(result.warnings),
            quality_hints=quality_hints,
            sweep_dimensions=sweep_dims,
            strategy_directives=strategy_dirs,
            strategy_type=strategy_type,
            parameters_json=_json.dumps(result.parameters or [], ensure_ascii=False),
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
        loop = asyncio.get_event_loop()
        signal_data = await asyncio.wait_for(
            loop.run_in_executor(None, _execute_inline, req.code or "", context),
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

# ── TranspileCode handler (RETIRED — use connectrpc_server.py) ───────
# The TranspileCode handler is now in connectrpc_server.py as _transpile().
# C1 (ADR-0020 D8): regex line-by-line fallback removed. AST-only with
# quality gate confidence (compiles + SDK imports + lint).
# See: connectrpc_server.py:_transpile
