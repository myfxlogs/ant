from typing import Optional

from fastapi import APIRouter

from app.schemas import ExternalMacroResponse, ExternalNewsResponse

router = APIRouter()


@router.get("/api/external/macro", response_model=ExternalMacroResponse)
async def get_external_macro(market: Optional[str] = None):
    return ExternalMacroResponse(data={
        "status": "reserved",
        "provider_status": "not_configured",
        "market": market or "",
        "intended_use": "future_financial_macro_data_for_ai_strategy_authoring",
        "items": [],
    })


@router.get("/api/external/news", response_model=ExternalNewsResponse)
async def get_external_news(market: Optional[str] = None, symbol: Optional[str] = None, limit: int = 20):
    return ExternalNewsResponse(
        items=[],
        status="reserved",
        provider_status="not_configured",
        market=market or "",
        symbol=symbol or "",
        intended_use="future_financial_news_data_for_ai_strategy_authoring",
    )
