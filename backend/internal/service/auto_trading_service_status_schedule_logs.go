package service

import (
	"context"

	"anttrader/internal/model"

	"github.com/google/uuid"
)

func (s *AutoTradingService) GetAutoTradingStatus(ctx context.Context, userID uuid.UUID) (*model.AutoTradingStatus, error) {
	status := &model.AutoTradingStatus{}

	settings, err := s.GetGlobalSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	status.GlobalEnabled = settings.AutoTradeEnabled

	strategies, err := s.strategyRepo.GetActiveByUserID(ctx, userID)
	if err == nil {
		status.ActiveStrategies = len(strategies)
	}

	signals, err := s.strategyRepo.GetSignalsByStatus(ctx, model.SignalStatusPending, 100)
	if err == nil {
		pendingCount := 0
		for _, sig := range signals {
			if sig.TemplateID != uuid.Nil {
				strat, stratErr := s.strategyRepo.GetByID(ctx, sig.TemplateID)
				if stratErr == nil && strat.UserID == userID {
					pendingCount++
				}
			}
		}
		status.PendingSignals = pendingCount
	}

	accounts, err := s.accountRepo.GetByUserID(ctx, userID)
	if err == nil && len(accounts) > 0 {
		todayExecutions := 0
		todayProfit := 0.0
		for _, acc := range accounts {
			count, _ := s.autoTradingRepo.GetTodayExecutionCount(ctx, acc.ID)
			todayExecutions += count
			profit, _ := s.autoTradingRepo.GetTodayProfit(ctx, acc.ID)
			todayProfit += profit
		}
		status.TodayExecutions = todayExecutions
		status.TodayProfit = todayProfit
	}

	return status, nil
}

func (s *AutoTradingService) GetTradingLogs(ctx context.Context, userID uuid.UUID, params *model.LogListParams) ([]*model.TradingLog, int, error) {
	return s.autoTradingRepo.GetTradingLogs(ctx, userID, params)
}

func (s *AutoTradingService) GetRecentTradingLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*model.TradingLog, error) {
	return s.autoTradingRepo.GetRecentTradingLogs(ctx, userID, limit)
}
