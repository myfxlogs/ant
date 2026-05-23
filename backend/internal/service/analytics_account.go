package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (s *AnalyticsService) GetAccountAnalytics(ctx context.Context, userID, accountID uuid.UUID, start, end time.Time, year int) (*model.AccountAnalytics, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, err
	}

	analytics := &model.AccountAnalytics{}

	tradeStats, err := s.GetTradeSummary(ctx, userID, accountID, start, end)
	if err == nil {
		analytics.TradeStats = tradeStats
	}

	riskMetrics, err := s.GetRiskMetrics(ctx, userID, accountID, start, end)
	if err == nil {
		analytics.RiskMetrics = riskMetrics
	}

	symbolStats, err := s.GetSymbolStats(ctx, userID, accountID, start, end)
	if err == nil {
		analytics.SymbolStats = symbolStats
	}

	monthlyPnL, err := s.analyticsRepo.GetMonthlyPnL(ctx, accountID, year)
	if err == nil {
		analytics.MonthlyPnL = monthlyPnL
	}

	dailyPnL, err := s.analyticsRepo.GetDailyPnL(ctx, accountID, start, end)
	if err == nil {
		analytics.DailyPnL = dailyPnL
	}

	hourlyStats, err := s.analyticsRepo.GetHourlyStats(ctx, accountID, start, end)
	if err == nil {
		analytics.HourlyStats = hourlyStats
	}

	weekdayPnL, err := s.analyticsRepo.GetWeekdayPnL(ctx, accountID, start, end)
	if err == nil {
		analytics.WeekdayPnL = weekdayPnL
	}

	equityCurve, err := s.analyticsRepo.GetEquityCurve(ctx, accountID, start, end)
	if err == nil {
		analytics.EquityCurve = equityCurve
	}

	recentTrades, err := s.analyticsRepo.GetTradeRecordsWithLimit(ctx, accountID, start, end, 20)
	if err == nil {
		analytics.RecentTrades = recentTrades
	}

	return analytics, nil
}

func (s *AnalyticsService) GetRecentTrades(ctx context.Context, userID, accountID uuid.UUID, limit int) ([]*model.TradeRecord, int, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, 0, err
	}

	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	total, err := s.analyticsRepo.GetTradeRecordsCount(ctx, accountID, start, end)
	if err != nil {
		return nil, 0, err
	}

	records, err := s.analyticsRepo.GetTradeRecordsWithLimit(ctx, accountID, start, end, limit)
	if err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (s *AnalyticsService) GetRecentTradesPaginated(ctx context.Context, userID, accountID uuid.UUID, page, pageSize int) ([]*model.TradeRecord, int, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, 0, err
	}

	end := time.Now()
	start := end.AddDate(-1, 0, 0)

	return s.analyticsRepo.GetTradeRecordsPaginated(ctx, accountID, start, end, page, pageSize)
}

func (s *AnalyticsService) GetMonthlyPnL(ctx context.Context, userID, accountID uuid.UUID, year int) ([]*model.MonthlyPnL, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, err
	}

	return s.analyticsRepo.GetMonthlyPnL(ctx, accountID, year)
}
