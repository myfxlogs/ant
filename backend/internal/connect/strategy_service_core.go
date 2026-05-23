package connect

import (
	"anttrader/internal/service"
)

type StrategyService struct {
	strategySvc *service.StrategyService
	schedule    *service.StrategyScheduleService
	templateSvc *service.StrategyTemplateService
	pythonSvc   *service.PythonStrategyService
	logSvc      *service.LogService
}

func NewStrategyService(strategySvc *service.StrategyService, schedule *service.StrategyScheduleService, templateSvc *service.StrategyTemplateService, pythonSvc *service.PythonStrategyService, logSvc *service.LogService) *StrategyService {
	return &StrategyService{
		strategySvc: strategySvc,
		schedule:    schedule,
		templateSvc: templateSvc,
		pythonSvc:   pythonSvc,
		logSvc:      logSvc,
	}
}
