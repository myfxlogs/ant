-- 130_ai_strategy_templates: AI strategy generation template library (spec/26 Phase 1)

CREATE TABLE IF NOT EXISTS ai_strategy_templates (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    category        VARCHAR(32) NOT NULL,
    name            VARCHAR(128) NOT NULL,
    description_zh  TEXT NOT NULL,
    python_skeleton TEXT NOT NULL,
    parameter_slots JSONB NOT NULL DEFAULT '[]',
    risk_level      VARCHAR(8) NOT NULL DEFAULT 'medium',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ck_ai_strategy_templates_category CHECK (category IN (
        'trend_following', 'mean_reversion', 'breakout', 'grid', 'martingale'
    )),
    CONSTRAINT ck_ai_strategy_templates_risk CHECK (risk_level IN (
        'low', 'medium', 'high', 'critical'
    ))
);

CREATE INDEX IF NOT EXISTS idx_ai_strategy_templates_category ON ai_strategy_templates(category);
CREATE INDEX IF NOT EXISTS idx_ai_strategy_templates_active ON ai_strategy_templates(is_active);

-- Seed: 5 strategy templates

INSERT INTO ai_strategy_templates (category, name, description_zh, python_skeleton, parameter_slots, risk_level) VALUES

-- 1. 双均线交叉 (Trend Following)
('trend_following', '双均线交叉', '经典趋势跟踪策略：快线上穿慢线做多，下穿做空',
$$import numpy as np
import pandas as pd

def initialize(state):
    state.entry_pct = {entry_pct}
    state.stop_loss_pct = {stop_loss_pct}
    state.take_profit_pct = {take_profit_pct}

@param fast_period {fast_period_default} range={fast_period_min}:{fast_period_max}:{fast_period_step}
@param slow_period {slow_period_default} range={slow_period_min}:{slow_period_max}:{slow_period_step}
def on_bar(state, bar, history):
    if len(history) < state.slow_period:
        return
    closes = np.array([h.close for h in history[-state.slow_period:]])
    fast_ma = np.mean(closes[-state.fast_period:])
    slow_ma = np.mean(closes)

    if fast_ma > slow_ma and state.position == 0:
        state.open_long(bar.close, entry_pct=state.entry_pct)
    elif fast_ma < slow_ma and state.position > 0:
        state.close_position(bar.close)
    elif fast_ma < slow_ma and state.position == 0:
        state.open_short(bar.close, entry_pct=state.entry_pct)
    elif fast_ma > slow_ma and state.position < 0:
        state.close_position(bar.close)
$$,
'[
    {"name": "fast_period", "type": "int", "default": 10, "min": 5, "max": 50, "step": 5, "description_zh": "快线周期"},
    {"name": "slow_period", "type": "int", "default": 30, "min": 20, "max": 100, "step": 10, "description_zh": "慢线周期"},
    {"name": "entry_pct", "type": "float", "default": 0.95, "min": 0.5, "max": 1.0, "step": 0.05, "description_zh": "仓位比例"},
    {"name": "stop_loss_pct", "type": "float", "default": 0.02, "min": 0.01, "max": 0.05, "step": 0.005, "description_zh": "止损比例"},
    {"name": "take_profit_pct", "type": "float", "default": 0.04, "min": 0.02, "max": 0.10, "step": 0.01, "description_zh": "止盈比例"}
]',
'medium'),

-- 2. 布林带突破 (Mean Reversion)
('mean_reversion', '布林带突破', '均值回归策略：价格触及下轨做多，触及上轨做空',
$$import numpy as np
import pandas as pd

def initialize(state):
    state.entry_pct = {entry_pct}
    state.stop_loss_pct = {stop_loss_pct}

@param period {period_default} range={period_min}:{period_max}:{period_step}
@param std_dev {std_dev_default} range={std_dev_min}:{std_dev_max}:{std_dev_step}
def on_bar(state, bar, history):
    if len(history) < state.period:
        return
    closes = np.array([h.close for h in history[-state.period:]])
    middle = np.mean(closes)
    std = np.std(closes)
    upper = middle + state.std_dev * std
    lower = middle - state.std_dev * std

    if bar.close <= lower and state.position == 0:
        state.open_long(bar.close, entry_pct=state.entry_pct)
    elif bar.close >= upper and state.position > 0:
        state.close_position(bar.close)
    elif bar.close >= upper and state.position == 0:
        state.open_short(bar.close, entry_pct=state.entry_pct)
    elif bar.close <= lower and state.position < 0:
        state.close_position(bar.close)
$$,
'[
    {"name": "period", "type": "int", "default": 20, "min": 10, "max": 50, "step": 5, "description_zh": "计算周期"},
    {"name": "std_dev", "type": "float", "default": 2.0, "min": 1.0, "max": 3.0, "step": 0.25, "description_zh": "标准差倍数"},
    {"name": "entry_pct", "type": "float", "default": 0.95, "min": 0.5, "max": 1.0, "step": 0.05, "description_zh": "仓位比例"},
    {"name": "stop_loss_pct", "type": "float", "default": 0.02, "min": 0.01, "max": 0.05, "step": 0.005, "description_zh": "止损比例"}
]',
'medium'),

