from fastapi import APIRouter

from app.schemas import ObjectiveScoreRequest, ObjectiveScoreResponse
from app.services.objective_score import calculate_objective_score

router = APIRouter()


@router.post("/api/objective-score", response_model=ObjectiveScoreResponse)
async def objective_score(req: ObjectiveScoreRequest):
    return calculate_objective_score(req)
