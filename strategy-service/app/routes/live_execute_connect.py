"""ConnectRPC handler for PythonStrategyService.ExecuteLive.

Maintains a LiveWorker pool (strategy_hash → LiveWorker) so that per-bar
strategy calls reuse a pre-compiled, long-lived sandbox process.  This is the
runtime counterpart to the one-shot Execute RPC — designed for <100ms per-bar
latency in live/paper trading.

Protocol: POST /ant.v1.PythonStrategyService/ExecuteLive
"""

from __future__ import annotations

import asyncio
import hashlib
import json
import logging
import os
import time
from typing import Dict

import numpy as np
from fastapi import APIRouter, Request, Response
from google.protobuf.json_format import MessageToDict, Parse

from app.python_strategy_pb2 import (
    ExecuteLiveRequest,
    ExecuteLiveResponse,
)
from app.strategy_signal_messages_pb2 import StrategySignal
from app.engine.live_sandbox import LiveWorker
from app.engine.sandbox import StrategyRuntimeError, validate_strategy_code
from app.engine.types import StrategyCompileError

logger = logging.getLogger(__name__)
router = APIRouter()

# ── LiveWorker pool: strategy_hash → LiveWorker ──
# Workers are created on first call and reused until the hash changes
# (code edit) or the process dies (auto-restart on next call).
_worker_pool: Dict[str, LiveWorker] = {}

# Max pool size — evict least-recently-used when exceeded.
_MAX_POOL_SIZE = int(os.getenv("LIVE_WORKER_POOL_SIZE", "16"))
_worker_last_used: Dict[str, float] = {}


def _strategy_hash(code: str) -> str:
    """SHA256 digest of the strategy code, used as pool key."""
    return hashlib.sha256(code.encode()).hexdigest()[:16]


def _evict_lru_if_needed() -> None:
    """Evict the least-recently-used worker when pool is full."""
    if len(_worker_pool) <= _MAX_POOL_SIZE:
        return
    lru_key = min(_worker_last_used, key=_worker_last_used.get)
    worker = _worker_pool.pop(lru_key, None)
    _worker_last_used.pop(lru_key, None)
    if worker is not None:
        try:
            worker.shutdown()
        except Exception:
            pass
    logger.info("LiveWorker pool: evicted LRU %s (size=%d)", lru_key, len(_worker_pool))


# ── Context conversion: proto LiveStrategyContext → dict ──


def _proto_context_to_dict(ctx) -> dict:
    """Convert proto-native LiveStrategyContext to the dict expected by run(context).

    Strategy code receives the same dict shape as backtest mode:
        context['close'] = np.ndarray (float64)
        context['position'] = dict or None
        context['mode'] = 'live' | 'paper'
        ...etc.
    """
    result: dict = {
        "symbol": ctx.symbol or "",
        "timeframe": ctx.timeframe or "1h",
        "mode": ctx.mode or "live",
        "current_price": ctx.current_price or 0.0,
        # OHLCV as numpy arrays
        "close": np.array(list(ctx.close), dtype=np.float64) if ctx.close else np.array([]),
        "open": np.array(list(ctx.open), dtype=np.float64) if ctx.open else np.array([]),
        "high": np.array(list(ctx.high), dtype=np.float64) if ctx.high else np.array([]),
        "low": np.array(list(ctx.low), dtype=np.float64) if ctx.low else np.array([]),
        "volume": np.array(list(ctx.volume), dtype=np.float64) if ctx.volume else np.array([]),
        "bar_times_ms": np.array(list(ctx.bar_times_ms), dtype=np.int64) if ctx.bar_times_ms else np.array([]),
        # Account state
        "equity": ctx.equity or 0.0,
        "cash": ctx.balance or 0.0,
        "account_balance": ctx.balance or 0.0,
        "account_equity": ctx.equity or 0.0,
        "params": {p.key: p.value for p in ctx.params} if ctx.params else {},
        # Position(s)
        "position": _proto_position_to_dict(ctx.position) if ctx.HasField("position") else None,
        "positions": [_proto_position_to_dict(p) for p in ctx.positions] if ctx.positions else [],
        "positions_total": len(ctx.positions) if ctx.positions else 0,
    }
    # Multi-symbol (Phase B2)
    if ctx.symbols:
        result["symbols"] = [s.symbol for s in ctx.symbols]
        for s in ctx.symbols:
            sym = s.symbol
            result[f"close_{sym}"] = np.array(list(s.close), dtype=np.float64)
            result[f"open_{sym}"] = np.array(list(s.open), dtype=np.float64)
            result[f"high_{sym}"] = np.array(list(s.high), dtype=np.float64)
            result[f"low_{sym}"] = np.array(list(s.low), dtype=np.float64)
            result[f"volume_{sym}"] = np.array(list(s.volume), dtype=np.float64)
    return result


