// share_service_metrics.go — Metrics computation helpers extracted from share_service.go.
package user

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"

	"alphaforge/internal/model"
)

func computeMaxDrawdownPct(equityPoints []*model.EquityPoint) decimal.Decimal {
	if len(equityPoints) < 2 {
		return decimal.Zero
	}
	var maxDD decimal.Decimal
	runningPeak := equityPoints[0].Equity
	for _, p := range equityPoints[1:] {
		if p.Equity.GreaterThan(runningPeak) {
			runningPeak = p.Equity
		}
		if runningPeak.GreaterThan(decimal.Zero) {
			dd := runningPeak.Sub(p.Equity).Div(runningPeak).Mul(decimal.NewFromInt(100))
			if dd.GreaterThan(maxDD) {
				maxDD = dd
			}
		}
	}
	return maxDD
}

// aggregateSymbolStats groups trades by symbol and computes count + net profit.
// REUSE: aggregation pattern from analytics_compute.go:228/258 (by-symbol grouping).
func aggregateSymbolStats(trades []*model.TradeRecord) []symbolStatPayload {
	if len(trades) == 0 {
		return nil
	}
	type acc struct {
		count int
		net   decimal.Decimal
	}
	m := make(map[string]*acc)
	order := make([]string, 0, len(trades))
	for _, t := range trades {
		sym := t.Symbol
		if sym == "" {
			sym = "-"
		}
		if _, ok := m[sym]; !ok {
			m[sym] = &acc{}
			order = append(order, sym)
		}
		m[sym].count++
		m[sym].net = m[sym].net.Add(t.Profit)
	}
	out := make([]symbolStatPayload, 0, len(order))
	for _, sym := range order {
		out = append(out, symbolStatPayload{
			Symbol: sym,
			Count:  m[sym].count,
			Net:    m[sym].net.String(),
		})
	}
	return out
}

// computeSharpe calculates annualized Sharpe ratio from equity curve points.
func computeSharpe(equityPoints []*model.EquityPoint) float64 {
	if len(equityPoints) < 2 {
		return 0
	}
	var sum, sumSq float64
	var returns []float64
	for i := 1; i < len(equityPoints); i++ {
		prev, ok1 := equityPoints[i-1].Equity.Float64()
		if !ok1 || prev == 0 {
			continue
		}
		curr, ok2 := equityPoints[i].Equity.Float64()
		if !ok2 {
			continue
		}
		r := (curr - prev) / prev
		returns = append(returns, r)
		sum += r
	}
	if len(returns) < 2 {
		return 0
	}
	n := float64(len(returns))
	mean := sum / n
	for _, r := range returns {
		diff := r - mean
		sumSq += diff * diff
	}
	variance := sumSq / (n - 1)
	if variance <= 0 {
		return 0
	}
	std := math.Sqrt(variance)
	if std == 0 {
		return 0
	}
	return mean / std * math.Sqrt(252)
}

// fmtShareURL returns the share URL path for a token.
func fmtShareURL(token string) string {
	return fmt.Sprintf("/share/%s", token)
}
