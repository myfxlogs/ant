package backtest

import (
	"math"

	"github.com/shopspring/decimal"
)

// CalculateMetrics computes performance statistics from equity curve and trades.
func CalculateMetrics(equity []EquityPoint, trades []Trade) Metrics {
	m := Metrics{TotalTrades: len(trades)}

	if len(equity) < 2 {
		return m
	}

	initial := equity[0].Equity
	final := equity[len(equity)-1].Equity

	if initial.IsPositive() {
		m.TotalReturn = final.Sub(initial).Div(initial)
	}

	// Max drawdown
	peak := initial
	for _, e := range equity {
		if e.Equity.GreaterThan(peak) {
			peak = e.Equity
		}
		dd := peak.Sub(e.Equity)
		if dd.GreaterThan(m.MaxDrawdown) {
			m.MaxDrawdown = dd
		}
	}
	if peak.IsPositive() {
		m.MaxDrawdownPct = m.MaxDrawdown.Div(peak)
	}

	// Trade analysis
	var totalProfit, totalLoss decimal.Decimal
	var sumWin, sumLoss decimal.Decimal
	for _, t := range trades {
		if t.Profit.IsPositive() {
			m.WinningTrades++
			totalProfit = totalProfit.Add(t.Profit)
			sumWin = sumWin.Add(t.Profit)
		} else {
			m.LosingTrades++
			totalLoss = totalLoss.Add(t.Profit.Neg())
			sumLoss = sumLoss.Add(t.Profit.Neg())
		}
		if t.Profit.GreaterThan(m.BestTrade) {
			m.BestTrade = t.Profit
		}
		if t.Profit.LessThan(m.WorstTrade) {
			m.WorstTrade = t.Profit
		}
	}

	if m.TotalTrades > 0 {
		m.WinRate = float64(m.WinningTrades) / float64(m.TotalTrades)
	}

	if m.WinningTrades > 0 {
		m.AvgWin = sumWin.Div(decimal.NewFromInt(int64(m.WinningTrades)))
	}
	if m.LosingTrades > 0 {
		m.AvgLoss = sumLoss.Div(decimal.NewFromInt(int64(m.LosingTrades)))
	}

	// Profit factor
	if totalLoss.IsPositive() {
		pf, exact := totalProfit.Div(totalLoss).Float64()
		if exact {
			m.ProfitFactor = pf
		}
		m.ProfitFactor = pf
	} else if totalProfit.IsPositive() {
		m.ProfitFactor = math.Inf(1)
	}

	// Sharpe ratio (simplified: based on trade returns)
	if m.TotalTrades > 1 {
		returns := make([]float64, len(trades))
		for i, t := range trades {
			returns[i] = t.ProfitPct
		}
		mean := meanFloat(returns)
		std := stdFloat(returns, mean)
		if std > 0 {
			m.SharpeRatio = mean / std * math.Sqrt(float64(m.TotalTrades))
		}
	}

	return m
}

func meanFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func stdFloat(vals []float64, mean float64) float64 {
	if len(vals) < 2 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		d := v - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(vals)-1))
}
