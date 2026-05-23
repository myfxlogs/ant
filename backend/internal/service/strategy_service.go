package service

import (
"context"
"errors"
"fmt"

"github.com/google/uuid"
"go.uber.org/zap"

"anttrader/internal/model"
"anttrader/internal/repository"
"anttrader/pkg/logger"
)

var (
ErrStrategyNotFound     = errors.New("strategy not found")
ErrInvalidStrategyInput = errors.New("invalid strategy input")
ErrAccountNotSpecified  = errors.New("account not specified")
)

// StrategyService 策略服务（仅提供 CRUD，策略生成由 AI workflow 处理）
type StrategyService struct {
strategyRepo *repository.StrategyRepository
accountRepo  *repository.AccountRepository
}

// NewStrategyService 创建策略服务
func NewStrategyService(
strategyRepo *repository.StrategyRepository,
accountRepo *repository.AccountRepository,
) *StrategyService {
return &StrategyService{
strategyRepo: strategyRepo,
accountRepo:  accountRepo,
}
}

// SaveStrategy 保存策略
func (s *StrategyService) SaveStrategy(ctx context.Context, strategy *model.Strategy) error {
if strategy == nil {
return ErrInvalidStrategyInput
}

if strategy.AccountID != (uuid.UUID{}) {
account, err := s.accountRepo.GetByID(ctx, strategy.AccountID)
if err != nil {
logger.Error("Failed to verify account",
zap.Error(err),
zap.String("account_id", strategy.AccountID.String()))
return fmt.Errorf("account verification failed: %w", err)
}
if account.UserID != strategy.UserID {
logger.Error("Account does not belong to user",
zap.String("account_user_id", account.UserID.String()),
zap.String("strategy_user_id", strategy.UserID.String()))
return errors.New("account does not belong to user")
}
}

if err := s.strategyRepo.Create(ctx, strategy); err != nil {
logger.Error("Failed to save strategy",
zap.Error(err),
zap.String("strategy_id", strategy.ID.String()))
return fmt.Errorf("failed to save strategy: %w", err)
}

return nil
}

// GetUserStrategies 获取用户策略列表
func (s *StrategyService) GetUserStrategies(ctx context.Context, userID uuid.UUID) ([]*model.Strategy, error) {
strategies, err := s.strategyRepo.GetByUserID(ctx, userID)
if err != nil {
logger.Error("Failed to get user strategies",
zap.Error(err),
zap.String("user_id", userID.String()))
return nil, fmt.Errorf("failed to get strategies: %w", err)
}
return strategies, nil
}

// GetStrategyByID 根据ID获取策略
func (s *StrategyService) GetStrategyByID(ctx context.Context, strategyID uuid.UUID) (*model.Strategy, error) {
strategy, err := s.strategyRepo.GetByID(ctx, strategyID)
if err != nil {
logger.Error("Failed to get strategy by ID",
zap.Error(err),
zap.String("strategy_id", strategyID.String()))
return nil, fmt.Errorf("%w: %v", ErrStrategyNotFound, err)
}
return strategy, nil
}

// UpdateStrategyStatus 更新策略状态
func (s *StrategyService) UpdateStrategyStatus(ctx context.Context, strategyID uuid.UUID, status string) error {
validStatuses := map[string]bool{
model.StrategyStatusActive:  true,
model.StrategyStatusPaused:  true,
model.StrategyStatusStopped: true,
}
if !validStatuses[status] {
return fmt.Errorf("invalid strategy status: %s", status)
}
if err := s.strategyRepo.UpdateStatus(ctx, strategyID, status); err != nil {
logger.Error("Failed to update strategy status",
zap.Error(err),
zap.String("strategy_id", strategyID.String()),
zap.String("status", status))
return fmt.Errorf("failed to update status: %w", err)
}
return nil
}

// UpdateStrategy 更新策略
func (s *StrategyService) UpdateStrategy(ctx context.Context, strategy *model.Strategy) error {
if strategy == nil {
return ErrInvalidStrategyInput
}
if err := s.strategyRepo.Update(ctx, strategy); err != nil {
logger.Error("Failed to update strategy",
zap.Error(err),
zap.String("strategy_id", strategy.ID.String()))
return fmt.Errorf("failed to update strategy: %w", err)
}
return nil
}

// DeleteStrategy 删除策略
func (s *StrategyService) DeleteStrategy(ctx context.Context, strategyID uuid.UUID) error {
if err := s.strategyRepo.Delete(ctx, strategyID); err != nil {
logger.Error("Failed to delete strategy",
zap.Error(err),
zap.String("strategy_id", strategyID.String()))
return fmt.Errorf("failed to delete strategy: %w", err)
}
return nil
}

// GetActiveStrategies 获取用户的激活策略
func (s *StrategyService) GetActiveStrategies(ctx context.Context, userID uuid.UUID) ([]*model.Strategy, error) {
strategies, err := s.strategyRepo.GetActiveByUserID(ctx, userID)
if err != nil {
logger.Error("Failed to get active strategies",
zap.Error(err),
zap.String("user_id", userID.String()))
return nil, fmt.Errorf("failed to get active strategies: %w", err)
}
return strategies, nil
}

// GetAccountStrategies 获取账户的策略列表
func (s *StrategyService) GetAccountStrategies(ctx context.Context, accountID uuid.UUID) ([]*model.Strategy, error) {
strategies, err := s.strategyRepo.GetByAccountID(ctx, accountID)
if err != nil {
logger.Error("Failed to get account strategies",
zap.Error(err),
zap.String("account_id", accountID.String()))
return nil, fmt.Errorf("failed to get account strategies: %w", err)
}
return strategies, nil
}
