"""ObjectiveScore REST endpoint superseded by ConnectRPC.
CalculateObjectiveScore: POST /ant.v1.ObjectiveScoreService/CalculateObjectiveScore (objective_score_connect.py)"""

from fastapi import APIRouter

router = APIRouter()
# The /api/objective-score REST endpoint has been migrated to ConnectRPC.
# See objective_score_connect.py for the ConnectRPC handler.
