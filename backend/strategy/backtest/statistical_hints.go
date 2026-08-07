package backtest

import antv1 "alphaforge/gen/proto/ant/v1"

const severityHint = "提示"

// CheckZeroEquityVariance detects equity curves with zero variance — all equity points
// are identical, meaning the strategy never made or lost money. This is a statistical hint
// (not a fatal invariant): it could be a legitimate flat period, but combined with trades
// it usually indicates a bug (e.g. volume=0 producing zero PnL).
// Returns a BlindSpot with severity "提示" if detected, nil otherwise.
func CheckZeroEquityVariance(result *Result) *antv1.BlindSpot {
	if len(result.Equity) < 2 {
		return nil
	}
	first := result.Equity[0].Equity
	for _, ep := range result.Equity[1:] {
		if !ep.Equity.Equal(first) {
			return nil
		}
	}
	return &antv1.BlindSpot{
		Id:          "zero_equity_variance",
		Category:    "statistical",
		Severity:    severityHint,
		Description: "净值曲线零波动：所有净值点相同，策略可能未产生任何盈亏（常见于手数=0或未成交）",
	}
}

// CheckMonotonicPositionGrowth detects strategies where trade volume monotonically
// increases across consecutive trades — a hallmark of grid/martingale strategies.
// This is a statistical hint: such strategies are legitimate but carry hidden tail risk.
// Returns a BlindSpot with severity "提示" if detected, nil otherwise.
func CheckMonotonicPositionGrowth(trades []Trade) *antv1.BlindSpot {
	if len(trades) < 3 {
		return nil
	}
	increasingCount := 0
	for i := 1; i < len(trades); i++ {
		if trades[i].Volume.GreaterThan(trades[i-1].Volume) {
			increasingCount++
		}
	}
	if increasingCount == len(trades)-1 {
		return &antv1.BlindSpot{
			Id:          "monotonic_position_growth",
			Category:    "statistical",
			Severity:    severityHint,
			Description: "持仓单调增长：连续交易手数递增，疑似网格/马丁策略，存在隐藏尾部风险",
		}
	}
	return nil
}

// CheckAllSameDirectionSameVolume detects strategies where every trade has the same
// direction and the same volume — could be legitimate (DCA) but also indicates
// a strategy that ignores market conditions.
// Returns a BlindSpot with severity "提示" if detected, nil otherwise.
func CheckAllSameDirectionSameVolume(trades []Trade) *antv1.BlindSpot {
	if len(trades) < 3 {
		return nil
	}
	firstSide := trades[0].Side
	firstVolume := trades[0].Volume
	for _, t := range trades[1:] {
		if t.Side != firstSide || !t.Volume.Equal(firstVolume) {
			return nil
		}
	}
	return &antv1.BlindSpot{
		Id:          "all_same_direction_same_volume",
		Category:    "statistical",
		Severity:    severityHint,
		Description: "全同向同额：所有交易方向和手数相同，策略可能忽略市场条件变化",
	}
}

// CheckAbnormalTradeFrequency detects strategies with abnormal trading frequency —
// either too few trades (< 3 over the backtest period) or extremely high frequency
// (> 1000 trades per day, indicating a tick-scalping or loop bug).
// Returns a BlindSpot with severity "提示" if detected, nil otherwise.
func CheckAbnormalTradeFrequency(result *Result) *antv1.BlindSpot {
	if len(result.Trades) == 0 {
		return nil
	}
	span := result.Config.EndDate.Sub(result.Config.StartDate)
	if span <= 0 {
		return nil
	}
	days := span.Hours() / 24
	if days < 1 {
		days = 1
	}
	tradesPerDay := float64(len(result.Trades)) / days

	if tradesPerDay > 1000 {
		return &antv1.BlindSpot{
			Id:          "abnormal_trade_frequency_high",
			Category:    "statistical",
			Severity:    severityHint,
			Description: "交易频率异常：日均交易数>1000，疑似tick级循环或高频剥头皮，回测结果可能不反映真实滑点",
		}
	}
	return nil
}

// CheckStatisticalHints runs all ADR-0028 §4.2 statistical-class checks on a backtest result.
// Returns hints (not fatal) as BlindSpots. These do NOT affect IsReliable — they are
// advisory only, appended to the compatibility report.
func CheckStatisticalHints(result *Result) []*antv1.BlindSpot {
	var hints []*antv1.BlindSpot
	if bs := CheckZeroEquityVariance(result); bs != nil {
		hints = append(hints, bs)
	}
	if bs := CheckMonotonicPositionGrowth(result.Trades); bs != nil {
		hints = append(hints, bs)
	}
	if bs := CheckAllSameDirectionSameVolume(result.Trades); bs != nil {
		hints = append(hints, bs)
	}
	if bs := CheckAbnormalTradeFrequency(result); bs != nil {
		hints = append(hints, bs)
	}
	return hints
}
