package server

func getExtraDefaultStrategyTemplates() []defaultStrategyTemplate {
	return []defaultStrategyTemplate{
		{
			Name:        "网格交易",
			Description: "在指定价格区间设置等距买卖网格，震荡行情中自动低买高卖",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]
    runtime = context.get("runtime") if isinstance(context.get("runtime"), dict) else {}

    grid_count = int(params.get("grid_count", 10))
    lower = float(params.get("lower_price", 0))
    upper = float(params.get("upper_price", 0))
    lot = float(params.get("lot", 0.01))

    if grid_count < 2:
        grid_count = 2
    if len(close) < 20:
        return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "low", "reason": "数据不足"}

    price = float(close[-1])

    # Auto range if not provided: use recent min/max.
    if upper <= lower:
        window = close[-300:] if len(close) > 300 else close
        lower = float(window.min())
        upper = float(window.max())
        if upper <= lower:
            return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "low", "reason": "区间无效"}

    step = (upper - lower) / float(grid_count - 1)

    levels = runtime.get("grid_levels")
    if not isinstance(levels, list) or len(levels) != grid_count:
        levels = [lower + i * step for i in range(grid_count)]
        runtime["grid_levels"] = levels

    placed = runtime.get("placed_levels")
    if not isinstance(placed, list):
        placed = []

    def key(x):
        try:
            return str(round(float(x), 6))
        except Exception:
            return str(x)

    # Decide next order: place one missing pending order per evaluation.
    buy_target = None
    sell_target = None

    for lv in reversed(levels):
        if float(lv) < price and key(lv) not in placed:
            buy_target = float(lv)
            break
    for lv in levels:
        if float(lv) > price and key(lv) not in placed:
            sell_target = float(lv)
            break

    if buy_target is None and sell_target is None:
        return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "low", "reason": "网格已布满"}

    if buy_target is not None:
        placed.append(key(buy_target))
        runtime["placed_levels"] = placed
        return {"signal": "buy_limit", "symbol": symbol, "price": buy_target, "volume": lot, "confidence": 0.6, "risk_level": "medium", "reason": "网格买入挂单"}

    placed.append(key(sell_target))
    runtime["placed_levels"] = placed
    return {"signal": "sell_limit", "symbol": symbol, "price": sell_target, "volume": lot, "confidence": 0.6, "risk_level": "medium", "reason": "网格卖出挂单"}`,
		},
		{
			Name:        "马丁格尔加仓",
			Description: "亏损后倍数加仓，盈利重置。高风险策略，严格控制最大层数",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    close = context["close"]
    runtime = context.get("runtime") if isinstance(context.get("runtime"), dict) else {}

    base_lot = float(params.get("base_lot", params.get("lot", 0.01)))
    multiplier = float(params.get("multiplier", 2.0))
    max_levels = int(params.get("max_levels", 5))
    adverse_step = float(params.get("adverse_price_step", 0))

    if len(close) < 20:
        return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "low", "reason": "数据不足"}

    price = float(close[-1])

    # Auto-estimate adverse_step from recent ATR-like range if not provided.
    if adverse_step <= 0:
        window = close[-50:] if len(close) >= 50 else close
        lo = float(window.min())
        hi = float(window.max())
        adverse_step = max(1e-6, (hi - lo) / 20.0)

    level = int(runtime.get("martingale_level") or 0)
    entry_price = runtime.get("entry_price")
    direction = runtime.get("direction") or 0  # 1=long, -1=short, 0=idle

    # First entry: buy on simple momentum (up-close).
    if level == 0 or direction == 0 or entry_price is None:
        prev = float(close[-2]) if len(close) >= 2 else price
        if price > prev:
            runtime["martingale_level"] = 1
            runtime["entry_price"] = price
            runtime["direction"] = 1
            return {"signal": "buy", "symbol": symbol, "volume": base_lot, "confidence": 0.55, "risk_level": "high", "reason": "马丁首单做多"}
        return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "low", "reason": "等待首单触发"}

    entry_price = float(entry_price)

    # Reset on profit (price moved favorably by one adverse_step worth).
    if direction == 1 and price >= entry_price + adverse_step:
        runtime["martingale_level"] = 0
        runtime["entry_price"] = None
        runtime["direction"] = 0
        return {"signal": "close", "symbol": symbol, "confidence": 0.7, "risk_level": "high", "reason": "盈利离场，重置马丁"}

    # Adverse move: add to position if not at cap.
    if direction == 1 and price <= entry_price - adverse_step * level:
        if level >= max_levels:
            # Cap reached: force close to avoid blow-up.
            runtime["martingale_level"] = 0
            runtime["entry_price"] = None
            runtime["direction"] = 0
            return {"signal": "close", "symbol": symbol, "confidence": 0.8, "risk_level": "high", "reason": "达到最大层数，强制平仓"}
        add_lot = base_lot * (multiplier ** level)
        runtime["martingale_level"] = level + 1
        return {"signal": "buy", "symbol": symbol, "volume": add_lot, "confidence": 0.6, "risk_level": "high", "reason": f"第{level+1}层加仓"}

    return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "high", "reason": "持仓观察"}`,
		},
		{
			Name:        "定投策略 (DCA)",
			Description: "定期买入，长期平摊成本（依赖 bar_time_ms 与 runtime 持久化）",
			IsPublic:    true,
			Code: `def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or ""
    runtime = context.get("runtime") if isinstance(context.get("runtime"), dict) else {}

    interval_hours = int(params.get("interval_hours", 168))
    lot = float(params.get("lot", 0.01))

    now_ms = context.get("bar_time_ms")
    if now_ms is None:
        return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "low", "reason": "缺少 bar_time_ms"}

    last_ms = runtime.get("last_dca_buy_ms")
    if last_ms is None:
        runtime["last_dca_buy_ms"] = int(now_ms)
        return {"signal": "buy", "symbol": symbol, "volume": lot, "confidence": 0.6, "risk_level": "low", "reason": "首次定投"}

    if int(now_ms) - int(last_ms) >= interval_hours * 3600 * 1000:
        runtime["last_dca_buy_ms"] = int(now_ms)
        return {"signal": "buy", "symbol": symbol, "volume": lot, "confidence": 0.6, "risk_level": "low", "reason": "定投间隔到达"}

    return {"signal": "hold", "symbol": symbol, "confidence": 0.0, "risk_level": "low", "reason": "等待下次定投"}`,
		},
	}
}
