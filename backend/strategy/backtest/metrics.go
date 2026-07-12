package backtest

import (
	"math"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// CalculateMetrics produces antv1.BacktestMetrics from equity curve and trades.
// Reuses the existing proto type defined in strategy_runtime.proto.
func CalculateMetrics(initialCapital decimal.Decimal, equity []EquityPoint, trades []Trade) *antv1.BacktestMetrics {
	m := &antv1.BacktestMetrics{
		TotalTrades: int32(len(trades)),
	}

	if len(equity) < 2 || !initialCapital.IsPositive() {
		return m
	}

	initial := equity[0].Equity
	final := equity[len(equity)-1].Equity

	// Total return
	if initial.IsPositive() {
		tr, _ := final.Sub(initial).Div(initial).Float64()
		m.TotalReturn = tr
		// Annualize: (1 + total_return)^(365/days) - 1
		duration := equity[len(equity)-1].Time.Sub(equity[0].Time)
		days := duration.Hours() / 24
		if days > 0 {
			m.AnnualReturn = math.Pow(1+tr, 365.0/days) - 1
		} else {
			m.AnnualReturn = tr
		}
	}

	// Max drawdown
	peak := initial
	var maxDD decimal.Decimal
	for _, e := range equity {
		if e.Equity.GreaterThan(peak) {
			peak = e.Equity
		}
		dd := peak.Sub(e.Equity)
		if dd.GreaterThan(maxDD) {
			maxDD = dd
		}
	}
	if peak.IsPositive() {
		m.MaxDrawdown, _ = maxDD.Div(peak).Float64()
	}

	// Trade analysis
	var totalProfit, totalLoss decimal.Decimal
	var totalCommission, totalSwap decimal.Decimal
	for _, t := range trades {
		totalCommission = totalCommission.Add(t.Commission)
		totalSwap = totalSwap.Add(t.Swap)
		// Net profit = gross profit - commission - swap
		netProfit := t.Profit.Sub(t.Commission).Sub(t.Swap)
		if netProfit.IsPositive() {
			m.WinningTrades++
			totalProfit = totalProfit.Add(netProfit)
		} else {
			m.LosingTrades++
			totalLoss = totalLoss.Add(netProfit.Neg())
		}
	}
	_ = totalCommission
	_ = totalSwap

	if m.TotalTrades > 0 {
		m.WinRate = float64(m.WinningTrades) / float64(m.TotalTrades)
	}

	// Profit factor
	if totalLoss.IsPositive() {
		m.ProfitFactor, _ = totalProfit.Div(totalLoss).Float64()
	} else if totalProfit.IsPositive() {
		m.ProfitFactor = math.Inf(1)
	}

	// Average profit/loss
	if m.WinningTrades > 0 {
		avgProfit := totalProfit.Div(decimal.NewFromInt(int64(m.WinningTrades)))
		m.AverageProfit = avgProfit.String()
	}
	if m.LosingTrades > 0 {
		avgLoss := totalLoss.Div(decimal.NewFromInt(int64(m.LosingTrades)))
		m.AverageLoss = avgLoss.String()
	}

	// Sharpe ratio (annualized: based on trade returns, scaled by trades per year)
	if len(trades) > 1 {
		returns := make([]float64, len(trades))
		for i, t := range trades {
			returns[i] = t.ProfitPct
		}
		mean := meanFloat(returns)
		std := stdFloat(returns, mean)
		if std > 0 {
			// Annualize: Sharpe = mean/std * sqrt(trades_per_year)
			duration := equity[len(equity)-1].Time.Sub(equity[0].Time)
			days := duration.Hours() / 24
			if days > 0 {
				tradesPerYear := float64(len(trades)) * 365.0 / days
				m.SharpeRatio = mean / std * math.Sqrt(tradesPerYear)
			} else {
				m.SharpeRatio = mean / std * math.Sqrt(float64(len(trades)))
			}
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
