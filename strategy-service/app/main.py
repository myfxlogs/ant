#!/usr/bin/env python3
import os
from datetime import datetime

from dotenv import load_dotenv
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.memory import get_backtest_memory
from app.routes import backtest, external, memory, objective_score, strategy

load_dotenv()

HOST = os.getenv('HOST', '0.0.0.0')
PORT = int(os.getenv('PORT', '8081'))
DEBUG = os.getenv('DEBUG', 'false').lower() == 'true'

app = FastAPI(
    title="AntTrader 策略服务",
    description="在安全沙箱中执行Python量化策略",
    version="2.0.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(strategy.router)
app.include_router(backtest.router)
app.include_router(memory.router)
app.include_router(external.router)
app.include_router(objective_score.router)


@app.on_event("startup")
async def startup():
    get_backtest_memory()


@app.on_event("shutdown")
async def shutdown():
    pass


@app.get("/health")
async def health_check():
    return {"status": "healthy", "timestamp": datetime.now().isoformat()}


if __name__ == "__main__":
    import uvicorn

    uvicorn.run("app.main:app", host=HOST, port=PORT, reload=DEBUG)
