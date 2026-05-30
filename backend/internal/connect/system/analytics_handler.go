package system

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type AnalyticsServer struct {
	repo *repository.AnalyticsRepository
	log  *zap.Logger
}

var _ antv1c.AnalyticsServiceHandler = (*AnalyticsServer)(nil)

func NewAnalyticsServer(repo *repository.AnalyticsRepository, log *zap.Logger) *AnalyticsServer {
	return &AnalyticsServer{repo: repo, log: log}
}

func (s *AnalyticsServer) GetAccountAnalytics(ctx context.Context, req *connect.Request[antv1.GetAccountAnalyticsRequest]) (*connect.Response[antv1.AccountAnalyticsResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0) // last 12 months

	// Trade stats from raw records
	trades, err := s.repo.GetTradeRecords(ctx, accountID, start, now)
	if err != nil {
		return nil, fmt.Errorf("get trade records: %w", err)
	}

	tradeStats := computeTradeStats(trades)

	// Consecutive wins/losses (SQL window-function — handler must call separately)
	maxWins, maxLosses, err := s.repo.GetConsecutiveStats(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("get consecutive stats failed", zap.Error(err))
	} else {
		tradeStats.MaxConsecutiveWins = maxWins
		tradeStats.MaxConsecutiveLosses = maxLosses
	}

	// Risk metrics — computed from daily percentage returns (not dollar amounts)
	_, maxDDPercent, err := s.repo.GetMaxDrawdown(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("get max drawdown failed", zap.Error(err))
	}
	// Get equity curve for computing daily percentage returns.
	// Uses full 12-month equity (not period-filtered) for stable risk metrics.
	eqFull, err := s.repo.GetEquityCurve(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("get equity curve for risk metrics failed", zap.Error(err))
	}
	dailyReturnPct := dailyReturnsToPercent(eqFull)
	sharpe, sortino, calmar, volatility, avgDailyReturn := computeRiskMetrics(dailyReturnPct, maxDDPercent)

	// Symbol stats
	symbolStats, err := s.repo.GetSymbolStats(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("get symbol stats failed", zap.Error(err))
	}

	// Equity curve — period-specific time window
	eqStart := start
	useHourly := false
	switch req.Msg.EquityCurvePeriod {
	case antv1.EquityCurvePeriod_EQUITY_CURVE_PERIOD_DAY:
		eqStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		useHourly = true
	case antv1.EquityCurvePeriod_EQUITY_CURVE_PERIOD_WEEK:
		eqStart = now.AddDate(0, 0, -7)
	case antv1.EquityCurvePeriod_EQUITY_CURVE_PERIOD_MONTH:
		eqStart = now.AddDate(0, 0, -30)
	}
	var equityCurve []*model.EquityPoint
	if useHourly {
		equityCurve, err = s.repo.GetHourlyEquityCurve(ctx, accountID, eqStart, now)
	} else {
		equityCurve, err = s.repo.GetEquityCurve(ctx, accountID, eqStart, now)
	}
	if err != nil {
		s.log.Warn("get equity curve failed", zap.Error(err))
	}
	// Daily PnL
	dailyPnL, err := s.repo.GetDailyPnL(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("get daily pnl failed", zap.Error(err))
	}

	// Hourly stats
	hourlyStats, err := s.repo.GetHourlyStats(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("get hourly stats failed", zap.Error(err))
	}

	return connect.NewResponse(&antv1.AccountAnalyticsResponse{
		TradeStats:  tradeStatsToProto(tradeStats),
		RiskMetrics: riskMetricsToProto(sharpe, sortino, calmar, volatility, avgDailyReturn, maxDDPercent),
		SymbolStats: symbolStatsToProto(symbolStats),
		EquityCurve: equityCurveToProto(equityCurve),
		DailyPnl:    dailyPnLToProto(dailyPnL),
		HourlyStats: hourlyStatsToProto(hourlyStats),
	}), nil
}

func (s *AnalyticsServer) GetRecentTrades(ctx context.Context, req *connect.Request[antv1.GetRecentTradesRequest]) (*connect.Response[antv1.GetRecentTradesResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}

	page := int(req.Msg.Page)
	pageSize := int(req.Msg.PageSize)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	start := time.Now().AddDate(-1, 0, 0)
	end := time.Now()

	records, total, err := s.repo.GetTradeRecordsPaginated(ctx, accountID, start, end, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("get trade records paginated: %w", err)
	}

	protoTrades := make([]*antv1.TradeRecord, 0, len(records))
	for _, r := range records {
		protoTrades = append(protoTrades, tradeRecordToProto(r))
	}

	return connect.NewResponse(&antv1.GetRecentTradesResponse{
		Trades: protoTrades,
		Total:  int64(total),
	}), nil
}

