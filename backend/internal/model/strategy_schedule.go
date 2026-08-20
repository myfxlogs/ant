package model

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

type StrategySchedule struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	TemplateID      uuid.UUID  `json:"template_id" db:"template_id"`
	AccountID       uuid.UUID  `json:"account_id" db:"account_id"`
	Name            string     `json:"name" db:"name"`
	Symbol          string     `json:"symbol" db:"symbol"`
	Timeframe       string     `json:"timeframe" db:"timeframe"`
	Parameters      JSONB      `json:"parameters" db:"parameters"`
	ScheduleType    string     `json:"schedule_type" db:"schedule_type"`
	ScheduleConfig  JSONB      `json:"schedule_config" db:"schedule_config"`
	BacktestMetrics JSONB      `json:"backtest_metrics" db:"backtest_metrics"`
	RiskScore       *int       `json:"risk_score" db:"risk_score"`
	RiskLevel       string     `json:"risk_level" db:"risk_level"`
	RiskReasons     JSONB      `json:"risk_reasons" db:"risk_reasons"`
	RiskWarnings    JSONB      `json:"risk_warnings" db:"risk_warnings"`
	LastBacktestAt  *time.Time `json:"last_backtest_at" db:"last_backtest_at"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	LastRunAt       *time.Time `json:"last_run_at" db:"last_run_at"`
	NextRunAt       *time.Time `json:"next_run_at" db:"next_run_at"`
	RunCount        int        `json:"run_count" db:"run_count"`
	LastError       string     `json:"last_error" db:"last_error"`
	EnableCount     int        `json:"enable_count" db:"enable_count"`
	MagicNumber     *int32     `json:"magic_number" db:"magic_number"`
	ManualRunCount  int        `json:"manual_run_count" db:"manual_run_count"`
	LastManualRunAt *time.Time `json:"last_manual_run_at" db:"last_manual_run_at"`
	LastManualError string     `json:"last_manual_error" db:"last_manual_error"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type BacktestMetrics struct {
	TotalReturn   decimal.Decimal `json:"total_return"`
	AnnualReturn  decimal.Decimal `json:"annual_return"`
	MaxDrawdown   decimal.Decimal `json:"max_drawdown"`
	SharpeRatio   decimal.Decimal `json:"sharpe_ratio"`
	WinRate       decimal.Decimal `json:"win_rate"`
	ProfitFactor  decimal.Decimal `json:"profit_factor"`
	TotalTrades   int             `json:"total_trades"`
	WinningTrades int             `json:"winning_trades"`
	LosingTrades  int             `json:"losing_trades"`
	AverageProfit decimal.Decimal `json:"average_profit"`
	AverageLoss   decimal.Decimal `json:"average_loss"`
}

type RiskAssessment struct {
	Score      int      `json:"score"`
	Level      string   `json:"level"`
	Reasons    []string `json:"reasons"`
	Warnings   []string `json:"warnings"`
	IsReliable bool     `json:"is_reliable"`
}

const (
	RiskLevelLow     = "low"
	RiskLevelMedium  = "medium"
	RiskLevelHigh    = "high"
	RiskLevelUnknown = "unknown"
)

func (s *StrategySchedule) GetParameters() (map[string]string, error) {
	if len(s.Parameters) == 0 {
		return make(map[string]string), nil
	}
	var params antv1.StrategyParams
	if err := proto.Unmarshal(s.Parameters, &params); err != nil {
		return make(map[string]string), nil
	}
	return params.GetValues(), nil
}

func (s *StrategySchedule) SetParameters(params map[string]string) error {
	data, err := proto.Marshal(&antv1.StrategyParams{Values: params})
	if err != nil {
		return fmt.Errorf("marshal strategy schedule parameters: %w", err)
	}
	s.Parameters = data
	return nil
}

