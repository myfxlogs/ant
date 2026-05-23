package service

import (
	"context"
)

func ResolveBacktestCostModel(ctx context.Context, dynamicCfg *DynamicConfigService) BacktestCostModel {
	cost := DefaultBacktestCostModelFromEnv()
	if dynamicCfg == nil {
		return cost
	}

	if v, enabled, _ := dynamicCfg.GetFloat64(ctx, "backtest.commission_rate", cost.CommissionRate); enabled {
		cost.CommissionRate = v
	}
	if v, enabled, _ := dynamicCfg.GetFloat64(ctx, "backtest.spread_rate", cost.SpreadRate); enabled {
		cost.SpreadRate = v
	}
	if v, enabled, _ := dynamicCfg.GetFloat64(ctx, "backtest.swap_rate", cost.SwapRate); enabled {
		cost.SwapRate = v
	}
	if v, enabled, _ := dynamicCfg.GetString(ctx, "backtest.server_timezone", cost.ServerTimezone); enabled {
		cost.ServerTimezone = v
	}
	if v, enabled, _ := dynamicCfg.GetInt(ctx, "backtest.rollover_hour", cost.RolloverHour); enabled {
		cost.RolloverHour = v
	}
	if v, enabled, _ := dynamicCfg.GetInt(ctx, "backtest.triple_swap_weekday", cost.TripleSwapWeekday); enabled {
		cost.TripleSwapWeekday = v
	}
	if v, enabled, _ := dynamicCfg.GetString(ctx, "backtest.slippage_mode", cost.SlippageMode); enabled {
		cost.SlippageMode = v
	}
	if v, enabled, _ := dynamicCfg.GetFloat64(ctx, "backtest.slippage_rate", cost.SlippageRate); enabled {
		cost.SlippageRate = v
	}
	if v, enabled, _ := dynamicCfg.GetInt64(ctx, "backtest.slippage_seed", cost.SlippageSeed); enabled {
		cost.SlippageSeed = v
	}

	return cost
}
