import logging

from fastapi import APIRouter

from app.memory import build_advice_text, build_situation_key, get_backtest_memory
from app.schemas import MemoryEntry, MemoryQueryRequest, MemoryQueryResponse, MemoryRecordRequest, MemoryRecordResponse

logger = logging.getLogger(__name__)
router = APIRouter()


@router.post("/api/memory/record", response_model=MemoryRecordResponse)
async def memory_record(request: MemoryRecordRequest):
    try:
        mem = get_backtest_memory()
        situation = build_situation_key(request.symbol, request.timeframe, request.strategy_code, request.metrics)
        advice = build_advice_text(request.metrics, request.strategy_code)
        if request.extra_advice:
            advice += "\n" + request.extra_advice
        mem.record(situation, advice)
        return MemoryRecordResponse(success=True, total_entries=mem.size())
    except Exception as exc:
        logger.warning("memory record failed: %s", exc)
        return MemoryRecordResponse(success=False, total_entries=0)


@router.post("/api/memory/query", response_model=MemoryQueryResponse)
async def memory_query(request: MemoryQueryRequest):
    try:
        mem = get_backtest_memory()
        situation = build_situation_key(request.symbol, request.timeframe, request.strategy_code)
        results = mem.query(situation, n=request.n)
        return MemoryQueryResponse(memories=[MemoryEntry(situation=r["situation"], advice=r["advice"], score=r["score"]) for r in results])
    except Exception as exc:
        logger.warning("memory query failed: %s", exc)
        return MemoryQueryResponse(memories=[])
