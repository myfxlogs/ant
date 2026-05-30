package system

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type AnalyticsServer struct {
	repo     *repository.AnalyticsRepository
	platform *service.PlatformService
	cache    *service.AnalyticsCache
	log      *zap.Logger
}

var _ antv1c.AnalyticsServiceHandler = (*AnalyticsServer)(nil)

func NewAnalyticsServer(repo *repository.AnalyticsRepository, platform *service.PlatformService, cache *service.AnalyticsCache, log *zap.Logger) *AnalyticsServer {
	return &AnalyticsServer{repo: repo, platform: platform, cache: cache, log: log}
}

// verifyAccountOwnership extracts userID and checks account ownership (#19).
func (s *AnalyticsServer) verifyAccountOwnership(ctx context.Context, accountID string) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	ok, err := s.platform.UserOwnsAccount(ctx, userID, accountID)
	if err != nil {
		s.log.Error("verifyAccountOwnership: check failed", zap.String("accountId", accountID), zap.Error(err))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("ownership check failed: %w", err))
	}
	if !ok {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account does not belong to user"))
	}
	return nil
}

func (s *AnalyticsServer) GetAccountAnalytics(ctx context.Context, req *connect.Request[antv1.GetAccountAnalyticsRequest]) (*connect.Response[antv1.AccountAnalyticsResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	// #19: Verify account ownership.
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	// Check analytics cache — return immediately on hit, bypassing all 7 SQL queries.
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, req.Msg.AccountId); err == nil {
			return connect.NewResponse(cached), nil
		}
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0) // last 12 months

	// Trade stats from raw records
	trades, err := s.repo.GetTradeRecords(ctx, accountID, start, now)
	if err != nil {
		// #20: Use connect.NewError instead of raw fmt.Errorf.
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get trade records: %w", err))
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

	// Risk metrics — computed from daily percentage returns (not dollar amounts).
	// #23: Propagate critical errors (getMaxDrawdown, getEquityCurve) instead of silently zeroing.
	_, maxDDPercent, err := s.repo.GetMaxDrawdown(ctx, accountID, start, now)
	if err != nil {
		s.log.Error("get max drawdown failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get max drawdown: %w", err))
	}
	eqFull, err := s.repo.GetEquityCurve(ctx, accountID, start, now)
	if err != nil {
		s.log.Error("get equity curve for risk metrics failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get equity curve: %w", err))
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

	resp := &antv1.AccountAnalyticsResponse{
		TradeStats:  tradeStatsToProto(tradeStats),
		RiskMetrics: riskMetricsToProto(sharpe, sortino, calmar, volatility, avgDailyReturn, maxDDPercent),
		SymbolStats: symbolStatsToProto(symbolStats),
		EquityCurve: equityCurveToProto(equityCurve),
		DailyPnl:    dailyPnLToProto(dailyPnL),
		HourlyStats: hourlyStatsToProto(hourlyStats),
	}

	// Cache the computed response so subsequent requests skip SQL queries.
	if s.cache != nil {
		if err := s.cache.Set(ctx, req.Msg.AccountId, resp); err != nil {
			s.log.Warn("analytics cache: set failed", zap.Error(err))
		}
	}

	return connect.NewResponse(resp), nil
}

func (s *AnalyticsServer) GetRecentTrades(ctx context.Context, req *connect.Request[antv1.GetRecentTradesRequest]) (*connect.Response[antv1.GetRecentTradesResponse], error) {
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	// #19: Verify account ownership.
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
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
		// #20: Use connect.NewError.
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get trade records paginated: %w", err))
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
	// #19: Verify account ownership.
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	year := int(req.Msg.Year)
	if year <= 0 {
		year = time.Now().Year()
	}

	monthlyData, err := s.repo.GetMonthlyPnL(ctx, accountID, year)
	if err != nil {
		// #20: Use connect.NewError.
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get monthly pnl: %w", err))
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
	// #19: Verify account ownership.
	if err := s.verifyAccountOwnership(ctx, req.Msg.AccountId); err != nil {
		return nil, err
	}

	years, err := s.repo.GetMonthlyAnalysisYears(ctx, accountID)
	if err != nil {
		// #20: Use connect.NewError.
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get monthly analysis years: %w", err))
	}

	points, err := s.repo.GetMonthlyAnalysisRaw(ctx, accountID)
	if err != nil {
		// #20: Use connect.NewError.
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get monthly analysis raw: %w", err))
	}

	data, err := json.Marshal(monthlyAnalysisToJSON(points))
	if err != nil {
		// #20: Use connect.NewError.
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal monthly analysis: %w", err))
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

// --- converters (proto mapping) ---

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
