package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (s *AnalyticsService) GetTradeSummary(ctx context.Context, userID, accountID uuid.UUID, start, end time.Time) (*model.TradeStats, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, err
	}

	records, err := s.analyticsRepo.GetTradeRecords(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	stats := s.calculateTradeStatsFromRecords(records)

	maxWins, maxLosses, err := s.analyticsRepo.GetConsecutiveStats(ctx, accountID, start, end)
	if err == nil {
		stats.MaxConsecutiveWins = maxWins
		stats.MaxConsecutiveLosses = maxLosses
	}

	avgHoldingSecs, err := s.analyticsRepo.GetHoldingTimeStats(ctx, accountID, start, end)
	if err == nil && avgHoldingSecs > 0 {
		stats.AverageHoldingTime = s.formatDuration(avgHoldingSecs)
	}

	return stats, nil
}

func (s *AnalyticsService) GetRiskMetrics(ctx context.Context, userID, accountID uuid.UUID, start, end time.Time) (*model.RiskMetrics, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, err
	}

	metrics := &model.RiskMetrics{}

	maxDD, maxDDPercent, err := s.analyticsRepo.GetMaxDrawdown(ctx, accountID, start, end)
	if err == nil {
		metrics.MaxDrawdown = maxDD
		metrics.MaxDrawdownPercent = maxDDPercent
	}

	dailyReturns, err := s.analyticsRepo.GetDailyReturns(ctx, accountID, start, end)
	if err == nil && len(dailyReturns) > 0 {
		metrics.AverageDailyReturn = s.mean(dailyReturns)
		metrics.ValueAtRisk95 = s.percentile(dailyReturns, 5)
		metrics.ExpectedShortfall = s.expectedShortfall(dailyReturns, 5)

		if len(dailyReturns) > 1 {
			metrics.ReturnStdDev = s.stdDev(dailyReturns)
			metrics.Volatility = metrics.ReturnStdDev * 252

			if metrics.ReturnStdDev > 0 {
				metrics.SharpeRatio = (metrics.AverageDailyReturn * 252) / metrics.ReturnStdDev
			}

			downsideReturns := s.filterNegative(dailyReturns)
			if len(downsideReturns) > 1 {
				downsideStd := s.stdDev(downsideReturns)
				if downsideStd > 0 {
					metrics.SortinoRatio = (metrics.AverageDailyReturn * 252) / downsideStd
				}
			}
		}

		if metrics.MaxDrawdownPercent > 0 {
			metrics.CalmarRatio = (metrics.AverageDailyReturn * 252) / (metrics.MaxDrawdownPercent / 100)
		}
	}

	return metrics, nil
}

func (s *AnalyticsService) GetSymbolStats(ctx context.Context, userID, accountID uuid.UUID, start, end time.Time) ([]*model.SymbolStats, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, err
	}

	stats, err := s.analyticsRepo.GetSymbolStats(ctx, accountID, start, end)
	if err != nil {
		return nil, err
	}

	for _, stat := range stats {
		if stat.TotalLoss > 0 {
			stat.ProfitFactor = stat.TotalProfit / stat.TotalLoss
		} else if stat.TotalProfit > 0 {
			stat.ProfitFactor = 999.99
		}
		if stat.TotalTrades > 0 {
			stat.AverageProfit = stat.NetProfit / float64(stat.TotalTrades)
			stat.AverageVolume = stat.TotalVolume / float64(stat.TotalTrades)
		}
	}

	return stats, nil
}

func (s *AnalyticsService) GetTradeReport(ctx context.Context, userID, accountID uuid.UUID, start, end time.Time) (*model.TradeReport, error) {
	if _, err := s.verifyAccountAccess(ctx, userID, accountID); err != nil {
		return nil, err
	}

	report := &model.TradeReport{
		AccountID: accountID.String(),
		StartDate: start.Format("2006-01-02"),
		EndDate:   end.Format("2006-01-02"),
	}

	tradeStats, err := s.GetTradeSummary(ctx, userID, accountID, start, end)
	if err != nil {
		return nil, err
	}
	report.TradeStats = *tradeStats

	riskMetrics, err := s.GetRiskMetrics(ctx, userID, accountID, start, end)
	if err != nil {
		return nil, err
	}
	report.RiskMetrics = *riskMetrics

	symbolStats, err := s.GetSymbolStats(ctx, userID, accountID, start, end)
	if err != nil {
		return nil, err
	}
	report.SymbolStats = symbolStats

	dailyEquity, err := s.analyticsRepo.GetDailyEquity(ctx, accountID, start, end)
	if err == nil {
		report.DailyEquity = dailyEquity
		report.EquityCurve = make([]float64, len(dailyEquity))
		report.DrawdownCurve = make([]float64, len(dailyEquity))
		peak := 0.0
		for i, de := range dailyEquity {
			report.EquityCurve[i] = de.Equity
			if de.Equity > peak {
				peak = de.Equity
			}
			if peak > 0 {
				report.DrawdownCurve[i] = (peak - de.Equity) / peak * 100
			}
		}
	}

	return report, nil
}