-- 3. Donchian Channel (Breakout)
('breakout', 'Donchian Channel 突破', '突破策略：价格突破 N 日最高价做多，跌破最低价做空',
$$import numpy as np
import pandas as pd

def initialize(state):
    state.entry_pct = {entry_pct}
    state.stop_loss_pct = {stop_loss_pct}
    state.take_profit_pct = {take_profit_pct}

@param period {period_default} range={period_min}:{period_max}:{period_step}
def on_bar(state, bar, history):
    if len(history) < state.period:
        return
    highs = np.array([h.high for h in history[-(state.period+1):-1]])
    lows = np.array([h.low for h in history[-(state.period+1):-1]])
    highest = np.max(highs)
    lowest = np.min(lows)

    if bar.close > highest and state.position == 0:
        state.open_long(bar.close, entry_pct=state.entry_pct)
    elif bar.close < lowest and state.position > 0:
        state.close_position(bar.close)
    elif bar.close < lowest and state.position == 0:
        state.open_short(bar.close, entry_pct=state.entry_pct)
    elif bar.close > highest and state.position < 0:
        state.close_position(bar.close)
$$,
'[
    {"name": "period", "type": "int", "default": 20, "min": 10, "max": 60, "step": 5, "description_zh": "突破周期"},
    {"name": "entry_pct", "type": "float", "default": 0.95, "min": 0.5, "max": 1.0, "step": 0.05, "description_zh": "仓位比例"},
    {"name": "stop_loss_pct", "type": "float", "default": 0.02, "min": 0.01, "max": 0.05, "step": 0.005, "description_zh": "止损比例"},
    {"name": "take_profit_pct", "type": "float", "default": 0.04, "min": 0.02, "max": 0.10, "step": 0.01, "description_zh": "止盈比例"}
]',
'medium'),

-- 4. 网格交易 (Grid)
('grid', '网格交易', '震荡市策略：在价格区间内等距挂单，低买高卖',
$$import numpy as np
import pandas as pd

def initialize(state):
    state.entry_pct = {entry_pct}
    state.stop_loss_pct = {stop_loss_pct}

@param grid_count {grid_count_default} range={grid_count_min}:{grid_count_max}:{grid_count_step}
@param grid_spacing_pct {grid_spacing_pct_default} range={grid_spacing_pct_min}:{grid_spacing_pct_max}:{grid_spacing_pct_step}
def on_bar(state, bar, history):
    if state.grid_initialized == 0:
        base = bar.close
        state.grid_levels = [base * (1 + state.grid_spacing_pct * (i - state.grid_count//2))
                             for i in range(state.grid_count)]
        state.grid_initialized = 1
        state.last_grid_idx = state.grid_count // 2

    for i, level in enumerate(state.grid_levels):
        if bar.close <= level and i < state.last_grid_idx and state.position >= 0:
            state.open_long(bar.close, entry_pct=state.entry_pct / state.grid_count)
            state.last_grid_idx = i
        elif bar.close >= level and i > state.last_grid_idx and state.position > 0:
            pct = state.entry_pct / state.grid_count
            state.close_partial(bar.close, pct)
            state.last_grid_idx = i
$$,
'[
    {"name": "grid_count", "type": "int", "default": 10, "min": 5, "max": 50, "step": 5, "description_zh": "网格数量"},
    {"name": "grid_spacing_pct", "type": "float", "default": 0.01, "min": 0.005, "max": 0.05, "step": 0.005, "description_zh": "网格间距"},
    {"name": "entry_pct", "type": "float", "default": 0.95, "min": 0.5, "max": 1.0, "step": 0.05, "description_zh": "仓位比例"},
    {"name": "stop_loss_pct", "type": "float", "default": 0.05, "min": 0.02, "max": 0.10, "step": 0.01, "description_zh": "止损比例"}
]',
'high'),

-- 5. 马丁策略 (Martingale) — marked critical risk
('martingale', '马丁格尔策略', '逆势加仓策略：亏损后加倍仓位，高风险——仅供学习参考',
$$import numpy as np
import pandas as pd

def initialize(state):
    state.base_entry_pct = {entry_pct}
    state.multiplier = {multiplier_default}
    state.max_levels = {max_levels_default}
    state.current_level = 0
    state.stop_loss_pct = {stop_loss_pct}

def on_bar(state, bar, history):
    if state.current_level >= state.max_levels:
        if state.position > 0:
            state.close_position(bar.close)
        return

    entry_pct = state.base_entry_pct * (state.multiplier ** state.current_level)

    if state.position == 0:
        state.open_long(bar.close, entry_pct=min(entry_pct, 1.0))
        state.current_level = 1
    elif bar.close < state.avg_entry_price * (1 - state.stop_loss_pct):
        state.open_long(bar.close, entry_pct=min(entry_pct, 1.0))
        state.current_level += 1
    elif bar.close > state.avg_entry_price * (1 + state.stop_loss_pct):
        state.close_position(bar.close)
        state.current_level = 0
$$,
'[
    {"name": "entry_pct", "type": "float", "default": 0.02, "min": 0.01, "max": 0.05, "step": 0.005, "description_zh": "初始仓位"},
    {"name": "multiplier", "type": "float", "default": 2.0, "min": 1.5, "max": 3.0, "step": 0.25, "description_zh": "加仓倍数"},
    {"name": "max_levels", "type": "int", "default": 5, "min": 3, "max": 10, "step": 1, "description_zh": "最大加仓层级"},
    {"name": "stop_loss_pct", "type": "float", "default": 0.03, "min": 0.01, "max": 0.05, "step": 0.005, "description_zh": "止损比例"}
]',
'critical');
