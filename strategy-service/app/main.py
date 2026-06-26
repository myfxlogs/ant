#!/usr/bin/env python3
"""AntTrader Strategy Service — ConnectRPC + raw HTTP for strategy import."""

import json
import os
from dotenv import load_dotenv
from app.connectrpc_server import create_app
from app.memory import get_backtest_memory

load_dotenv()

HOST = os.getenv('HOST', '0.0.0.0')
PORT = int(os.getenv('PORT', '8081'))
DEBUG = os.getenv('DEBUG', 'false').lower() == 'true'

_connectrpc = create_app()

# Import API routes (raw JSON HTTP, not ConnectRPC)
_IMPORT_ROUTES = {
    "/api/strategy/import/analyze": "analyze",
    "/api/strategy/import/generate": "generate",
    "/api/strategy/import/import": "import",
}


class _HealthWrapper:
    """ASGI3 app: /health, /api/strategy/import/*, else → ConnectRPC."""

    def __init__(self, connectrpc_app):
        self._connectrpc = connectrpc_app

    async def __call__(self, scope, receive, send):
        if scope["type"] != "http":
            await self._connectrpc(scope, receive, send)
            return

        path = scope["path"]

        if path in ("/health", "/healthz"):
            await self._json_response(send, 200, {"status": "healthy"})
            return

        if path in _IMPORT_ROUTES:
            await self._handle_import(path, scope, receive, send)
            return

        await self._connectrpc(scope, receive, send)

    async def _handle_import(self, path, scope, receive, send):
        try:
            body = await self._read_body(receive)
            data = json.loads(body) if body else {}
        except json.JSONDecodeError:
            await self._json_response(send, 400, {"error": "Invalid JSON"})
            return

        try:
            from app.routes.strategy_import_connect import (
                _analyze_code, _generate_code, _import_strategy,
            )
            action = _IMPORT_ROUTES[path]

            if action == "analyze":
                req = _FakeReq(data)
                resp = await _analyze_code(req, None)
                result = _dataclass_to_dict(resp)
            elif action == "generate":
                req = _FakeReq(data)
                resp = await _generate_code(req, None)
                result = _dataclass_to_dict(resp)
            elif action == "import":
                req = _FakeReq(data)
                resp = await _import_strategy(req, None)
                result = _dataclass_to_dict(resp)
            else:
                result = {"error": "Unknown action"}
            await self._json_response(send, 200, result)
        except Exception as e:
            await self._json_response(send, 500, {"error": str(e)})

    @staticmethod
    async def _read_body(receive) -> bytes:
        body = b""
        more_body = True
        while more_body:
            message = await receive()
            if message["type"] == "http.request":
                body += message.get("body", b"")
                more_body = message.get("more_body", False)
        return body

    @staticmethod
    async def _json_response(send, status: int, data: dict):
        body = json.dumps(data, ensure_ascii=False, default=str).encode()
        await send({
            "type": "http.response.start",
            "status": status,
            "headers": [
                (b"content-type", b"application/json"),
                (b"access-control-allow-origin", b"*"),
            ],
        })
        await send({"type": "http.response.body", "body": body})


class _FakeReq:
    """Mimics protobuf request with attribute access."""
    def __init__(self, data: dict):
        self.source_code = data.get("sourceCode", data.get("source_code", ""))
        self.source_name = data.get("sourceName", data.get("source_name", "untitled.mq4"))
        self.source_lang = data.get("sourceLang", data.get("source_lang", "mql4"))


def _dataclass_to_dict(obj) -> dict:
    """Convert a dataclass response to camelCase dict for JSON."""
    import dataclasses
    if dataclasses.is_dataclass(obj):
        result = {}
        for f in dataclasses.fields(obj):
            value = getattr(obj, f.name)
            if dataclasses.is_dataclass(value):
                value = _dataclass_to_dict(value)
            elif isinstance(value, list):
                value = [_dataclass_to_dict(v) if dataclasses.is_dataclass(v) else v for v in value]
            # Convert snake_case to camelCase
            parts = f.name.split("_")
            key = parts[0] + "".join(p.capitalize() for p in parts[1:])
            result[key] = value
        return result
    return obj


app = _HealthWrapper(_connectrpc)

get_backtest_memory()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host=HOST, port=PORT, reload=DEBUG)