func (s *StrategySchedule) GetBacktestMetrics() (*BacktestMetrics, error) {
	if len(s.BacktestMetrics) == 0 {
		return nil, nil
	}
	var m antv1.BacktestMetrics
	if err := proto.Unmarshal(s.BacktestMetrics, &m); err != nil {
		return nil, err
	}
	return &BacktestMetrics{
		TotalReturn:   safeDecimal(m.GetTotalReturn()),
		AnnualReturn:  safeDecimal(m.GetAnnualReturn()),
		MaxDrawdown:   safeDecimal(m.GetMaxDrawdown()),
		SharpeRatio:   safeDecimal(m.GetSharpeRatio()),
		WinRate:       safeDecimal(m.GetWinRate()),
		ProfitFactor:  safeDecimal(m.GetProfitFactor()),
		TotalTrades:   int(m.GetTotalTrades()),
		WinningTrades: int(m.GetWinningTrades()),
		LosingTrades:  int(m.GetLosingTrades()),
		AverageProfit: safeDecimal(m.GetAverageProfit()),
		AverageLoss:   safeDecimal(m.GetAverageLoss()),
	}, nil
}

func safeDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func (s *StrategySchedule) SetBacktestMetrics(metrics *BacktestMetrics) error {
	if metrics == nil {
		s.BacktestMetrics = nil
		return nil
	}
	data, err := proto.Marshal(&antv1.BacktestMetrics{
		TotalReturn:   metrics.TotalReturn.String(),
		AnnualReturn:  metrics.AnnualReturn.String(),
		MaxDrawdown:   metrics.MaxDrawdown.String(),
		SharpeRatio:   metrics.SharpeRatio.String(),
		WinRate:       metrics.WinRate.String(),
		ProfitFactor:  metrics.ProfitFactor.String(),
		TotalTrades:   int32(metrics.TotalTrades),
		WinningTrades: int32(metrics.WinningTrades),
		LosingTrades:  int32(metrics.LosingTrades),
		AverageProfit: metrics.AverageProfit.String(),
		AverageLoss:   metrics.AverageLoss.String(),
	})
	if err != nil {
		return fmt.Errorf("marshal backtest metrics: %w", err)
	}
	s.BacktestMetrics = data
	return nil
}

func (s *StrategySchedule) GetRiskReasons() ([]string, error) {
	if len(s.RiskReasons) == 0 {
		return []string{}, nil
	}
	var lst antv1.BacktestRisk
	if err := proto.Unmarshal(s.RiskReasons, &lst); err != nil {
		return []string{}, nil
	}
	return lst.GetReasons(), nil
}

func (s *StrategySchedule) SetRiskReasons(reasons []string) error {
	data, err := proto.Marshal(&antv1.BacktestRisk{Reasons: reasons})
	if err != nil {
		return fmt.Errorf("marshal risk reasons: %w", err)
	}
	s.RiskReasons = data
	return nil
}

func (s *StrategySchedule) GetRiskWarnings() ([]string, error) {
	if len(s.RiskWarnings) == 0 {
		return []string{}, nil
	}
	var lst antv1.BacktestRisk
	if err := proto.Unmarshal(s.RiskWarnings, &lst); err != nil {
		return []string{}, nil
	}
	return lst.GetWarnings(), nil
}

func (s *StrategySchedule) SetRiskWarnings(warnings []string) error {
	data, err := proto.Marshal(&antv1.BacktestRisk{Warnings: warnings})
	if err != nil {
		return fmt.Errorf("marshal risk warnings: %w", err)
	}
	s.RiskWarnings = data
	return nil
}

func (s *StrategySchedule) GetScheduleConfig() (*antv1.ScheduleConfig, error) {
	if len(s.ScheduleConfig) == 0 {
		return &antv1.ScheduleConfig{}, nil
	}
	var cfg antv1.ScheduleConfig
	if err := proto.Unmarshal(s.ScheduleConfig, &cfg); err != nil {
		return &antv1.ScheduleConfig{}, nil
	}
	return &cfg, nil
}

func (s *StrategySchedule) SetScheduleConfig(cfg *antv1.ScheduleConfig) error {
	data, err := proto.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal schedule config: %w", err)
	}
	s.ScheduleConfig = data
	return nil
}

// ComputeNextRunAt returns the next execution time based on schedule_type and schedule_config.
// Returns zero time for event-driven schedules (kline_close, hf_quote) that don't use timer-based firing.
func (s *StrategySchedule) ComputeNextRunAt() (time.Time, error) {
	return ComputeNextRunAtFromConfig(s.ScheduleType, s.ScheduleConfig)
}

