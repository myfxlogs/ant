package backtest

import (
	"math"
	"sort"
)

// Trade is a completed backtest trade with entry and exit prices.
type Trade struct {
	EntryPrice   float64
	ExitPrice    float64
	Volume       float64
	Direction    int // 1=buy/long, -1=sell/short
	GrossPnL     float64
	NetPnL       float64
	Commission   float64
	Swap         float64
	Slippage     float64
	EntryTime    int64 // unix ms
	ExitTime     int64 // unix ms
}

// Metrics holds standard backtest performance statistics.
type Metrics struct {
	TotalReturn   float64
	AnnualReturn  float64
	MaxDrawdown   float64
	SharpeRatio   float64
	WinRate       float64
	ProfitFactor  float64
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	AverageProfit float64
	AverageLoss   float64
	GrossPnL      float64
	NetPnL        float64
	TotalCosts    float64 // sum of commission+swap+slippage
}

// ComputeMetrics calculates standard backtest metrics from a list of trades.
func ComputeMetrics(trades []Trade, initialCapital float64, equityCurve []float64) *Metrics {
	m := &Metrics{TotalTrades: len(trades)}
	if len(trades) == 0 {
		return m
	}

	totalProfit := 0.0
	totalLoss := 0.0

	for _, t := range trades {
		m.GrossPnL += t.GrossPnL
		m.NetPnL += t.NetPnL
		m.TotalCosts += t.Commission + t.Swap + t.Slippage

		if t.NetPnL > 0 {
			m.WinningTrades++
			totalProfit += t.NetPnL
		} else if t.NetPnL < 0 {
			m.LosingTrades++
			totalLoss += -t.NetPnL
		}
	}

	if initialCapital > 0 {
		m.TotalReturn = m.NetPnL / initialCapital
	}

	if m.TotalTrades > 0 {
		m.WinRate = float64(m.WinningTrades) / float64(m.TotalTrades)
	}
	if m.WinningTrades > 0 {
		m.AverageProfit = totalProfit / float64(m.WinningTrades)
	}
	if m.LosingTrades > 0 {
		m.AverageLoss = totalLoss / float64(m.LosingTrades)
	}
	if totalLoss > 0 {
		m.ProfitFactor = totalProfit / totalLoss
	} else if totalProfit > 0 {
		m.ProfitFactor = math.Inf(1)
	}

	m.MaxDrawdown = computeMaxDrawdown(equityCurve)
	m.AnnualReturn = m.TotalReturn // caller should annualize with actual trading days
	m.SharpeRatio = computeSharpe(equityCurve, initialCapital)

	return m
}

func computeMaxDrawdown(equity []float64) float64 {
	if len(equity) < 2 {
		return 0
	}
	peak := equity[0]
	maxDD := 0.0
	for _, e := range equity {
		if e > peak {
			peak = e
		}
		dd := (peak - e) / peak
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

func computeSharpe(equity []float64, initialCapital float64) float64 {
	if len(equity) < 3 || initialCapital <= 0 {
		return 0
	}
	returns := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		if equity[i-1] > 0 {
			returns[i-1] = (equity[i] - equity[i-1]) / equity[i-1]
		}
	}
	meanRet := 0.0
	for _, r := range returns {
		meanRet += r
	}
	meanRet /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - meanRet) * (r - meanRet)
	}
	variance /= float64(len(returns) - 1)
	if variance <= 0 {
		return 0
	}
	// Sharpe ratio (assuming risk-free rate = 0)
	return meanRet / math.Sqrt(variance) * math.Sqrt(252)
}

// SortTradesByEntryTime sorts trades in chronological order.
func SortTradesByEntryTime(trades []Trade) {
	sort.Slice(trades, func(i, j int) bool {
		return trades[i].EntryTime < trades[j].EntryTime
	})
}
