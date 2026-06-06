"""ConnectRPC handler for ObjectiveScoreService.CalculateObjectiveScore.
Replaces the previous FastAPI /api/objective-score REST endpoint.
Protocol: POST /ant.v1.ObjectiveScoreService/CalculateObjectiveScore"""

import json
import logging

from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse
from google.protobuf.json_format import MessageToDict, Parse

from app.objective_score_pb2 import (
    ObjectiveScoreRequest as ProtoObjectiveScoreRequest,
    ObjectiveScoreResponse as ProtoObjectiveScoreResponse,
    ObjectiveSignals as ProtoObjectiveSignals,
    RSISignal as ProtoRSISignal,
    MACDSignal as ProtoMACDSignal,
    MASignal as ProtoMASignal,
)
from app.schemas import (
    KlineEntry,
    ObjectiveScoreRequest as PydanticObjectiveScoreRequest,
)
from app.services.objective_score import calculate_objective_score

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


@router.post("/ant.v1.ObjectiveScoreService/CalculateObjectiveScore")
async def objective_score_connect(request: Request):
    """Calculate objective strategy score via ConnectRPC."""
    proto_req = await _parse_request(request, ProtoObjectiveScoreRequest)

    # Convert proto → pydantic for the existing service layer
    pydantic_req = PydanticObjectiveScoreRequest(
        symbol=proto_req.symbol,
        timeframe=proto_req.timeframe,
        klines=[KlineEntry(
            open_time=k.open_time,
            close_time=k.close_time,
            open_price=k.open_price,
            high_price=k.high_price,
            low_price=k.low_price,
            close_price=k.close_price,
            volume=k.volume,
        ) for k in proto_req.klines],
    )

    py_resp = calculate_objective_score(pydantic_req)

    # Convert pydantic → proto response
    proto_signals = None
    if py_resp.signals:
        proto_signals = ProtoObjectiveSignals(
            rsi=ProtoRSISignal(value=py_resp.signals.rsi.value, signal=py_resp.signals.rsi.signal),
            macd=ProtoMACDSignal(
                value=py_resp.signals.macd.value,
                signal_line=py_resp.signals.macd.signal_line,
                histogram=py_resp.signals.macd.histogram,
                signal=py_resp.signals.macd.signal,
                trend=py_resp.signals.macd.trend,
            ),
            ma=ProtoMASignal(
                ma5=py_resp.signals.ma.ma5,
                ma10=py_resp.signals.ma.ma10,
                ma20=py_resp.signals.ma.ma20,
                trend=py_resp.signals.ma.trend,
            ),
        )

    proto_resp = ProtoObjectiveScoreResponse(
        decision=py_resp.decision,
        overall_score=py_resp.overall_score,
        technical_score=py_resp.technical_score,
    )
    if proto_signals:
        proto_resp.signals.CopyFrom(proto_signals)

    return _respond(proto_resp, request)
