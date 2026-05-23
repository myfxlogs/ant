import asyncio
import os
import time
from typing import Any, Dict

import numpy as np
from fastapi import APIRouter

from app.engine.params_extractor import extract_required_params
from app.engine.sandbox import StrategyRunner, StrategyRuntimeError, validate_strategy_code
from app.schemas import StrategyExecuteRequest, StrategyExecuteResponse, StrategyValidateRequest, StrategyValidateResponse, TradeSignal

router = APIRouter()


@router.post("/api/strategy/validate", response_model=StrategyValidateResponse)
async def validate_strategy(request: StrategyValidateRequest):
    try:
        result = validate_strategy_code(request.strategy_code)
        params = extract_required_params(request.strategy_code) if result.valid else []
        return StrategyValidateResponse(
            valid=result.valid,
            errors=list(result.errors),
            warnings=list(result.warnings),
            parameters=params,
        )
    except Exception as e:
        return StrategyValidateResponse(valid=False, errors=[f"验证错误: {e}"], warnings=[], parameters=[])


@router.post("/api/strategy/execute", response_model=StrategyExecuteResponse)
async def execute_strategy(request: StrategyExecuteRequest):
    start_time = time.time()
    try:
        klines = request.market_data.klines
        close_prices = np.array([k.close_price for k in klines])
        open_prices = np.array([k.open_price for k in klines], dtype=float)
        high_prices = np.array([k.high_price for k in klines], dtype=float)
        low_prices = np.array([k.low_price for k in klines], dtype=float)
        volumes = np.array([k.volume for k in klines], dtype=float)
        bar_times_ms = [int(k.close_time.timestamp() * 1000) for k in klines]
        context: Dict[str, Any] = {
            'close': close_prices,
            'open': open_prices,
            'high': high_prices,
            'low': low_prices,
            'volume': volumes,
            'bar_times_ms': bar_times_ms,
            'bar_time_ms': bar_times_ms[-1] if len(bar_times_ms) > 0 else None,
            'symbol': request.market_data.symbol,
            'timeframe': request.market_data.timeframe,
            'current_price': request.market_data.current_price if request.market_data.current_price is not None else (float(close_prices[-1]) if len(close_prices) > 0 else 0),
        }
        if request.context:
            context.update(request.context)
        timeout_seconds = int(os.getenv('BACKTEST_TIMEOUT', '120'))
        runner = StrategyRunner(request.strategy_code, timeout_ms=timeout_seconds * 1000)
        loop = asyncio.get_event_loop()
        signal_data = await loop.run_in_executor(None, runner.call, context)
        elapsed = (time.time() - start_time) * 1000
        if signal_data is None or not isinstance(signal_data, dict):
            return StrategyExecuteResponse(success=False, error="策略未返回有效信号", execution_time_ms=elapsed, logs=[])
        action = str(signal_data.get('signal', 'hold')).strip().lower()
        allowed = {'buy', 'sell', 'hold', 'close', 'buy_limit', 'sell_limit', 'buy_stop', 'sell_stop', 'buy_stop_limit', 'sell_stop_limit', 'cancel_pending'}
        if action not in allowed:
            return StrategyExecuteResponse(success=False, error="signal 字段不支持，允许: buy/sell/hold/close 与挂单类型", execution_time_ms=elapsed, logs=[])
        runtime = context.get('runtime') if isinstance(context.get('runtime'), dict) else None
        signal = TradeSignal(
            signal=action,
            symbol=signal_data.get('symbol', request.market_data.symbol),
            price=signal_data.get('price'),
            volume=signal_data.get('volume'),
            stop_loss=signal_data.get('stop_loss'),
            take_profit=signal_data.get('take_profit'),
            confidence=float(signal_data.get('confidence', 0.5)),
            reason=signal_data.get('reason'),
            risk_level=signal_data.get('risk_level', 'medium'),
        )
        return StrategyExecuteResponse(success=True, signal=signal, runtime=runtime, execution_time_ms=elapsed, logs=[])
    except StrategyRuntimeError as e:
        elapsed = (time.time() - start_time) * 1000
        return StrategyExecuteResponse(success=False, error=str(e), execution_time_ms=elapsed, logs=[])
    except Exception as e:
        elapsed = (time.time() - start_time) * 1000
        return StrategyExecuteResponse(success=False, error=f"服务器错误: {e}", execution_time_ms=elapsed, logs=[])
