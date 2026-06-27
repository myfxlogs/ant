// Package ai — equity curve to daily returns conversion.
package ai

import (
	"sort"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
)

// EquityCurveToDailyReturns extracts daily bar-level equity diffs from the
// proto binary ExecuteBacktestResponse.  Falls back to DailyReturnsFromTrades
// when the equity_curve field has fewer than 2 points.
func EquityCurveToDailyReturns(protoResp []byte) []float64 {
	if len(protoResp) == 0 {
		return nil
	}
	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(protoResp, &resp); err != nil {
		return nil
	}
	equity := resp.GetEquityCurve()
	if len(equity) >= 2 {
		return equityDiffs(equity)
	}
	return DailyReturnsFromTrades(resp.GetTrades())
}

func equityDiffs(equity []float64) []float64 {
	rets := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		rets[i-1] = equity[i] - equity[i-1]
	}
	return rets
}

// DailyReturnsFromTrades reconstructs daily P&L from closed trades,
// grouped by close date (UTC).  Returns nil when no trades are available.
func DailyReturnsFromTrades(trades []*antv1.ExecuteBacktestTrade) []float64 {
	if len(trades) == 0 {
		return nil
	}
	dayPnL := make(map[string]float64)
	for _, t := range trades {
		if t.CloseTsMs == 0 {
			continue
		}
		day := time.UnixMilli(t.CloseTsMs).UTC().Format("2006-01-02")
		dayPnL[day] += parseFloat(t.Pnl)
	}
	if len(dayPnL) == 0 {
		return nil
	}
	days := make([]string, 0, len(dayPnL))
	for d := range dayPnL {
		days = append(days, d)
	}
	sort.Strings(days)
	rets := make([]float64, len(days))
	for i, d := range days {
		rets[i] = dayPnL[d]
	}
	return rets
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
