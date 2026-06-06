package system

import (
	"github.com/shopspring/decimal"
	"context"
	"fmt"
	"math"
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
		if cached, err := s.cache.Get(ctx, req.Msg.AccountId); err == nil && cached != nil {
			return connect.NewResponse(cached), nil
		}
	}

	now := time.Now()
	start := now.AddDate(-1, 0, 0)

	tradeStats, err := s.fetchTradeStats(ctx, accountID, start, now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get trade records: %w", err))
	}

	sharpe, sortino, calmar, volatility, avgDailyReturn, maxDDPercent, err := s.fetchRiskMetrics(ctx, accountID, start, now)
	if err != nil {
		return nil, err
	}

	symbolStats, _ := s.repo.GetSymbolStats(ctx, accountID, start, now)

	equityCurve := s.fetchEquityCurve(ctx, req, accountID, start, now)

	resp := &antv1.AccountAnalyticsResponse{
		TradeStats:  tradeStatsToProto(tradeStats),
		RiskMetrics: riskMetricsToProto(sharpe, sortino, calmar, volatility, avgDailyReturn, maxDDPercent),
		SymbolStats: symbolStatsToProto(symbolStats),
		EquityCurve: equityCurveToProto(equityCurve),
		DailyPnl:    dailyPnLToProto(s.fetchDailyPnL(ctx, accountID, start, now)),
		HourlyStats: hourlyStatsToProto(s.fetchHourlyStats(ctx, accountID, start, now)),
	}

	if s.cache != nil {
		if err := s.cache.Set(ctx, req.Msg.AccountId, resp); err != nil {
			s.log.Warn("analytics cache: set failed", zap.Error(err))
		}
	}

	return connect.NewResponse(resp), nil
}

func (s *AnalyticsServer) fetchTradeStats(ctx context.Context, accountID uuid.UUID, start, now time.Time) (*model.TradeStats, error) {
	trades, err := s.repo.GetTradeRecords(ctx, accountID, start, now)
	if err != nil { return nil, err }
	tradeStats := computeTradeStats(trades)
	maxWins, maxLosses, err := s.repo.GetConsecutiveStats(ctx, accountID, start, now)
	if err != nil {
		s.log.Warn("get consecutive stats failed", zap.Error(err))
	} else {
		tradeStats.MaxConsecutiveWins = maxWins
		tradeStats.MaxConsecutiveLosses = maxLosses
	}
	return tradeStats, nil
}

func (s *AnalyticsServer) fetchRiskMetrics(ctx context.Context, accountID uuid.UUID, start, now time.Time) (float64, float64, float64, float64, float64, float64, error) {
	_, maxDDPercent, err := s.repo.GetMaxDrawdown(ctx, accountID, start, now)
	if err != nil {
		s.log.Error("get max drawdown failed", zap.Error(err))
		return 0, 0, 0, 0, 0, 0, connect.NewError(connect.CodeInternal, fmt.Errorf("get max drawdown: %w", err))
	}
	eqFull, err := s.repo.GetEquityCurve(ctx, accountID, start, now)
	if err != nil {
		s.log.Error("get equity curve for risk metrics failed", zap.Error(err))
		return 0, 0, 0, 0, 0, 0, connect.NewError(connect.CodeInternal, fmt.Errorf("get equity curve: %w", err))
	}
	dailyReturnPct := dailyReturnsToPercent(eqFull)
	sharpe, sortino, calmar, volatility, avgDailyReturn := computeRiskMetrics(dailyReturnPct, maxDDPercent)
	return sharpe, sortino, calmar, volatility, avgDailyReturn, maxDDPercent, nil
}

func (s *AnalyticsServer) fetchEquityCurve(ctx context.Context, req *connect.Request[antv1.GetAccountAnalyticsRequest], accountID uuid.UUID, start, now time.Time) []*model.EquityPoint {
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
	var err error
	if useHourly {
		equityCurve, err = s.repo.GetHourlyEquityCurve(ctx, accountID, eqStart, now)
	} else {
		equityCurve, err = s.repo.GetEquityCurve(ctx, accountID, eqStart, now)
	}
	if err != nil {
		s.log.Warn("get equity curve failed", zap.Error(err))
	} else {
		equityCurve = appendLiveEquity(ctx, s.repo, accountID, equityCurve)
	}
	return equityCurve
}

func (s *AnalyticsServer) fetchDailyPnL(ctx context.Context, accountID uuid.UUID, start, now time.Time) []*model.DailyPnL {
	pnl, _ := s.repo.GetDailyPnL(ctx, accountID, start, now)
	return pnl
}

func (s *AnalyticsServer) fetchHourlyStats(ctx context.Context, accountID uuid.UUID, start, now time.Time) []*model.HourlyStats {
	stats, _ := s.repo.GetHourlyStats(ctx, accountID, start, now)
	return stats
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
			Equity:  p.Equity.InexactFloat64(),
			Balance: p.Balance.InexactFloat64(),
			Profit:  p.Profit.InexactFloat64(),
		})
	}
	return result
}

func hourlyStatsToProto(stats []*model.HourlyStats) []*antv1.HourlyStat {
	result := make([]*antv1.HourlyStat, 0, len(stats))
	for _, h := range stats {
		result = append(result, &antv1.HourlyStat{
			Hour:                   int32(h.HourStart),
			Lots:                   h.Lots.InexactFloat64(),
			Balance:                h.Balance.InexactFloat64(),
			ProfitFactor:           h.ProfitFactor.InexactFloat64(),
			MaxFloatingLossAmount:  h.MaxFloatingLossAmount,
			MaxFloatingLossRatio:   h.MaxFloatingLossRatio,
			MaxFloatingProfitAmount: h.MaxFloatingProfitAmount,
			MaxFloatingProfitRatio: h.MaxFloatingProfitRatio,
		})
	}
	return result
}

// appendLiveEquity updates the last equity-curve point with the current live
// balance/equity/profit from mt_accounts so the chart always reaches the
// real-time value.  Essential for newly-bound accounts that have few historical
// snapshots in account_balance_history.
func appendLiveEquity(ctx context.Context, repo *repository.AnalyticsRepository, accountID uuid.UUID, curve []*model.EquityPoint) []*model.EquityPoint {
	live, err := repo.GetCurrentAccountMetrics(ctx, accountID)
	if err != nil || live == nil {
		return curve
	}
	today := time.Now().Format("2006-01-02")

	if len(curve) > 0 && curve[len(curve)-1].Date == today {
		curve[len(curve)-1].Equity = decimal.NewFromFloat(live.Equity)
		curve[len(curve)-1].Balance = decimal.NewFromFloat(live.Balance)
		curve[len(curve)-1].Profit = decimal.NewFromFloat(live.Profit)
	} else {
		curve = append(curve, &model.EquityPoint{
			Date:    today,
			Equity:  decimal.NewFromFloat(math.Round(live.Equity*100) / 100),
			Balance: decimal.NewFromFloat(math.Round(live.Balance*100) / 100),
			Profit:  decimal.NewFromFloat(math.Round(live.Profit*100) / 100),
		})
	}
	return curve
}
