package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

type defaultStrategyTemplate struct {
	Name        string
	Description string
	Code        string
	IsPublic    bool
}

func seedDefaultStrategyTemplates(ctx context.Context, db *sqlx.DB, templateRepo *repository.StrategyTemplateRepository) {
	if db == nil || templateRepo == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var userIDs []uuid.UUID
	if err := db.SelectContext(ctx, &userIDs, `SELECT id FROM users ORDER BY created_at ASC`); err != nil {
		logger.Warn("Seed default templates: failed to list users", zap.Error(err))
		return
	}

	defaults := getDefaultStrategyTemplates()
	i18nMap := getTemplateI18nMap()
	created := 0
	updated := 0
	skipped := 0
	failed := 0

	for _, userID := range userIDs {
		for _, tpl := range defaults {
			m := model.NewStrategyTemplate(userID, tpl.Name, tpl.Code)
			m.Description = tpl.Description
			m.IsPublic = tpl.IsPublic
			m.IsSystem = true
			// Basic tagging so the frontend can highlight presets.
			m.Tags = []string{"preset"}

			// Attach parameter schema and i18n per known template name.
			switch tpl.Name {
			case "双均线交叉策略":
				m.Tags = append(m.Tags, "trend", "moving-average", "beginner")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "fast_period", Type: "int", Default: "10", Min: "2", Max: "200", Step: "1", Label: "快线周期", Description: "用于计算快均线的周期"},
					{Name: "slow_period", Type: "int", Default: "30", Min: "3", Max: "400", Step: "1", Label: "慢线周期", Description: "用于计算慢均线的周期"},
				})
			case "RSI超买超卖策略":
				m.Tags = append(m.Tags, "mean-reversion", "RSI", "beginner")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "rsi_period", Type: "int", Default: "14", Min: "2", Max: "200", Step: "1", Label: "RSI 周期", Description: "用于计算 RSI 的回溯长度"},
					{Name: "oversold", Type: "float", Default: "30", Min: "0", Max: "50", Step: "1", Label: "超卖阈值", Description: "低于该值视为超卖"},
					{Name: "overbought", Type: "float", Default: "70", Min: "50", Max: "100", Step: "1", Label: "超买阈值", Description: "高于该值视为超买"},
				})
			case "MACD策略":
				m.Tags = append(m.Tags, "trend", "MACD", "intermediate")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "fast_period", Type: "int", Default: "12", Min: "2", Max: "200", Step: "1", Label: "快线周期", Description: "MACD 快线 EMA 周期"},
					{Name: "slow_period", Type: "int", Default: "26", Min: "3", Max: "400", Step: "1", Label: "慢线周期", Description: "MACD 慢线 EMA 周期"},
					{Name: "signal_period", Type: "int", Default: "9", Min: "2", Max: "200", Step: "1", Label: "信号线周期", Description: "MACD 信号线 EMA 周期"},
				})
			case "布林带收缩突破":
				m.Tags = append(m.Tags, "volatility", "bollinger", "breakout", "intermediate")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "bb_period", Type: "int", Default: "20", Min: "2", Max: "500", Step: "1", Label: "布林带周期", Description: "用于计算布林带的回溯周期"},
					{Name: "bb_std", Type: "float", Default: "2.0", Min: "0.1", Max: "10.0", Step: "0.1", Label: "标准差倍数", Description: "布林带偏离倍数"},
					{Name: "squeeze_threshold", Type: "float", Default: "0.05", Min: "0.0", Max: "1.0", Step: "0.01", Label: "收缩阈值", Description: "相对带宽低于该值视为收缩"},
				})
			case "布林带均值回归":
				m.Tags = append(m.Tags, "mean-reversion", "bollinger", "beginner")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "bb_period", Type: "int", Default: "20", Min: "2", Max: "500", Step: "1", Label: "布林带周期", Description: "用于计算布林带的回溯周期"},
					{Name: "bb_std", Type: "float", Default: "2.0", Min: "0.1", Max: "10.0", Step: "0.1", Label: "标准差倍数", Description: "布林带偏离倍数"},
				})
			case "放量突破":
				m.Tags = append(m.Tags, "breakout", "volume", "intermediate")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "lookback", Type: "int", Default: "20", Min: "2", Max: "500", Step: "1", Label: "回看周期", Description: "用于统计近期高点和平均成交量的周期"},
					{Name: "volume_multiplier", Type: "float", Default: "1.5", Min: "1.0", Max: "10.0", Step: "0.1", Label: "放量倍数", Description: "当前成交量相对平均成交量的放大量阈值"},
				})
			case "海龟交易法":
				m.Tags = append(m.Tags, "trend", "turtle", "classic", "advanced")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "entry_period", Type: "int", Default: "20", Min: "2", Max: "200", Step: "1", Label: "入场通道周期", Description: "用于计算入场突破通道高点的周期"},
					{Name: "exit_period", Type: "int", Default: "10", Min: "2", Max: "200", Step: "1", Label: "退出通道周期", Description: "用于计算退出通道低点的周期"},
				})
			case "网格交易":
				m.Tags = append(m.Tags, "grid", "market-making", "range", "intermediate")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "grid_count", Type: "int", Default: "10", Min: "2", Max: "200", Step: "1", Label: "网格数量", Description: "上下区间内的网格层数"},
					{Name: "lower_price", Type: "float", Default: "0", Min: "0", Max: "0", Step: "0.0001", Label: "下边界价格", Description: "网格下边界（0 表示自动估计）"},
					{Name: "upper_price", Type: "float", Default: "0", Min: "0", Max: "0", Step: "0.0001", Label: "上边界价格", Description: "网格上边界（0 表示自动估计）"},
					{Name: "lot", Type: "float", Default: "0.01", Min: "0.01", Max: "100", Step: "0.01", Label: "下单手数", Description: "每层网格默认下单量"},
				})
			case "马丁格尔加仓":
				m.Tags = append(m.Tags, "martingale", "risk-heavy", "advanced")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "base_lot", Type: "float", Default: "0.01", Min: "0.01", Max: "100", Step: "0.01", Label: "基础手数", Description: "首单下单量"},
					{Name: "multiplier", Type: "float", Default: "2.0", Min: "1.1", Max: "5.0", Step: "0.1", Label: "加倍倍数", Description: "每次亏损后乘的倍数"},
					{Name: "max_levels", Type: "int", Default: "5", Min: "1", Max: "15", Step: "1", Label: "最大加仓层数", Description: "达到后重置，避免无限加仓"},
					{Name: "adverse_price_step", Type: "float", Default: "0", Min: "0", Max: "10000", Step: "0.0001", Label: "反向价距", Description: "每层加仓所需的不利价差（0=自动按近期波动估算）"},
				})
			case "定投策略 (DCA)":
				m.Tags = append(m.Tags, "passive", "DCA", "long-term", "beginner")
				_ = m.SetParameters([]model.TemplateParameter{
					{Name: "interval_hours", Type: "int", Default: "168", Min: "1", Max: "8760", Step: "1", Label: "间隔小时", Description: "每隔多少小时买入一次"},
					{Name: "lot", Type: "float", Default: "0.01", Min: "0.01", Max: "100", Step: "0.01", Label: "下单手数", Description: "每次定投下单量"},
				})
			}

			// Attach i18n dictionary (zh-CN / zh-TW / en / ja / vi) from the
			// centralized table in seed_default_templates_i18n.go.
			if v, ok := i18nMap[tpl.Name]; ok && v != nil {
				_ = m.SetI18n(v)
			}

			// Only match the user's own SYSTEM-templates by name. A user may
			// have a custom (non-system) template with the same name as a
			// preset; we must not touch that row. System templates are
			// DELETE-protected at the repo layer (see
			// StrategyTemplateRepository.Delete), so on steady state we either
			// find the existing system row and UPDATE it, or we INSERT a new
			// is_system=true row for a fresh user.
			var exists bool
			if err := db.GetContext(ctx, &exists,
				`SELECT EXISTS(
					SELECT 1 FROM strategy_templates
					WHERE user_id = $1 AND name = $2 AND is_system = TRUE
				)`,
				userID, tpl.Name,
			); err != nil {
				failed++
				continue
			}
			if exists {
				if _, err := db.ExecContext(ctx,
					`UPDATE strategy_templates
					 SET code = $3, description = $4, is_public = $5,
					     parameters = $6, i18n = $7, tags = $8,
					     updated_at = CURRENT_TIMESTAMP
					 WHERE user_id = $1 AND name = $2 AND is_system = TRUE`,
					userID, tpl.Name, tpl.Code, tpl.Description, tpl.IsPublic,
					m.Parameters, m.I18n, m.Tags,
				); err != nil {
					failed++
					continue
				}
				updated++
				continue
			}

			if err := templateRepo.Create(ctx, m); err != nil {
				failed++
				continue
			}
			created++
		}
	}
	logger.Info("Seed default templates complete",
		zap.Int("users", len(userIDs)),
		zap.Int("created", created),
		zap.Int("updated", updated),
		zap.Int("skipped", skipped),
		zap.Int("failed", failed),
	)
}

