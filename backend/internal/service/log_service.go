package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type LogService struct {
	logRepo *repository.LogRepository
}

func NewLogService(logRepo *repository.LogRepository) *LogService {
	return &LogService{logRepo: logRepo}
}

func (s *LogService) LogConnection(ctx context.Context, log *model.AccountConnectionLog) error {
	return s.logRepo.CreateConnectionLog(ctx, log)
}

func (s *LogService) GetConnectionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.AccountConnectionLog, int, error) {
	return s.logRepo.GetConnectionLogs(ctx, userID, params)
}

func (s *LogService) LogExecution(ctx context.Context, log *model.StrategyExecutionLog) error {
	return s.logRepo.CreateExecutionLog(ctx, log)
}

func (s *LogService) UpdateExecution(ctx context.Context, log *model.StrategyExecutionLog) error {
	return s.logRepo.UpdateExecutionLog(ctx, log)
}

func (s *LogService) GetExecutionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.StrategyExecutionLog, int, error) {
	return s.logRepo.GetExecutionLogs(ctx, userID, params)
}

func (s *LogService) LogOrder(ctx context.Context, order *model.OrderHistory) error {
	return s.logRepo.CreateOrderHistory(ctx, order)
}

// UpdateOrderHistoryClose updates the open row for a schedule ticket after a successful close.
func (s *LogService) UpdateOrderHistoryClose(ctx context.Context, userID, accountID, scheduleID uuid.UUID, ticket int64, closePrice, profit, swap, commission float64, closeTime time.Time) (int64, error) {
	if s == nil || s.logRepo == nil {
		return 0, nil
	}
	return s.logRepo.UpdateOrderHistoryClose(ctx, userID, accountID, scheduleID, ticket, closePrice, profit, swap, commission, closeTime)
}

func (s *LogService) GetOrderHistory(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.OrderHistory, int, error) {
	return s.logRepo.GetOrderHistory(ctx, userID, params)
}

func (s *LogService) LogOperation(ctx context.Context, log *model.SystemOperationLog) error {
	return s.logRepo.CreateOperationLog(ctx, log)
}

func (s *LogService) GetOperationLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.SystemOperationLog, int, error) {
	return s.logRepo.GetOperationLogs(ctx, userID, params)
}

func (s *LogService) GetScheduleRunLogs(ctx context.Context, userID uuid.UUID, scheduleID uuid.UUID, page, pageSize int) ([]*repository.ScheduleRunLogRow, int, error) {
	return s.logRepo.GetScheduleRunLogs(ctx, userID, scheduleID, page, pageSize)
}

func (s *LogService) GetAllLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	connLogs, connTotal, err := s.logRepo.GetConnectionLogs(ctx, userID, params)
	if err != nil {
		return nil, err
	}
	result["connection_logs"] = connLogs
	result["connection_total"] = connTotal

	execLogs, execTotal, err := s.logRepo.GetExecutionLogs(ctx, userID, params)
	if err != nil {
		return nil, err
	}
	result["execution_logs"] = execLogs
	result["execution_total"] = execTotal

	orders, orderTotal, err := s.logRepo.GetOrderHistory(ctx, userID, params)
	if err != nil {
		return nil, err
	}
	result["order_history"] = orders
	result["order_total"] = orderTotal

	opLogs, opTotal, err := s.logRepo.GetOperationLogs(ctx, userID, params)
	if err != nil {
		return nil, err
	}
	result["operation_logs"] = opLogs
	result["operation_total"] = opTotal

	return result, nil
}
