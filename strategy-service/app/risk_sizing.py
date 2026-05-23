"""
仓位计算模块

参考 nautilus_trader risk/sizing.pyx 的 FixedRiskSizer 设计，
将「账户权益 × 风险百分比 / 止损点数」换算为手数，
同时提供 ATR 波动率仓位（VolatilityRiskSizer）。

策略代码中可直接调用（通过沙箱注入）：
  vol = risk_size(equity, risk_pct, entry, stop_loss, pip_value)
  vol = atr_size(equity, risk_pct, atr_value, pip_value)
"""

from __future__ import annotations

import math
from typing import Optional


# --- 常量 ---
MIN_LOT = 0.01          # 最小手数
LOT_STEP = 0.01         # 手数精度
MAX_LOT = 100.0         # 最大手数（安全上限）


def _round_lot(size: float, step: float = LOT_STEP, min_lot: float = MIN_LOT) -> float:
    """向下取整到最近的 step，并保证 >= min_lot。"""
    if size <= 0:
        return 0.0
    rounded = math.floor(size / step) * step
    return max(min_lot, min(rounded, MAX_LOT))


def risk_size(
    equity: float,
    risk_pct: float,
    entry_price: float,
    stop_loss_price: float,
    pip_value: float = 10.0,
    contract_size: float = 100000.0,
    commission_rate: float = 0.0,
    min_lot: float = MIN_LOT,
    max_lot: Optional[float] = None,
) -> float:
    """
    基于固定风险百分比计算手数（参考 nautilus FixedRiskSizer）。

    公式：
        risk_money = equity × risk_pct
        sl_pips    = |entry - stop_loss| / pip_size
        lot_size   = risk_money / (sl_pips × pip_value)

    Parameters
    ----------
    equity          : 账户净值（如 10000.0 USD）
    risk_pct        : 每笔风险比例（如 0.01 = 1%）
    entry_price     : 入场价格
    stop_loss_price : 止损价格
    pip_value       : 每手每 pip 盈亏价值（默认 10 USD，标准手 EURUSD）
    contract_size   : 合约面值（标准手 = 100000）
    commission_rate : 手续费率（占 risk_money 的比例，round-turn）
    min_lot         : 最小手数
    max_lot         : 最大手数限制（None = 用全局上限）

    Returns
    -------
    float : 建议手数，0.0 表示不开仓
    """
    if equity <= 0 or risk_pct <= 0:
        return 0.0
    if entry_price <= 0 or stop_loss_price <= 0:
        return 0.0

    sl_distance = abs(entry_price - stop_loss_price)
    if sl_distance < 1e-10:
        return 0.0

    risk_money = equity * risk_pct
    # 扣除手续费（round-turn）
    commission = risk_money * commission_rate * 2
    riskable = max(0.0, risk_money - commission)

    # pip_value 对应每手每点盈亏，sl_distance / pip_size = pip 数
    # 对于 5 位报价品种，pip_size = 0.0001，对于日元 = 0.01
    # 这里用价格差直接乘以 contract_size / pip_value 换算
    lot_size = riskable / (sl_distance * contract_size / pip_value * pip_value)
    # 化简：lot = riskable / (sl_distance * contract_size)  ← USD 计价
    lot_size = riskable / (sl_distance * contract_size)

    hard_limit = max_lot if max_lot is not None else MAX_LOT
    lot_size = min(lot_size, hard_limit)

    return _round_lot(lot_size, LOT_STEP, min_lot)


def atr_size(
    equity: float,
    risk_pct: float,
    atr_value: float,
    atr_multiplier: float = 1.5,
    contract_size: float = 100000.0,
    commission_rate: float = 0.0,
    min_lot: float = MIN_LOT,
    max_lot: Optional[float] = None,
) -> float:
    """
    基于 ATR 波动率的动态仓位计算。

    将 stop_loss_distance = atr_value × atr_multiplier 代入 risk_size 逻辑。

    Parameters
    ----------
    equity         : 账户净值
    risk_pct       : 每笔风险比例
    atr_value      : 当前 ATR 值（价格单位）
    atr_multiplier : ATR 倍数（止损宽度 = atr × multiplier，默认 1.5x）
    contract_size  : 合约面值
    commission_rate: 手续费率
    min_lot        : 最小手数
    max_lot        : 最大手数限制

    Returns
    -------
    float : 建议手数
    """
    if atr_value <= 0:
        return min_lot

    sl_distance = atr_value * atr_multiplier
    risk_money = equity * risk_pct
    commission = risk_money * commission_rate * 2
    riskable = max(0.0, risk_money - commission)

    lot_size = riskable / (sl_distance * contract_size)

    hard_limit = max_lot if max_lot is not None else MAX_LOT
    lot_size = min(lot_size, hard_limit)

    return _round_lot(lot_size, LOT_STEP, min_lot)


def kelly_size(
    equity: float,
    win_rate: float,
    avg_win: float,
    avg_loss: float,
    kelly_fraction: float = 0.5,
    min_lot: float = MIN_LOT,
    max_lot: Optional[float] = None,
    contract_size: float = 100000.0,
    current_price: float = 1.0,
) -> float:
    """
    Kelly 准则仓位计算（半凯利，减少波动）。

    f* = W/|L| - (1-W)/W  （简化公式）

    Parameters
    ----------
    equity         : 账户净值
    win_rate       : 历史胜率（0~1）
    avg_win        : 平均盈利（价格点数）
    avg_loss       : 平均亏损（价格点数，正值）
    kelly_fraction : Kelly 缩减系数（0.5 = 半凯利）
    min_lot        : 最小手数
    max_lot        : 最大手数限制
    contract_size  : 合约面值
    current_price  : 当前价格（用于折算手数）

    Returns
    -------
    float : 建议手数
    """
    if win_rate <= 0 or win_rate >= 1 or avg_win <= 0 or avg_loss <= 0:
        return min_lot

    loss_rate = 1.0 - win_rate
    kelly_f = (win_rate / avg_loss) - (loss_rate / avg_win)
    kelly_f = max(0.0, kelly_f) * kelly_fraction

    # kelly_f 是权益分配比例
    risk_money = equity * kelly_f
    if risk_money <= 0:
        return min_lot

    lot_size = risk_money / (current_price * contract_size)

    hard_limit = max_lot if max_lot is not None else MAX_LOT
    lot_size = min(lot_size, hard_limit)

    return _round_lot(lot_size, LOT_STEP, min_lot)
