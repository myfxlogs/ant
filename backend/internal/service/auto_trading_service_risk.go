package service

import (
	"context"
	"errors"
	"math"

	"anttrader/internal/model"
	"anttrader/internal/repository"

	"github.com/google/uuid"
)

func (s *AutoTradingService) GetRiskConfig(ctx context.Context, userID, accountID uuid.UUID) (*model.RiskConfig, error) {
	if accountID != uuid.Nil {
		config, err := s.autoTradingRepo.GetRiskConfigByAccountID(ctx, accountID)
		if err == nil {
			return config, nil
		}
	}

	config, err := s.autoTradingRepo.GetRiskConfigByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrRiskConfigNotFound) {
			config = model.NewRiskConfig(userID, uuid.Nil)
			if err := s.autoTradingRepo.CreateRiskConfig(ctx, config); err != nil {
				return nil, err
			}
			return config, nil
		}
		return nil, err
	}
	return config, nil
}

func (s *AutoTradingService) UpdateRiskConfig(ctx context.Context, userID, accountID uuid.UUID, config *model.RiskConfig) error {
	config.UserID = userID
	config.AccountID = accountID

	if accountID != uuid.Nil {
		existing, err := s.autoTradingRepo.GetRiskConfigByAccountID(ctx, accountID)
		if err == nil {
			config.ID = existing.ID
			return s.autoTradingRepo.UpdateRiskConfig(ctx, config)
		}
	} else {
		existing, err := s.autoTradingRepo.GetRiskConfigByUserID(ctx, userID)
		if err == nil {
			config.ID = existing.ID
			return s.autoTradingRepo.UpdateRiskConfig(ctx, config)
		}
	}

	return s.autoTradingRepo.CreateRiskConfig(ctx, config)
}

func (s *AutoTradingService) CheckRiskLimits(ctx context.Context, req *model.RiskCheckRequest) (*model.RiskCheckResult, error) {
	result := &model.RiskCheckResult{
		Allowed:        true,
		IsWithinLimits: true,
		Decision:       model.AllowRiskDecision(model.RiskDecisionSourceAuto),
	}
	if req != nil {
		result.PositionCount = req.OpenPositions
	}
	if s == nil || s.accountRepo == nil || s.autoTradingRepo == nil || req == nil || req.AccountID == uuid.Nil {
		return result, nil
	}
	account, err := s.accountRepo.GetByID(ctx, req.AccountID)
	if err != nil || account == nil {
		return result, nil
	}
	riskConfig, err := s.GetRiskConfig(ctx, account.UserID, req.AccountID)
	if err != nil || riskConfig == nil {
		return result, nil
	}

	if account.IsDisabled {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_ACCOUNT_DISABLED", "account is disabled", false))
		return result, nil
	}
	if account.IsInvestor {
		result.SetDecision(model.RejectRiskDecision(model.RiskDecisionSourceAuto, "RISK_INVESTOR_ACCOUNT", "investor account cannot trade", false))
		return result, nil
	}
	return s.riskEngine.CheckAuto(req, riskConfig), nil
}

func (s *AutoTradingService) CalculatePositionSize(ctx context.Context, req *model.PositionSizingRequest) (*model.PositionSizingResult, error) {
	result := &model.PositionSizingResult{}

	if req.StopLossPips <= 0 {
		return nil, errors.New("stop loss pips must be positive")
	}

	riskPercent := req.RiskPercent
	if riskPercent <= 0 {
		riskPercent = 2.0
	}

	account, err := s.accountRepo.GetByID(ctx, req.AccountID)
	if err != nil {
		return nil, err
	}

	riskConfig, err := s.GetRiskConfig(ctx, account.UserID, req.AccountID)
	if err == nil && riskConfig.MaxRiskPercent > 0 {
		if riskPercent > riskConfig.MaxRiskPercent {
			riskPercent = riskConfig.MaxRiskPercent
		}
	}

	balance := req.AccountBalance
	if balance <= 0 {
		balance = account.Balance
	}

	result.RiskAmount = balance * (riskPercent / 100.0)

	pipValue := 10.0
	if req.Symbol != "" {
		if val, err := s.getPipValue(ctx, req.AccountID, req.Symbol); err == nil {
			pipValue = val
		}
	}
	result.PipValue = pipValue

	pipValuePerLot := pipValue
	result.Volume = result.RiskAmount / (req.StopLossPips * pipValuePerLot)

	result.MinVolume = 0.01
	result.MaxVolume = 100.0

	if riskConfig != nil && riskConfig.MaxLotSize > 0 {
		result.MaxVolume = riskConfig.MaxLotSize
	}

	if result.Volume > result.MaxVolume {
		result.Volume = result.MaxVolume
	}
	if result.Volume < result.MinVolume {
		result.Volume = result.MinVolume
	}

	volumeStep := 0.01
	result.Volume = math.Floor(result.Volume/volumeStep) * volumeStep

	return result, nil
}

func (s *AutoTradingService) getPipValue(ctx context.Context, accountID uuid.UUID, symbol string) (float64, error) {
	return 10.0, nil
}
