package backtest

import (
	"math"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// safeDecimal converts a float64 to decimal, returning "0" for NaN/Inf.
// This prevents the panic "Cannot create a Decimal from NaN" when metrics
// computation produces non-finite values (e.g., std=0 → sharpe=NaN).
func safeDecimal(f float64) decimal.Decimal {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(f)
}

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
		m.TotalReturn = safeDecimal(tr).String()
		// Annualize: (1 + total_return)^(365/days) - 1
		duration := equity[len(equity)-1].Time.Sub(equity[0].Time)
		days := duration.Hours() / 24
		var annualReturn float64
		if days > 0 {
			annualReturn = math.Pow(1+tr, 365.0/days) - 1
		} else {
			annualReturn = tr
		}
		m.AnnualReturn = safeDecimal(annualReturn).String()
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
		m.MaxDrawdown = maxDD.Div(peak).String()
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
		m.WinRate = safeDecimal(float64(m.WinningTrades) / float64(m.TotalTrades)).String()
	}

	// Profit factor
	if totalLoss.IsPositive() {
		m.ProfitFactor = totalProfit.Div(totalLoss).String()
	} else if totalProfit.IsPositive() {
		m.ProfitFactor = "999" // no losing trades — use large finite value, not "Infinity"
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
		if std > 0 && !math.IsNaN(mean) && !math.IsInf(mean, 0) {
			// Annualize: Sharpe = mean/std * sqrt(trades_per_year)
			duration := equity[len(equity)-1].Time.Sub(equity[0].Time)
			days := duration.Hours() / 24
			var sharpe float64
			if days > 0 {
				tradesPerYear := float64(len(trades)) * 365.0 / days
				sharpe = mean / std * math.Sqrt(tradesPerYear)
			} else {
				sharpe = mean / std * math.Sqrt(float64(len(trades)))
			}
			m.SharpeRatio = safeDecimal(sharpe).String()
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
