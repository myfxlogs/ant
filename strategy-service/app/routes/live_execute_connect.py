"""ConnectRPC handler for PythonStrategyService.ExecuteLive.

Pure SDK path (ADR-0020): all strategies run through StrategyRuntime → LiveBroker.
No RestrictedPython. No def run(context). No signal dict. No LiveWorker pool.
"""

from __future__ import annotations

import asyncio
import hashlib
import logging
import time
from typing import Dict

import numpy as np
from fastapi import APIRouter, Request, Response
from google.protobuf.json_format import MessageToDict

from app.engine.sdk_worker import process_bar
from app.python_strategy_pb2 import ExecuteLiveRequest, ExecuteLiveResponse
from app.strategy_signal_messages_pb2 import StrategySignal

logger = logging.getLogger(__name__)
router = APIRouter()


def _strategy_hash(code: str) -> str:
    return hashlib.sha256(code.encode()).hexdigest()[:16]


def _proto_context_to_dict(ctx) -> dict:
    """Convert proto LiveStrategyContext to a plain dict for the SDK worker."""
    d = MessageToDict(ctx, preserving_proto_field_name=True)
    # Flatten nested structures.
    if "params" in d:
        d["params"] = [{"key": p.get("key", ""), "value": p.get("value", "")} for p in d["params"]]
    if "positions" in d:
        d["positions"] = d["positions"]
    return d


def _build_signals_from_intents(intents: list) -> list:
    """Convert all SDK intents to StrategySignals."""
    signals = []
    for intent in intents:
        action = intent.get("action", "hold")
        if not action or action == "hold":
            continue
        volume = float(intent.get("volume", 0))
        sl = float(intent.get("sl", 0))
        tp = float(intent.get("tp", 0))
        price = float(intent.get("price", 0))
        ticket_str = str(intent.get("ticket", "0")).strip()
        try:
            ticket = int(ticket_str) if ticket_str else 0
        except ValueError:
            ticket = 0
        signals.append(StrategySignal(
            signal_type=action,
            volume=volume,
            stop_loss=sl,
            take_profit=tp,
            price=price,
            executed_ticket=ticket,
        ))
    return signals


@router.post("/ant.v1.PythonStrategyService/ExecuteLive")
async def execute_live(request: Request):
    """Execute strategy code via SDK path — pure StrategyRuntime."""
    req_bytes = await request.body()
    req = ExecuteLiveRequest()
    req.ParseFromString(req_bytes)

    if not req.strategy_code:
        return _respond(ExecuteLiveResponse(success=False, error="strategy_code is required"))

    code_hash = _strategy_hash(req.strategy_code)
    t0 = time.time()

    try:
        context = _proto_context_to_dict(req.context) if req.HasField("context") else {}
        result = process_bar(req.strategy_code, context)

        if result.get("error"):
            return _respond(
                ExecuteLiveResponse(success=False, error=result["error"], strategy_hash=code_hash))

        intents = result.get("intents", [])
        signals = _build_signals_from_intents(intents)

        elapsed = (time.time() - t0) * 1000
        if elapsed > 200:
            logger.warning("ExecuteLive: slow %.0fms hash=%s intents=%d", elapsed, code_hash, len(intents))

        # Set first signal for backward compat, repeated for multi-intent
        first_sig = signals[0] if signals else StrategySignal(signal_type="hold")
        return _respond(ExecuteLiveResponse(
            success=True, signal=first_sig, strategy_hash=code_hash,
            signals=signals))

    except Exception as e:
        logger.exception("ExecuteLive: error hash=%s", code_hash)
        return _respond(ExecuteLiveResponse(success=False, error=str(e), strategy_hash=code_hash))


# ── Response helpers ──────────────────────────────────────────────────

def _respond(proto_resp, _request: Request = None) -> Response:
    """Serialize proto response to binary."""
    return Response(
        content=proto_resp.SerializeToString(),
        media_type="application/proto",
    )
