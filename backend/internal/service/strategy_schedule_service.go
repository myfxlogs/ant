package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"
)

var (
	ErrScheduleNameEmpty      = errors.New("schedule name cannot be empty")
	ErrScheduleSymbolEmpty    = errors.New("schedule symbol cannot be empty")
	ErrTemplateNotFound       = errors.New("template not found")
	ErrBacktestFailed         = errors.New("backtest failed")
	ErrScheduleAlreadyExists  = errors.New("schedule already exists")
	ErrTemplateNotPublished   = errors.New("template not published")
	ErrActiveScheduleConflict = errors.New("another active schedule already exists for this account/strategy")
)

type StrategyScheduleService struct {
	scheduleRepo *repository.StrategyScheduleRepository
	templateRepo *repository.StrategyTemplateRepository
	accountRepo  *repository.AccountRepository
	dynamicCfg   *DynamicConfigService
	pythonSvc    *PythonStrategyService
	klineSvc     *KlineService
	datasetSvc   *BacktestDatasetService
	backtestRun  *BacktestRunService
}

func NewStrategyScheduleService(
	scheduleRepo *repository.StrategyScheduleRepository,
	templateRepo *repository.StrategyTemplateRepository,
	accountRepo *repository.AccountRepository,
	dynamicCfg *DynamicConfigService,
	pythonSvc *PythonStrategyService,
	klineSvc *KlineService,
	datasetSvc *BacktestDatasetService,
	backtestRun *BacktestRunService,
) *StrategyScheduleService {
	return &StrategyScheduleService{
		scheduleRepo: scheduleRepo,
		templateRepo: templateRepo,
		accountRepo:  accountRepo,
		dynamicCfg:   dynamicCfg,
		pythonSvc:    pythonSvc,
		klineSvc:     klineSvc,
		datasetSvc:   datasetSvc,
		backtestRun:  backtestRun,
	}
}

func (s *StrategyScheduleService) CreateSchedule(ctx context.Context, userID uuid.UUID, req *CreateScheduleRequest) (*model.StrategySchedule, error) {
	tpl, err := s.templateRepo.GetByID(ctx, req.TemplateID)
	if err != nil {
		return nil, ErrTemplateNotFound
	}
	if tpl.Status != model.StrategyTemplateStatusPublished {
		return nil, ErrTemplateNotPublished
	}

	account, err := s.accountRepo.GetByID(ctx, req.AccountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	if account.UserID != userID {
		return nil, errors.New("unauthorized access to account")
	}

	if s.scheduleRepo != nil {
		if existing, e := s.scheduleRepo.GetByUniqueKey(ctx, userID, req.AccountID, req.TemplateID, req.Symbol, req.Timeframe); e == nil && existing != nil {
			return nil, ErrScheduleAlreadyExists
		}
	}

	schedule := model.NewStrategySchedule(userID, req.TemplateID, req.AccountID, req.Symbol, req.Timeframe)
	schedule.Name = req.Name
	schedule.ScheduleType = req.ScheduleType

	// Creating a schedule does not automatically enable it.
	schedule.IsActive = false
	schedule.EnableCount = 0

	if err := schedule.SetScheduleConfig(req.ScheduleConfig); err != nil {
		return nil, err
	}

	if err := schedule.SetParameters(req.Parameters); err != nil {
		return nil, err
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		logger.Error("Failed to create schedule", zap.Error(err))
		return nil, err
	}

	if err := s.templateRepo.IncrementUseCount(ctx, req.TemplateID); err != nil {
		logger.Warn("Failed to increment template use count", zap.Error(err))
	}

	return schedule, nil
}

func (s *StrategyScheduleService) GetSchedule(ctx context.Context, scheduleID uuid.UUID) (*model.StrategySchedule, error) {
	return s.scheduleRepo.GetByID(ctx, scheduleID)
}

func (s *StrategyScheduleService) GetSchedulesByUser(ctx context.Context, userID uuid.UUID) ([]*model.StrategySchedule, error) {
	return s.scheduleRepo.GetByUserID(ctx, userID)
}

func (s *StrategyScheduleService) GetSchedulesByTemplate(ctx context.Context, templateID uuid.UUID) ([]*model.StrategySchedule, error) {
	return s.scheduleRepo.GetByTemplateID(ctx, templateID)
}

func (s *StrategyScheduleService) UpdateSchedule(ctx context.Context, schedule *model.StrategySchedule) error {
	return s.scheduleRepo.Update(ctx, schedule)
}

func (s *StrategyScheduleService) DeleteSchedule(ctx context.Context, scheduleID uuid.UUID) error {
	return s.scheduleRepo.Delete(ctx, scheduleID)
}

func (s *StrategyScheduleService) SetScheduleActive(ctx context.Context, scheduleID uuid.UUID, active bool) error {
	return s.scheduleRepo.SetActive(ctx, scheduleID, active)
}

func (s *StrategyScheduleService) ToggleSchedule(ctx context.Context, userID, scheduleID uuid.UUID, active bool) (*model.StrategySchedule, error) {
	item, err := s.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return nil, err
	}
	if item.UserID != userID {
		return nil, errors.New("unauthorized access to schedule")
	}

	item.IsActive = active
	if active {
		existing, e := s.scheduleRepo.GetByUserID(ctx, userID)
		if e == nil {
			for _, it := range existing {
				if it == nil || it.ID == item.ID || !it.IsActive {
					continue
				}
				if it.AccountID == item.AccountID && it.TemplateID == item.TemplateID && it.Symbol == item.Symbol && it.Timeframe == item.Timeframe {
					return nil, ErrActiveScheduleConflict
				}
			}
		}
		_ = setNextRunAtForInterval(item)
	} else {
		item.NextRunAt = nil
	}

	if err := s.scheduleRepo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func setNextRunAtForInterval(sch *model.StrategySchedule) error {
	if sch == nil {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(sch.ScheduleType), model.ScheduleTypeEvent) {
		sch.NextRunAt = nil
		return nil
	}
	conf, err := sch.GetScheduleConfig()
	if err != nil {
		return err
	}
	intervalMs := toInt64Schedule(conf["stable_override_interval_ms"])
	if intervalMs <= 0 {
		intervalMs = toInt64Schedule(conf["interval_ms"])
	}
	if intervalMs <= 0 {
		next := time.Now().Add(timeframeToDurationForScheduleService(sch.Timeframe))
		sch.NextRunAt = &next
		return nil
	}
	next := time.Now().Add(time.Duration(intervalMs) * time.Millisecond)
	sch.NextRunAt = &next
	return nil
}

func toInt64Schedule(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		return 0
	}
}