func (s *AnalyticsServer) GetMonthlyPnL(ctx context.Context, req *connect.Request[antv1.GetMonthlyPnLRequest]) (*connect.Response[antv1.GetMonthlyPnLResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}

	year := int(req.Msg.Year)
	if year <= 0 {
		year = time.Now().Year()
	}

	monthlyData, err := s.repo.GetMonthlyPnL(ctx, accountID, year)
	if err != nil {
		return nil, fmt.Errorf("get monthly pnl: %w", err)
	}

	items := make([]*antv1.MonthlyPnLItem, 0, len(monthlyData))
	for _, m := range monthlyData {
		items = append(items, &antv1.MonthlyPnLItem{
			Month:  int32(m.MonthNum),
			Profit: m.Profit,
			Trades: int64(m.Trades),
		})
	}

	return connect.NewResponse(&antv1.GetMonthlyPnLResponse{
		MonthlyPnl: items,
	}), nil
}

func (s *AnalyticsServer) GetMonthlyAnalysis(ctx context.Context, req *connect.Request[antv1.GetMonthlyAnalysisRequest]) (*connect.Response[antv1.GetMonthlyAnalysisResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}

	years, err := s.repo.GetMonthlyAnalysisYears(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get monthly analysis years: %w", err)
	}

	points, err := s.repo.GetMonthlyAnalysisRaw(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get monthly analysis raw: %w", err)
	}

	data, err := json.Marshal(monthlyAnalysisToJSON(points))
	if err != nil {
		return nil, fmt.Errorf("marshal monthly analysis: %w", err)
	}

	protoYears := make([]int32, len(years))
	for i, y := range years {
		protoYears[i] = int32(y)
	}

	return connect.NewResponse(&antv1.GetMonthlyAnalysisResponse{
		Years: protoYears,
		Data:  data,
	}), nil
}

// --- converters ---

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
	switch orderType {
	case "balance", "credit", "BALANCE", "CREDIT", "Balance", "Credit":
		return true
	}
	return false
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
		if profit == 0 {
			continue // skip no-trade days
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

func riskMetricsToProto(sharpe, sortino, calmar, volatility, avgDailyReturn, maxDDPercent float64) *antv1.RiskMetrics {
	return &antv1.RiskMetrics{
		MaxDrawdownPercent: maxDDPercent,
		SharpeRatio:        sharpe,
		SortinoRatio:       sortino,
		CalmarRatio:        calmar,
		Volatility:         volatility,
		AverageDailyReturn: avgDailyReturn,
	}
}

func symbolStatsToProto(stats []*model.SymbolStats) []*antv1.SymbolStat {
	result := make([]*antv1.SymbolStat, 0, len(stats))

	// 计算总交易笔数，用于计算占比
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

func equityCurveToProto(points []*model.EquityPoint) []*antv1.EquityPoint {
	result := make([]*antv1.EquityPoint, 0, len(points))
	for _, p := range points {
		result = append(result, &antv1.EquityPoint{
			Date:    p.Date,
			Equity:  p.Equity,
			Balance: p.Balance,
			Profit:  p.Profit,
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

func hourlyStatsToProto(stats []*model.HourlyStats) []*antv1.HourlyStat {
	result := make([]*antv1.HourlyStat, 0, len(stats))
	for _, h := range stats {
		result = append(result, &antv1.HourlyStat{
			Hour:                   int32(h.HourStart),
			Lots:                   h.Lots,
			Balance:                h.Balance,
			ProfitFactor:           h.ProfitFactor,
			MaxFloatingLossAmount:  h.MaxFloatingLossAmount,
			MaxFloatingLossRatio:   h.MaxFloatingLossRatio,
			MaxFloatingProfitAmount: h.MaxFloatingProfitAmount,
			MaxFloatingProfitRatio: h.MaxFloatingProfitRatio,
		})
	}
	return result
}

func tradeRecordToProto(r *model.TradeRecord) *antv1.TradeRecord {
	return &antv1.TradeRecord{
		Ticket:     fmt.Sprintf("%d", r.Ticket),
		Symbol:     r.Symbol,
		Type:       r.OrderType,
		Volume:     r.Volume,
		OpenPrice:  r.OpenPrice,
		ClosePrice: r.ClosePrice,
		Profit:     r.Profit,
		OpenTime:   r.OpenTime.Format(time.RFC3339),
		CloseTime:  r.CloseTime.Format(time.RFC3339),
		Swap:       r.Swap,
		Commission: r.Commission,
		Comment:    r.OrderComment,
	}
}

func monthlyAnalysisToJSON(points []*model.MonthlyAnalysisPoint) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(points))
	for _, p := range points {
		result = append(result, map[string]interface{}{
			"year":   p.Year,
			"month":  p.Month,
			"profit": p.Profit,
			"lots":   p.Lots,
			"pips":   p.Pips,
			"trades": p.Trades,
			"change": p.Change,
		})
	}
	return result
}
