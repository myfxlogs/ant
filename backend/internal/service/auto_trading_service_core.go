package service

import (
	"errors"

	"anttrader/internal/connection"
	"anttrader/internal/event"
	"anttrader/internal/repository"
)

var (
	ErrAutoTradingDisabled    = errors.New("auto trading is disabled")
	ErrRiskLimitExceeded      = errors.New("risk limit exceeded")
	ErrMaxPositionsReached    = errors.New("maximum positions reached")
	ErrDailyLossLimitExceeded = errors.New("daily loss limit exceeded")
	ErrMaxDrawdownExceeded    = errors.New("maximum drawdown exceeded")
	ErrMaxLotSizeExceeded     = errors.New("maximum lot size exceeded")
)

type AutoTradingService struct {
	autoTradingRepo *repository.AutoTradingRepository
	strategyRepo    *repository.StrategyRepository
	accountRepo     *repository.AccountRepository
	engine          MatchingEngine
	gateway         *ExecutionGateway
	connMgr         *connection.ConnectionManager
	eventBus        *event.Bus
	riskEngine      *RiskEngine
}

func NewAutoTradingService(
	autoTradingRepo *repository.AutoTradingRepository,
	strategyRepo *repository.StrategyRepository,
	accountRepo *repository.AccountRepository,
	engine MatchingEngine,
	gateway *ExecutionGateway,
	connMgr *connection.ConnectionManager,
) *AutoTradingService {
	return &AutoTradingService{
		autoTradingRepo: autoTradingRepo,
		strategyRepo:    strategyRepo,
		accountRepo:     accountRepo,
		engine:          engine,
		gateway:         gateway,
		connMgr:         connMgr,
		eventBus:        event.GetBus(),
		riskEngine:      NewRiskEngine(),
	}
}

func (s *AutoTradingService) GetTradingService() MatchingEngine {
	return s.engine
}
