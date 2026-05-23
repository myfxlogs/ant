package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"anttrader/internal/event"
	"anttrader/internal/model"
	"anttrader/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *AutoTradingService) ExecuteSignal(ctx context.Context, userID uuid.UUID, signal *model.StrategySignal) (*model.ExecutionResult, error) {
	settings, err := s.GetGlobalSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !settings.AutoTradeEnabled {
		return nil, ErrAutoTradingDisabled
	}

	strategy, err := s.strategyRepo.GetByID(ctx, signal.TemplateID)
	if err != nil {
		return nil, err
	}

	if !strategy.AutoExecute {
		return nil, ErrAutoTradingDisabled
	}

	account, err := s.accountRepo.GetByID(ctx, signal.AccountID)
	if err != nil {
		return nil, err
	}

	positions, err := s.engine.GetPositions(ctx, userID, signal.AccountID)
	if err != nil {
		logger.Warn("Failed to get positions for risk check", zap.Error(err))
	}

	riskResult, err := s.CheckRiskLimits(ctx, &model.RiskCheckRequest{
		AccountID:      signal.AccountID,
		Symbol:         signal.Symbol,
		Volume:         signal.Volume,
		CurrentBalance: account.Balance,
		CurrentEquity:  account.Equity,
		OpenPositions:  len(positions),
	})
	if err != nil {
		return nil, err
	}

	if !riskResult.Allowed {
		return nil, fmt.Errorf("%w: %s", ErrRiskLimitExceeded, riskResult.Reason)
	}

	execution := model.NewStrategyExecution(signal.UserID, signal.TemplateID, signal.AccountID)
	if err := s.autoTradingRepo.CreateExecution(ctx, execution); err != nil {
		return nil, err
	}

	defer func() {
		if err := s.autoTradingRepo.UpdateExecution(ctx, execution); err != nil {
			logger.Error("Failed to update execution", zap.Error(err))
		}
	}()

	var orderReq *OrderSendRequest
	switch signal.SignalType {
	case model.SignalTypeBuy:
		orderReq = &OrderSendRequest{
			AccountID:  signal.AccountID.String(),
			Symbol:     signal.Symbol,
			Type:       "buy",
			Volume:     signal.Volume,
			StopLoss:   signal.StopLoss,
			TakeProfit: signal.TakeProfit,
		}
	case model.SignalTypeSell:
		orderReq = &OrderSendRequest{
			AccountID:  signal.AccountID.String(),
			Symbol:     signal.Symbol,
			Type:       "sell",
			Volume:     signal.Volume,
			StopLoss:   signal.StopLoss,
			TakeProfit: signal.TakeProfit,
		}
	case model.SignalTypeClose:
		return s.executeCloseSignal(ctx, userID, signal, execution)
	default:
		return nil, errors.New("invalid signal type")
	}

	callCtx := WithTradeTriggerSource(ctx, TriggerSourceStrategy)
	orderResp, err := s.engine.OrderSend(callCtx, userID, orderReq)
	if s.gateway != nil {
		op := model.NewSystemOperationLog(userID, model.OperationTypeCreate, "execution", "auto_trading_execute_signal")
		op.ResourceType = "execution"
		op.ResourceID = execution.ID
		op.Status = model.OperationStatusRunning
		orderResp, err = s.gateway.OrderSend(callCtx, userID, orderReq, op)
	}
	if err != nil {
		execution.Status = model.ExecutionStatusFailed
		execution.ErrorMessage = err.Error()
		now := time.Now()
		execution.CompletedAt = &now
		return nil, err
	}

	execution.Status = model.ExecutionStatusCompleted
	now := time.Now()
	execution.CompletedAt = &now

	result := &model.ExecutionResult{
		Ticket: orderResp.Ticket,
	}

	s.logExecution(ctx, userID, signal.AccountID, signal.TemplateID, execution.ID, signal.SignalType, signal.Symbol, signal.Volume, orderResp.Price, orderResp.Ticket, 0)

	return result, nil
}

func (s *AutoTradingService) executeCloseSignal(ctx context.Context, userID uuid.UUID, signal *model.StrategySignal, execution *model.StrategyExecution) (*model.ExecutionResult, error) {
	if signal.Ticket == 0 {
		return nil, errors.New("ticket is required for close signal")
	}

	closeReq := &OrderCloseRequest{
		AccountID:   signal.AccountID.String(),
		Ticket:      signal.Ticket,
		Volume:      signal.Volume,
		CloseReason: "strategy_exit",
	}

	callCtx := WithTradeTriggerSource(ctx, TriggerSourceStrategy)
	orderResp, err := s.engine.OrderClose(callCtx, userID, closeReq)
	if s.gateway != nil {
		op := model.NewSystemOperationLog(userID, model.OperationTypeUpdate, "execution", "auto_trading_close")
		op.ResourceType = "execution"
		op.ResourceID = execution.ID
		op.Status = model.OperationStatusRunning
		orderResp, err = s.gateway.OrderClose(callCtx, userID, closeReq, op)
	}
	if err != nil {
		execution.Status = model.ExecutionStatusFailed
		execution.ErrorMessage = err.Error()
		now := time.Now()
		execution.CompletedAt = &now
		return nil, err
	}

	execution.Status = model.ExecutionStatusCompleted
	now := time.Now()
	execution.CompletedAt = &now

	result := &model.ExecutionResult{
		Ticket: orderResp.Ticket,
		Profit: orderResp.Profit,
	}

	s.logExecution(ctx, userID, signal.AccountID, signal.TemplateID, execution.ID, "close", signal.Symbol, signal.Volume, orderResp.Price, orderResp.Ticket, orderResp.Profit)

	return result, nil
}

func (s *AutoTradingService) logExecution(ctx context.Context, userID, accountID, templateID, executionID uuid.UUID, action, symbol string, volume, price float64, ticket int64, profit float64) {
	log := model.NewTradingLog(userID, model.LogTypeTrade, action, symbol, fmt.Sprintf("Auto executed: %s", action))
	log.AccountID = accountID
	log.Volume = volume
	log.Price = price
	log.Ticket = ticket
	log.Profit = profit

	if err := s.autoTradingRepo.CreateTradingLog(ctx, log); err != nil {
		logger.Error("Failed to create trading log", zap.Error(err))
	}

	if s.eventBus != nil {
		s.eventBus.Publish(accountID.String(), &event.Event{
			Type:      event.EventStrategyExecution,
			AccountID: accountID.String(),
			Data: &event.StrategyExecutionData{
				TemplateID:  templateID.String(),
				ExecutionID: executionID.String(),
				AccountID:   accountID.String(),
				Status:      model.ExecutionStatusCompleted,
				Symbol:      symbol,
				Action:      action,
				Ticket:      ticket,
				Volume:      fmt.Sprintf("%.2f", volume),
				Price:       fmt.Sprintf("%.5f", price),
				Profit:      fmt.Sprintf("%.2f", profit),
			},
		})
	}
}