func timeframeToDurationForScheduleService(tf string) time.Duration {
	switch strings.ToUpper(strings.TrimSpace(tf)) {
	case "M1":
		return 1 * time.Minute
	case "M5":
		return 5 * time.Minute
	case "M15":
		return 15 * time.Minute
	case "M30":
		return 30 * time.Minute
	case "H1":
		return 1 * time.Hour
	case "H4":
		return 4 * time.Hour
	case "D1":
		return 24 * time.Hour
	default:
		return 5 * time.Minute
	}
}

func (s *StrategyScheduleService) RunBacktest(ctx context.Context, userID uuid.UUID, req *RunBacktestRequest) (*BacktestWithRiskResponse, error) {
	ctx = WithEngineMode(ctx, EngineModeBacktest)

	template, err := s.templateRepo.GetByID(ctx, req.TemplateID)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	account, err := s.accountRepo.GetByID(ctx, req.AccountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	if account.UserID != userID {
		return nil, errors.New("unauthorized access to account")
	}

	var (
		klines    []*KlineResponse
		datasetID string
		costPtr   *BacktestCostModel
	)
	if req.DatasetID != "" {
		if s.datasetSvc == nil {
			return nil, errors.New("dataset service not available")
		}
		dsid, err := uuid.Parse(req.DatasetID)
		if err != nil {
			return nil, errors.New("invalid dataset_id")
		}
		klines, err = s.datasetSvc.GetFrozenDatasetKlines(ctx, userID, dsid, 0)
		if err != nil {
			return nil, err
		}
		if c, ok, _ := s.datasetSvc.GetFrozenDatasetCostModel(ctx, userID, dsid); ok {
			costPtr = c
		}
		datasetID = dsid.String()
	} else {
		klines, err = s.klineSvc.GetKlines(ctx, userID, req.AccountID, &KlineRequest{
			Symbol:    req.Symbol,
			Timeframe: req.Timeframe,
			Count:     500,
		})
		if err != nil {
			return nil, err
		}
		if s.datasetSvc != nil {
			cost := ResolveBacktestCostModel(ctx, s.dynamicCfg)
			dsid, err := s.datasetSvc.CreateFrozenDatasetFromKlines(ctx, userID, req.AccountID, req.Symbol, req.Timeframe, nil, nil, 500, klines, &cost)
			if err == nil {
				datasetID = dsid.String()
				costPtr = &cost
			}
		}
	}

	if len(klines) < 10 {
		return nil, errors.New("insufficient kline data for backtest")
	}

	paramsJSON, _ := json.Marshal(req.Parameters)
	var params map[string]interface{}
	json.Unmarshal(paramsJSON, &params)

	if costPtr == nil {
		cost := ResolveBacktestCostModel(ctx, s.dynamicCfg)
		costPtr = &cost
	}
	backtestResp, err := s.pythonSvc.RunBacktest(
		ctx,
		req.TemplateID,
		template.Code,
		klines,
		nil,
		req.Symbol,
		req.Timeframe,
		req.InitialCapital,
		costPtr,
		nil,
	)
	if err != nil {
		return nil, err
	}

	if !backtestResp.Success {
		return nil, errors.New(backtestResp.Error)
	}

	if s.backtestRun != nil {
		var dsUUID *uuid.UUID
		if datasetID != "" {
			if id, perr := uuid.Parse(datasetID); perr == nil {
				dsUUID = &id
			}
		}
		_, _ = s.backtestRun.Record(ctx, &BacktestRunRecordRequest{
			UserID:       userID,
			AccountID:    req.AccountID,
			Symbol:       req.Symbol,
			Timeframe:    req.Timeframe,
			DatasetID:    dsUUID,
			StrategyCode: template.Code,
			CostModel:    costPtr,
			Metrics:      backtestResp.Metrics,
			EquityCurve:  backtestResp.EquityCurve,
		})
	}

	return &BacktestWithRiskResponse{
		Success:      true,
		Metrics:      backtestResp.Metrics,
		RiskScore:    backtestResp.RiskAssessment.Score,
		RiskLevel:    backtestResp.RiskAssessment.Level,
		RiskReasons:  backtestResp.RiskAssessment.Reasons,
		RiskWarnings: backtestResp.RiskAssessment.Warnings,
		IsReliable:   backtestResp.RiskAssessment.IsReliable,
		DatasetID:    datasetID,
	}, nil
}

func (s *StrategyScheduleService) UpdateRiskAssessment(ctx context.Context, scheduleID uuid.UUID) error {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return err
	}

	template, err := s.templateRepo.GetByID(ctx, schedule.TemplateID)
	if err != nil {
		return err
	}

	klines, err := s.klineSvc.GetKlines(ctx, schedule.UserID, schedule.AccountID, &KlineRequest{
		Symbol:    schedule.Symbol,
		Timeframe: schedule.Timeframe,
		Count:     500,
	})
	if err != nil {
		return err
	}

	cost := ResolveBacktestCostModel(ctx, s.dynamicCfg)
	backtestResp, err := s.pythonSvc.RunBacktest(
		ctx,
		schedule.TemplateID,
		template.Code,
		klines,
		nil,
		schedule.Symbol,
		schedule.Timeframe,
		10000.0,
		&cost,
		nil,
	)
	if err != nil {
		return err
	}

	if !backtestResp.Success {
		return ErrBacktestFailed
	}

	metrics := &model.BacktestMetrics{
		TotalReturn:   backtestResp.Metrics.TotalReturn,
		AnnualReturn:  backtestResp.Metrics.AnnualReturn,
		MaxDrawdown:   backtestResp.Metrics.MaxDrawdown,
		SharpeRatio:   backtestResp.Metrics.SharpeRatio,
		WinRate:       backtestResp.Metrics.WinRate,
		ProfitFactor:  backtestResp.Metrics.ProfitFactor,
		TotalTrades:   backtestResp.Metrics.TotalTrades,
		WinningTrades: backtestResp.Metrics.WinningTrades,
		LosingTrades:  backtestResp.Metrics.LosingTrades,
		AverageProfit: backtestResp.Metrics.AverageProfit,
		AverageLoss:   backtestResp.Metrics.AverageLoss,
	}

	assessment := &model.RiskAssessment{
		Score:      backtestResp.RiskAssessment.Score,
		Level:      backtestResp.RiskAssessment.Level,
		Reasons:    backtestResp.RiskAssessment.Reasons,
		Warnings:   backtestResp.RiskAssessment.Warnings,
		IsReliable: backtestResp.RiskAssessment.IsReliable,
	}

	return s.scheduleRepo.UpdateRiskAssessment(ctx, scheduleID, assessment, metrics)
}