func getDefaultStrategyTemplates() []defaultStrategyTemplate {
	base := []defaultStrategyTemplate{
		{
			Name:        "双均线交叉策略",
			Description: "当快均线上穿慢均线时买入，下穿时卖出",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]

    fast_period = int(params.get("fast_period", 10))
    slow_period = int(params.get("slow_period", 20))

    if slow_period < 2:
        slow_period = 2
    if fast_period < 1:
        fast_period = 1

    if len(close) < slow_period + 2:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    ma_fast = float(close[-fast_period:].mean())
    ma_slow = float(close[-slow_period:].mean())
    ma_fast_prev = float(close[-fast_period-1:-1].mean())
    ma_slow_prev = float(close[-slow_period-1:-1].mean())

    action = "hold"
    reason = "无信号"

    if ma_fast > ma_slow and ma_fast_prev <= ma_slow_prev:
        action = "buy"
        reason = "金叉买入信号"
    elif ma_fast < ma_slow and ma_fast_prev >= ma_slow_prev:
        action = "sell"
        reason = "死叉卖出信号"

    return {
        "signal": action,
        "symbol": symbol,
        "confidence": 0.7,
        "risk_level": "medium",
        "reason": reason,
    }`,
		},
		{
			Name:        "RSI超买超卖策略",
			Description: "RSI低于30超买区买入，高于70超卖区卖出",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]

    period = int(params.get("rsi_period", 14))
    oversold = float(params.get("oversold", 30))
    overbought = float(params.get("overbought", 70))

    if len(close) < period + 1:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    deltas = np.diff(close)
    gains = np.where(deltas > 0, deltas, 0.0)
    losses = np.where(deltas < 0, -deltas, 0.0)
    avg_gain = float(np.mean(gains[-period:]))
    avg_loss = float(np.mean(losses[-period:]))
    if avg_loss == 0:
        rsi = 100.0
    else:
        rs = avg_gain / avg_loss
        rsi = 100.0 - (100.0 / (1.0 + rs))

    action = "hold"
    risk_level = "low"
    reason = f"RSI={rsi:.2f} 无信号"

    if rsi < oversold:
        action = "buy"
        risk_level = "medium"
        reason = f"RSI={rsi:.2f} 超卖区买入信号"
    elif rsi > overbought:
        action = "sell"
        risk_level = "medium"
        reason = f"RSI={rsi:.2f} 超买区卖出信号"

    return {
        "signal": action,
        "symbol": symbol,
        "confidence": 0.6,
        "risk_level": risk_level,
        "reason": reason,
        "rsi": round(rsi, 2),
    }`,
		},
		{
			Name:        "MACD策略",
			Description: "MACD金叉买入，死叉卖出",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]

    fast_period = int(params.get("fast_period", 12))
    slow_period = int(params.get("slow_period", 26))
    signal_period = int(params.get("signal_period", 9))

    if len(close) < slow_period + signal_period + 2:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    def ema(series, period):
        series = series[-period*4:]
        if len(series) == 0:
            return float(close[-1])
        multiplier = 2.0 / (period + 1.0)
        ema_val = float(series[0])
        for price in series[1:]:
            ema_val = (float(price) - ema_val) * multiplier + ema_val
        return ema_val

    ema_fast = ema(close, fast_period)
    ema_slow = ema(close, slow_period)
    macd_line = ema_fast - ema_slow

    # 简化的信号线和历史
    macd_history = []
    for i in range(slow_period, len(close)):
        window = close[: i + 1]
        ef = ema(window, fast_period)
        es = ema(window, slow_period)
        macd_history.append(ef - es)

    if len(macd_history) < signal_period + 2:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    signal_line = ema(np.array(macd_history), signal_period)
    macd_prev = macd_history[-2]
    signal_prev = signal_line

    action = "hold"
    reason = "无信号"

    if macd_line > signal_line and macd_prev <= signal_prev:
        action = "buy"
        reason = "MACD金叉买入信号"
    elif macd_line < signal_line and macd_prev >= signal_prev:
        action = "sell"
        reason = "MACD死叉卖出信号"

    return {
        "signal": action,
        "symbol": symbol,
        "price": float(close[-1]),
        "confidence": 0.65,
        "reason": reason,
        "risk_level": "medium",
        "macd": round(macd_line, 5),
        "signal_line": round(signal_line, 5),
    }`,
		},
		// Bollinger band squeeze breakout
		{
			Name:        "布林带收缩突破",
			Description: "布林带收缩后突破上轨做多、下轨做空",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]

    bb_period = int(params.get("bb_period", 20))
    bb_std = float(params.get("bb_std", 2.0))
    squeeze_threshold = float(params.get("squeeze_threshold", 0.05))

    if bb_period < 2:
        bb_period = 2

    if len(close) < bb_period + 2:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    window = close[-bb_period:]
    mid = float(window.mean())
    std = float(window.std())
    upper = mid + bb_std * std
    lower = mid - bb_std * std
    width = upper - lower
    rel_width = width / mid if mid > 0 else 0.0

    price_prev = float(close[-2])
    price = float(close[-1])

    action = "hold"
    reason = "无信号"

    if rel_width <= squeeze_threshold and price > upper and price_prev <= upper:
        action = "buy"
        reason = "布林带收缩后向上突破"
    elif rel_width <= squeeze_threshold and price < lower and price_prev >= lower:
        action = "sell"
        reason = "布林带收缩后向下突破"

    return {
        "signal": action,
        "symbol": symbol,
        "confidence": 0.65,
        "risk_level": "medium",
        "reason": reason,
    }`,
		},
		// Bollinger mean reversion
		{
			Name:        "布林带均值回归",
			Description: "价格触碰下轨做多，上轨平仓/做空，适合震荡市",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]

    bb_period = int(params.get("bb_period", 20))
    bb_std = float(params.get("bb_std", 2.0))

    if bb_period < 2:
        bb_period = 2

    if len(close) < bb_period + 2:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    window = close[-bb_period:]
    mid = float(window.mean())
    std = float(window.std())
    upper = mid + bb_std * std
    lower = mid - bb_std * std

    price_prev = float(close[-2])
    price = float(close[-1])

    action = "hold"
    reason = "无信号"

    # 价格从带内下穿到下轨以下，视为偏离过大，做多均值回归
    if price < lower and price_prev >= lower:
        action = "buy"
        reason = "价格触及布林带下轨，做多均值回归"
    # 价格从带内上穿到上轨以上，平多/做空
    elif price > upper and price_prev <= upper:
        action = "sell"
        reason = "价格触及布林带上轨，做空或平多"

    return {
        "signal": action,
        "symbol": symbol,
        "confidence": 0.6,
        "risk_level": "medium",
        "reason": reason,
    }`,
		},
		// Volume breakout with ATR flavour
		{
			Name:        "放量突破",
			Description: "价格突破近期高点且成交量放大时入场，结合ATR止损思路",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]
    high = context["high"]
    volume = context["volume"]

    lookback = int(params.get("lookback", 20))
    vol_mult = float(params.get("volume_multiplier", 1.5))

    if lookback < 2:
        lookback = 2

    if len(close) < lookback + 2:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    recent_high = float(high[-lookback-1:-1].max())
    price_prev = float(close[-2])
    price = float(close[-1])
    vol = float(volume[-1])
    avg_vol = float(volume[-lookback:].mean())

    action = "hold"
    reason = "无信号"

    if price > recent_high and price_prev <= recent_high and vol > vol_mult * avg_vol:
        action = "buy"
        reason = "放量向上突破近期高点"

    return {
        "signal": action,
        "symbol": symbol,
        "confidence": 0.7 if action != "hold" else 0.0,
        "risk_level": "medium" if action != "hold" else "low",
        "reason": reason,
    }`,
		},
		// Turtle trading (long-only simplified)
		{
			Name:        "海龟交易法",
			Description: "突破入场、跌破退出的经典趋势策略，使用通道突破信号",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]
    high = context["high"]
    low = context["low"]

    entry_period = int(params.get("entry_period", 20))
    exit_period = int(params.get("exit_period", 10))

    if entry_period < 2:
        entry_period = 2
    if exit_period < 2:
        exit_period = 2

    min_len = max(entry_period, exit_period) + 2
    if len(close) < min_len:
        return {
            "signal": "hold",
            "symbol": symbol,
            "confidence": 0.0,
            "risk_level": "low",
            "reason": "数据不足",
        }

    # 多头入场：突破 entry_period 最高价
    channel_high = float(high[-entry_period-1:-1].max())
    channel_low_exit = float(low[-exit_period-1:-1].min())

    price_prev = float(close[-2])
    price = float(close[-1])

    pos = context.get("position") or None

    action = "hold"
    reason = "无信号"

    if pos is None:
        if price > channel_high and price_prev <= channel_high:
            action = "buy"
            reason = "突破通道上轨，海龟入场信号"
    else:
        # 简化：只有多头，跌破 exit 通道下轨时退出
        if price < channel_low_exit and price_prev >= channel_low_exit:
            action = "close"
            reason = "跌破退出通道，下车信号"

    return {
        "signal": action,
        "symbol": symbol,
        "confidence": 0.7 if action != "hold" else 0.0,
        "risk_level": "medium" if action != "hold" else "low",
        "reason": reason,
    }`,
		},
	}
	return append(base, getExtraDefaultStrategyTemplates()...)
}
