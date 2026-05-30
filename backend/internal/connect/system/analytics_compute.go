package system

import (
	"fmt"
	"math"
	"strings"
	"time"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

// --- compute functions ---

func computeTradeStats(trades []*repository.TradeRecord) *model.TradeStats {
	s := &model.TradeStats{}
	if len(trades) == 0 {
		return s
	}

	var totalProfit, totalLoss float64
	var winCount, lossCount int
	var sumHoldingSeconds float64
	var holdingCount int

	for _, t := range trades {
		// Deposit/withdrawal records are meta-transactions, not trades
		if isBalanceType(t.OrderType) {
			if t.Profit > 0 {
				s.TotalDeposit += t.Profit
			} else {
				s.TotalWithdrawal += math.Abs(t.Profit)
			}
			continue
		}

		s.TotalTrades++
		s.TotalVolume += t.Volume
		s.NetProfit += t.Profit

		if t.Profit > 0 {
			winCount++
			totalProfit += t.Profit
			if t.Profit > s.LargestWin {
				s.LargestWin = t.Profit
			}
		} else if t.Profit < 0 {
			lossCount++
			totalLoss += math.Abs(t.Profit)
			if math.Abs(t.Profit) > s.LargestLoss {
				s.LargestLoss = math.Abs(t.Profit)
			}
		}

		if !t.OpenTime.IsZero() && !t.CloseTime.IsZero() {
			sumHoldingSeconds += t.CloseTime.Sub(t.OpenTime).Seconds()
			holdingCount++
		}
	}

	s.WinningTrades = winCount
	s.LosingTrades = lossCount
	s.TotalProfit = totalProfit
	s.TotalLoss = totalLoss

	if s.TotalTrades > 0 {
		s.WinRate = float64(winCount) / float64(s.TotalTrades) * 100
		s.AverageTrade = s.NetProfit / float64(s.TotalTrades)
	}
	if winCount > 0 {
		s.AverageProfit = totalProfit / float64(winCount)
	}
	if lossCount > 0 {
		s.AverageLoss = totalLoss / float64(lossCount)
	}
	if totalLoss > 0 {
		s.ProfitFactor = totalProfit / totalLoss
	}
	if holdingCount > 0 {
		avgSec := sumHoldingSeconds / float64(holdingCount)
		s.AverageHoldingTime = formatDuration(avgSec)
	}

	s.NetDeposit = s.TotalDeposit - s.TotalWithdrawal

	return s
}

func isBalanceType(orderType string) bool {
	t := strings.ToLower(orderType)
	return t == "balance" || t == "credit"
}

func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.0fm", seconds/60)
	}
	if seconds < 86400 {
		h := seconds / 3600
		return fmt.Sprintf("%.1fh", h)
	}
	d := seconds / 86400
	return fmt.Sprintf("%.1fd", d)
}

// dailyReturnsToPercent computes daily percentage returns from the equity curve.
// Formula: pct[i] = equityCurve[i].Profit / equityCurve[i-1].Equity
// Uses the equity curve directly because GetDailyReturns returns dollar amounts
// that don't align day-by-day with the equity timeline.
//
// #25: Zero-profit days are intentionally excluded — days with zero profit indicate
// no trading activity, and including them would dilute daily-return-based risk
// metrics (Sharpe, Sortino, Calmar ratios) by adding zero-return periods that
// do not represent actual trading performance.
func dailyReturnsToPercent(equityCurve []*model.EquityPoint) []float64 {
	if len(equityCurve) < 2 {
		return nil
	}
	result := make([]float64, 0, len(equityCurve)-1)
	for i := 1; i < len(equityCurve); i++ {
		prev := equityCurve[i-1].Equity
		if prev <= 0 {
			continue
		}
		profit := equityCurve[i].Profit
		// #25: Zero-profit days are excluded intentionally (see doc comment above).
		if profit == 0 {
			continue
		}
		result = append(result, profit/prev)
	}
	return result
}

