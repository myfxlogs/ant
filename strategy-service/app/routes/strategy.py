"""ConnectRPC handler stubs — REST endpoints migrated to ConnectRPC.
Execute: POST /ant.v1.PythonStrategyService/Execute (strategy_connect.py)
Validate: POST /ant.v1.PythonStrategyService/Validate (strategy_connect.py)"""

from fastapi import APIRouter

router = APIRouter()
# All strategy endpoints have been migrated to ConnectRPC (see strategy_connect.py).
# This router is retained for future REST endpoints if needed.