def _proto_position_to_dict(pos) -> dict | None:
    """Convert LivePosition proto to context-compatible position dict."""
    if pos is None:
        return None
    return {
        "ticket": pos.ticket,
        "side": pos.side,
        "volume": pos.volume,
        "open_price": pos.open_price,
        "sl": pos.sl,
        "tp": pos.tp,
        "swap": pos.swap,
        "commission": pos.commission,
    }


# ── Request parsing (reuse pattern from strategy_connect.py) ──


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
    return Response(
        content=json.dumps(
            MessageToDict(proto_resp, preserving_proto_field_name=True)
        ).encode(),
        media_type="application/json",
        headers={"Connect-Protocol-Version": "1"},
    )


# ── RPC handler ──


@router.post("/ant.v1.PythonStrategyService/ExecuteLive")
async def execute_live(request: Request):
    """Execute strategy code with proto-native context — live/paper mode.

    Maintains a LiveWorker pool keyed by strategy SHA256.  First call for a new
    code hash spawns a persistent subprocess; subsequent calls reuse it.
    """
    req = await _parse_request(request, ExecuteLiveRequest)
    start_time = time.time()

    if not req.strategy_code:
        return _respond(ExecuteLiveResponse(success=False, error="strategy_code is required"), request)

    code_hash = _strategy_hash(req.strategy_code)

    # ── SDK strategy path (ADR-0020, pure StrategyRuntime) ──
    if _is_sdk_strategy_code(req.strategy_code):
        return await _execute_sdk(req, request)

    # ── Legacy signal-dict path ──
    try:
        # Validate code before compiling (only on first use for this hash).
        if code_hash not in _worker_pool:
            validation = validate_strategy_code(req.strategy_code)
            if not validation.valid:
                return _respond(
                    ExecuteLiveResponse(
                        success=False,
                        error="; ".join(validation.errors),
                        strategy_hash=code_hash,
                    ),
                    request,
                )
            # Create LiveWorker (spawns subprocess, pre-compiles bytecode).
            _evict_lru_if_needed()
            timeout_ms = int(os.getenv("LIVE_WORKER_TIMEOUT_MS", "5000"))
            _worker_pool[code_hash] = LiveWorker(req.strategy_code, timeout_ms=timeout_ms)
            logger.info("LiveWorker pool: created worker for hash %s (size=%d)", code_hash, len(_worker_pool))

        _worker_last_used[code_hash] = time.time()
        worker = _worker_pool[code_hash]

        # Convert proto context → dict → call strategy.
        context = _proto_context_to_dict(req.context) if req.HasField("context") else {}
        loop = asyncio.get_event_loop()
        signal_data = await asyncio.wait_for(
            loop.run_in_executor(None, worker.call, context),
            timeout=10.0,  # per-bar timeout
        )

        elapsed = (time.time() - start_time) * 1000
        if elapsed > 200:
            logger.warning("ExecuteLive: slow call %.0fms hash=%s", elapsed, code_hash)

        if signal_data is None or not isinstance(signal_data, dict):
            return _respond(
                ExecuteLiveResponse(success=False, error="strategy returned no signal", strategy_hash=code_hash),
                request,
            )

        action = str(signal_data.get("signal", "hold")).strip().lower()
        allowed = {
            "buy", "sell", "hold", "close", "buy_limit", "sell_limit",
            "buy_stop", "sell_stop", "buy_stop_limit", "sell_stop_limit",
            "cancel_pending",
        }
        if action not in allowed:
            return _respond(
                ExecuteLiveResponse(success=False, error=f"invalid signal: {action}", strategy_hash=code_hash),
                request,
            )

        signal = StrategySignal(
            signal_type=action,
            volume=float(signal_data.get("volume", 0.0)),
            price=float(signal_data.get("price", 0.0)),
            stop_loss=float(signal_data.get("stop_loss", 0.0)),
            take_profit=float(signal_data.get("take_profit", 0.0)),
            reason=str(signal_data.get("reason", "")),
        )
        return _respond(
            ExecuteLiveResponse(success=True, signal=signal, strategy_hash=code_hash),
            request,
        )

    except (StrategyCompileError, StrategyRuntimeError) as e:
        # Remove failed worker from pool — next call will re-create.
        dead = _worker_pool.pop(code_hash, None)
        _worker_last_used.pop(code_hash, None)
        if dead is not None:
            try:
                dead.shutdown()
            except Exception:
                pass
        return _respond(
            ExecuteLiveResponse(success=False, error=str(e), strategy_hash=code_hash),
            request,
        )
    except Exception as e:
        logger.exception("ExecuteLive: unexpected error hash=%s", code_hash)
        return _respond(
            ExecuteLiveResponse(success=False, error=f"live execution error: {e}", strategy_hash=code_hash),
            request,
        )


