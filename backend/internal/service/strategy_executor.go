package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

var (
	ErrSignalNotFound    = errors.New("signal not found")
	ErrSignalAlreadyExec = errors.New("signal already executed")
	ErrSignalNotPending  = errors.New("signal not pending")
)

type ExecutionResult struct {
	SignalID   uuid.UUID `json:"signal_id"`
	Ticket     int64     `json:"ticket"`
	Symbol     string    `json:"symbol"`
	Type       string    `json:"type"`
	Volume     float64   `json:"volume"`
	Price      float64   `json:"price"`
	Profit     float64   `json:"profit"`
	ExecutedAt time.Time `json:"executed_at"`
}

type StrategyExecutor struct {
	strategyRepo *repository.StrategyRepository
	engine       MatchingEngine
	gateway      *ExecutionGateway
	accountRepo  *repository.AccountRepository
	logService   *LogService
}

func NewStrategyExecutor(
	strategyRepo *repository.StrategyRepository,
	engine MatchingEngine,
	gateway *ExecutionGateway,
	accountRepo *repository.AccountRepository,
	logService *LogService,
) *StrategyExecutor {
	return &StrategyExecutor{
		strategyRepo: strategyRepo,
		engine:       engine,
		gateway:      gateway,
		accountRepo:  accountRepo,
		logService:   logService,
	}
}

func (e *StrategyExecutor) ExecuteSignal(ctx context.Context, userID uuid.UUID, signalID uuid.UUID) (*ExecutionResult, error) {
	signal, err := e.strategyRepo.GetSignalByID(ctx, signalID)
	if err != nil {
		return nil, ErrSignalNotFound
	}

	if signal.Status != model.SignalStatusPending && signal.Status != model.SignalStatusConfirmed {
		return nil, ErrSignalAlreadyExec
	}

	account, err := e.accountRepo.GetByID(ctx, signal.AccountID)
	if err != nil {
		return nil, err
	}

	if account.UserID != userID {
		return nil, errors.New("unauthorized access to account")
	}

	// 记录执行开始
	executionLog := model.NewStrategyExecutionLog(userID, signal.Symbol, "H1")
	executionLog.AccountID = &signal.AccountID
	executionLog.Status = model.StrategyExecutionStatusRunning
	executionLog.SignalType = model.StrategyExecutionLogSignalType(signal.SignalType)
	executionLog.SignalPrice = signal.Price
	executionLog.SignalVolume = signal.Volume
	executionLog.SignalStopLoss = signal.StopLoss
	executionLog.SignalTakeProfit = signal.TakeProfit

	if e.logService != nil {
		e.logService.LogExecution(ctx, executionLog)
	}

	orderReq := &OrderSendRequest{
		AccountID:  signal.AccountID.String(),
		Symbol:     signal.Symbol,
		Type:       signal.SignalType,
		Volume:     signal.Volume,
		Price:      signal.Price,
		StopLoss:   signal.StopLoss,
		TakeProfit: signal.TakeProfit,
		Comment:    signal.Reason,
	}

	startTime := time.Now()
	orderResp, err := e.engine.OrderSend(ctx, userID, orderReq)
	if e.gateway != nil {
		op := model.NewSystemOperationLog(userID, model.OperationTypeCreate, "execution", "strategy_execute_signal")
		op.ResourceType = "signal"
		op.ResourceID = signalID
		op.Status = model.OperationStatusRunning
		orderResp, err = e.gateway.OrderSend(ctx, userID, orderReq, op)
	}
	executionTime := time.Since(startTime).Milliseconds()

	if err != nil {
		logger.Error("Failed to execute signal",
			zap.String("signal_id", signalID.String()),
			zap.Error(err))

		// 记录执行失败
		if e.logService != nil {
			executionLog.Status = model.StrategyExecutionStatusFailed
			executionLog.ErrorMessage = err.Error()
			executionLog.ExecutionTimeMs = executionTime
			e.logService.UpdateExecution(ctx, executionLog)
		}

		return nil, err
	}

	// 记录执行成功
	if e.logService != nil {
		executionLog.Status = model.StrategyExecutionStatusCompleted
		executionLog.ExecutedOrderID = fmt.Sprintf("%d", orderResp.Ticket)
		executionLog.ExecutedPrice = orderResp.Price
		executionLog.ExecutionTimeMs = executionTime
		e.logService.UpdateExecution(ctx, executionLog)
	}

	if err := e.strategyRepo.UpdateSignalStatus(ctx, signalID, model.SignalStatusExecuted, orderResp.Ticket, 0); err != nil {
		logger.Error("Failed to update signal status",
			zap.String("signal_id", signalID.String()),
			zap.Error(err))
	}

	return &ExecutionResult{
		SignalID:   signalID,
		Ticket:     orderResp.Ticket,
		Symbol:     orderResp.Symbol,
		Type:       orderResp.Type,
		Volume:     orderResp.Volume,
		Price:      orderResp.Price,
		ExecutedAt: time.Now(),
	}, nil
}

func (e *StrategyExecutor) ConfirmSignal(ctx context.Context, userID uuid.UUID, signalID uuid.UUID) error {
	signal, err := e.strategyRepo.GetSignalByID(ctx, signalID)
	if err != nil {
		return ErrSignalNotFound
	}

	if signal.Status != model.SignalStatusPending {
		return ErrSignalNotPending
	}

	account, err := e.accountRepo.GetByID(ctx, signal.AccountID)
	if err != nil {
		return err
	}

	if account.UserID != userID {
		return errors.New("unauthorized access to account")
	}

	return e.strategyRepo.ConfirmSignal(ctx, signalID)
}

func (e *StrategyExecutor) CancelSignal(ctx context.Context, userID uuid.UUID, signalID uuid.UUID) error {
	signal, err := e.strategyRepo.GetSignalByID(ctx, signalID)
	if err != nil {
		return ErrSignalNotFound
	}

	account, err := e.accountRepo.GetByID(ctx, signal.AccountID)
	if err != nil {
		return err
	}

	if account.UserID != userID {
		return errors.New("unauthorized access to account")
	}

	return e.strategyRepo.CancelSignal(ctx, signalID)
}

func (e *StrategyExecutor) GetPendingSignals(ctx context.Context, userID uuid.UUID) ([]*model.StrategySignal, error) {
	accounts, err := e.accountRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var allSignals []*model.StrategySignal
	for _, account := range accounts {
		signals, err := e.strategyRepo.GetPendingSignals(ctx, account.ID)
		if err != nil {
			continue
		}
		allSignals = append(allSignals, signals...)
	}

	return allSignals, nil
}