func (s *StrategyScheduleService) GetDueSchedules(ctx context.Context) ([]*model.StrategySchedule, error) {
	return s.scheduleRepo.GetDueSchedules(ctx, time.Now())
}

type CreateScheduleRequest struct {
	TemplateID     uuid.UUID              `json:"template_id"`
	AccountID      uuid.UUID              `json:"account_id"`
	Name           string                 `json:"name"`
	Symbol         string                 `json:"symbol"`
	Timeframe      string                 `json:"timeframe"`
	Parameters     map[string]interface{} `json:"parameters"`
	ScheduleType   string                 `json:"schedule_type"`
	ScheduleConfig map[string]interface{} `json:"schedule_config"`
}

type RunBacktestRequest struct {
	TemplateID     uuid.UUID              `json:"template_id"`
	AccountID      uuid.UUID              `json:"account_id"`
	Symbol         string                 `json:"symbol"`
	Timeframe      string                 `json:"timeframe"`
	Parameters     map[string]interface{} `json:"parameters"`
	InitialCapital float64                `json:"initial_capital"`
	DatasetID      string                 `json:"dataset_id,omitempty"`
}

type BacktestWithRiskResponse struct {
	Success      bool                   `json:"success"`
	Metrics      *BacktestMetricsPython `json:"metrics"`
	RiskScore    int                    `json:"risk_score"`
	RiskLevel    string                 `json:"risk_level"`
	RiskReasons  []string               `json:"risk_reasons"`
	RiskWarnings []string               `json:"risk_warnings"`
	IsReliable   bool                   `json:"is_reliable"`
	Error        string                 `json:"error,omitempty"`
	DatasetID    string                 `json:"dataset_id,omitempty"`
}
