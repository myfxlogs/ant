import numpy as np

from app.schemas import (
    MACDSignal,
    MASignal,
    ObjectiveScoreRequest,
    ObjectiveScoreResponse,
    ObjectiveSignals,
    RSISignal,
)


def calculate_objective_score(req: ObjectiveScoreRequest) -> ObjectiveScoreResponse:
    closes = [float(k.close_price) for k in req.klines if k and k.close_price is not None]
    if len(closes) < 5:
        last = closes[-1] if closes else 0.0
        sigs = ObjectiveSignals(
            rsi=RSISignal(value=50.0, signal="neutral"),
            macd=MACDSignal(value=0.0, signal_line=0.0, histogram=0.0, signal="neutral", trend="consolidating"),
            ma=MASignal(ma5=last, ma10=last, ma20=last, trend="sideways"),
        )
        return ObjectiveScoreResponse(decision="HOLD", overall_score=0.0, technical_score=0.0, signals=sigs)

    prices = np.asarray(closes, dtype=float)
    rsi_val = _rsi(prices, 14)
    rsi_flag = "oversold" if rsi_val < 30 else "overbought" if rsi_val > 70 else "neutral"
    rsi_sig = RSISignal(value=float(round(rsi_val, 2)), signal=rsi_flag)

    fast = _ema_series(prices, 12)
    slow = _ema_series(prices, 26)
    mlen = min(len(fast), len(slow))
    macd_line = fast[-mlen:] - slow[-mlen:]
    if len(macd_line) < 2:
        macd_val = float(macd_line[-1]) if len(macd_line) > 0 else 0.0
        sig_line = 0.0
        hist = 0.0
    else:
        sig_series = _ema_series(macd_line, 9)
        macd_val = float(macd_line[-1])
        sig_line = float(sig_series[-1])
        hist = float(macd_val - sig_line)
    if macd_val > sig_line and hist > 0:
        macd_flag = "bullish"
        macd_trend = "golden_cross"
    elif macd_val < sig_line and hist < 0:
        macd_flag = "bearish"
        macd_trend = "death_cross"
    else:
        macd_flag = "neutral"
        macd_trend = "consolidating"
    macd_sig = MACDSignal(
        value=float(round(macd_val, 6)),
        signal_line=float(round(sig_line, 6)),
        histogram=float(round(hist, 6)),
        signal=macd_flag,
        trend=macd_trend,
    )

    price = float(prices[-1])
    ma5 = float(np.mean(prices[-5:])) if len(prices) >= 5 else price
    ma10 = float(np.mean(prices[-10:])) if len(prices) >= 10 else price
    ma20 = float(np.mean(prices[-20:])) if len(prices) >= 20 else price
    if price > ma5 > ma10 > ma20:
        ma_trend = "strong_uptrend"
    elif price > ma20:
        ma_trend = "uptrend"
    elif price < ma5 < ma10 < ma20:
        ma_trend = "strong_downtrend"
    elif price < ma20:
        ma_trend = "downtrend"
    else:
        ma_trend = "sideways"
    ma_sig = MASignal(ma5=float(round(ma5, 6)), ma10=float(round(ma10, 6)), ma20=float(round(ma20, 6)), trend=ma_trend)

    score = _score(rsi_flag, macd_flag, ma_trend)
    decision = "BUY" if score >= 20 else "SELL" if score <= -20 else "HOLD"
    return ObjectiveScoreResponse(
        decision=decision,
        overall_score=score,
        technical_score=score,
        signals=ObjectiveSignals(rsi=rsi_sig, macd=macd_sig, ma=ma_sig),
    )


def _rsi(arr, period=14):
    if len(arr) <= period:
        return 50.0
    deltas = np.diff(arr)
    gains = np.where(deltas > 0, deltas, 0.0)
    losses = np.where(deltas < 0, -deltas, 0.0)
    avg_gain = np.mean(gains[-period:]) if len(gains) >= period else np.mean(gains)
    avg_loss = np.mean(losses[-period:]) if len(losses) >= period else np.mean(losses)
    if avg_loss == 0:
        return 100.0
    rs = avg_gain / avg_loss
    return float(100.0 - 100.0 / (1.0 + rs))


def _ema_series(arr, n):
    if len(arr) < n:
        return np.array([float(np.mean(arr))])
    k = 2.0 / (n + 1)
    out = [float(np.mean(arr[:n]))]
    for v in arr[n:]:
        out.append(v * k + out[-1] * (1 - k))
    return np.array(out)


def _score(rsi_flag: str, macd_flag: str, ma_trend: str) -> float:
    score = 0.0
    if rsi_flag == "oversold":
        score += 20
    elif rsi_flag == "overbought":
        score -= 20
    if macd_flag == "bullish":
        score += 25
    elif macd_flag == "bearish":
        score -= 25
    if ma_trend == "strong_uptrend":
        score += 30
    elif ma_trend == "uptrend":
        score += 15
    elif ma_trend == "strong_downtrend":
        score -= 30
    elif ma_trend == "downtrend":
        score -= 15
    return max(-100.0, min(100.0, score))