// ComputeNextRunAtFromConfig is a standalone helper for computing next_run_at from raw fields.
// Used by both the model (StrategySchedule.ComputeNextRunAt) and the service layer (ScheduleRow).
// Accepts proto-encoded ScheduleConfig bytes. Wraps ComputeNextRunAtFromConfigAt with time.Now.
func ComputeNextRunAtFromConfig(scheduleType string, scheduleConfig []byte) (time.Time, error) {
	return ComputeNextRunAtFromConfigAt(scheduleType, scheduleConfig, time.Now())
}

// ComputeNextRunAtFromConfigAt is the deterministic core: computes the next run time
// strictly after `now` based on schedule_type and config. Pure function — no time.Now,
// no side effects. Used by ScheduleEngine with an injectable now() for testability.
// Returns zero time for event-driven schedules (kline_close, hf_quote) that don't use
// timer-based firing. Returns error for unknown schedule_type or unparseable config.
func ComputeNextRunAtFromConfigAt(scheduleType string, scheduleConfig []byte, now time.Time) (time.Time, error) {
	var cfg antv1.ScheduleConfig
	if len(scheduleConfig) > 0 {
		if err := proto.Unmarshal(scheduleConfig, &cfg); err != nil {
			return time.Time{}, fmt.Errorf("compute next_run_at: parse config: %w", err)
		}
	}

	switch scheduleType {
	case ScheduleTypeInterval:
		ms := cfg.GetIntervalMs()
		if ms == 0 {
			ms = 3600_000 // default: 1 hour
		}
		if ms < 1000 {
			ms = 3600_000
		}
		return now.Add(time.Duration(ms) * time.Millisecond), nil

	case ScheduleTypeCron:
		// Backward compat: old records mapped kline_close/hf_quote as "cron" with triggerMode.
		triggerMode := cfg.GetTriggerMode()
		if triggerMode == "stable_kline" || triggerMode == "hf_quote_stream" {
			return time.Time{}, nil // event-driven, no next_run_at
		}
		// Cron expression — for now fallback to interval mode.
		ms := cfg.GetIntervalMs()
		if ms == 0 {
			ms = 3600_000
		}
		if ms < 1000 {
			ms = 3600_000
		}
		return now.Add(time.Duration(ms) * time.Millisecond), nil

	case ScheduleTypeEvent:
		return time.Time{}, nil // event-driven: kline_close / hf_quote

	default:
		return time.Time{}, fmt.Errorf("unknown schedule_type: %s", scheduleType)
	}
}

// StrategyMagic derives a deterministic 32-bit magic number from a ScheduleID.
// This allows multiple strategies on the same account to attribute positions
// correctly — each strategy's orders carry a unique magic, and dispatchCloseAll
// filters by magic to avoid cross-strategy position interference.
// Returns 0 when ScheduleID is zero (backward compat for callers that don't set it).
func StrategyMagic(scheduleID uuid.UUID) int32 {
	if scheduleID == uuid.Nil {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write(scheduleID[:])
	return int32(h.Sum32())
}

func NewStrategySchedule(userID, templateID, accountID uuid.UUID, symbol, timeframe string) *StrategySchedule {
	now := time.Now()
	defaultParams, _ := proto.Marshal(&antv1.StrategyParams{Values: map[string]string{}})
	defaultCfg, _ := proto.Marshal(&antv1.ScheduleConfig{IntervalMs: 3600_000})
	defaultReasons, _ := proto.Marshal(&antv1.BacktestRisk{})
	return &StrategySchedule{
		ID:             uuid.New(),
		UserID:         userID,
		TemplateID:     templateID,
		AccountID:      accountID,
		Symbol:         symbol,
		Timeframe:      timeframe,
		Parameters:     defaultParams,
		ScheduleType:   ScheduleTypeInterval,
		ScheduleConfig: defaultCfg,
		RiskReasons:    defaultReasons,
		RiskWarnings:   defaultReasons,
		RiskLevel:      RiskLevelUnknown,
		IsActive:       false,
		RunCount:       0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