# ── SDK strategy path (ADR-0020) ──────────────────────────────────────

def _is_sdk_strategy_code(code: str) -> bool:
    """Detect SDK-format strategies: class X(StrategyBase)."""
    return "StrategyBase" in code and "class " in code


async def _execute_sdk(req, request: Request) -> Response:
    """Execute an SDK strategy via StrategyRuntime → LiveBroker → intents."""
    import time as _time
    from app.engine.sdk_worker import process_bar, reset_runtime

    t0 = _time.time()
    code_hash = _strategy_hash(req.strategy_code)

    try:
        context = _proto_context_to_dict(req.context) if req.HasField("context") else {}
        result = process_bar(req.strategy_code, context)

        if result.get("error"):
            return _respond(
                ExecuteLiveResponse(success=False, error=result["error"], strategy_hash=code_hash),
                request,
            )

        intents = result.get("intents", [])
        if not intents:
            return _respond(
                ExecuteLiveResponse(success=True, strategy_hash=code_hash,
                    signal=_build_signal_from_intents([])),
                request,
            )

        sig = _build_signal_from_intents(intents)
        elapsed = (_time.time() - t0) * 1000
        if elapsed > 200:
            logger.warning("ExecuteLive(SDK): slow %.0fms hash=%s intents=%d", elapsed, code_hash, len(intents))

        return _respond(
            ExecuteLiveResponse(success=True, signal=sig, strategy_hash=code_hash),
            request,
        )

    except Exception as e:
        logger.exception("ExecuteLive(SDK): error hash=%s", code_hash)
        return _respond(
            ExecuteLiveResponse(success=False, error=str(e), strategy_hash=code_hash),
            request,
        )


def _build_signal_from_intents(intents: list):
    """Convert SDK intents list to a StrategySignal proto.

    The first intent becomes the primary signal; additional intents
    are logged but only one signal is returned per bar (Go dispatches it).
    For multi-intent bars, intents after the first are queued for next bar.
    """
    from app.strategy_signal_messages_pb2 import StrategySignal

    if not intents:
        return StrategySignal(signal_type="hold")

    first = intents[0]
    action = first.get("action", "hold")
    volume = float(first.get("volume", 0))
    sl = float(first.get("sl", 0))
    tp = float(first.get("tp", 0))
    price = float(first.get("price", 0))
    ticket = int(first.get("ticket", 0)) if first.get("ticket", "").lstrip("-").isdigit() else 0

    return StrategySignal(
        signal_type=action,
        volume=volume,
        stop_loss=sl,
        take_profit=tp,
        price=price,
        executed_ticket=ticket,
    )
