#!/usr/bin/env python3
"""AntTrader Strategy Service — ConnectRPC only."""

import os
from dotenv import load_dotenv
from app.connectrpc_server import create_app
from app.memory import get_backtest_memory

load_dotenv()

HOST = os.getenv('HOST', '0.0.0.0')
PORT = int(os.getenv('PORT', '8081'))
DEBUG = os.getenv('DEBUG', 'false').lower() == 'true'

_connectrpc = create_app()


class _HealthWrapper:
    """ASGI3 app that handles /health and delegates everything else to ConnectRPC."""

    def __init__(self, connectrpc_app):
        self._connectrpc = connectrpc_app

    async def __call__(self, scope, receive, send):
        if scope["type"] == "http" and scope["path"] in ("/health", "/healthz"):
            await send({
                "type": "http.response.start",
                "status": 200,
                "headers": [(b"content-type", b"application/json")],
            })
            await send({"type": "http.response.body", "body": b'{"status":"healthy"}'})
            return
        await self._connectrpc(scope, receive, send)


app = _HealthWrapper(_connectrpc)

get_backtest_memory()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host=HOST, port=PORT, reload=DEBUG)
