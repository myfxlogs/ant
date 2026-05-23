"""Metrics & risk assessment builders.

契约：docs/domains/backtest-system.md §7.4.3 · metrics.py

公式严格复刻旧 ``app/backtest.py:341-421`` 以保证 I9 金标准 bit-equal：

* ``total_return`` = (eq[-1] - eq[0]) / eq[0]
* ``max_drawdown`` = max((peak - eq) / peak)，peak = np.maximum.accumulate(eq)
* ``sharpe_ratio`` = mean(step_ret) / std(step_ret) * sqrt(bars_per_year)
    - step_ret = np.diff(eq) / eq[:-1]
    - bars_per_year = 365 * 24 * 3600 / Δt(前两根 bar 的 open 时间差，秒)
* ``annual_return`` = (1 + total_return) ** (1 / years) - 1
    - years = (bars[-1].close_time - bars[0].open_time) / (365.25 * 24 * 3600)
* ``profit_factor`` = gross_profit / gross_loss，inf 或 > 99 封顶 99.0
* ``win_rate`` = winning / total
* ``risk_score`` = clamp(100 - max_dd * 200, 0, 100)
* ``risk_level`` ≥ 70 low, ≥ 40 medium, else high
* ``is_reliable`` = total_trades ≥ 10
* warnings 文案：total<10 / max_dd>0.3 / sharpe<0
"""

from __future__ import annotations

from typing import List, Tuple

import numpy as np

from app.engine.types import Bar, Metrics, RiskAssessment, Trade


def _total_return(eq: np.ndarray) -> float:
    if len(eq) == 0 or eq[0] == 0:
        return 0.0
    return float((eq[-1] - eq[0]) / eq[0])


def _max_drawdown(eq: np.ndarray) -> float:
    if len(eq) == 0:
        return 0.0
    peak = np.maximum.accumulate(eq)
    drawdowns = np.where(peak > 0, (peak - eq) / peak, 0.0)
    return float(drawdowns.max()) if len(drawdowns) > 0 else 0.0


def _bars_per_year(bars: List[Bar]) -> int:
    if len(bars) < 2:
        return 365 * 24  # fallback = hourly
    dt = bars[1].open_time - bars[0].open_time  # milliseconds
    if dt <= 0:
        return 365 * 24
    tf_seconds = dt / 1000.0
    return max(1, int(365 * 24 * 3600 / tf_seconds))


def _sharpe_ratio(eq: np.ndarray, bars: List[Bar]) -> float:
    if len(eq) < 2:
        return 0.0
    step_returns = np.diff(eq) / np.where(eq[:-1] != 0, eq[:-1], 1.0)
    if len(step_returns) < 2 or np.std(step_returns) == 0:
        return 0.0
    bpy = _bars_per_year(bars)
    return float(np.mean(step_returns) / np.std(step_returns) * np.sqrt(bpy))


def _annual_return(total_return: float, bars: List[Bar]) -> float:
    if len(bars) < 2:
        return total_return
    duration_sec = (bars[-1].close_time - bars[0].open_time) / 1000.0
    years = max(duration_sec / (365.25 * 24 * 3600), 1e-6)
    if total_return <= -1.0:
        return -1.0
    try:
        return float((1 + total_return) ** (1.0 / years) - 1)
    except OverflowError:
        # Pathological: huge return over near-zero horizon. Fall back to raw total.
        return total_return


def _trade_stats(trades: List[Trade]) -> dict:
    winning = 0
    losing = 0
    gross_profit = 0.0
    gross_loss = 0.0
    profits: List[float] = []
    losses: List[float] = []
    for t in trades:
        if t.pnl > 0:
            winning += 1
            gross_profit += t.pnl
            profits.append(t.pnl)
        elif t.pnl < 0:
            losing += 1
            loss_abs = abs(t.pnl)
            gross_loss += loss_abs
            losses.append(loss_abs)
    total = len(trades)
    win_rate = winning / total if total > 0 else 0.0
    if gross_loss > 0:
        pf = gross_profit / gross_loss
    else:
        pf = float("inf") if gross_profit > 0 else 0.0
    # Cap inf / large to 99.0 (legacy behaviour preserves serializability).
    if pf == float("inf") or pf > 99.0:
        pf = 99.0
    return {
        "total": total,
        "winning": winning,
        "losing": losing,
        "win_rate": win_rate,
        "profit_factor": pf,
        "average_profit": float(np.mean(profits)) if profits else 0.0,
        "average_loss": float(np.mean(losses)) if losses else 0.0,
    }


def _build_risk(max_dd: float, sharpe: float, total_trades: int) -> RiskAssessment:
    score = max(0, min(100, int(100 - max_dd * 200)))
    level = "low" if score >= 70 else ("medium" if score >= 40 else "high")
    warnings: List[str] = []
    if total_trades < 10:
        warnings.append("样本数据较少（< 10 笔），统计结果不稳定")
    if max_dd > 0.3:
        warnings.append(f"最大回撤 {max_dd * 100:.1f}% 偏高，风险较大")
    if sharpe < 0:
        warnings.append("夏普比率为负，收益不抵波动风险")
    return RiskAssessment(
        score=score,
        level=level,  # type: ignore[arg-type]
        reasons=["基于权益曲线和成交记录分析"],
        warnings=warnings,
        is_reliable=total_trades >= 10,
    )


def build_metrics(
    equity_curve: List[float],
    trades: List[Trade],
    bars: List[Bar],
) -> Tuple[Metrics, RiskAssessment]:
    """Compute metrics + risk assessment. See module docstring for formulas."""
    eq = np.asarray(equity_curve, dtype=float)
    total_return = _total_return(eq)
    max_dd = _max_drawdown(eq)
    sharpe = _sharpe_ratio(eq, bars)
    annual = _annual_return(total_return, bars)
    stats = _trade_stats(trades)

    metrics = Metrics(
        total_return=total_return,
        annual_return=annual,
        max_drawdown=max_dd,
        sharpe_ratio=sharpe,
        win_rate=stats["win_rate"],
        profit_factor=stats["profit_factor"],
        total_trades=stats["total"],
        winning_trades=stats["winning"],
        losing_trades=stats["losing"],
        average_profit=stats["average_profit"],
        average_loss=stats["average_loss"],
    )
    risk = _build_risk(max_dd, sharpe, stats["total"])
    return metrics, risk
