package connect

import (
	"anttrader/internal/service"
)

type AutoTradingService struct {
	autoTradingSvc *service.AutoTradingService
	schedule       *service.StrategyScheduleService
}

func NewAutoTradingService(autoTradingSvc *service.AutoTradingService, schedule *service.StrategyScheduleService) *AutoTradingService {
	return &AutoTradingService{
		autoTradingSvc: autoTradingSvc,
		schedule:       schedule,
	}
}
