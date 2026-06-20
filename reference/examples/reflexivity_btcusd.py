# Reflexivity Strategy — BTCUSD 反身性交易策略
# 基于索罗斯反身性理论：自我强化阶段入场，偏离均衡或拐点离场

import math
import numpy as np

# ── 可调参数 ──
# @param ema_fast 20 range=5:50:5
# @param ema_slow 50 range=20:100:10
# @param atr_period 20 range=10:30:5
# @param streak_bars 3 range=2:5:1
# @param deviation_atr 2.5 range=1.5:4.0:0.5
# @param rsi_floor 30.0 range=20:40:5
# @param rsi_ceil 70.0 range=60:80:5

# 风控交给引擎
# @strategy stopLossPct 0.02
# @strategy takeProfitPct 0.04
# @strategy entryPct 0.25
# @strategy tradeDirection both


def ema(series, period):
    """指数移动平均"""
    k = 2.0 / (period + 1)
    result = [series[0]]
    for i in range(1, len(series)):
        result.append(series[i] * k + result[-1] * (1 - k))
    return result


def run(context):
    ema_fast = 20
    ema_slow = 50
    atr_period = 20
    streak_bars = 3
    deviation_atr = 2.5
    rsi_floor = 30.0
    rsi_ceil = 70.0

    close = context.get('close', [])
    high  = context.get('high', [])
    low   = context.get('low', [])
    pos   = context.get('position')

    n = len(close)
    if n < ema_slow + atr_period + 5:
        return {'signal': 'hold', 'volume': 0}

    # ── 指标 ──
    ema20 = ema(close, ema_fast)
    ema50 = ema(close, ema_slow)

    # ATR (Wilder's)
    tr_list = []
    for i in range(1, n):
        tr = max(
            abs(high[i] - low[i]),
            abs(high[i] - close[i-1]),
            abs(low[i] - close[i-1])
        )
        tr_list.append(tr)
    atr_vals = [sum(tr_list[:atr_period]) / atr_period] * atr_period
    for i in range(atr_period, len(tr_list)):
        atr_vals.append((atr_vals[-1] * (atr_period - 1) + tr_list[i]) / atr_period)

    # RSI
    gains, losses = [], []
    for i in range(1, n):
        delta = close[i] - close[i-1]
        gains.append(max(delta, 0))
        losses.append(max(-delta, 0))
    rsi_vals = [50.0] * 14
    avg_gain = sum(gains[:14]) / 14
    avg_loss = sum(losses[:14]) / 14
    rs = avg_gain / max(avg_loss, 1e-10)
    rsi_vals.append(100 - 100 / (1 + rs))
    for i in range(14, len(gains)):
        avg_gain = (avg_gain * 13 + gains[i]) / 14
        avg_loss = (avg_loss * 13 + losses[i]) / 14
        rs = avg_gain / max(avg_loss, 1e-10)
        rsi_vals.append(100 - 100 / (1 + rs))

    idx = -1
    ema20_now = ema20[idx]
    ema50_now = ema50[idx]
    atr_now   = atr_vals[idx]
    rsi_now   = rsi_vals[idx]
    price     = close[idx]

    # ── 反身性状态机 ──
    trend_up   = ema20_now > ema50_now
    trend_down = ema20_now < ema50_now

    bullish_streak = all(close[-i] >= close[-i-1] for i in range(1, streak_bars + 1))
    bearish_streak = all(close[-i] <= close[-i-1] for i in range(1, streak_bars + 1))

    atr_ma5  = sum(atr_vals[-6:-1]) / 5
    atr_ma10 = sum(atr_vals[-11:-1]) / 10
    atr_accel = atr_ma5 > atr_ma10 * 1.1

    deviation = abs(price - ema50_now) / max(atr_now, 1e-10)
    far_equilibrium = deviation > deviation_atr

    prev_high = high[-2]
    prev_low  = low[-2]
    bearish_reversal = close[-1] < prev_low and rsi_now < rsi_floor
    bullish_reversal = close[-1] > prev_high and rsi_now > rsi_ceil

    # ── 信号 ──
    if pos is None:
        if trend_up and price > ema20_now and bullish_streak and atr_accel and not far_equilibrium:
            return {
                'signal': 'buy', 'volume': 1.0,
                'stop_loss': price - atr_now * 2,
                'take_profit': price + atr_now * 4,
            }
        if trend_down and price < ema20_now and bearish_streak and atr_accel and not far_equilibrium:
            return {
                'signal': 'sell', 'volume': 1.0,
                'stop_loss': price + atr_now * 2,
                'take_profit': price - atr_now * 4,
            }
    else:
        should_close = False
        if pos.get('type') == 'buy':
            if trend_down or far_equilibrium or bearish_reversal:
                should_close = True
        elif pos.get('type') == 'sell':
            if trend_up or far_equilibrium or bullish_reversal:
                should_close = True
        if should_close:
            return {'signal': 'hold', 'volume': 0, 'close': True}

    return {'signal': 'hold', 'volume': 0}
