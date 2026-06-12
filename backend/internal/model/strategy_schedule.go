package model

import (
	"github.com/shopspring/decimal"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
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

func (s *StrategySchedule) GetParameters() (map[string]interface{}, error) {
	if len(s.Parameters) == 0 {
		return make(map[string]interface{}), nil
	}
	var params map[string]interface{}
	err := json.Unmarshal(s.Parameters, &params)
	return params, err
}

func (s *StrategySchedule) SetParameters(params map[string]interface{}) error {
	data, err := json.Marshal(params)
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
	var metrics BacktestMetrics
	err := json.Unmarshal(s.BacktestMetrics, &metrics)
	return &metrics, err
}

func (s *StrategySchedule) SetBacktestMetrics(metrics *BacktestMetrics) error {
	if metrics == nil {
		s.BacktestMetrics = nil
		return nil
	}
	data, err := json.Marshal(metrics)
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
	var reasons []string
	err := json.Unmarshal(s.RiskReasons, &reasons)
	return reasons, err
}

func (s *StrategySchedule) SetRiskReasons(reasons []string) error {
	data, err := json.Marshal(reasons)
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
	var warnings []string
	err := json.Unmarshal(s.RiskWarnings, &warnings)
	return warnings, err
}

func (s *StrategySchedule) SetRiskWarnings(warnings []string) error {
	data, err := json.Marshal(warnings)
	if err != nil {
		return fmt.Errorf("marshal risk warnings: %w", err)
	}
	s.RiskWarnings = data
	return nil
}

func (s *StrategySchedule) GetScheduleConfig() (map[string]interface{}, error) {
	if len(s.ScheduleConfig) == 0 {
		return make(map[string]interface{}), nil
	}
	var config map[string]interface{}
	err := json.Unmarshal(s.ScheduleConfig, &config)
	return config, err
}

func (s *StrategySchedule) SetScheduleConfig(config map[string]interface{}) error {
	data, err := json.Marshal(config)
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

// getInt64FromConfig extracts an int64 value from a decoded JSONB config map.
// Handles float64 (JSON default), int64 (proto), and bigint (frontend via BigInt(n)).
func getInt64FromConfig(cfg map[string]interface{}, key string) int64 {
	v, ok := cfg[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

// ComputeNextRunAtFromConfig is a standalone helper for computing next_run_at from raw fields.
// Used by both the model (StrategySchedule.ComputeNextRunAt) and the service layer (ScheduleRow).
func ComputeNextRunAtFromConfig(scheduleType string, scheduleConfig []byte) (time.Time, error) {
	var cfg map[string]interface{}
	if len(scheduleConfig) > 0 {
		if err := json.Unmarshal(scheduleConfig, &cfg); err != nil {
			return time.Time{}, fmt.Errorf("compute next_run_at: parse config: %w", err)
		}
	}
	if cfg == nil {
		cfg = make(map[string]interface{})
	}

	switch scheduleType {
	case ScheduleTypeInterval:
		ms := getInt64FromConfig(cfg, "intervalMs")
		if ms == 0 {
			ms = getInt64FromConfig(cfg, "interval_seconds") * 1000 // legacy
		}
		if ms < 1000 {
			ms = 3600_000 // default: 1 hour
		}
		return time.Now().Add(time.Duration(ms) * time.Millisecond), nil

	case ScheduleTypeCron:
		// Backward compat: old records mapped kline_close/hf_quote as "cron" with triggerMode.
		if triggerMode, _ := cfg["triggerMode"].(string); triggerMode == "stable_kline" || triggerMode == "hf_quote_stream" {
			return time.Time{}, nil // event-driven, no next_run_at
		}
		// Cron expression — for now fallback to interval mode.
		ms := getInt64FromConfig(cfg, "intervalMs")
		if ms == 0 {
			ms = getInt64FromConfig(cfg, "interval_seconds") * 1000
		}
		if ms < 1000 {
			ms = 3600_000
		}
		return time.Now().Add(time.Duration(ms) * time.Millisecond), nil

	case ScheduleTypeEvent:
		return time.Time{}, nil // event-driven: kline_close / hf_quote

	default:
		return time.Time{}, fmt.Errorf("unknown schedule_type: %s", scheduleType)
	}
}

func NewStrategySchedule(userID, templateID, accountID uuid.UUID, symbol, timeframe string) *StrategySchedule {
	now := time.Now()
	return &StrategySchedule{
		ID:             uuid.New(),
		UserID:         userID,
		TemplateID:     templateID,
		AccountID:      accountID,
		Symbol:         symbol,
		Timeframe:      timeframe,
		Parameters:     []byte("{}"),
		ScheduleType:   ScheduleTypeInterval,
		ScheduleConfig: []byte(`{"interval_seconds": 3600}`),
		RiskReasons:    []byte("[]"),
		RiskWarnings:   []byte("[]"),
		RiskLevel:      RiskLevelUnknown,
		IsActive:       false,
		RunCount:       0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