func computeRiskMetrics(dailyReturns []float64, maxDDPercent float64) (sharpe, sortino, calmar, volatility, avgDailyReturn float64) {
	if len(dailyReturns) == 0 {
		return 0, 0, 0, 0, 0
	}

	// Average daily return
	sum := 0.0
	for _, r := range dailyReturns {
		sum += r
	}
	avgDailyReturn = sum / float64(len(dailyReturns))

	// Std dev
	sumSq := 0.0
	for _, r := range dailyReturns {
		sumSq += (r - avgDailyReturn) * (r - avgDailyReturn)
	}
	variance := sumSq / float64(len(dailyReturns))
	volatility = math.Sqrt(variance)

	// Sharpe (assuming 0 risk-free rate, annualized with 252 trading days)
	if volatility > 0 {
		sharpe = (avgDailyReturn / volatility) * math.Sqrt(252)
	}

	// Sortino (downside deviation only)
	downSq := 0.0
	downCount := 0
	for _, r := range dailyReturns {
		if r < 0 {
			downSq += r * r
			downCount++
		}
	}
	if downCount > 0 {
		downStd := math.Sqrt(downSq / float64(downCount))
		if downStd > 0 {
			sortino = (avgDailyReturn / downStd) * math.Sqrt(252)
		}
	}

	// Calmar
	if maxDDPercent > 0 {
		annualizedReturn := avgDailyReturn * 252
		calmar = annualizedReturn / maxDDPercent
	}

	return
}

// --- converters ---

func tradeStatsToProto(s *model.TradeStats) *antv1.TradeStats {
	if s == nil {
		return nil
	}
	return &antv1.TradeStats{
		TotalTrades:          int64(s.TotalTrades),
		WinRate:              s.WinRate,
		ProfitFactor:         s.ProfitFactor,
		AverageProfit:        s.AverageProfit,
		AverageLoss:          s.AverageLoss,
		LargestWin:           s.LargestWin,
		LargestLoss:          s.LargestLoss,
		MaxConsecutiveWins:   int64(s.MaxConsecutiveWins),
		MaxConsecutiveLosses: int64(s.MaxConsecutiveLosses),
		AverageHoldingTime:   s.AverageHoldingTime,
		NetProfit:            s.NetProfit,
		TotalDeposit:         s.TotalDeposit,
		TotalWithdrawal:      s.TotalWithdrawal,
		NetDeposit:           s.NetDeposit,
	}
}

func symbolStatsToProto(stats []*model.SymbolStats) []*antv1.SymbolStat {
	result := make([]*antv1.SymbolStat, 0, len(stats))

	// calculate total trades for share percentage
	var totalTrades int
	for _, s := range stats {
		totalTrades += s.TotalTrades
	}

	for _, s := range stats {
		tradeSharePct := 0.0
		if totalTrades > 0 {
			tradeSharePct = float64(s.TotalTrades) / float64(totalTrades) * 100.0
		}
		result = append(result, &antv1.SymbolStat{
			Symbol:            s.Symbol,
			Profit:            s.NetProfit,
			TradeSharePercent: tradeSharePct,
		})
	}
	return result
}

func dailyPnLToProto(items []*model.DailyPnL) []*antv1.DailyPnL {
	result := make([]*antv1.DailyPnL, 0, len(items))
	for _, d := range items {
		result = append(result, &antv1.DailyPnL{
			Day:                    d.Day,
			Date:                   d.Date,
			Pnl:                    d.PnL,
			Trades:                 int64(d.Trades),
			Lots:                   d.Lots,
			Balance:                d.Balance,
			ProfitFactor:           d.ProfitFactor,
			MaxFloatingLossAmount:  d.MaxFloatingLossAmount,
			MaxFloatingLossRatio:   d.MaxFloatingLossRatio,
			MaxFloatingProfitAmount: d.MaxFloatingProfitAmount,
			MaxFloatingProfitRatio: d.MaxFloatingProfitRatio,
		})
	}
	return result
}

// #26: Check CloseTime is not zero before formatting to avoid "0001-01-01" output.
func tradeRecordToProto(r *model.TradeRecord) *antv1.TradeRecord {
	closeTimeStr := ""
	if !r.CloseTime.IsZero() {
		closeTimeStr = r.CloseTime.Format(time.RFC3339)
	}
	openTimeStr := ""
	if !r.OpenTime.IsZero() {
		openTimeStr = r.OpenTime.Format(time.RFC3339)
	}
	return &antv1.TradeRecord{
		Ticket:     fmt.Sprintf("%d", r.Ticket),
		Symbol:     r.Symbol,
		Type:       r.OrderType,
		Volume:     r.Volume,
		OpenPrice:  r.OpenPrice,
		ClosePrice: r.ClosePrice,
		Profit:     r.Profit,
		OpenTime:   openTimeStr,
		CloseTime:  closeTimeStr,
		Swap:       r.Swap,
		Commission: r.Commission,
		Comment:    r.OrderComment,
	}
}

