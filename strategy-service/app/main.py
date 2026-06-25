#!/usr/bin/env python3
"""AntTrader Strategy Service — ConnectRPC + protobuf only. Zero JSON, zero FastAPI."""

import os
from datetime import datetime

from dotenv import load_dotenv
from starlette.applications import Starlette
from starlette.responses import Response as StarletteResponse
from starlette.routing import Route

from app.connectrpc_server import create_app
from app.memory import get_backtest_memory

load_dotenv()

HOST = os.getenv('HOST', '0.0.0.0')
PORT = int(os.getenv('PORT', '8081'))
DEBUG = os.getenv('DEBUG', 'false').lower() == 'true'

connectrpc_app = create_app()


async def health_check(_request):
    return StarletteResponse(
        content='{"status":"healthy"}',
        media_type="application/json",
    )


async def startup():
    get_backtest_memory()


app = Starlette(
    debug=DEBUG,
    routes=[
        Route("/health", health_check, methods=["GET"]),
        Route("/healthz", health_check, methods=["GET"]),
    ],
    on_startup=[startup],
    middleware=[],
)

# Mount the ConnectRPC ASGI app directly onto the Starlette app.
# ConnectASGIApplication is an ASGI3 app, so it works as middleware.
app.mount("/", connectrpc_app)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host=HOST, port=PORT, reload=DEBUG)
